package btp

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	// DefaultCatalogueMaxBytes is the maximum BTP XML catalogue response size accepted by default.
	DefaultCatalogueMaxBytes int64 = 50 * 1024 * 1024
	// DefaultCatalogueMaxProducts is the maximum number of products parsed from one BTP XML catalogue.
	DefaultCatalogueMaxProducts = 50_000
)

var (
	// ErrCatalogueTooLarge is returned when a BTP XML catalogue exceeds the configured byte limit.
	ErrCatalogueTooLarge = errors.New("btp: catalogue XML exceeds size limit")
	// ErrCatalogueProductLimitExceeded is returned when a BTP XML catalogue has too many products.
	ErrCatalogueProductLimitExceeded = errors.New("btp: catalogue product limit exceeded")
)

// CatalogueParseOptions controls BTP XML catalogue parser limits.
type CatalogueParseOptions struct {
	// MaxBytes is the maximum number of bytes to read from the XML feed. Values <= 0 use DefaultCatalogueMaxBytes.
	MaxBytes int64
	// MaxProducts is the maximum number of products to parse. Values <= 0 use DefaultCatalogueMaxProducts.
	MaxProducts int
}

// CatalogueProduct represents a product parsed from a BTP XML catalogue feed.
type CatalogueProduct struct {
	EAN                  string
	SupplierItemCode     string
	ManufacturerItemCode string
	ItemDescription      string
	LongItemDescription  string
	ProductGroup         string
	PrimaryProductGroup  string
	BrandName            string
	Weight               float64
	UnitNetPrice         float64
	UnitRetailPrice      float64
	TaxRate              float64
	CustomsTariffNumber  string
	Guarantee            int
	Pictures             []string
	Attributes           map[string]string
}

// XML structures for parsing BTP catalogue feed (Comarch EDI format).

type xmlLine struct {
	Item xmlLineItem `xml:"Line-Item"`
}

type xmlLineItem struct {
	EAN                  string      `xml:"EAN"`
	SupplierItemCode     string      `xml:"SupplierItemCode"`
	ManufacturerItemCode string      `xml:"ManufacturerItemCode"`
	ItemDescription      string      `xml:"ItemDescription"`
	LongItemDescription  string      `xml:"LongItemDescription"`
	ProductGroup         string      `xml:"ProductGroup"`
	PrimaryProductGroup  string      `xml:"PrimaryProductGroup"`
	BrandName            string      `xml:"BrandName"`
	Weight               string      `xml:"Weight"`
	UnitNetPrice         string      `xml:"UnitNetPrice"`
	UnitRetailPrice      string      `xml:"UnitRetailPrice"`
	TaxRate              string      `xml:"TaxRate"`
	CustomsTariffNumber  string      `xml:"CustomsTariffNumber"`
	Guarantee            string      `xml:"Guarantee"`
	Pictures             xmlPictures `xml:"Pictures"`
	Specification        xmlSpec     `xml:"Specification"`
}

type xmlPictures struct {
	Pictures []xmlPicture `xml:"Picture"`
}

type xmlPicture struct {
	URL string `xml:"Url"`
}

type xmlSpec struct {
	Attributes []xmlAttribute `xml:"Attributes>Attribute"`
}

type xmlAttribute struct {
	Name   string   `xml:"Name"`
	Values []string `xml:"Values>Value"`
}

// ParseCatalogueXML parses a BTP XML catalogue feed from the given reader.
func ParseCatalogueXML(r io.Reader) ([]CatalogueProduct, error) {
	return ParseCatalogueXMLWithOptions(r, CatalogueParseOptions{})
}

// ParseCatalogueXMLWithOptions parses a BTP XML catalogue feed with explicit safety limits.
func ParseCatalogueXMLWithOptions(r io.Reader, opts CatalogueParseOptions) ([]CatalogueProduct, error) {
	opts = normalizeCatalogueParseOptions(opts)
	decoder := xml.NewDecoder(&byteLimitReader{
		r:         r,
		remaining: opts.MaxBytes + 1,
		err:       ErrCatalogueTooLarge,
	})

	products := make([]CatalogueProduct, 0)
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			return products, nil
		}
		if err != nil {
			return nil, wrapCatalogueDecodeError(err, opts.MaxBytes)
		}

		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "Line" {
			continue
		}
		if len(products) >= opts.MaxProducts {
			return nil, fmt.Errorf("%w: max %d products", ErrCatalogueProductLimitExceeded, opts.MaxProducts)
		}

		var line xmlLine
		if err := decoder.DecodeElement(&line, &start); err != nil {
			return nil, wrapCatalogueDecodeError(err, opts.MaxBytes)
		}
		products = append(products, catalogueProductFromLine(line))
	}
}

func catalogueProductFromLine(line xmlLine) CatalogueProduct {
	item := line.Item
	p := CatalogueProduct{
		EAN:                  strings.TrimSpace(item.EAN),
		SupplierItemCode:     strings.TrimSpace(item.SupplierItemCode),
		ManufacturerItemCode: strings.TrimSpace(item.ManufacturerItemCode),
		ItemDescription:      strings.TrimSpace(item.ItemDescription),
		LongItemDescription:  strings.TrimSpace(item.LongItemDescription),
		ProductGroup:         strings.TrimSpace(item.ProductGroup),
		PrimaryProductGroup:  strings.TrimSpace(item.PrimaryProductGroup),
		BrandName:            strings.TrimSpace(item.BrandName),
		CustomsTariffNumber:  strings.TrimSpace(item.CustomsTariffNumber),
		Attributes:           map[string]string{},
	}

	if v, err := strconv.ParseFloat(item.Weight, 64); err == nil {
		p.Weight = v
	}
	if v, err := strconv.ParseFloat(item.UnitNetPrice, 64); err == nil {
		p.UnitNetPrice = v
	}
	if v, err := strconv.ParseFloat(item.UnitRetailPrice, 64); err == nil {
		p.UnitRetailPrice = v
	}
	if v, err := strconv.ParseFloat(item.TaxRate, 64); err == nil {
		p.TaxRate = v
	}
	if v, err := strconv.Atoi(strings.TrimSpace(item.Guarantee)); err == nil {
		p.Guarantee = v
	}

	for _, pic := range item.Pictures.Pictures {
		url := strings.TrimSpace(pic.URL)
		if url != "" {
			p.Pictures = append(p.Pictures, url)
		}
	}

	for _, attr := range item.Specification.Attributes {
		name := strings.TrimSpace(attr.Name)
		if name == "" {
			continue
		}
		values := make([]string, 0, len(attr.Values))
		for _, v := range attr.Values {
			v = strings.TrimSpace(v)
			if v != "" {
				values = append(values, v)
			}
		}
		if len(values) > 0 {
			p.Attributes[name] = strings.Join(values, ", ")
		}
	}

	return p
}

// ParseCatalogueURL fetches and parses a BTP XML catalogue feed from the given URL.
// The provided http.Client should have SSRF protections (e.g. netutil.SafeHTTPClient).
// Additionally, this function validates the URL host against private IP ranges as defense-in-depth.
func ParseCatalogueURL(ctx context.Context, rawURL string, client *http.Client) ([]CatalogueProduct, error) {
	return ParseCatalogueURLWithOptions(ctx, rawURL, client, CatalogueParseOptions{})
}

// ParseCatalogueURLWithOptions fetches and parses a BTP XML catalogue feed with explicit safety limits.
func ParseCatalogueURLWithOptions(ctx context.Context, rawURL string, client *http.Client, opts CatalogueParseOptions) ([]CatalogueProduct, error) {
	opts = normalizeCatalogueParseOptions(opts)
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("btp: invalid catalogue URL: %w", err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, fmt.Errorf("btp: catalogue URL must use http or https scheme, got %q", parsed.Scheme)
	}
	if err := urlHostValidator(ctx, parsed.Hostname()); err != nil {
		return nil, fmt.Errorf("btp: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("btp: create catalogue request: %w", err)
	}

	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("btp: fetch catalogue URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("btp: catalogue URL returned status %d", resp.StatusCode)
	}
	if resp.ContentLength > opts.MaxBytes {
		return nil, fmt.Errorf("%w: content length %d exceeds max %d bytes", ErrCatalogueTooLarge, resp.ContentLength, opts.MaxBytes)
	}

	return ParseCatalogueXMLWithOptions(resp.Body, opts)
}

func normalizeCatalogueParseOptions(opts CatalogueParseOptions) CatalogueParseOptions {
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = DefaultCatalogueMaxBytes
	}
	if opts.MaxProducts <= 0 {
		opts.MaxProducts = DefaultCatalogueMaxProducts
	}
	return opts
}

func wrapCatalogueDecodeError(err error, maxBytes int64) error {
	if errors.Is(err, ErrCatalogueTooLarge) {
		return fmt.Errorf("%w: max %d bytes", ErrCatalogueTooLarge, maxBytes)
	}
	return fmt.Errorf("btp: decode catalogue XML: %w", err)
}

type byteLimitReader struct {
	r         io.Reader
	remaining int64
	err       error
}

func (r *byteLimitReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, r.err
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.r.Read(p)
	r.remaining -= int64(n)
	if r.remaining <= 0 && n > 0 {
		return n, r.err
	}
	return n, err
}

// privateCIDRs lists IP ranges that must not be targeted by catalogue fetches (SSRF protection).
var privateCIDRs = func() []*net.IPNet {
	cidrs := []string{
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"127.0.0.0/8", "169.254.0.0/16", "0.0.0.0/8",
		"100.64.0.0/10", "::1/128", "fc00::/7", "fe80::/10",
	}
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, ipNet, err := net.ParseCIDR(c)
		if err == nil {
			nets = append(nets, ipNet)
		}
	}
	return nets
}()

// urlHostValidator is the function used to validate URL hosts. It can be replaced in tests.
var urlHostValidator = validateURLHost

// validateURLHost resolves the hostname and rejects private/internal IP addresses.
func validateURLHost(ctx context.Context, host string) error {
	if host == "" {
		return fmt.Errorf("empty hostname")
	}
	ips, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return fmt.Errorf("DNS lookup failed for %q: %w", host, err)
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		for _, cidr := range privateCIDRs {
			if cidr.Contains(ip) {
				return fmt.Errorf("URL host %q resolves to private IP %s", host, ipStr)
			}
		}
	}
	return nil
}

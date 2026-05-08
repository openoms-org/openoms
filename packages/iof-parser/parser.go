package iof

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const (
	// DefaultMaxBytes is the maximum IOF XML response size accepted by default.
	DefaultMaxBytes int64 = 50 * 1024 * 1024
	// DefaultMaxProducts is the maximum number of products parsed from one IOF XML feed.
	DefaultMaxProducts = 50_000
)

var (
	// ErrFeedTooLarge is returned when an IOF feed exceeds the configured byte limit.
	ErrFeedTooLarge = errors.New("IOF feed exceeds size limit")
	// ErrProductLimitExceeded is returned when an IOF feed has too many products.
	ErrProductLimitExceeded = errors.New("IOF product limit exceeded")
)

// ParseOptions controls IOF parser limits.
type ParseOptions struct {
	// MaxBytes is the maximum number of bytes to read from the XML feed. Values <= 0 use DefaultMaxBytes.
	MaxBytes int64
	// MaxProducts is the maximum number of products to parse. Values <= 0 use DefaultMaxProducts.
	MaxProducts int
}

// Product represents a parsed product from an IOF XML feed.
type Product struct {
	ID          string
	Name        string
	EAN         string
	SKU         string
	Price       float64
	Stock       int
	Description string
	Category    string
	ImageURL    string
	Weight      float64
	Attributes  map[string]string
}

// IOF XML structures

type xmlProduct struct {
	ID           string         `xml:"id,attr"`
	CodeProducer string         `xml:"code_producer,attr"`
	Producer     xmlProducer    `xml:"producer"`
	Category     xmlCategory    `xml:"category"`
	Description  xmlDescription `xml:"description"`
	Price        xmlPrice       `xml:"price"`
	Sizes        xmlSizes       `xml:"sizes"`
	Images       xmlImages      `xml:"images"`
	Codes        []xmlCode      `xml:"code"`
	Attrs        xmlAttrs       `xml:"attrs"`
}

type xmlAttrs struct {
	Attrs []xmlAttr `xml:"a"`
}

type xmlAttr struct {
	Name   string   `xml:"name,attr"`
	Values []string `xml:"a_value"`
}

type xmlProducer struct {
	Name string `xml:"name,attr"`
}

type xmlCategory struct {
	Name string `xml:"name,attr"`
}

type xmlDescription struct {
	Name     string `xml:"name"`
	LongDesc string `xml:"long_desc"`
}

type xmlPrice struct {
	Gross string `xml:"gross,attr"`
	Net   string `xml:"net,attr"`
}

type xmlSizes struct {
	Sizes []xmlSize `xml:"size"`
}

type xmlSize struct {
	CodeProducer string   `xml:"code_producer,attr"`
	Stock        string   `xml:"stock,attr"`
	Weight       string   `xml:"weight,attr"`
	Price        xmlPrice `xml:"price"`
}

type xmlImages struct {
	Large []xmlImage `xml:"large"`
}

type xmlImage struct {
	URL string `xml:"url,attr"`
}

type xmlCode struct {
	Code     string `xml:"code,attr"`
	CodeType string `xml:"code_type,attr"`
}

// Parse reads an IOF XML feed from an io.Reader and returns products.
func Parse(r io.Reader) ([]Product, error) {
	return ParseWithOptions(r, ParseOptions{})
}

// ParseWithOptions reads an IOF XML feed with explicit safety limits.
func ParseWithOptions(r io.Reader, opts ParseOptions) ([]Product, error) {
	opts = normalizeParseOptions(opts)
	decoder := xml.NewDecoder(&byteLimitReader{
		r:         r,
		remaining: opts.MaxBytes + 1,
		err:       ErrFeedTooLarge,
	})

	products := make([]Product, 0)
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			return products, nil
		}
		if err != nil {
			return nil, wrapDecodeError(err, opts.MaxBytes)
		}

		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "product" {
			continue
		}
		if len(products) >= opts.MaxProducts {
			return nil, fmt.Errorf("%w: max %d products", ErrProductLimitExceeded, opts.MaxProducts)
		}

		var xp xmlProduct
		if err := decoder.DecodeElement(&xp, &start); err != nil {
			return nil, wrapDecodeError(err, opts.MaxBytes)
		}
		products = append(products, convertProduct(xp))
	}
}

// ParseURL fetches and parses an IOF XML feed from a URL.
// If client is nil, http.DefaultClient is used (not recommended — pass an SSRF-safe client).
func ParseURL(ctx context.Context, url string, client *http.Client) ([]Product, error) {
	return ParseURLWithOptions(ctx, url, client, ParseOptions{})
}

// ParseURLWithOptions fetches and parses an IOF XML feed with explicit safety limits.
func ParseURLWithOptions(ctx context.Context, url string, client *http.Client, opts ParseOptions) ([]Product, error) {
	opts = normalizeParseOptions(opts)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch IOF feed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IOF feed returned status %d", resp.StatusCode)
	}
	if resp.ContentLength > opts.MaxBytes {
		return nil, fmt.Errorf("%w: content length %d exceeds max %d bytes", ErrFeedTooLarge, resp.ContentLength, opts.MaxBytes)
	}

	return ParseWithOptions(resp.Body, opts)
}

func normalizeParseOptions(opts ParseOptions) ParseOptions {
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = DefaultMaxBytes
	}
	if opts.MaxProducts <= 0 {
		opts.MaxProducts = DefaultMaxProducts
	}
	return opts
}

func wrapDecodeError(err error, maxBytes int64) error {
	if errors.Is(err, ErrFeedTooLarge) {
		return fmt.Errorf("%w: max %d bytes", ErrFeedTooLarge, maxBytes)
	}
	return fmt.Errorf("decode IOF XML: %w", err)
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

func convertProduct(xp xmlProduct) Product {
	p := Product{
		ID:         xp.ID,
		SKU:        xp.CodeProducer,
		Name:       strings.TrimSpace(xp.Description.Name),
		Category:   xp.Category.Name,
		Attributes: make(map[string]string),
	}

	// Description
	if desc := strings.TrimSpace(xp.Description.LongDesc); desc != "" {
		p.Description = desc
	}

	// Price from top-level or first size
	if gross := xp.Price.Gross; gross != "" {
		p.Price, _ = strconv.ParseFloat(gross, 64)
	}

	// EAN from codes
	for _, c := range xp.Codes {
		if strings.EqualFold(c.CodeType, "EAN") {
			p.EAN = c.Code
			break
		}
	}

	// First image
	if len(xp.Images.Large) > 0 {
		p.ImageURL = xp.Images.Large[0].URL
	}

	// Producer as attribute
	if xp.Producer.Name != "" {
		p.Attributes["producer"] = xp.Producer.Name
	}

	// Product attributes from <attrs><a name="..."><a_value>...</a_value></a></attrs>
	for _, attr := range xp.Attrs.Attrs {
		name := strings.TrimSpace(attr.Name)
		if name == "" {
			continue
		}
		values := make([]string, 0, len(attr.Values))
		for _, v := range attr.Values {
			if v = strings.TrimSpace(v); v != "" {
				values = append(values, v)
			}
		}
		if len(values) > 0 {
			p.Attributes[name] = strings.Join(values, ", ")
		}
	}

	// Size data (stock, weight, price override)
	if len(xp.Sizes.Sizes) > 0 {
		size := xp.Sizes.Sizes[0]
		if stock := size.Stock; stock != "" {
			p.Stock, _ = strconv.Atoi(stock)
		}
		if weight := size.Weight; weight != "" {
			p.Weight, _ = strconv.ParseFloat(weight, 64)
		}
		// Size price overrides top-level price if present
		if gross := size.Price.Gross; gross != "" {
			p.Price, _ = strconv.ParseFloat(gross, 64)
		}
		if size.CodeProducer != "" && p.SKU == "" {
			p.SKU = size.CodeProducer
		}
	}

	return p
}

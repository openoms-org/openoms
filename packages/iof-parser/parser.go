package iof

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

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

type xmlOffer struct {
	XMLName  xml.Name     `xml:"offer"`
	Products xmlProducts  `xml:"products"`
}

type xmlProducts struct {
	Products []xmlProduct `xml:"product"`
}

type xmlProduct struct {
	ID           string          `xml:"id,attr"`
	CodeProducer string          `xml:"code_producer,attr"`
	Producer     xmlProducer     `xml:"producer"`
	Category     xmlCategory     `xml:"category"`
	Description  xmlDescription  `xml:"description"`
	Price        xmlPrice        `xml:"price"`
	Sizes        xmlSizes        `xml:"sizes"`
	Images       xmlImages       `xml:"images"`
	Codes        []xmlCode       `xml:"code"`
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
	var offer xmlOffer
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&offer); err != nil {
		return nil, fmt.Errorf("decode IOF XML: %w", err)
	}

	products := make([]Product, 0, len(offer.Products.Products))
	for _, xp := range offer.Products.Products {
		p := convertProduct(xp)
		products = append(products, p)
	}
	return products, nil
}

// ParseURL fetches and parses an IOF XML feed from a URL.
func ParseURL(ctx context.Context, url string) ([]Product, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch IOF feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IOF feed returned status %d", resp.StatusCode)
	}

	return Parse(resp.Body)
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

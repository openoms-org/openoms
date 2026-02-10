package iof

import (
	"strings"
	"testing"
)

const sampleIOF = `<?xml version="1.0" encoding="UTF-8"?>
<offer file_format="IOF" version="2.6">
  <products>
    <product id="P001" code_producer="SKU-PHONE-1">
      <producer name="TechBrand"/>
      <category name="Electronics > Phones"/>
      <description>
        <name>Smartfon TechBrand X100</name>
        <long_desc>Nowoczesny smartfon z ekranem AMOLED</long_desc>
      </description>
      <price gross="1299.99" net="1056.91"/>
      <sizes>
        <size code_producer="SKU-PHONE-1" stock="25" weight="0.185">
          <price gross="1299.99"/>
        </size>
      </sizes>
      <images>
        <large url="https://example.com/images/phone1.jpg"/>
      </images>
      <code code="5901234567890" code_type="EAN"/>
    </product>
    <product id="P002" code_producer="SKU-CASE-1">
      <producer name="AccessoryPro"/>
      <category name="Electronics > Accessories"/>
      <description>
        <name>Etui na telefon</name>
        <long_desc></long_desc>
      </description>
      <price gross="49.99" net="40.64"/>
      <sizes>
        <size code_producer="SKU-CASE-1" stock="100" weight="0.05">
          <price gross="49.99"/>
        </size>
      </sizes>
      <images>
        <large url="https://example.com/images/case1.jpg"/>
      </images>
      <code code="5901234567891" code_type="EAN"/>
    </product>
    <product id="P003" code_producer="">
      <producer name=""/>
      <category name=""/>
      <description>
        <name>Produkt minimalny</name>
      </description>
      <price gross="10.00" net="8.13"/>
      <sizes></sizes>
      <images></images>
    </product>
  </products>
</offer>`

func TestParse(t *testing.T) {
	products, err := Parse(strings.NewReader(sampleIOF))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(products) != 3 {
		t.Fatalf("expected 3 products, got %d", len(products))
	}

	// First product - full data
	p := products[0]
	if p.ID != "P001" {
		t.Errorf("product[0].ID = %q, want %q", p.ID, "P001")
	}
	if p.Name != "Smartfon TechBrand X100" {
		t.Errorf("product[0].Name = %q, want %q", p.Name, "Smartfon TechBrand X100")
	}
	if p.EAN != "5901234567890" {
		t.Errorf("product[0].EAN = %q, want %q", p.EAN, "5901234567890")
	}
	if p.SKU != "SKU-PHONE-1" {
		t.Errorf("product[0].SKU = %q, want %q", p.SKU, "SKU-PHONE-1")
	}
	if p.Price != 1299.99 {
		t.Errorf("product[0].Price = %f, want %f", p.Price, 1299.99)
	}
	if p.Stock != 25 {
		t.Errorf("product[0].Stock = %d, want %d", p.Stock, 25)
	}
	if p.Weight != 0.185 {
		t.Errorf("product[0].Weight = %f, want %f", p.Weight, 0.185)
	}
	if p.Category != "Electronics > Phones" {
		t.Errorf("product[0].Category = %q, want %q", p.Category, "Electronics > Phones")
	}
	if p.ImageURL != "https://example.com/images/phone1.jpg" {
		t.Errorf("product[0].ImageURL = %q, want %q", p.ImageURL, "https://example.com/images/phone1.jpg")
	}
	if p.Description != "Nowoczesny smartfon z ekranem AMOLED" {
		t.Errorf("product[0].Description = %q", p.Description)
	}
	if p.Attributes["producer"] != "TechBrand" {
		t.Errorf("product[0].Attributes[producer] = %q, want %q", p.Attributes["producer"], "TechBrand")
	}

	// Second product
	p2 := products[1]
	if p2.ID != "P002" {
		t.Errorf("product[1].ID = %q, want %q", p2.ID, "P002")
	}
	if p2.Stock != 100 {
		t.Errorf("product[1].Stock = %d, want %d", p2.Stock, 100)
	}

	// Third product - minimal data
	p3 := products[2]
	if p3.ID != "P003" {
		t.Errorf("product[2].ID = %q, want %q", p3.ID, "P003")
	}
	if p3.Name != "Produkt minimalny" {
		t.Errorf("product[2].Name = %q, want %q", p3.Name, "Produkt minimalny")
	}
	if p3.EAN != "" {
		t.Errorf("product[2].EAN should be empty, got %q", p3.EAN)
	}
	if p3.Stock != 0 {
		t.Errorf("product[2].Stock = %d, want 0", p3.Stock)
	}
	if p3.Price != 10.00 {
		t.Errorf("product[2].Price = %f, want %f", p3.Price, 10.00)
	}
}

func TestParseInvalidXML(t *testing.T) {
	_, err := Parse(strings.NewReader("<invalid"))
	if err == nil {
		t.Fatal("expected error for invalid XML")
	}
}

func TestParseEmptyOffer(t *testing.T) {
	xml := `<?xml version="1.0"?><offer><products></products></offer>`
	products, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(products) != 0 {
		t.Fatalf("expected 0 products, got %d", len(products))
	}
}

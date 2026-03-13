package btp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const sampleCatalogueXML = `<?xml version="1.0" encoding="UTF-8"?>
<Document-ProductCatalogue>
  <ProductCatalogue-Lines>
    <Line><Line-Item>
      <EAN>888488284408</EAN>
      <SupplierItemCode>SUP-001</SupplierItemCode>
      <ManufacturerItemCode>MFR-ZZ-001</ManufacturerItemCode>
      <ItemDescription><![CDATA[Etui Zizo Bolt iPhone X]]></ItemDescription>
      <LongItemDescription><![CDATA[<p>Pancerne etui <b>Zizo Bolt</b> dedykowane dla iPhone X.</p>]]></LongItemDescription>
      <ProductGroup>Telefony i akcesoria</ProductGroup>
      <PrimaryProductGroup>Elektronika</PrimaryProductGroup>
      <BrandName>Zizo</BrandName>
      <Weight>0.075</Weight>
      <UnitNetPrice>12.44</UnitNetPrice>
      <UnitRetailPrice>14.31</UnitRetailPrice>
      <TaxRate>23.0</TaxRate>
      <CustomsTariffNumber>3926909790</CustomsTariffNumber>
      <Guarantee>24</Guarantee>
      <Pictures>
        <Picture><Url>https://pub.btp.pro/Public/Resource/ItemPic2/img1.jpg</Url></Picture>
        <Picture><Url>https://pub.btp.pro/Public/Resource/ItemPic2/img2.jpg</Url></Picture>
        <Picture><Url>https://pub.btp.pro/Public/Resource/ItemPic2/img3.jpg</Url></Picture>
      </Pictures>
      <Specification><Attributes>
        <Attribute><Name>Compatibility</Name><Values><Value>Apple iPhone X</Value></Values></Attribute>
        <Attribute><Name>Color</Name><Values><Value>Black</Value><Value>Red</Value></Values></Attribute>
      </Attributes></Specification>
    </Line-Item></Line>
    <Line><Line-Item>
      <EAN>5901234567890</EAN>
      <SupplierItemCode>SUP-002</SupplierItemCode>
      <ManufacturerItemCode>MFR-KB-002</ManufacturerItemCode>
      <ItemDescription><![CDATA[Kabel USB-C 1m]]></ItemDescription>
      <LongItemDescription></LongItemDescription>
      <ProductGroup>Kable</ProductGroup>
      <PrimaryProductGroup>Akcesoria</PrimaryProductGroup>
      <BrandName>Baseus</BrandName>
      <Weight>0.032</Weight>
      <UnitNetPrice>5.50</UnitNetPrice>
      <UnitRetailPrice>6.77</UnitRetailPrice>
      <TaxRate>23.0</TaxRate>
      <CustomsTariffNumber>8544429090</CustomsTariffNumber>
      <Guarantee>12</Guarantee>
      <Pictures>
        <Picture><Url>https://pub.btp.pro/Public/Resource/ItemPic2/cable1.jpg</Url></Picture>
      </Pictures>
      <Specification><Attributes></Attributes></Specification>
    </Line-Item></Line>
  </ProductCatalogue-Lines>
</Document-ProductCatalogue>`

func TestParseCatalogueXML(t *testing.T) {
	products, err := ParseCatalogueXML(strings.NewReader(sampleCatalogueXML))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(products))
	}

	// First product — full data
	p := products[0]
	if p.EAN != "888488284408" {
		t.Errorf("expected EAN 888488284408, got %s", p.EAN)
	}
	if p.SupplierItemCode != "SUP-001" {
		t.Errorf("expected SupplierItemCode SUP-001, got %s", p.SupplierItemCode)
	}
	if p.ManufacturerItemCode != "MFR-ZZ-001" {
		t.Errorf("expected ManufacturerItemCode MFR-ZZ-001, got %s", p.ManufacturerItemCode)
	}
	if p.ItemDescription != "Etui Zizo Bolt iPhone X" {
		t.Errorf("unexpected ItemDescription: %s", p.ItemDescription)
	}
	if !strings.Contains(p.LongItemDescription, "Zizo Bolt") {
		t.Errorf("expected LongItemDescription to contain 'Zizo Bolt', got: %s", p.LongItemDescription)
	}
	if p.ProductGroup != "Telefony i akcesoria" {
		t.Errorf("expected ProductGroup 'Telefony i akcesoria', got %s", p.ProductGroup)
	}
	if p.PrimaryProductGroup != "Elektronika" {
		t.Errorf("expected PrimaryProductGroup 'Elektronika', got %s", p.PrimaryProductGroup)
	}
	if p.BrandName != "Zizo" {
		t.Errorf("expected BrandName 'Zizo', got %s", p.BrandName)
	}
	if p.Weight != 0.075 {
		t.Errorf("expected Weight 0.075, got %f", p.Weight)
	}
	if p.UnitNetPrice != 12.44 {
		t.Errorf("expected UnitNetPrice 12.44, got %f", p.UnitNetPrice)
	}
	if p.UnitRetailPrice != 14.31 {
		t.Errorf("expected UnitRetailPrice 14.31, got %f", p.UnitRetailPrice)
	}
	if p.TaxRate != 23.0 {
		t.Errorf("expected TaxRate 23.0, got %f", p.TaxRate)
	}
	if p.CustomsTariffNumber != "3926909790" {
		t.Errorf("expected CustomsTariffNumber 3926909790, got %s", p.CustomsTariffNumber)
	}
	if p.Guarantee != 24 {
		t.Errorf("expected Guarantee 24, got %d", p.Guarantee)
	}

	// Pictures
	if len(p.Pictures) != 3 {
		t.Fatalf("expected 3 pictures, got %d", len(p.Pictures))
	}
	if p.Pictures[0] != "https://pub.btp.pro/Public/Resource/ItemPic2/img1.jpg" {
		t.Errorf("unexpected picture[0]: %s", p.Pictures[0])
	}

	// Attributes
	if p.Attributes["Compatibility"] != "Apple iPhone X" {
		t.Errorf("unexpected Compatibility attribute: %s", p.Attributes["Compatibility"])
	}
	if p.Attributes["Color"] != "Black, Red" {
		t.Errorf("expected multi-value attribute 'Black, Red', got %s", p.Attributes["Color"])
	}

	// Second product — minimal data
	p2 := products[1]
	if p2.EAN != "5901234567890" {
		t.Errorf("expected EAN 5901234567890, got %s", p2.EAN)
	}
	if p2.LongItemDescription != "" {
		t.Errorf("expected empty LongItemDescription, got %s", p2.LongItemDescription)
	}
	if len(p2.Pictures) != 1 {
		t.Errorf("expected 1 picture, got %d", len(p2.Pictures))
	}
	if len(p2.Attributes) != 0 {
		t.Errorf("expected 0 attributes, got %d", len(p2.Attributes))
	}
	if p2.Guarantee != 12 {
		t.Errorf("expected Guarantee 12, got %d", p2.Guarantee)
	}
}

func TestParseCatalogueXML_Empty(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<Document-ProductCatalogue>
  <ProductCatalogue-Lines></ProductCatalogue-Lines>
</Document-ProductCatalogue>`

	products, err := ParseCatalogueXML(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 0 {
		t.Errorf("expected 0 products, got %d", len(products))
	}
}

func TestParseCatalogueXML_InvalidXML(t *testing.T) {
	_, err := ParseCatalogueXML(strings.NewReader("<invalid"))
	if err == nil {
		t.Fatal("expected error for invalid XML")
	}
}

func TestParseCatalogueXML_EmptyPictureURL(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<Document-ProductCatalogue>
  <ProductCatalogue-Lines>
    <Line><Line-Item>
      <EAN>1234567890123</EAN>
      <Pictures>
        <Picture><Url>https://example.com/img.jpg</Url></Picture>
        <Picture><Url>  </Url></Picture>
        <Picture><Url></Url></Picture>
      </Pictures>
    </Line-Item></Line>
  </ProductCatalogue-Lines>
</Document-ProductCatalogue>`

	products, err := ParseCatalogueXML(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}
	if len(products[0].Pictures) != 1 {
		t.Errorf("expected 1 non-empty picture, got %d", len(products[0].Pictures))
	}
}

func TestParseCatalogueURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(sampleCatalogueXML))
	}))
	defer srv.Close()

	products, err := ParseCatalogueURL(context.Background(), srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(products))
	}
}

func TestParseCatalogueURL_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := ParseCatalogueURL(context.Background(), srv.URL, srv.Client())
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

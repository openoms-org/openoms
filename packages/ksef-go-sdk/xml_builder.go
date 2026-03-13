package ksef

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

// InvoiceData holds all data needed to build a KSeF-compliant FA(2) XML invoice.
type InvoiceData struct {
	// Header
	InvoiceDate   time.Time
	InvoiceNumber string
	Currency      string

	// Seller (Podmiot1)
	SellerNIP     string
	SellerName    string
	SellerStreet  string
	SellerCity    string
	SellerPostal  string
	SellerCountry string // ISO 3166-1 alpha-2, e.g. "PL"

	// Buyer (Podmiot2)
	BuyerNIP     string
	BuyerName    string
	BuyerStreet  string
	BuyerCity    string
	BuyerPostal  string
	BuyerCountry string

	// Invoice body
	Items       []InvoiceLineItem
	TotalNet    float64
	TotalVAT    float64
	TotalGross  float64
	PaymentDate time.Time
	PaymentType string // "przelew", "gotowka", etc.
	Notes       string
}

// InvoiceLineItem represents a single line item on the invoice.
type InvoiceLineItem struct {
	LineNumber  int
	Name        string
	Quantity    float64
	Unit        string // "szt.", "kg", "usł.", etc.
	NetPrice    float64
	NetAmount   float64
	VATRate     string // "23", "8", "5", "0", "zw" (exempt), "np" (not subject)
	VATAmount   float64
	GrossAmount float64
}

// BuildInvoiceXML generates a KSeF-compliant FA(2) structured invoice XML.
func BuildInvoiceXML(data InvoiceData) ([]byte, error) {
	if data.SellerNIP == "" {
		return nil, fmt.Errorf("ksef: seller NIP is required")
	}
	if len(data.Items) == 0 {
		return nil, fmt.Errorf("ksef: at least one line item is required")
	}
	if data.Currency == "" {
		data.Currency = "PLN"
	}
	if data.SellerCountry == "" {
		data.SellerCountry = "PL"
	}
	if data.BuyerCountry == "" {
		data.BuyerCountry = "PL"
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteString(`<Faktura xmlns="http://crd.gov.pl/wzor/2023/06/29/12648/"` + "\n")
	buf.WriteString(`  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"` + "\n")
	buf.WriteString(`  xsi:schemaLocation="http://crd.gov.pl/wzor/2023/06/29/12648/ http://crd.gov.pl/wzor/2023/06/29/12648/schemat.xsd">` + "\n")

	// Naglowek (Header)
	writeHeader(&buf, data)

	// Podmiot1 (Seller)
	writeSubject1(&buf, data)

	// Podmiot2 (Buyer)
	writeSubject2(&buf, data)

	// Fa (Invoice body)
	writeInvoiceBody(&buf, data)

	buf.WriteString("</Faktura>\n")

	return buf.Bytes(), nil
}

func writeHeader(w io.Writer, _ InvoiceData) {
	_, _ = fmt.Fprintf(w, "  <Naglowek>\n")
	_, _ = fmt.Fprintf(w, "    <KodFormularza kodSystemowy=\"FA (2)\" wersjaSchemy=\"1-0E\">FA</KodFormularza>\n")
	_, _ = fmt.Fprintf(w, "    <WariantFormularza>2</WariantFormularza>\n")
	_, _ = fmt.Fprintf(w, "    <DataWytworzeniaFa>%s</DataWytworzeniaFa>\n", time.Now().Format("2006-01-02T15:04:05"))
	_, _ = fmt.Fprintf(w, "    <SystemInfo>OpenOMS</SystemInfo>\n")
	_, _ = fmt.Fprintf(w, "  </Naglowek>\n")
}

func writeSubject1(w io.Writer, data InvoiceData) {
	_, _ = fmt.Fprintf(w, "  <Podmiot1>\n")
	_, _ = fmt.Fprintf(w, "    <DaneIdentyfikacyjne>\n")
	_, _ = fmt.Fprintf(w, "      <NIP>%s</NIP>\n", escapeXML(data.SellerNIP))
	_, _ = fmt.Fprintf(w, "      <Nazwa>%s</Nazwa>\n", escapeXML(data.SellerName))
	_, _ = fmt.Fprintf(w, "    </DaneIdentyfikacyjne>\n")
	_, _ = fmt.Fprintf(w, "    <Adres>\n") //nolint:misspell // Polish KSeF XML element name
	_, _ = fmt.Fprintf(w, "      <KodKraju>%s</KodKraju>\n", escapeXML(data.SellerCountry))
	_, _ = fmt.Fprintf(w, "      <AdresL1>%s</AdresL1>\n", escapeXML(data.SellerStreet))
	_, _ = fmt.Fprintf(w, "      <AdresL2>%s %s</AdresL2>\n", escapeXML(data.SellerPostal), escapeXML(data.SellerCity))
	_, _ = fmt.Fprintf(w, "    </Adres>\n") //nolint:misspell // Polish KSeF XML element name
	_, _ = fmt.Fprintf(w, "  </Podmiot1>\n")
}

func writeSubject2(w io.Writer, data InvoiceData) {
	_, _ = fmt.Fprintf(w, "  <Podmiot2>\n")
	_, _ = fmt.Fprintf(w, "    <DaneIdentyfikacyjne>\n")
	if data.BuyerNIP != "" {
		_, _ = fmt.Fprintf(w, "      <NIP>%s</NIP>\n", escapeXML(data.BuyerNIP))
	}
	_, _ = fmt.Fprintf(w, "      <Nazwa>%s</Nazwa>\n", escapeXML(data.BuyerName))
	_, _ = fmt.Fprintf(w, "    </DaneIdentyfikacyjne>\n")
	if data.BuyerStreet != "" || data.BuyerCity != "" {
		_, _ = fmt.Fprintf(w, "    <Adres>\n") //nolint:misspell // Polish KSeF XML element name //nolint:misspell // Polish KSeF XML element name
		_, _ = fmt.Fprintf(w, "      <KodKraju>%s</KodKraju>\n", escapeXML(data.BuyerCountry))
		if data.BuyerStreet != "" {
			_, _ = fmt.Fprintf(w, "      <AdresL1>%s</AdresL1>\n", escapeXML(data.BuyerStreet))
		}
		if data.BuyerCity != "" || data.BuyerPostal != "" {
			_, _ = fmt.Fprintf(w, "      <AdresL2>%s %s</AdresL2>\n", escapeXML(data.BuyerPostal), escapeXML(data.BuyerCity))
		}
		_, _ = fmt.Fprintf(w, "    </Adres>\n") //nolint:misspell // Polish KSeF XML element name //nolint:misspell // Polish KSeF XML element name
	}
	_, _ = fmt.Fprintf(w, "  </Podmiot2>\n")
}

func writeInvoiceBody(w io.Writer, data InvoiceData) {
	_, _ = fmt.Fprintf(w, "  <Fa>\n")
	_, _ = fmt.Fprintf(w, "    <KodWaluty>%s</KodWaluty>\n", escapeXML(data.Currency))
	_, _ = fmt.Fprintf(w, "    <P_1>%s</P_1>\n", data.InvoiceDate.Format("2006-01-02"))
	_, _ = fmt.Fprintf(w, "    <P_2>%s</P_2>\n", escapeXML(data.InvoiceNumber))

	// VAT summary amounts per rate
	vatSummary := computeVATSummary(data.Items)
	for _, vs := range vatSummary {
		writeVATSummaryLine(w, vs)
	}

	// Totals
	_, _ = fmt.Fprintf(w, "    <P_15>%.2f</P_15>\n", data.TotalGross)

	// Payment
	if !data.PaymentDate.IsZero() {
		_, _ = fmt.Fprintf(w, "    <TerminPlatnosci>\n")
		_, _ = fmt.Fprintf(w, "      <Termin>%s</Termin>\n", data.PaymentDate.Format("2006-01-02"))
		_, _ = fmt.Fprintf(w, "    </TerminPlatnosci>\n")
	}

	if data.PaymentType != "" {
		_, _ = fmt.Fprintf(w, "    <FormaPlatnosci>%s</FormaPlatnosci>\n", escapeXML(mapPaymentType(data.PaymentType)))
	}

	// Line items
	for i, item := range data.Items {
		writeLineItem(w, item, i+1)
	}

	if data.Notes != "" {
		_, _ = fmt.Fprintf(w, "    <Adnotacje>\n")
		_, _ = fmt.Fprintf(w, "      <P_16>2</P_16>\n")
		_, _ = fmt.Fprintf(w, "      <P_17>2</P_17>\n")
		_, _ = fmt.Fprintf(w, "      <P_18>2</P_18>\n")
		_, _ = fmt.Fprintf(w, "      <P_18A>2</P_18A>\n")
		_, _ = fmt.Fprintf(w, "      <Zwolnienie>\n")
		_, _ = fmt.Fprintf(w, "        <P_19N>1</P_19N>\n")
		_, _ = fmt.Fprintf(w, "      </Zwolnienie>\n")
		_, _ = fmt.Fprintf(w, "      <NoweSrodkiTransportu>\n")
		_, _ = fmt.Fprintf(w, "        <P_22N>1</P_22N>\n")
		_, _ = fmt.Fprintf(w, "      </NoweSrodkiTransportu>\n")
		_, _ = fmt.Fprintf(w, "      <P_23>2</P_23>\n")
		_, _ = fmt.Fprintf(w, "      <PMarzy>\n")
		_, _ = fmt.Fprintf(w, "        <P_PMarzyN>1</P_PMarzyN>\n")
		_, _ = fmt.Fprintf(w, "      </PMarzy>\n")
		_, _ = fmt.Fprintf(w, "    </Adnotacje>\n")
	}

	_, _ = fmt.Fprintf(w, "  </Fa>\n")
}

func writeLineItem(w io.Writer, item InvoiceLineItem, lineNum int) {
	_, _ = fmt.Fprintf(w, "    <FaWiersz>\n")
	_, _ = fmt.Fprintf(w, "      <NrWierszaFa>%d</NrWierszaFa>\n", lineNum)
	_, _ = fmt.Fprintf(w, "      <P_7>%s</P_7>\n", escapeXML(item.Name))
	_, _ = fmt.Fprintf(w, "      <P_8A>%s</P_8A>\n", escapeXML(item.Unit))
	_, _ = fmt.Fprintf(w, "      <P_8B>%.4f</P_8B>\n", item.Quantity)
	_, _ = fmt.Fprintf(w, "      <P_9A>%.2f</P_9A>\n", item.NetPrice)
	_, _ = fmt.Fprintf(w, "      <P_11>%.2f</P_11>\n", item.NetAmount)
	_, _ = fmt.Fprintf(w, "      <P_12>%s</P_12>\n", escapeXML(item.VATRate))
	_, _ = fmt.Fprintf(w, "    </FaWiersz>\n")
}

type vatSummaryEntry struct {
	Rate      string
	NetAmount float64
	VATAmount float64
}

func computeVATSummary(items []InvoiceLineItem) []vatSummaryEntry {
	summaryMap := make(map[string]*vatSummaryEntry)
	var order []string

	for _, item := range items {
		rate := item.VATRate
		entry, ok := summaryMap[rate]
		if !ok {
			entry = &vatSummaryEntry{Rate: rate}
			summaryMap[rate] = entry
			order = append(order, rate)
		}
		entry.NetAmount += item.NetAmount
		entry.VATAmount += item.VATAmount
	}

	result := make([]vatSummaryEntry, 0, len(order))
	for _, rate := range order {
		result = append(result, *summaryMap[rate])
	}
	return result
}

func writeVATSummaryLine(w io.Writer, vs vatSummaryEntry) {
	// Map VAT rates to the appropriate P_13/P_14 element pairs
	switch vs.Rate {
	case "23":
		_, _ = fmt.Fprintf(w, "    <P_13_1>%.2f</P_13_1>\n", vs.NetAmount)
		_, _ = fmt.Fprintf(w, "    <P_14_1>%.2f</P_14_1>\n", vs.VATAmount)
	case "22":
		_, _ = fmt.Fprintf(w, "    <P_13_1>%.2f</P_13_1>\n", vs.NetAmount)
		_, _ = fmt.Fprintf(w, "    <P_14_1>%.2f</P_14_1>\n", vs.VATAmount)
	case "8":
		_, _ = fmt.Fprintf(w, "    <P_13_2>%.2f</P_13_2>\n", vs.NetAmount)
		_, _ = fmt.Fprintf(w, "    <P_14_2>%.2f</P_14_2>\n", vs.VATAmount)
	case "7":
		_, _ = fmt.Fprintf(w, "    <P_13_2>%.2f</P_13_2>\n", vs.NetAmount)
		_, _ = fmt.Fprintf(w, "    <P_14_2>%.2f</P_14_2>\n", vs.VATAmount)
	case "5":
		_, _ = fmt.Fprintf(w, "    <P_13_3>%.2f</P_13_3>\n", vs.NetAmount)
		_, _ = fmt.Fprintf(w, "    <P_14_3>%.2f</P_14_3>\n", vs.VATAmount)
	case "0":
		_, _ = fmt.Fprintf(w, "    <P_13_6_1>%.2f</P_13_6_1>\n", vs.NetAmount)
	case "zw":
		_, _ = fmt.Fprintf(w, "    <P_13_7>%.2f</P_13_7>\n", vs.NetAmount)
	}
}

func mapPaymentType(pt string) string {
	switch strings.ToLower(pt) {
	case "przelew", "transfer", "bank_transfer":
		return "6"
	case "gotowka", "cash":
		return "1"
	case "karta", "card":
		return "2"
	case "pobranie", "cod":
		return "6"
	default:
		return "6" // Default: bank transfer
	}
}

func escapeXML(s string) string {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
}

// buildInitTokenXML builds the XML body for the InitToken endpoint.
func buildInitTokenXML(nip, encryptedToken, challenge string) io.Reader {
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<ns3:InitSessionTokenRequest xmlns="http://ksef.mf.gov.pl/schema/gtw/svc/online/types/2021/10/01/0001"
  xmlns:ns2="http://ksef.mf.gov.pl/schema/gtw/svc/types/2021/10/01/0001"
  xmlns:ns3="http://ksef.mf.gov.pl/schema/gtw/svc/online/auth/request/2021/10/01/0001">
  <ns3:Context>
    <Challenge>%s</Challenge>
    <Identifier xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="ns2:SubjectIdentifierByCompanyType">
      <ns2:Identifier>%s</ns2:Identifier>
    </Identifier>
    <DocumentType>
      <ns2:Service>KSeF</ns2:Service>
      <ns2:FormCode>
        <ns2:SystemCode>FA (2)</ns2:SystemCode>
        <ns2:SchemaVersion>1-0E</ns2:SchemaVersion>
        <ns2:TargetNamespace>http://crd.gov.pl/wzor/2023/06/29/12648/</ns2:TargetNamespace>
        <ns2:Value>FA</ns2:Value>
      </ns2:FormCode>
    </DocumentType>
    <Token>%s</Token>
  </ns3:Context>
</ns3:InitSessionTokenRequest>`, escapeXML(challenge), escapeXML(nip), escapeXML(encryptedToken))

	return strings.NewReader(xml)
}

// parseJSON is a helper to unmarshal JSON from bytes.
func parseJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

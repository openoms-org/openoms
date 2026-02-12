package ksef

import "time"

// Environment represents the KSeF API environment.
type Environment string

const (
	// EnvironmentTest is the KSeF test (sandbox) environment.
	EnvironmentTest Environment = "test"
	// EnvironmentProduction is the KSeF production environment.
	EnvironmentProduction Environment = "production"
)

// BaseURL returns the KSeF API base URL for the environment.
func (e Environment) BaseURL() string {
	switch e {
	case EnvironmentProduction:
		return "https://ksef.mf.gov.pl/api"
	default:
		return "https://ksef-test.mf.gov.pl/api"
	}
}

// AuthorisationChallengeRequest is the request body for the authorisation challenge.
type AuthorisationChallengeRequest struct {
	ContextIdentifier ContextIdentifier `json:"contextIdentifier"`
}

// ContextIdentifier holds the NIP used in the challenge request.
type ContextIdentifier struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

// AuthorisationChallengeResponse is the response from the authorisation challenge.
type AuthorisationChallengeResponse struct {
	Timestamp string `json:"timestamp"`
	Challenge string `json:"challenge"`
}

// InitSessionTokenRequest is the XML-based request body for session init.
type InitSessionTokenRequest struct {
	Context InitSessionContext `json:"context"`
}

// InitSessionContext holds the nested context for init session.
type InitSessionContext struct {
	Challenge   string         `json:"challenge"`
	Identifier  ContextIdentifier `json:"identifier"`
	DocumentType DocumentType  `json:"documentType"`
	Token       string         `json:"token"`
}

// DocumentType specifies the invoice schema in use.
type DocumentType struct {
	Service    string `json:"service"`
	FormCode   FormCode `json:"formCode"`
}

// FormCode identifies the schema version.
type FormCode struct {
	SystemCode  string `json:"systemCode"`
	SchemaVersion string `json:"schemaVersion"`
	TargetNamespace string `json:"targetNamespace"`
	Value       string `json:"value"`
}

// InitSessionResponse is the response from session initialization.
type InitSessionResponse struct {
	SessionToken SessionToken `json:"sessionToken"`
	ReferenceNumber string `json:"referenceNumber"`
	Timestamp string `json:"timestamp"`
}

// SessionToken holds the session token details.
type SessionToken struct {
	Token  string `json:"token"`
	Context SessionContext `json:"context"`
}

// SessionContext holds session metadata.
type SessionContext struct {
	ContextIdentifier ContextIdentifier `json:"contextIdentifier"`
	ContextName string `json:"contextName,omitempty"`
	CredentialsRoleList []CredentialsRole `json:"credentialsRoleList,omitempty"`
}

// CredentialsRole defines a role in the session.
type CredentialsRole struct {
	Type         string `json:"type"`
	RoleType     string `json:"roleType"`
	RoleDescription string `json:"roleDescription,omitempty"`
}

// SessionStatusResponse is the response from session status.
type SessionStatusResponse struct {
	Timestamp       string `json:"timestamp"`
	ReferenceNumber string `json:"referenceNumber"`
	NumberOfElements int   `json:"numberOfElements"`
	PageSize        int    `json:"pageSize"`
	PageOffset      int    `json:"pageOffset"`
	ProcessingCode  int    `json:"processingCode"`
	ProcessingDescription string `json:"processingDescription"`
	InvoiceStatusList []InvoiceStatus `json:"invoiceStatusList,omitempty"`
}

// InvoiceStatus represents the status of a submitted invoice.
type InvoiceStatus struct {
	InvoiceNumber     string `json:"invoiceNumber,omitempty"`
	KsefReferenceNumber string `json:"ksefReferenceNumber,omitempty"`
	AcquisitionTimestamp string `json:"acquisitionTimestamp,omitempty"`
	ProcessingCode    int    `json:"processingCode"`
	ProcessingDescription string `json:"processingDescription"`
	ElementReferenceNumber string `json:"elementReferenceNumber,omitempty"`
}

// SendInvoiceResponse is the response from sending an invoice to KSeF.
type SendInvoiceResponse struct {
	ElementReferenceNumber string `json:"elementReferenceNumber"`
	ReferenceNumber       string `json:"referenceNumber"`
	ProcessingCode        int    `json:"processingCode"`
	ProcessingDescription string `json:"processingDescription"`
	Timestamp             string `json:"timestamp"`
}

// TerminateSessionResponse is the response from terminating a session.
type TerminateSessionResponse struct {
	Timestamp       string `json:"timestamp"`
	ReferenceNumber string `json:"referenceNumber"`
	ProcessingCode  int    `json:"processingCode"`
	ProcessingDescription string `json:"processingDescription"`
}

// UPOResponse represents the UPO (Urzedowe Poswiadczenie Odbioru) data.
type UPOResponse struct {
	Timestamp       string `json:"timestamp"`
	ReferenceNumber string `json:"referenceNumber"`
	ProcessingCode  int    `json:"processingCode"`
	ProcessingDescription string `json:"processingDescription"`
	UPO             string `json:"upo"` // base64-encoded UPO document
}

// InvoiceQueryRequest defines parameters for querying invoices.
type InvoiceQueryRequest struct {
	SubjectType    string    `json:"subjectType"` // "subject1" (seller) or "subject2" (buyer)
	DateFrom       time.Time `json:"-"`
	DateTo         time.Time `json:"-"`
	SubjectNIP     string    `json:"subjectNip,omitempty"`
}

// InvoiceQueryResponse is the response from an invoice query.
type InvoiceQueryResponse struct {
	Timestamp       string `json:"timestamp"`
	ReferenceNumber string `json:"referenceNumber"`
	NumberOfElements int   `json:"numberOfElements"`
	PageSize        int    `json:"pageSize"`
	PageOffset      int    `json:"pageOffset"`
	InvoiceHeaderList []InvoiceHeader `json:"invoiceHeaderList,omitempty"`
}

// InvoiceHeader contains summary info about a KSeF invoice.
type InvoiceHeader struct {
	InvoiceReferenceNumber string `json:"invoiceReferenceNumber"`
	KsefReferenceNumber    string `json:"ksefReferenceNumber"`
	InvoiceNumber          string `json:"invoiceNumber"`
	SubjectBy              Subject `json:"subjectBy,omitempty"`
	SubjectTo              Subject `json:"subjectTo,omitempty"`
	Net                    string `json:"net,omitempty"`
	Vat                    string `json:"vat,omitempty"`
	Gross                  string `json:"gross,omitempty"`
	InvoicingDate          string `json:"invoicingDate,omitempty"`
}

// Subject represents a party (buyer or seller) in query results.
type Subject struct {
	IssuedByIdentifier SubjectIdentifier `json:"issuedByIdentifier,omitempty"`
	IssuedByName       string            `json:"issuedByName,omitempty"`
}

// SubjectIdentifier holds NIP or other identifiers.
type SubjectIdentifier struct {
	Type       string `json:"type,omitempty"`
	Identifier string `json:"identifier,omitempty"`
}

// APIError represents an error returned by the KSeF API.
type APIError struct {
	StatusCode  int    `json:"-"`
	Code        string `json:"code,omitempty"`
	Message     string `json:"message,omitempty"`
	Description string `json:"description,omitempty"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Message != "" {
		return "ksef: " + e.Message
	}
	if e.Description != "" {
		return "ksef: " + e.Description
	}
	return "ksef: API error"
}

package btp

// APIResult is the standard result envelope returned by most BTP API endpoints.
type APIResult struct {
	ResultType string             `json:"resultType"` // INFORMATION, WARNING, ERROR
	Messages   []APIResultMessage `json:"messages"`
	Code       int                `json:"code"`
}

// APIResultMessage represents a single message in an API result.
type APIResultMessage struct {
	MessageType string `json:"messageType"` // INFORMATION, WARNING, ERROR
	ObjectKey   string `json:"objectKey"`
	Message     string `json:"message"`
	Code        int    `json:"code"`
}

// --- Inventory Report ---

// InvReportDocument is the response from InventoryReportGet.
type InvReportDocument struct {
	Header  InvReportHeader  `json:"header"`
	Parties InvReportParties `json:"parties"`
	Lines   []InvReportLine  `json:"lines"`
}

// InvReportHeader contains metadata about the inventory report.
type InvReportHeader struct {
	InventoryReportNumber string `json:"inventoryReportNumber"`
	InventoryReportDate   string `json:"inventoryReportDate"`
	InventoryReportTime   string `json:"inventoryReportTime"`
	Currency              string `json:"currency"`
}

// InvReportParties contains buyer/seller/warehouse information.
type InvReportParties struct {
	Buyer     Party `json:"buyer"`
	Seller    Party `json:"seller"`
	Warehouse Party `json:"warehouse"`
}

// Party represents a business entity (buyer, seller, or warehouse).
type Party struct {
	Name       string `json:"name"`
	ILN        string `json:"iln"`
	Address    string `json:"address"`
	City       string `json:"city"`
	PostalCode string `json:"postalCode"`
	CountryID  string `json:"countryId"`
}

// InvReportLine represents a single product line in the inventory report.
type InvReportLine struct {
	LineNumber           int64         `json:"lineNumber"`
	ItemID               string        `json:"itemId"`
	EAN                  string        `json:"ean"`
	ManufacturerItemCode string        `json:"manufacturerItemCode"`
	ItemName             string        `json:"itemName"`
	QuantityOnHand       *float64      `json:"quantityOnHand"`
	ActualStock          *float64      `json:"actualStock"`
	UnitOfMeasure        string        `json:"unitOfMeasure"`
	UnitNetPrice         float64       `json:"unitNetPrice"`
	UnitRetailPrice      float64       `json:"unitRetailPrice"`
	Availability         *Availability `json:"availability"`
}

// Availability contains scheduled stock information.
type Availability struct {
	StockSchedules []StockSchedule `json:"stockSchedules"`
}

// StockSchedule represents a future stock availability entry.
type StockSchedule struct {
	ScheduleDate  string  `json:"scheduleDate"`
	Quantity      float64 `json:"quantity"`
	WarehouseName string  `json:"warehouseName"`
}

// --- Product Catalogue ---

// ProdCatalogueDocument is the response from ProductCatalogueGet.
type ProdCatalogueDocument struct {
	Header  ProdCatalogueHeader  `json:"header"`
	Parties ProdCatalogueParties `json:"parties"`
	Lines   []ProdCatalogueLine  `json:"lines"`
}

// ProdCatalogueHeader contains metadata about the product catalogue.
type ProdCatalogueHeader struct {
	CatalogueNumber string `json:"catalogueNumber"`
	CatalogueDate   string `json:"catalogueDate"`
	CatalogueTime   string `json:"catalogueTime"`
	Currency        string `json:"currency"`
}

// ProdCatalogueParties contains buyer/seller/warehouse information.
type ProdCatalogueParties struct {
	Buyer     Party `json:"buyer"`
	Seller    Party `json:"seller"`
	Warehouse Party `json:"warehouse"`
}

// ProdCatalogueLine represents a single product in the catalogue.
type ProdCatalogueLine struct {
	LineNumber           int64          `json:"lineNumber"`
	ItemID               string         `json:"itemId"`
	EAN                  string         `json:"ean"`
	ManufacturerItemCode string         `json:"manufacturerItemCode"`
	ItemName             string         `json:"itemName"`
	ItemDescription      string         `json:"itemDescription"`
	UnitOfMeasure        string         `json:"unitOfMeasure"`
	UnitNetPrice         float64        `json:"unitNetPrice"`
	UnitRetailPrice      float64        `json:"unitRetailPrice"`
	CategoryNodes        []CategoryNode `json:"categoryNodes"`
	BrandNodes           []BrandNode    `json:"brandNodes"`
	Availability         *Availability  `json:"availability"`
}

// CategoryNode represents a product category in the hierarchy.
type CategoryNode struct {
	CategoryNodeID   string `json:"categoryNodeId"`
	CategoryNodeCode string `json:"categoryNodeCode"`
	CategoryNodeName string `json:"categoryNodeName"`
}

// BrandNode represents a product brand in the hierarchy.
type BrandNode struct {
	BrandNodeID   string `json:"brandNodeId"`
	BrandNodeCode string `json:"brandNodeCode"`
	BrandNodeName string `json:"brandNodeName"`
}

// --- Client Info ---

// ClientInfo is the response from ClientGet.
type ClientInfo struct {
	ClientID           string                `json:"clientId"`
	Name               string                `json:"name"`
	TaxID              string                `json:"taxId"`
	DefaultCurrencyID  string                `json:"defaultCurrencyId"`
	DefaultLangID      string                `json:"defaultLangId"`
	DefaultPaymentDays int                   `json:"defaultPaymentDays"`
	DeliveryPoints     []ClientDeliveryPoint `json:"deliveryPoints"`
}

// ClientDeliveryPoint represents a delivery address.
type ClientDeliveryPoint struct {
	DeliveryPointID string `json:"deliveryPointId"`
	Name            string `json:"name"`
	AddressLine     string `json:"addressLine"`
	City            string `json:"city"`
	PostalCode      string `json:"postalCode"`
	Country         string `json:"country"`
	ContactPerson   string `json:"contactPerson"`
	OwnName         string `json:"ownName"`
	Note            string `json:"note"`
	IsDisposable    *bool  `json:"isDisposable"`
}

// DeliveryMode represents a delivery mode option.
type DeliveryMode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// DeliveryModesResponse is the response from DeliveryModesGet.
type DeliveryModesResponse struct {
	Modes []DeliveryMode `json:"modes"`
}

// Carrier represents a carrier option.
type Carrier struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CarriersResponse is the response from CarriersGet.
type CarriersResponse struct {
	Carriers []Carrier `json:"carriers"`
}

// PaymentMethod represents a payment method option.
type PaymentMethod struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PaymentMethodsResponse is the response from PaymentMethodsGet.
type PaymentMethodsResponse struct {
	PaymentMethods []PaymentMethod `json:"paymentMethods"`
}

// CountryGovArea represents a governing/territorial area.
type CountryGovArea struct {
	CountryID      string    `json:"countryId"`
	GovAreaLevel   int       `json:"govAreaLevel"`
	GovAreaID      string    `json:"govAreaId"`
	ParentGovArea  string    `json:"parentGovAreaId"`
	Name           string    `json:"name"`
	Enabled        bool      `json:"enabled"`
	SortID         int       `json:"sortId"`
	Code           string    `json:"code"`
	ExtCode        string    `json:"extCode"`
	EntryInfo      EntryInfo `json:"entryInfo"`
}

// EntryInfo contains create/update metadata for an entity.
type EntryInfo struct {
	CreatedAt  string `json:"createdAt"`
	CreatedBy  string `json:"createdBy"`
	ModifiedAt string `json:"modifiedAt"`
	ModifiedBy string `json:"modifiedBy"`
}

// GovAreasResponse is the response from CountryGovAreasGet.
type GovAreasResponse struct {
	GovAreas []CountryGovArea `json:"govAreas"`
}

// --- Orders ---

// OrderRequest is the request body for OrderCreate.
type OrderRequest struct {
	Lines              []OrderRequestLine `json:"lines"`
	DeliveryPointID    string             `json:"deliveryPointId,omitempty"`
	CurrencyID         string             `json:"currencyId,omitempty"`
	LangID             string             `json:"langId,omitempty"`
	PaymentMethodID    string             `json:"paymentMethodId,omitempty"`
	DeliveryModeID     string             `json:"deliveryModeId,omitempty"`
	CarrierID          string             `json:"carrierId,omitempty"`
	ClientOrderNumber  string             `json:"clientOrderNumber,omitempty"`
	EndUserOrderNumber string             `json:"endUserOrderNumber,omitempty"`
	Note               string             `json:"note,omitempty"`
	IsTesting          bool               `json:"isTesting,omitempty"`
	AttachmentFileIDs  []string           `json:"attachmentFileIds,omitempty"`
}

// OrderRequestLine represents a single line item in an order.
type OrderRequestLine struct {
	BarCode      string   `json:"barCode,omitempty"`
	ItemID       string   `json:"itemId,omitempty"`
	Quantity     float64  `json:"quantity"`
	UnitNetPrice *float64 `json:"unitNetPrice,omitempty"`
}

// OrderResponse is the response from OrderCreate.
type OrderResponse struct {
	Result      APIResult          `json:"result"`
	OrderID     string             `json:"orderId"`
	OrderNumber string             `json:"orderNumber"`
	OrderDate   string             `json:"orderDate"`
	Warnings    []APIResultMessage `json:"warnings"`
}

// OrderAttachmentsRequest is the request body for OrderAttachmentsSet.
type OrderAttachmentsRequest struct {
	OrderID           string   `json:"orderId"`
	AttachmentFileIDs []string `json:"attachmentFileIds"`
}

// OrderAttachmentsResponse is the response from OrderAttachmentsSet.
type OrderAttachmentsResponse struct {
	Result          APIResult `json:"result"`
	OrderID         string    `json:"orderId"`
	AttachmentCount int       `json:"attachmentCount"`
}

// --- Invoices ---

// InvoicesRequest is the request body for InvoicesGet.
type InvoicesRequest struct {
	Documents []InvoicesRequestDocument `json:"documents"`
}

// InvoicesRequestDocument identifies an invoice to retrieve.
type InvoicesRequestDocument struct {
	DocumentID   string `json:"documentId"`
	DocumentType string `json:"documentType"` // INV or CR
}

// InvoicesResponse is the response from InvoicesGet.
type InvoicesResponse struct {
	Invoices          []InvoiceDocument         `json:"invoices"`
	NotFoundDocuments []InvoicesRequestDocument `json:"notFoundDocuments"`
}

// InvoiceDocument represents a single invoice.
type InvoiceDocument struct {
	InvoiceCreatedOperation string               `json:"invoiceCreatedOperation"` // CREATED, MODIFIED
	Header                  InvoiceDocumentHeader `json:"header"`
	Parties                 InvoiceParties        `json:"parties"`
	Lines                   []InvoiceLine         `json:"lines"`
	Summary                 *InvoiceSummary       `json:"summary"`
}

// InvoiceDocumentHeader contains invoice metadata.
type InvoiceDocumentHeader struct {
	InvoiceNumber string `json:"invoiceNumber"`
	InvoiceDate   string `json:"invoiceDate"`
	DueDate       string `json:"dueDate"`
	Currency      string `json:"currency"`
	OrderNumber   string `json:"orderNumber"`
	SellerID      string `json:"sellerId"`
}

// InvoiceParties contains invoice party information.
type InvoiceParties struct {
	Buyer  Party `json:"buyer"`
	Seller Party `json:"seller"`
}

// InvoiceLine represents a single line item on an invoice.
type InvoiceLine struct {
	LineNumber    int64   `json:"lineNumber"`
	ItemID        string  `json:"itemId"`
	EAN           string  `json:"ean"`
	ItemName      string  `json:"itemName"`
	Quantity      float64 `json:"quantity"`
	UnitNetPrice  float64 `json:"unitNetPrice"`
	NetAmount     float64 `json:"netAmount"`
	TaxRate       float64 `json:"taxRate"`
	TaxAmount     float64 `json:"taxAmount"`
	GrossAmount   float64 `json:"grossAmount"`
	UnitOfMeasure string  `json:"unitOfMeasure"`
}

// InvoiceSummary contains invoice total amounts.
type InvoiceSummary struct {
	TotalNetAmount   float64 `json:"totalNetAmount"`
	TotalTaxAmount   float64 `json:"totalTaxAmount"`
	TotalGrossAmount float64 `json:"totalGrossAmount"`
}

// InvoiceImageRequest contains parameters for InvoiceImageGet.
type InvoiceImageRequest struct {
	DocumentID   string `json:"documentId"`
	DocumentType string `json:"documentType"` // INV or CR
	ImageFormat  string `json:"imageFormat"`  // PDF
	SellerID     string `json:"sellerId"`
}

// InvoiceImageResponse is the response from InvoiceImageGet.
type InvoiceImageResponse struct {
	ImageBase64 string `json:"imageBase64"`
}

// --- Waybills ---

// WaybillsRequest is the request body for WaybillsGet.
type WaybillsRequest struct {
	WaybillNumbers []string `json:"waybillNumbers"`
}

// WaybillsResponse is the response from WaybillsGet.
type WaybillsResponse struct {
	Waybills         []Waybill `json:"waybills"`
	NotFoundWaybills []string  `json:"notFoundWaybills"`
}

// Waybill represents a shipping waybill document.
type Waybill struct {
	WaybillNumber  string `json:"waybillNumber"`
	OrderNumber    string `json:"orderNumber"`
	CarrierName    string `json:"carrierName"`
	TrackingNumber string `json:"trackingNumber"`
	Status         string `json:"status"`
}

// --- Notification Configuration ---

// DocumentNotifyConfig is used to configure invoice/waybill webhook notifications.
type DocumentNotifyConfig struct {
	EndpointURL    string                    `json:"endpointUrl"`
	HTTPMethod     string                    `json:"httpMethod"`
	Mode           string                    `json:"mode"`   // KeyOnly, WholeBody
	Format         string                    `json:"format"` // JSON, XML
	Authentication DocumentNotifyAuth        `json:"authentication"`
}

// DocumentNotifyAuth contains authentication settings for webhook notifications.
type DocumentNotifyAuth struct {
	AuthKey1     string `json:"authKey1"`
	AuthKey2     string `json:"authKey2"`
	AuthType     string `json:"authType"`     // Basic, Bearer, ApiKey
	AuthKey1Name string `json:"authKey1Name"`
	AuthKey2Name string `json:"authKey2Name"`
	AuthSendIn   string `json:"authSendIn"`   // Query, Header, Cookie
}

// --- Files ---

// FileAddResponse is the response from FileAdd.
type FileAddResponse struct {
	Result APIResult `json:"result"`
	FileID string    `json:"fileId"`
}

// FilesDeleteRequest is the request body for FilesDelete.
type FilesDeleteRequest struct {
	FileIDs []string `json:"fileIds"`
}

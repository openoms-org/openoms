package service

import "github.com/openoms-org/openoms/apps/api-server/internal/model"

// ProviderCatalogEntry is a static, class-first description of an existing
// provider, used to seed draft registry definitions (OPE-412 / design §526).
type ProviderCatalogEntry struct {
	ProviderKey     string
	DisplayName     string
	ProviderType    string
	Version         string
	Notes           string
	Regions         []string
	BusinessDomains []string
	Schema          []model.ProviderFieldGroup
	Capabilities    []model.ProviderCapability
}

// unknownCap builds a capability stub (support unknown — verified later by the
// Studio validation engine, never fabricated as supported).
func unknownCap(key string) model.ProviderCapability {
	return model.ProviderCapability{CapabilityKey: key, SupportStatus: model.SupportStatusUnknown}
}

func secretField(key, label string) model.ProviderField {
	return model.ProviderField{Key: key, Label: label, Type: model.FieldTypePassword, Required: true, Secret: true}
}

func settingField(key, label, typ string) model.ProviderField {
	return model.ProviderField{Key: key, Label: label, Type: typ}
}

// ProviderRegistryCatalog returns the class-first representative providers to
// seed into the registry (design §526). Capabilities are intentionally seeded
// as `unknown`; status mappings and capability verification happen in the Studio
// (OPE-407 workbench / OPE-408 validation) by platform admins.
func ProviderRegistryCatalog() []ProviderCatalogEntry {
	return []ProviderCatalogEntry{
		{
			ProviderKey: "allegro", DisplayName: "Allegro", ProviderType: model.ProviderTypeMarketplace, Version: "1.0.0",
			Notes:   "Marketplace capability-class representative (OAuth2). Seeded from existing integration.",
			Regions: []string{"PL"}, BusinessDomains: []string{"marketplace"},
			Schema: []model.ProviderFieldGroup{
				{Key: model.FieldGroupSecretCredentials, Label: "Credentials", Fields: []model.ProviderField{
					secretField("client_id", "Client ID"),
					secretField("client_secret", "Client Secret"),
				}},
				{Key: model.FieldGroupEnvironment, Label: "Environment", Fields: []model.ProviderField{
					{Key: "environment", Label: "Environment", Type: model.FieldTypeEnum, Required: true,
						Validation: model.ProviderFieldValidation{Enum: []string{"sandbox", "production"}}},
				}},
			},
			Capabilities: []model.ProviderCapability{
				unknownCap("marketplace.order.pull"), unknownCap("marketplace.order.status.push"),
			},
		},
		{
			ProviderKey: "inpost", DisplayName: "InPost", ProviderType: model.ProviderTypeCarrier, Version: "1.0.0",
			Notes:   "Carrier capability-class representative (labels, tracking, pickup points). Seeded from existing integration.",
			Regions: []string{"PL"}, BusinessDomains: []string{"carrier"},
			Schema: []model.ProviderFieldGroup{
				{Key: model.FieldGroupSecretCredentials, Label: "Credentials", Fields: []model.ProviderField{
					secretField("api_token", "API Token"),
				}},
				{Key: model.FieldGroupSettings, Label: "Settings", Fields: []model.ProviderField{
					settingField("organization_id", "Organization ID", model.FieldTypeString),
				}},
			},
			Capabilities: []model.ProviderCapability{
				unknownCap("carrier.shipment.create"), unknownCap("carrier.tracking.read"),
			},
		},
		{
			ProviderKey: "btp", DisplayName: "BTP.pro", ProviderType: model.ProviderTypeSupplier, Version: "1.0.0",
			Notes:   "Hybrid supplier feed/API capability-class representative. Seeded from existing integration.",
			Regions: []string{"PL"}, BusinessDomains: []string{"supplier"},
			Schema: []model.ProviderFieldGroup{
				{Key: model.FieldGroupSecretCredentials, Label: "Credentials", Fields: []model.ProviderField{
					{Key: "username", Label: "Username", Type: model.FieldTypeString, Required: true, Secret: true},
					secretField("password", "Password"),
				}},
				{Key: model.FieldGroupSettings, Label: "Feed", Fields: []model.ProviderField{
					{Key: "feed_url", Label: "Feed URL", Type: model.FieldTypeURL, Required: true},
				}},
			},
			Capabilities: []model.ProviderCapability{
				unknownCap("supplier.catalog.read"), unknownCap("supplier.availability.read"), unknownCap("supplier.price.read"),
			},
		},
		{
			ProviderKey: "fakturownia", DisplayName: "Fakturownia", ProviderType: model.ProviderTypeInvoicing, Version: "1.0.0",
			Notes:   "Invoice capability-class representative. Seeded from existing integration.",
			Regions: []string{"PL"}, BusinessDomains: []string{"invoicing"},
			Schema: []model.ProviderFieldGroup{
				{Key: model.FieldGroupSecretCredentials, Label: "Credentials", Fields: []model.ProviderField{
					secretField("api_token", "API Token"),
				}},
				{Key: model.FieldGroupSettings, Label: "Settings", Fields: []model.ProviderField{
					settingField("domain", "Account Domain", model.FieldTypeString),
				}},
			},
			Capabilities: []model.ProviderCapability{
				unknownCap("invoice.issue"), unknownCap("invoice.pdf.read"),
			},
		},
		{
			ProviderKey: "shopify", DisplayName: "Shopify", ProviderType: model.ProviderTypeShop, Version: "1.0.0",
			Notes:   "Hosted shop capability-class representative. Seeded from existing integration.",
			Regions: []string{}, BusinessDomains: []string{"shop"},
			Schema: []model.ProviderFieldGroup{
				{Key: model.FieldGroupSettings, Label: "Store", Fields: []model.ProviderField{
					{Key: "shop_url", Label: "Shop URL", Type: model.FieldTypeURL, Required: true},
				}},
				{Key: model.FieldGroupSecretCredentials, Label: "Credentials", Fields: []model.ProviderField{
					secretField("access_token", "Admin Access Token"),
				}},
			},
			Capabilities: []model.ProviderCapability{
				unknownCap("shop.order.pull"), unknownCap("shop.product.push"),
			},
		},
	}
}

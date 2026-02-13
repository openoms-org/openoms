package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// PrintTemplatesConfig holds the custom print templates stored in tenant settings.
type PrintTemplatesConfig struct {
	PackingSlipHTML  string `json:"packing_slip_html"`
	OrderSummaryHTML string `json:"order_summary_html"`
	ReturnSlipHTML   string `json:"return_slip_html"`
}

// PrintHandler handles HTML print template endpoints.
type PrintHandler struct {
	tenantRepo repository.TenantRepo
	orderRepo  repository.OrderRepo
	returnRepo repository.ReturnRepo
	pool       *pgxpool.Pool
}

// NewPrintHandler creates a new PrintHandler.
func NewPrintHandler(tenantRepo repository.TenantRepo, orderRepo repository.OrderRepo, returnRepo repository.ReturnRepo, pool *pgxpool.Pool) *PrintHandler {
	return &PrintHandler{tenantRepo: tenantRepo, orderRepo: orderRepo, returnRepo: returnRepo, pool: pool}
}

// --- Template data structs ---

type printItem struct {
	Name     string
	SKU      string
	Quantity int
	Price    string
	Total    string
}

type packingSlipData struct {
	CompanyName     string
	CompanyAddress  string
	CompanyNIP      string
	OrderID         string
	OrderDate       string
	Source          string
	CustomerName    string
	ShippingAddress string
	Items           []printItem
	TotalAmount     string
	Currency        string
	Notes           string
}

type orderSummaryData struct {
	CompanyName     string
	CompanyAddress  string
	CompanyNIP      string
	OrderID         string
	OrderDate       string
	Source          string
	Status          string
	CustomerName    string
	CustomerEmail   string
	CustomerPhone   string
	ShippingAddress string
	BillingAddress  string
	Items           []printItem
	TotalAmount     string
	Currency        string
	PaymentStatus   string
	PaymentMethod   string
	Notes           string
}

type returnSlipData struct {
	CompanyName    string
	CompanyAddress string
	CompanyNIP     string
	ReturnID       string
	OrderID        string
	ReturnDate     string
	Status         string
	Reason         string
	Items          []printItem
	RefundAmount   string
	Notes          string
}

// --- Default templates ---

const defaultPackingSlipTemplate = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>List przewozowy</title>
<style>
body { font-family: Arial, sans-serif; font-size: 12px; margin: 20px; }
.header { display: flex; justify-content: space-between; margin-bottom: 20px; }
.company { font-weight: bold; }
table { width: 100%; border-collapse: collapse; margin-top: 10px; }
th, td { border: 1px solid #ccc; padding: 6px 8px; text-align: left; }
th { background: #f5f5f5; }
.total { text-align: right; font-weight: bold; margin-top: 10px; }
@media print { body { margin: 0; } }
</style></head><body>
<div class="header">
  <div class="company">{{.CompanyName}}<br>{{.CompanyAddress}}<br>NIP: {{.CompanyNIP}}</div>
  <div>Zamówienie: {{.OrderID}}<br>Data: {{.OrderDate}}<br>Źródło: {{.Source}}</div>
</div>
<h3>Dane klienta</h3>
<p>{{.CustomerName}}<br>{{.ShippingAddress}}</p>
<h3>Pozycje</h3>
<table><thead><tr><th>Lp.</th><th>Nazwa</th><th>SKU</th><th>Ilość</th><th>Cena</th><th>Wartość</th></tr></thead>
<tbody>{{range $i, $item := .Items}}<tr><td>{{inc $i}}</td><td>{{$item.Name}}</td><td>{{$item.SKU}}</td><td>{{$item.Quantity}}</td><td>{{$item.Price}}</td><td>{{$item.Total}}</td></tr>{{end}}</tbody></table>
<p class="total">Razem: {{.TotalAmount}} {{.Currency}}</p>
{{if .Notes}}<p><strong>Uwagi:</strong> {{.Notes}}</p>{{end}}
</body></html>`

const defaultOrderSummaryTemplate = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Podsumowanie zamówienia</title>
<style>
body { font-family: Arial, sans-serif; font-size: 12px; margin: 20px; }
.header { display: flex; justify-content: space-between; margin-bottom: 20px; }
.company { font-weight: bold; }
.section { margin-bottom: 15px; }
.section h3 { margin-bottom: 5px; border-bottom: 1px solid #ccc; padding-bottom: 3px; }
table { width: 100%; border-collapse: collapse; margin-top: 10px; }
th, td { border: 1px solid #ccc; padding: 6px 8px; text-align: left; }
th { background: #f5f5f5; }
.total { text-align: right; font-weight: bold; margin-top: 10px; }
.grid { display: flex; gap: 40px; }
.grid div { flex: 1; }
@media print { body { margin: 0; } }
</style></head><body>
<div class="header">
  <div class="company">{{.CompanyName}}<br>{{.CompanyAddress}}<br>NIP: {{.CompanyNIP}}</div>
  <div>Zamówienie: {{.OrderID}}<br>Data: {{.OrderDate}}<br>Status: {{.Status}}</div>
</div>
<div class="section">
<h3>Dane klienta</h3>
<div class="grid">
<div><p><strong>{{.CustomerName}}</strong></p>
{{if .CustomerEmail}}<p>Email: {{.CustomerEmail}}</p>{{end}}
{{if .CustomerPhone}}<p>Tel: {{.CustomerPhone}}</p>{{end}}
</div>
<div>
{{if .ShippingAddress}}<p><strong>Address dostawy:</strong><br>{{.ShippingAddress}}</p>{{end}}
{{if .BillingAddress}}<p><strong>Address rozliczeniowy:</strong><br>{{.BillingAddress}}</p>{{end}}
</div>
</div>
</div>
<div class="section">
<h3>Pozycje</h3>
<table><thead><tr><th>Lp.</th><th>Nazwa</th><th>SKU</th><th>Ilość</th><th>Cena</th><th>Wartość</th></tr></thead>
<tbody>{{range $i, $item := .Items}}<tr><td>{{inc $i}}</td><td>{{$item.Name}}</td><td>{{$item.SKU}}</td><td>{{$item.Quantity}}</td><td>{{$item.Price}}</td><td>{{$item.Total}}</td></tr>{{end}}</tbody></table>
<p class="total">Razem: {{.TotalAmount}} {{.Currency}}</p>
</div>
<div class="section">
<h3>Płatność</h3>
<p>Status: {{.PaymentStatus}}{{if .PaymentMethod}} | Metoda: {{.PaymentMethod}}{{end}}</p>
</div>
{{if .Notes}}<div class="section"><h3>Uwagi</h3><p>{{.Notes}}</p></div>{{end}}
</body></html>`

const defaultReturnSlipTemplate = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Formularz zwrotu</title>
<style>
body { font-family: Arial, sans-serif; font-size: 12px; margin: 20px; }
.header { display: flex; justify-content: space-between; margin-bottom: 20px; }
.company { font-weight: bold; }
table { width: 100%; border-collapse: collapse; margin-top: 10px; }
th, td { border: 1px solid #ccc; padding: 6px 8px; text-align: left; }
th { background: #f5f5f5; }
.total { text-align: right; font-weight: bold; margin-top: 10px; }
@media print { body { margin: 0; } }
</style></head><body>
<div class="header">
  <div class="company">{{.CompanyName}}<br>{{.CompanyAddress}}<br>NIP: {{.CompanyNIP}}</div>
  <div>Zwrot: {{.ReturnID}}<br>Zamówienie: {{.OrderID}}<br>Data: {{.ReturnDate}}<br>Status: {{.Status}}</div>
</div>
<h3>Powód zwrotu</h3>
<p>{{.Reason}}</p>
{{if .Items}}<h3>Pozycje</h3>
<table><thead><tr><th>Lp.</th><th>Nazwa</th><th>SKU</th><th>Ilość</th></tr></thead>
<tbody>{{range $i, $item := .Items}}<tr><td>{{inc $i}}</td><td>{{$item.Name}}</td><td>{{$item.SKU}}</td><td>{{$item.Quantity}}</td></tr>{{end}}</tbody></table>{{end}}
<p class="total">Kwota zwrotu: {{.RefundAmount}}</p>
{{if .Notes}}<p><strong>Uwagi:</strong> {{.Notes}}</p>{{end}}
</body></html>`

// templateFuncs provides helper functions for templates.
var templateFuncs = template.FuncMap{
	"inc": func(i int) int { return i + 1 },
}

// --- Helpers ---

func (h *PrintHandler) getSettingsSection(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, key string, dest any) error {
	settings, err := h.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil {
		return err
	}
	if settings == nil {
		return nil
	}
	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(settings, &allSettings); err != nil {
		return nil
	}
	raw, ok := allSettings[key]
	if !ok {
		return nil
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		slog.Warn("failed to unmarshal print settings section", "key", key, "error", err)
	}
	return nil
}

func (h *PrintHandler) updateSettingsSection(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, key string, value any) error {
	existing, err := h.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil {
		return err
	}
	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(existing, &allSettings); err != nil {
		allSettings = make(map[string]json.RawMessage)
	}
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}
	allSettings[key] = valueJSON
	newSettings, err := json.Marshal(allSettings)
	if err != nil {
		return err
	}
	return h.tenantRepo.UpdateSettings(ctx, tx, tenantID, newSettings)
}

func (h *PrintHandler) loadCompanySettings(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) model.CompanySettings {
	var cs model.CompanySettings
	h.getSettingsSection(ctx, tx, tenantID, "company", &cs) //nolint:errcheck
	return cs
}

func (h *PrintHandler) loadPrintTemplates(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) PrintTemplatesConfig {
	var cfg PrintTemplatesConfig
	h.getSettingsSection(ctx, tx, tenantID, "print_templates", &cfg) //nolint:errcheck
	return cfg
}

func renderTemplate(tmplStr string, data any) ([]byte, error) {
	tmpl, err := template.New("print").Funcs(templateFuncs).Parse(tmplStr)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return buf.Bytes(), nil
}

func formatAddress(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var addr map[string]any
	if err := json.Unmarshal(raw, &addr); err != nil {
		return ""
	}
	parts := []string{}
	for _, key := range []string{"name", "street", "line1", "line2", "city", "post_code", "postal_code", "country"} {
		if v, ok := addr[key]; ok && v != nil {
			s := fmt.Sprintf("%v", v)
			if s != "" {
				parts = append(parts, s)
			}
		}
	}
	var result strings.Builder
	for i, p := range parts {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(p)
	}
	return result.String()
}

func parseOrderItems(raw json.RawMessage) []printItem {
	if len(raw) == 0 {
		return nil
	}
	var items []struct {
		Name     string  `json:"name"`
		SKU      string  `json:"sku"`
		Quantity int     `json:"quantity"`
		Price    float64 `json:"price"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	result := make([]printItem, len(items))
	for i, it := range items {
		result[i] = printItem{
			Name:     it.Name,
			SKU:      it.SKU,
			Quantity: it.Quantity,
			Price:    fmt.Sprintf("%.2f", it.Price),
			Total:    fmt.Sprintf("%.2f", it.Price*float64(it.Quantity)),
		}
	}
	return result
}

func parseReturnItems(raw json.RawMessage) []printItem {
	if len(raw) == 0 {
		return nil
	}
	var items []struct {
		Name     string `json:"name"`
		SKU      string `json:"sku"`
		Quantity int    `json:"quantity"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	result := make([]printItem, len(items))
	for i, it := range items {
		result[i] = printItem{
			Name:     it.Name,
			SKU:      it.SKU,
			Quantity: it.Quantity,
		}
	}
	return result
}

func writeHTML(w http.ResponseWriter, html []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.WriteHeader(http.StatusOK)
	w.Write(html) //nolint:errcheck
}

func shortUUID(id uuid.UUID) string {
	s := id.String()
	if len(s) >= 8 {
		return s[:8]
	}
	return s
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// --- Handlers ---

// GetPackingSlip renders an HTML packing slip for an order.
func (h *PrintHandler) GetPackingSlip(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var html []byte
	err = database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		order, err := h.orderRepo.FindByID(r.Context(), tx, orderID)
		if err != nil {
			return err
		}
		if order == nil {
			return fmt.Errorf("not_found")
		}

		company := h.loadCompanySettings(r.Context(), tx, tenantID)
		templates := h.loadPrintTemplates(r.Context(), tx, tenantID)

		companyAddr := company.Address
		if company.City != "" {
			companyAddr += ", " + company.City
		}
		if company.PostCode != "" {
			companyAddr += " " + company.PostCode
		}

		data := packingSlipData{
			CompanyName:     company.CompanyName,
			CompanyAddress:  companyAddr,
			CompanyNIP:      company.NIP,
			OrderID:         shortUUID(order.ID),
			OrderDate:       order.CreatedAt.Format("2006-01-02"),
			Source:          order.Source,
			CustomerName:    order.CustomerName,
			ShippingAddress: formatAddress(order.ShippingAddress),
			Items:           parseOrderItems(order.Items),
			TotalAmount:     fmt.Sprintf("%.2f", order.TotalAmount),
			Currency:        order.Currency,
			Notes:           derefStr(order.Notes),
		}

		tmplStr := defaultPackingSlipTemplate
		if templates.PackingSlipHTML != "" {
			tmplStr = templates.PackingSlipHTML
		}

		html, err = renderTemplate(tmplStr, data)
		return err
	})

	if err != nil {
		if err.Error() == "not_found" {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to render packing slip")
		return
	}

	writeHTML(w, html)
}

// GetOrderSummary renders a printable order summary.
func (h *PrintHandler) GetOrderSummary(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var html []byte
	err = database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		order, err := h.orderRepo.FindByID(r.Context(), tx, orderID)
		if err != nil {
			return err
		}
		if order == nil {
			return fmt.Errorf("not_found")
		}

		company := h.loadCompanySettings(r.Context(), tx, tenantID)
		templates := h.loadPrintTemplates(r.Context(), tx, tenantID)

		companyAddr := company.Address
		if company.City != "" {
			companyAddr += ", " + company.City
		}
		if company.PostCode != "" {
			companyAddr += " " + company.PostCode
		}

		data := orderSummaryData{
			CompanyName:     company.CompanyName,
			CompanyAddress:  companyAddr,
			CompanyNIP:      company.NIP,
			OrderID:         shortUUID(order.ID),
			OrderDate:       order.CreatedAt.Format("2006-01-02"),
			Source:          order.Source,
			Status:          order.Status,
			CustomerName:    order.CustomerName,
			CustomerEmail:   derefStr(order.CustomerEmail),
			CustomerPhone:   derefStr(order.CustomerPhone),
			ShippingAddress: formatAddress(order.ShippingAddress),
			BillingAddress:  formatAddress(order.BillingAddress),
			Items:           parseOrderItems(order.Items),
			TotalAmount:     fmt.Sprintf("%.2f", order.TotalAmount),
			Currency:        order.Currency,
			PaymentStatus:   order.PaymentStatus,
			PaymentMethod:   derefStr(order.PaymentMethod),
			Notes:           derefStr(order.Notes),
		}

		tmplStr := defaultOrderSummaryTemplate
		if templates.OrderSummaryHTML != "" {
			tmplStr = templates.OrderSummaryHTML
		}

		html, err = renderTemplate(tmplStr, data)
		return err
	})

	if err != nil {
		if err.Error() == "not_found" {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to render order summary")
		return
	}

	writeHTML(w, html)
}

// GetReturnSlip renders a printable return form.
func (h *PrintHandler) GetReturnSlip(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	returnID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid return ID")
		return
	}

	var html []byte
	err = database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		ret, err := h.returnRepo.FindByID(r.Context(), tx, returnID)
		if err != nil {
			return err
		}
		if ret == nil {
			return fmt.Errorf("not_found")
		}

		company := h.loadCompanySettings(r.Context(), tx, tenantID)
		templates := h.loadPrintTemplates(r.Context(), tx, tenantID)

		companyAddr := company.Address
		if company.City != "" {
			companyAddr += ", " + company.City
		}
		if company.PostCode != "" {
			companyAddr += " " + company.PostCode
		}

		data := returnSlipData{
			CompanyName:    company.CompanyName,
			CompanyAddress: companyAddr,
			CompanyNIP:     company.NIP,
			ReturnID:       shortUUID(ret.ID),
			OrderID:        shortUUID(ret.OrderID),
			ReturnDate:     ret.CreatedAt.Format("2006-01-02"),
			Status:         ret.Status,
			Reason:         ret.Reason,
			Items:          parseReturnItems(ret.Items),
			RefundAmount:   fmt.Sprintf("%.2f", ret.RefundAmount),
			Notes:          derefStr(ret.Notes),
		}

		tmplStr := defaultReturnSlipTemplate
		if templates.ReturnSlipHTML != "" {
			tmplStr = templates.ReturnSlipHTML
		}

		html, err = renderTemplate(tmplStr, data)
		return err
	})

	if err != nil {
		if err.Error() == "not_found" {
			writeError(w, http.StatusNotFound, "return not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to render return slip")
		return
	}

	writeHTML(w, html)
}

// GetPrintTemplates returns the current print templates configuration.
func (h *PrintHandler) GetPrintTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var cfg PrintTemplatesConfig
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		cfg = h.loadPrintTemplates(r.Context(), tx, tenantID)
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load print templates")
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

// UpdatePrintTemplates saves the print templates configuration.
func (h *PrintHandler) UpdatePrintTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var cfg PrintTemplatesConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate templates by trying to parse them
	if cfg.PackingSlipHTML != "" {
		if _, err := template.New("test").Funcs(templateFuncs).Parse(cfg.PackingSlipHTML); err != nil {
			writeError(w, http.StatusBadRequest, "invalid packing slip template: "+err.Error())
			return
		}
	}
	if cfg.OrderSummaryHTML != "" {
		if _, err := template.New("test").Funcs(templateFuncs).Parse(cfg.OrderSummaryHTML); err != nil {
			writeError(w, http.StatusBadRequest, "invalid order summary template: "+err.Error())
			return
		}
	}
	if cfg.ReturnSlipHTML != "" {
		if _, err := template.New("test").Funcs(templateFuncs).Parse(cfg.ReturnSlipHTML); err != nil {
			writeError(w, http.StatusBadRequest, "invalid return slip template: "+err.Error())
			return
		}
	}

	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		return h.updateSettingsSection(r.Context(), tx, tenantID, "print_templates", cfg)
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save print templates")
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

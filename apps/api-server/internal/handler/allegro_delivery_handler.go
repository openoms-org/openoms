package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// AllegroDeliveryHandler handles Allegro delivery settings and shipping rate endpoints.
type AllegroDeliveryHandler struct {
	integrationService *service.IntegrationService
	encryptionKey      []byte
}

// NewAllegroDeliveryHandler creates a new AllegroDeliveryHandler.
func NewAllegroDeliveryHandler(integrationService *service.IntegrationService, encryptionKey []byte) *AllegroDeliveryHandler {
	return &AllegroDeliveryHandler{
		integrationService: integrationService,
		encryptionKey:      encryptionKey,
	}
}

// newAllegroClient creates an authenticated Allegro SDK client from the integration credentials.
func (h *AllegroDeliveryHandler) newAllegroClient(r *http.Request) (*allegrosdk.Client, error) {
	return buildAllegroClient(r, h.integrationService, h.encryptionKey)
}

// GetDeliverySettings retrieves the seller's delivery settings.
// GET /v1/integrations/allegro/delivery-settings
func (h *AllegroDeliveryHandler) GetDeliverySettings(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro delivery: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	settings, err := client.DeliverySettings.Get(r.Context())
	if err != nil {
		slog.Error("allegro delivery: failed to get delivery settings", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac ustawien dostawy")
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

// UpdateDeliverySettings updates the seller's delivery settings.
// PUT /v1/integrations/allegro/delivery-settings
func (h *AllegroDeliveryHandler) UpdateDeliverySettings(w http.ResponseWriter, r *http.Request) {
	var body allegrosdk.DeliverySettings
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidlowe dane wejsciowe")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro delivery: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	if err := client.DeliverySettings.Update(r.Context(), body); err != nil {
		slog.Error("allegro delivery: failed to update delivery settings", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie zaktualizowac ustawien dostawy")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// ListShippingRates lists the seller's shipping rate tables.
// GET /v1/integrations/allegro/shipping-rates
func (h *AllegroDeliveryHandler) ListShippingRates(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro delivery: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	rates, err := client.DeliverySettings.ListShippingRates(r.Context())
	if err != nil {
		slog.Error("allegro delivery: failed to list shipping rates", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac cennikow wysylki")
		return
	}

	writeJSON(w, http.StatusOK, rates)
}

// GetShippingRate retrieves a single shipping rate table by ID.
// GET /v1/integrations/allegro/shipping-rates/{rateId}
func (h *AllegroDeliveryHandler) GetShippingRate(w http.ResponseWriter, r *http.Request) {
	rateID := chi.URLParam(r, "rateId")
	if rateID == "" {
		writeError(w, http.StatusBadRequest, "rateId jest wymagane")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro delivery: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	rate, err := client.DeliverySettings.GetShippingRate(r.Context(), rateID)
	if err != nil {
		slog.Error("allegro delivery: failed to get shipping rate", "error", err, "rate_id", rateID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac cennika wysylki")
		return
	}

	writeJSON(w, http.StatusOK, rate)
}

// CreateShippingRate creates a new shipping rate table.
// POST /v1/integrations/allegro/shipping-rates
func (h *AllegroDeliveryHandler) CreateShippingRate(w http.ResponseWriter, r *http.Request) {
	var body allegrosdk.CreateShippingRateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidlowe dane wejsciowe")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro delivery: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	rate, err := client.DeliverySettings.CreateShippingRate(r.Context(), body)
	if err != nil {
		slog.Error("allegro delivery: failed to create shipping rate", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie utworzyc cennika wysylki")
		return
	}

	writeJSON(w, http.StatusCreated, rate)
}

// UpdateShippingRate updates an existing shipping rate table.
// PUT /v1/integrations/allegro/shipping-rates/{rateId}
func (h *AllegroDeliveryHandler) UpdateShippingRate(w http.ResponseWriter, r *http.Request) {
	rateID := chi.URLParam(r, "rateId")
	if rateID == "" {
		writeError(w, http.StatusBadRequest, "rateId jest wymagane")
		return
	}

	var body allegrosdk.CreateShippingRateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidlowe dane wejsciowe")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro delivery: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	rate, err := client.DeliverySettings.UpdateShippingRate(r.Context(), rateID, body)
	if err != nil {
		slog.Error("allegro delivery: failed to update shipping rate", "error", err, "rate_id", rateID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie zaktualizowac cennika wysylki")
		return
	}

	writeJSON(w, http.StatusOK, rate)
}

// ListDeliveryMethods lists all available Allegro delivery methods.
// GET /v1/integrations/allegro/delivery-methods
func (h *AllegroDeliveryHandler) ListDeliveryMethods(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro delivery: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	methods, err := client.DeliverySettings.ListDeliveryMethods(r.Context())
	if err != nil {
		slog.Error("allegro delivery: failed to list delivery methods", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac metod dostawy")
		return
	}

	writeJSON(w, http.StatusOK, methods)
}

// autoGenerateShippingRateRequest is the payload for auto-generating a shipping rate from InPost pricing.
type autoGenerateShippingRateRequest struct {
	WeightKg float64 `json:"weight_kg"`
	WidthCm  float64 `json:"width_cm"`
	HeightCm float64 `json:"height_cm"`
	LengthCm float64 `json:"length_cm"`
	Name     string  `json:"name"` // optional name for the rate table
}

// AutoGenerateShippingRate creates a shipping rate table on Allegro based on InPost pricing
// for the given product dimensions.
// POST /v1/integrations/allegro/shipping-rates/auto-generate
func (h *AllegroDeliveryHandler) AutoGenerateShippingRate(w http.ResponseWriter, r *http.Request) {
	var body autoGenerateShippingRateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidlowe dane wejsciowe")
		return
	}

	if body.WeightKg <= 0 {
		writeError(w, http.StatusBadRequest, "Waga produktu jest wymagana")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro delivery: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	// 1. Fetch Allegro delivery methods and find InPost ones
	methods, err := client.DeliverySettings.ListDeliveryMethods(r.Context())
	if err != nil {
		slog.Error("allegro delivery: failed to list delivery methods", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac metod dostawy Allegro")
		return
	}

	// Find InPost delivery methods (prepaid only, PLN, domestic)
	type inpostMethod struct {
		id          string
		isLocker    bool
		constraints *allegrosdk.ShippingConstraints
	}
	var inpostMethods []inpostMethod
	for _, m := range methods.DeliveryMethods {
		if m.PaymentPolicy != "IN_ADVANCE" {
			continue
		}
		if m.ShippingRatesConstraints != nil && !m.ShippingRatesConstraints.Allowed {
			continue
		}
		nameLower := strings.ToLower(m.Name)
		// Skip international methods
		if strings.Contains(nameLower, "international") || strings.Contains(nameLower, "czechy") {
			continue
		}
		// Skip non-PLN currency methods
		if m.ShippingRatesConstraints != nil && m.ShippingRatesConstraints.FirstItemRate != nil {
			if m.ShippingRatesConstraints.FirstItemRate.Currency != "" && m.ShippingRatesConstraints.FirstItemRate.Currency != "PLN" {
				continue
			}
		}
		isInPost := strings.Contains(nameLower, "inpost")
		isPaczkomat := strings.Contains(nameLower, "paczkomat") || strings.Contains(nameLower, "paczkomaty") || strings.Contains(nameLower, "paczkopunkt")
		isKurier := strings.Contains(nameLower, "kurier")
		if isInPost && (isPaczkomat || isKurier) {
			inpostMethods = append(inpostMethods, inpostMethod{
				id:          m.ID,
				isLocker:    isPaczkomat,
				constraints: m.ShippingRatesConstraints,
			})
		}
	}

	if len(inpostMethods) == 0 {
		writeError(w, http.StatusBadRequest, "Nie znaleziono metod dostawy InPost na Allegro. Sprawdz czy InPost jest dostepny w metodach dostawy.")
		return
	}

	// 2. Classify product into InPost sizes and get prices
	w2 := body.WeightKg
	width := body.WidthCm
	height := body.HeightCm
	length := body.LengthCm

	fitsA := w2 <= 8 && width <= 38 && height <= 8 && length <= 64
	fitsB := w2 <= 25 && width <= 38 && height <= 19 && length <= 64
	fitsC := w2 <= 25 && width <= 41 && height <= 38 && length <= 64

	// 3. Build shipping rate entries respecting Allegro constraints
	var rates []allegrosdk.ShippingRateEntry

	for _, m := range inpostMethods {
		var basePrice float64
		if m.isLocker {
			// Best paczkomat price (smallest fitting size)
			if fitsA {
				basePrice = 12.99
			} else if fitsB {
				basePrice = 13.99
			} else if fitsC {
				basePrice = 15.49
			}
		} else {
			// Courier
			if w2 <= 25 {
				basePrice = 16.99
				if w2 > 10 {
					basePrice = 19.99
				}
			}
		}
		if basePrice == 0 {
			continue
		}

		// Cap price at Allegro's max constraint
		firstPrice := basePrice
		if m.constraints != nil && m.constraints.FirstItemRate != nil && m.constraints.FirstItemRate.Max != "" {
			if maxVal, err := strconv.ParseFloat(m.constraints.FirstItemRate.Max, 64); err == nil && firstPrice > maxVal {
				firstPrice = maxVal
			}
		}

		// Next item rate: respect max (usually 0.00 for endorsed methods)
		nextPrice := 0.00
		if m.constraints != nil && m.constraints.NextItemRate != nil && m.constraints.NextItemRate.Max != "" {
			if maxVal, err := strconv.ParseFloat(m.constraints.NextItemRate.Max, 64); err == nil {
				nextPrice = math.Min(firstPrice, maxVal)
			}
		}

		entry := allegrosdk.ShippingRateEntry{
			DeliveryMethod:        allegrosdk.ShippingDeliveryMethod{ID: m.id},
			MaxQuantityPerPackage: 1,
			FirstItemRate:         allegrosdk.Amount{Amount: fmt.Sprintf("%.2f", firstPrice), Currency: "PLN"},
			NextItemRate:          allegrosdk.Amount{Amount: fmt.Sprintf("%.2f", nextPrice), Currency: "PLN"},
		}

		// Only set shipping time if method allows customization
		if m.constraints != nil && m.constraints.ShippingTime != nil && m.constraints.ShippingTime.Customizable {
			if m.isLocker {
				entry.ShippingTime = &allegrosdk.ShippingTime{From: "PT24H", To: "PT72H"}
			} else {
				entry.ShippingTime = &allegrosdk.ShippingTime{From: "PT24H", To: "PT48H"}
			}
		}

		rates = append(rates, entry)
	}

	if len(rates) == 0 {
		writeError(w, http.StatusBadRequest, "Produkt nie miesci sie w zadnym rozmiarze przesylki InPost (maks. 25 kg, 41x38x64 cm)")
		return
	}

	// 4. Create shipping rate on Allegro
	rateName := body.Name
	if rateName == "" {
		rateName = "InPost auto"
	}

	result, err := client.DeliverySettings.CreateShippingRate(r.Context(), allegrosdk.CreateShippingRateRequest{
		Name:  rateName,
		Rates: rates,
	})
	if err != nil {
		slog.Error("allegro delivery: failed to create auto shipping rate", "error", err)
		writeError(w, http.StatusBadGateway, allegroErrorMessage("Nie udalo sie utworzyc cennika wysylki", err))
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

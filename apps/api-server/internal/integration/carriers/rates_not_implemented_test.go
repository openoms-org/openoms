package carriers

import (
	"context"
	"errors"
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

func TestCarrierGetRates_NotImplementedProviders(t *testing.T) {
	tests := []struct {
		name     string
		provider integration.CarrierProvider
	}{
		{name: "ups", provider: &UPSProvider{}},
		{name: "poczta_polska", provider: &PocztaPolskaProvider{}},
		{name: "orlen_paczka", provider: &OrlenPaczkaProvider{}},
		{name: "fedex", provider: &FedExProvider{}},
	}

	req := integration.RateRequest{
		FromCountry: "PL",
		ToCountry:   "PL",
		Weight:      5,
		COD:         100,
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rates, err := tc.provider.GetRates(context.Background(), req)
			if !errors.Is(err, integration.ErrCarrierRatesNotImplemented) {
				t.Fatalf("GetRates() error = %v, want ErrCarrierRatesNotImplemented", err)
			}
			if rates != nil {
				t.Fatalf("GetRates() rates = %#v, want nil", rates)
			}
		})
	}
}

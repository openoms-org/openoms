package inpost

import (
	"errors"
	"testing"
)

func TestValidateCreateShipment(t *testing.T) {
	tests := []struct {
		name    string
		req     *CreateShipmentRequest
		wantErr error
	}{
		{
			name: "valid locker shipment",
			req: &CreateShipmentRequest{
				Receiver: Receiver{Name: "Jan", Phone: "500100200"},
				Parcels:  []Parcel{{Template: ParcelSmall, Weight: Weight{Amount: 2.0, Unit: "kg"}}},
				Service:  ServiceLockerStandard,
				CustomAttributes: &CustomAttributes{
					TargetPoint: "KRA010",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid courier shipment",
			req: &CreateShipmentRequest{
				Receiver: Receiver{
					Name:  "Anna",
					Phone: "600200300",
					Address: &Address{
						Street:      "ul. Testowa 1",
						City:        "Krakow",
						PostCode:    "30-001",
						CountryCode: "PL",
					},
				},
				Parcels: []Parcel{{Template: ParcelMedium, Weight: Weight{Amount: 10.0, Unit: "kg"}}},
				Service: ServiceCourierStandard,
			},
			wantErr: nil,
		},
		{
			name: "missing receiver name",
			req: &CreateShipmentRequest{
				Receiver: Receiver{Phone: "500100200"},
				Parcels:  []Parcel{{Weight: Weight{Amount: 1.0, Unit: "kg"}}},
				Service:  ServiceLockerStandard,
				CustomAttributes: &CustomAttributes{
					TargetPoint: "KRA010",
				},
			},
			wantErr: ErrMissingReceiver,
		},
		{
			name: "missing receiver phone",
			req: &CreateShipmentRequest{
				Receiver: Receiver{Name: "Jan"},
				Parcels:  []Parcel{{Weight: Weight{Amount: 1.0, Unit: "kg"}}},
				Service:  ServiceLockerStandard,
				CustomAttributes: &CustomAttributes{
					TargetPoint: "KRA010",
				},
			},
			wantErr: ErrMissingReceiver,
		},
		{
			name: "weight exceeded",
			req: &CreateShipmentRequest{
				Receiver: Receiver{Name: "Jan", Phone: "500100200"},
				Parcels:  []Parcel{{Template: ParcelLarge, Weight: Weight{Amount: 30.0, Unit: "kg"}}},
				Service:  ServiceLockerStandard,
				CustomAttributes: &CustomAttributes{
					TargetPoint: "KRA010",
				},
			},
			wantErr: ErrWeightExceeded,
		},
		{
			name: "missing target_point for locker",
			req: &CreateShipmentRequest{
				Receiver: Receiver{Name: "Jan", Phone: "500100200"},
				Parcels:  []Parcel{{Template: ParcelSmall, Weight: Weight{Amount: 1.0, Unit: "kg"}}},
				Service:  ServiceLockerStandard,
			},
			wantErr: ErrMissingTargetPoint,
		},
		{
			name: "missing address for courier",
			req: &CreateShipmentRequest{
				Receiver: Receiver{Name: "Anna", Phone: "600200300"},
				Parcels:  []Parcel{{Template: ParcelSmall, Weight: Weight{Amount: 1.0, Unit: "kg"}}},
				Service:  ServiceCourierStandard,
			},
			wantErr: ErrMissingAddress,
		},
		{
			name: "invalid template",
			req: &CreateShipmentRequest{
				Receiver: Receiver{Name: "Jan", Phone: "500100200"},
				Parcels:  []Parcel{{Template: ParcelTemplate("xlarge"), Weight: Weight{Amount: 1.0, Unit: "kg"}}},
				Service:  ServiceLockerStandard,
				CustomAttributes: &CustomAttributes{
					TargetPoint: "KRA010",
				},
			},
			wantErr: ErrInvalidTemplate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateShipment(tt.req)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.wantErr)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

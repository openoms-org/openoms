package inpost

import "errors"

const MaxParcelWeightKg = 25.0

// ParcelDimensions maps standard templates to their maximum dimensions in mm.
var ParcelDimensions = map[ParcelTemplate]Dimensions{
	ParcelSmall:  {Height: 80, Width: 380, Length: 640},
	ParcelMedium: {Height: 190, Width: 380, Length: 640},
	ParcelLarge:  {Height: 410, Width: 380, Length: 640},
}

var (
	ErrWeightExceeded     = errors.New("inpost: parcel weight exceeds 25 kg")
	ErrInvalidTemplate    = errors.New("inpost: invalid parcel template")
	ErrMissingTargetPoint = errors.New("inpost: target_point required for locker delivery")
	ErrMissingAddress     = errors.New("inpost: address required for courier delivery")
	ErrMissingReceiver    = errors.New("inpost: receiver name and phone are required")
)

// ValidateCreateShipment checks a CreateShipmentRequest for common errors
// before sending it to the API.
func ValidateCreateShipment(req *CreateShipmentRequest) error {
	if req.Receiver.Name == "" || req.Receiver.Phone == "" {
		return ErrMissingReceiver
	}

	switch req.Service {
	case ServiceLockerStandard:
		if req.CustomAttributes == nil || req.CustomAttributes.TargetPoint == "" {
			return ErrMissingTargetPoint
		}
	case ServiceCourierStandard:
		if req.Receiver.Address == nil {
			return ErrMissingAddress
		}
	}

	for _, p := range req.Parcels {
		if p.Weight.Amount > MaxParcelWeightKg {
			return ErrWeightExceeded
		}

		if p.Template != "" {
			if _, ok := ParcelDimensions[p.Template]; !ok {
				return ErrInvalidTemplate
			}
		}
	}

	return nil
}

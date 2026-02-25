package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestInvoiceService_SetKSeFService(t *testing.T) {
	svc := NewInvoiceService(nil, nil, nil, nil, nil, nil)
	assert.Nil(t, svc.ksefService)

	ksefSvc := NewKSeFService(nil, nil, nil, nil, nil)
	svc.SetKSeFService(ksefSvc)
	assert.NotNil(t, svc.ksefService)
	assert.Equal(t, ksefSvc, svc.ksefService)
}

func TestInvoiceService_TriggerKSeFAutoSend_NilKSeFService(t *testing.T) {
	// Should not panic when ksefService is nil — this is the normal state
	// before SetKSeFService is called during wiring.
	svc := NewInvoiceService(nil, nil, nil, nil, nil, nil)
	assert.NotPanics(t, func() {
		svc.triggerKSeFAutoSend(uuid.New(), uuid.New())
	})
}

func TestInvoiceService_SetKSeFService_Overwrite(t *testing.T) {
	svc := NewInvoiceService(nil, nil, nil, nil, nil, nil)

	ksefSvc1 := NewKSeFService(nil, nil, nil, nil, nil)
	svc.SetKSeFService(ksefSvc1)
	assert.Equal(t, ksefSvc1, svc.ksefService)

	ksefSvc2 := NewKSeFService(nil, nil, nil, nil, nil)
	svc.SetKSeFService(ksefSvc2)
	assert.Equal(t, ksefSvc2, svc.ksefService)
}

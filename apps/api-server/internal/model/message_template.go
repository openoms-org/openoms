package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// MessageTemplate represents a reusable message template for marketplace messaging.
type MessageTemplate struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	Name            string    `json:"name"`
	Channel         string    `json:"channel"`
	Subject         *string   `json:"subject,omitempty"`
	Body            string    `json:"body"`
	Variables       []string  `json:"variables"`
	IsAutoresponder bool      `json:"is_autoresponder"`
	TriggerEvent    *string   `json:"trigger_event,omitempty"`
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateMessageTemplateRequest is the payload for creating a message template.
type CreateMessageTemplateRequest struct {
	Name            string   `json:"name"`
	Channel         string   `json:"channel"`
	Subject         *string  `json:"subject,omitempty"`
	Body            string   `json:"body"`
	Variables       []string `json:"variables"`
	IsAutoresponder bool     `json:"is_autoresponder"`
	TriggerEvent    *string  `json:"trigger_event,omitempty"`
	Enabled         *bool    `json:"enabled,omitempty"`
}

// Maximum body length for message templates (50 KB).
const MaxTemplateBodyLength = 50000

// Validate validates the create message template request.
func (r *CreateMessageTemplateRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	if err := validateMaxLength("name", r.Name, MaxNameLength); err != nil {
		return err
	}
	if r.Body == "" {
		return errors.New("body is required")
	}
	if err := validateMaxLength("body", r.Body, MaxTemplateBodyLength); err != nil {
		return err
	}
	if r.Channel == "" {
		return errors.New("channel is required")
	}
	validChannels := map[string]bool{"allegro": true, "email": true, "sms": true}
	if !validChannels[r.Channel] {
		return errors.New("channel must be allegro, email, or sms")
	}
	return nil
}

// UpdateMessageTemplateRequest is the payload for updating a message template.
type UpdateMessageTemplateRequest struct {
	Name            *string  `json:"name,omitempty"`
	Channel         *string  `json:"channel,omitempty"`
	Subject         *string  `json:"subject,omitempty"`
	Body            *string  `json:"body,omitempty"`
	Variables       []string `json:"variables,omitempty"`
	IsAutoresponder *bool    `json:"is_autoresponder,omitempty"`
	TriggerEvent    *string  `json:"trigger_event,omitempty"`
	Enabled         *bool    `json:"enabled,omitempty"`
}

// Validate validates the update message template request.
func (r *UpdateMessageTemplateRequest) Validate() error {
	if r.Name != nil && *r.Name == "" {
		return errors.New("name must not be empty")
	}
	if r.Name != nil {
		if err := validateMaxLength("name", *r.Name, MaxNameLength); err != nil {
			return err
		}
	}
	if r.Channel != nil {
		validChannels := map[string]bool{"allegro": true, "email": true, "sms": true}
		if !validChannels[*r.Channel] {
			return errors.New("channel must be allegro, email, or sms")
		}
	}
	if r.Body != nil && *r.Body == "" {
		return errors.New("body must not be empty")
	}
	if err := validateMaxLengthPtr("body", r.Body, MaxTemplateBodyLength); err != nil {
		return err
	}
	return nil
}

// MessageTemplateListFilter holds the filtering/pagination for listing message templates.
type MessageTemplateListFilter struct {
	PaginationParams
	Channel *string
	Enabled *bool
}

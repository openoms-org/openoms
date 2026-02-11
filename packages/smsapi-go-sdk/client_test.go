package smsapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendSMS_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/sms.do" {
			t.Errorf("expected /sms.do, got %s", r.URL.Path)
		}

		// Verify Authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", auth)
		}

		// Verify Content-Type
		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded" {
			t.Errorf("expected application/x-www-form-urlencoded, got %s", ct)
		}

		// Parse form body
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("to") != "48123456789" {
			t.Errorf("expected to=48123456789, got %s", r.FormValue("to"))
		}
		if r.FormValue("message") != "Testowa wiadomosc SMS" {
			t.Errorf("expected message=Testowa wiadomosc SMS, got %s", r.FormValue("message"))
		}
		if r.FormValue("from") != "OpenOMS" {
			t.Errorf("expected from=OpenOMS, got %s", r.FormValue("from"))
		}
		if r.FormValue("format") != "json" {
			t.Errorf("expected format=json, got %s", r.FormValue("format"))
		}
		if r.FormValue("encoding") != "utf-8" {
			t.Errorf("expected encoding=utf-8, got %s", r.FormValue("encoding"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SendSMSResponse{
			Count: 1,
			List: []SMSResult{
				{
					ID:     "abc123",
					Points: 0.16,
					Number: "48123456789",
					Status: "QUEUE",
				},
			},
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL), WithFrom("OpenOMS"))

	result, err := client.SendSMS(context.Background(), SendSMSRequest{
		To:      "48123456789",
		Message: "Testowa wiadomosc SMS",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 1 {
		t.Errorf("expected count 1, got %d", result.Count)
	}
	if len(result.List) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.List))
	}
	if result.List[0].ID != "abc123" {
		t.Errorf("expected ID abc123, got %s", result.List[0].ID)
	}
	if result.List[0].Status != "QUEUE" {
		t.Errorf("expected status QUEUE, got %s", result.List[0].Status)
	}
	if result.List[0].Number != "48123456789" {
		t.Errorf("expected number 48123456789, got %s", result.List[0].Number)
	}
}

func TestSendSMS_WithRequestFrom(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		// Request-level from should override client default
		if r.FormValue("from") != "CustomSender" {
			t.Errorf("expected from=CustomSender, got %s", r.FormValue("from"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SendSMSResponse{
			Count: 1,
			List: []SMSResult{
				{ID: "def456", Points: 0.16, Number: "48987654321", Status: "QUEUE"},
			},
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL), WithFrom("DefaultSender"))

	_, err := client.SendSMS(context.Background(), SendSMSRequest{
		To:      "48987654321",
		Message: "Test z nadawca",
		From:    "CustomSender",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendSMS_APIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   101,
			"message": "Authorization failed",
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient("bad-token", WithBaseURL(server.URL))

	_, err := client.SendSMS(context.Background(), SendSMSRequest{
		To:      "48123456789",
		Message: "Test",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "Authorization failed" {
		t.Errorf("expected message 'Authorization failed', got %s", apiErr.Message)
	}
}

func TestSendSMS_MessageError(t *testing.T) {
	errorCode := 13
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SendSMSResponse{
			Count: 1,
			List: []SMSResult{
				{
					ID:     "err123",
					Number: "48123456789",
					Status: "ERROR",
					Error:  &errorCode,
				},
			},
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL))

	result, err := client.SendSMS(context.Background(), SendSMSRequest{
		To:      "48123456789",
		Message: "Test",
	})
	if err == nil {
		t.Fatal("expected error for message-level error, got nil")
	}
	if result == nil {
		t.Fatal("expected result even with message-level error")
	}
}

func TestSendSMS_ValidationErrors(t *testing.T) {
	client := NewClient("test-token")

	// Missing "to"
	_, err := client.SendSMS(context.Background(), SendSMSRequest{
		Message: "Test",
	})
	if err == nil {
		t.Fatal("expected error for missing 'to', got nil")
	}

	// Missing "message"
	_, err = client.SendSMS(context.Background(), SendSMSRequest{
		To: "48123456789",
	})
	if err == nil {
		t.Fatal("expected error for missing 'message', got nil")
	}
}

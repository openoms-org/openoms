package apaczka

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetServiceStructure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{
			"status": 200,
			"response": {
				"services": [
					{
						"service_id": "svc1",
						"name": "DHL Domestic",
						"supplier": "DHL",
						"delivery_time": "1",
						"domestic": "1",
						"door_to_door": "1",
						"door_to_point": "0",
						"point_to_point": "0",
						"point_to_door": "0"
					},
					{
						"service_id": "svc2",
						"name": "InPost Locker",
						"supplier": "INPOST",
						"delivery_time": "2",
						"domestic": "1",
						"door_to_door": "0",
						"door_to_point": "1",
						"point_to_point": "1",
						"point_to_door": "0"
					}
				]
			}
		}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	resp, err := c.GetServiceStructure(context.Background())
	if err != nil {
		t.Fatalf("GetServiceStructure() error: %v", err)
	}

	if len(resp.Services) != 2 {
		t.Fatalf("len(Services) = %d, want 2", len(resp.Services))
	}

	svc := resp.Services[0]
	if svc.ServiceID != "svc1" {
		t.Errorf("Services[0].ServiceID = %q, want svc1", svc.ServiceID)
	}
	if svc.Supplier != "DHL" {
		t.Errorf("Services[0].Supplier = %q, want DHL", svc.Supplier)
	}
	if svc.DoorToDoor != "1" {
		t.Errorf("Services[0].DoorToDoor = %q, want \"1\"", svc.DoorToDoor)
	}
	if svc.Domestic != "1" {
		t.Errorf("Services[0].Domestic = %q, want \"1\"", svc.Domestic)
	}

	svc2 := resp.Services[1]
	if svc2.ServiceID != "svc2" {
		t.Errorf("Services[1].ServiceID = %q, want svc2", svc2.ServiceID)
	}
	if svc2.DoorToPoint != "1" {
		t.Errorf("Services[1].DoorToPoint = %q, want \"1\"", svc2.DoorToPoint)
	}
}

func TestGetServiceStructureAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":401,"message":"invalid credentials"}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	_, err := c.GetServiceStructure(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Status != 401 {
		t.Errorf("Status = %d, want 401", apiErr.Status)
	}
}

package inpost

import "testing"

func TestMapStatus(t *testing.T) {
	tests := []struct {
		inpost  string
		openoms string
		ok      bool
	}{
		{"created", "created", true},
		{"offers_prepared", "created", true},
		{"offer_selected", "created", true},
		{"confirmed", "label_ready", true},
		{"dispatched_by_sender", "picked_up", true},
		{"collected_from_sender", "picked_up", true},
		{"taken_by_courier", "in_transit", true},
		{"adopted_at_source_branch", "in_transit", true},
		{"sent_from_source_branch", "in_transit", true},
		{"adopted_at_sorting_center", "in_transit", true},
		{"sent_from_sorting_center", "in_transit", true},
		{"adopted_at_target_branch", "in_transit", true},
		{"out_for_delivery", "out_for_delivery", true},
		{"ready_to_pickup", "out_for_delivery", true},
		{"delivered", "delivered", true},
		{"picked_up", "delivered", true},
		{"returned_to_sender", "returned", true},
		{"avizo", "out_for_delivery", true},
		{"missing", "failed", true},
		{"claim_rejected", "failed", true},
		{"unknown_status", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.inpost, func(t *testing.T) {
			got, ok := MapStatus(tt.inpost)
			if ok != tt.ok {
				t.Fatalf("MapStatus(%q): ok = %v, want %v", tt.inpost, ok, tt.ok)
			}
			if got != tt.openoms {
				t.Fatalf("MapStatus(%q) = %q, want %q", tt.inpost, got, tt.openoms)
			}
		})
	}
}

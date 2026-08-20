package handler

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestAllegroTokenRefreshContextDetachesCancellationAndPreservesValues(t *testing.T) {
	type ctxKey struct{}
	key := ctxKey{}
	parent, cancelParent := context.WithCancel(context.WithValue(context.Background(), key, "request-id"))
	cancelParent()

	refreshCtx, cancel := allegroTokenRefreshContext(parent)
	defer cancel()

	if parent.Err() == nil {
		t.Fatal("expected parent context to be canceled")
	}
	if err := refreshCtx.Err(); err != nil {
		t.Fatalf("expected token refresh context to ignore parent cancellation, got %v", err)
	}
	select {
	case <-refreshCtx.Done():
		t.Fatal("expected token refresh context to remain active until its own timeout")
	default:
	}
	if got := refreshCtx.Value(key); got != "request-id" {
		t.Fatalf("expected token refresh context to preserve request values, got %v", got)
	}
	deadline, ok := refreshCtx.Deadline()
	if !ok {
		t.Fatal("expected token refresh context to have its own timeout")
	}
	remaining := time.Until(deadline)
	if remaining <= 0 || remaining > allegroTokenRefreshPersistTimeout {
		t.Fatalf("unexpected token refresh context deadline: %v", remaining)
	}
}

func TestAllegroTokenRefreshCallbackDoesNotUseRequestContext(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	filePath := filepath.Join(filepath.Dir(testFile), "allegro_common.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		t.Fatalf("parse allegro_common.go: %v", err)
	}

	var foundCallback bool
	var requestContextCalls []token.Position
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isAllegroOnTokenRefreshCall(call) {
			return true
		}
		foundCallback = true
		if len(call.Args) == 0 {
			return true
		}
		fn, ok := call.Args[0].(*ast.FuncLit)
		if !ok {
			return true
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if isRequestContextCall(n) {
				requestContextCalls = append(requestContextCalls, fset.Position(n.Pos()))
			}
			return true
		})
		return true
	})

	if !foundCallback {
		t.Fatal("expected buildAllegroClient to register an Allegro token refresh callback")
	}
	if len(requestContextCalls) > 0 {
		t.Fatalf("Allegro token refresh callback must not use request context directly; found r.Context() at %v", requestContextCalls)
	}
}

func isAllegroOnTokenRefreshCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "WithOnTokenRefresh" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "allegrosdk"
}

func isRequestContextCall(n ast.Node) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Context" {
		return false
	}
	receiver, ok := sel.X.(*ast.Ident)
	return ok && receiver.Name == "r"
}

func TestListingStatusFromAllegroPublication(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"ACTIVE", "active"},
		{"active", "active"},
		{"INACTIVE", "inactive"},
		{"DRAFT", "inactive"},
		{"IN_PROGRESS", "inactive"},
		{"ACTIVATING", "inactive"},
		{"CHECKING", "inactive"},
		{"szkic", "inactive"},
		{"ENDED", "ended"},
		{"", "inactive"},
		{"UNKNOWN", "inactive"},
	}
	for _, tt := range tests {
		if got := listingStatusFromAllegroPublication(tt.in); got != tt.want {
			t.Errorf("listingStatusFromAllegroPublication(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestListingStatusFromAllegroOffer_MissingPublicationIsInactive(t *testing.T) {
	if got := listingStatusFromAllegroOffer(nil); got != "inactive" {
		t.Fatalf("nil offer: got %q, want inactive", got)
	}
	if got := listingStatusFromAllegroOffer(&allegrosdk.Offer{ID: "7781994292"}); got != "inactive" {
		t.Fatalf("offer without publication: got %q, want inactive", got)
	}
	if got := listingStatusFromAllegroOffer(&allegrosdk.Offer{
		ID:          "7781994292",
		Publication: &allegrosdk.OfferPublication{},
	}); got != "inactive" {
		t.Fatalf("empty publication status: got %q, want inactive", got)
	}
}

func TestListingStatusFromAllegroOffer_UsesPublicationReality(t *testing.T) {
	if got := listingStatusFromAllegroOffer(&allegrosdk.Offer{
		Publication: &allegrosdk.OfferPublication{Status: "INACTIVE"},
	}); got != "inactive" {
		t.Fatalf("INACTIVE: got %q, want inactive", got)
	}
	if got := listingStatusFromAllegroOffer(&allegrosdk.Offer{
		Publication: &allegrosdk.OfferPublication{Status: "ACTIVE"},
	}); got != "active" {
		t.Fatalf("ACTIVE: got %q, want active", got)
	}
}

func TestAllegroListingNeedsPublicationHeal(t *testing.T) {
	ext := "7781994292"
	oldURL := "https://allegro.pl.allegrosandbox.pl/moje-allegro/sprzedaz/oferty/7781994292"
	newURL := "https://allegro.pl.allegrosandbox.pl/oferta/7781994292"

	if !allegroListingNeedsPublicationHeal(&model.ProductListing{
		Status: "active", ExternalID: &ext, URL: &oldURL,
	}) {
		t.Fatal("expected heal for active listing still on the seller-panel URL")
	}
	if allegroListingNeedsPublicationHeal(&model.ProductListing{
		Status: "active", ExternalID: &ext, URL: &newURL,
	}) {
		t.Fatal("must not keep refreshing after the public /oferta URL is stored")
	}
	if allegroListingNeedsPublicationHeal(&model.ProductListing{
		Status: "inactive", ExternalID: &ext, URL: &oldURL,
	}) {
		t.Fatal("inactive leftover is already not Aktywna")
	}
	if allegroListingNeedsPublicationHeal(&model.ProductListing{Status: "active", URL: &oldURL}) {
		t.Fatal("listing without an offer id cannot be healed from Allegro")
	}
	if allegroListingNeedsPublicationHeal(nil) {
		t.Fatal("nil listing")
	}
}

func TestAllegroOfferURLIsPublicOfertaPath(t *testing.T) {
	if got := allegroOfferURL("7781994292", false); got != "https://allegro.pl/oferta/7781994292" {
		t.Fatalf("prod URL = %q", got)
	}
	if got := allegroOfferURL("7781994292", true); got != "https://allegro.pl.allegrosandbox.pl/oferta/7781994292" {
		t.Fatalf("sandbox URL = %q", got)
	}
}

func TestAllegroSalesCenterCreateShipmentURL(t *testing.T) {
	const checkoutFormID = "19829450-9c54-11f1-bd08-9328d2ed1733"
	const sellerID = "110974929"

	gotSandbox := allegroSalesCenterCreateShipmentURL(checkoutFormID, sellerID, true)
	wantSandbox := "https://salescenter.allegro.com.allegrosandbox.pl/ship-with-allegro/swa/create-shipment/19829450-9c54-11f1-bd08-9328d2ed1733?sellerId=110974929"
	if gotSandbox != wantSandbox {
		t.Fatalf("sandbox URL = %q, want %q", gotSandbox, wantSandbox)
	}

	gotProd := allegroSalesCenterCreateShipmentURL(checkoutFormID, sellerID, false)
	wantProd := "https://salescenter.allegro.com/ship-with-allegro/swa/create-shipment/19829450-9c54-11f1-bd08-9328d2ed1733?sellerId=110974929"
	if gotProd != wantProd {
		t.Fatalf("prod URL = %q, want %q", gotProd, wantProd)
	}

	if strings.Contains(gotSandbox, "nadaj-paczke") || strings.Contains(gotSandbox, "orderId=") {
		t.Fatalf("must not use marketplace nadaj-paczke?orderId= (404s): %q", gotSandbox)
	}

	if got := allegroSalesCenterCreateShipmentURL("", sellerID, true); got != "" {
		t.Fatalf("empty checkoutFormID: got %q", got)
	}
	if got := allegroSalesCenterCreateShipmentURL(checkoutFormID, "  ", false); got != "" {
		t.Fatalf("empty sellerID: got %q", got)
	}
}

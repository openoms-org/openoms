// Package erli provides a Go client for the Erli.pl marketplace REST API.
//
// Integration tests against the Erli sandbox (api-sandbox.erli.pl/v2) are
// available in integration_test.go and can be run with:
//
//	ERLI_SANDBOX_TOKEN=<token> go test -tags=integration -v ./...
//
// Optional env vars:
//   - ERLI_SANDBOX_OFFER_ID  — existing offer ID for update tests
//   - ERLI_SANDBOX_ORDER_ID  — existing order ID for get/structure tests
package erli

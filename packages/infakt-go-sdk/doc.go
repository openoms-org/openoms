// Package infakt provides a Go client for the inFakt invoicing API.
//
// This is a standalone package licensed under MIT. It can be imported
// independently of the main OpenOMS application.
//
// Status: In Development — this package has been implemented but not yet
// verified against the real API in a production environment.
//
// Features:
//   - API key authentication (X-inFakt-ApiKey header)
//   - Invoice creation (VAT, proforma, final, correction)
//   - Invoice retrieval and listing
//   - PDF download
//   - Email sending
package infakt

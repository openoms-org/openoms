// Package btp provides a Go client for the BTP.pro (Business Trading Platform) Client API.
//
// BTP.pro is a B2B platform used by Polish wholesalers and distributors. The API provides
// access to product catalogues, inventory reports with stock levels, order placement,
// invoice retrieval, and waybill management.
//
// Authentication uses HTTP Basic Auth (username + password).
//
// Usage:
//
//	client := btp.NewClient("username", "password")
//	report, err := client.Inventory.GetReport(ctx, nil)
//	catalogue, err := client.Catalogue.GetCatalogue(ctx, nil)
//	order, err := client.Orders.Create(ctx, &btp.OrderRequest{...})
package btp

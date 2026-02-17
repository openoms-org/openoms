// Package shopify provides a lightweight Go SDK for the Shopify Admin REST API.
//
// Shopify is a global e-commerce platform with growing adoption in Poland.
// This SDK supports product, order, and inventory management via the
// Shopify Admin API (REST, version 2025-01).
//
// Authentication uses a private app access token:
// X-Shopify-Access-Token: {token}
//
// Endpoint pattern: https://{shop}.myshopify.com/admin/api/{version}/
package shopify

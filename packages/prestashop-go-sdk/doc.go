// Package prestashop provides a lightweight Go SDK for the PrestaShop Web Service API.
//
// PrestaShop is an open-source e-commerce platform popular in Europe,
// powering thousands of stores in Poland. This SDK communicates with
// the PrestaShop API using JSON output format (output_format=JSON).
//
// Authentication uses a simple API key via HTTP Basic Auth (key as username).
// Endpoint pattern: https://{shop}/api/{resource}?output_format=JSON
package prestashop

// Package engine implements the order state machine and domain events
// for the OpenOMS order management system.
//
// Standalone package licensed under MIT.
//
// Features (planned):
//   - Order status transitions with validation
//   - Allowed transitions map
//   - Domain events (OrderStatusChanged, etc.)
//   - Side effects (shipped_at, delivered_at timestamps)
package engine

import type { CreateOrderRequest, UpdateOrderRequest } from "@/types/api";

export function mapCreateOrderRequestToUpdateOrderRequest(
  data: CreateOrderRequest
): UpdateOrderRequest {
  const updateRequest: UpdateOrderRequest = {};

  if (data.external_id !== undefined) updateRequest.external_id = data.external_id;
  if (data.customer_name !== undefined) updateRequest.customer_name = data.customer_name;
  if (data.customer_email !== undefined) updateRequest.customer_email = data.customer_email;
  if (data.customer_phone !== undefined) updateRequest.customer_phone = data.customer_phone;
  if (data.shipping_address !== undefined) updateRequest.shipping_address = data.shipping_address;
  if (data.billing_address !== undefined) updateRequest.billing_address = data.billing_address;
  if (data.items !== undefined) updateRequest.items = data.items;
  if (data.total_amount !== undefined) updateRequest.total_amount = data.total_amount;
  if (data.currency !== undefined) updateRequest.currency = data.currency;
  if (data.notes !== undefined) updateRequest.notes = data.notes;
  if (data.internal_notes !== undefined) updateRequest.internal_notes = data.internal_notes;
  if (data.priority !== undefined) updateRequest.priority = data.priority;
  if (data.metadata !== undefined) updateRequest.metadata = data.metadata;
  if (data.tags !== undefined) updateRequest.tags = data.tags;
  if (data.delivery_method !== undefined) updateRequest.delivery_method = data.delivery_method;
  if (data.pickup_point_id !== undefined) updateRequest.pickup_point_id = data.pickup_point_id;
  if (data.payment_status !== undefined) updateRequest.payment_status = data.payment_status;
  if (data.payment_method !== undefined) updateRequest.payment_method = data.payment_method;

  return updateRequest;
}

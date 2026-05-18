import type { PaginationParams } from "./common";

// === Shipments ===
export interface Shipment {
  id: string;
  tenant_id: string;
  order_id: string;
  provider: string;
  integration_id?: string;
  tracking_number?: string;
  status: string;
  label_url?: string;
  carrier_data?: Record<string, unknown>;
  warehouse_id?: string;
  package_number: number;
  weight?: number;
  length?: number;
  width?: number;
  height?: number;
  notes: string;
  carbon_kg?: number;
  distance_km?: number;
  carbon_method?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateShipmentRequest {
  order_id: string;
  provider: string;
  integration_id?: string;
  tracking_number?: string;
  label_url?: string;
  carrier_data?: Record<string, unknown>;
  warehouse_id?: string;
  weight?: number;
  length?: number;
  width?: number;
  height?: number;
  notes?: string;
}

export interface UpdateShipmentRequest {
  tracking_number?: string;
  label_url?: string;
  carrier_data?: Record<string, unknown>;
  weight?: number;
  length?: number;
  width?: number;
  height?: number;
  notes?: string;
}

export interface ShipmentListParams extends PaginationParams {
  status?: string;
  provider?: string;
  order_id?: string;
  order_ids?: string[];
}

export interface TrackingEvent {
  status: string;
  location?: string;
  timestamp: string;
  details?: string;
}

export interface GenerateLabelRequest {
  service_type: string;
  parcel_size?: string;
  target_point?: string;
  sending_method?: string;
  label_format: string;
  weight_kg?: number;
  width_cm?: number;
  height_cm?: number;
  depth_cm?: number;
  cod_amount?: number;
  insured_value?: number;
}

// === InPost Points ===
interface InPostPointAddress {
  line1: string;
  line2: string;
}

interface InPostPointAddressDetails {
  city: string;
  province: string;
  post_code: string;
  street: string;
  building_number: string;
}

export interface InPostPoint {
  name: string;
  type: string[];
  address: InPostPointAddress;
  address_details?: InPostPointAddressDetails;
  location_description: string;
  opening_hours: string;
  status: string;
}

export interface InPostPointSearchResponse {
  items: InPostPoint[];
  count: number;
  page: number;
  per_page: number;
  total_pages: number;
}

// === Dispatch Orders ===
export interface BatchLabelsRequest {
  shipment_ids: string[];
}

export interface CreateDispatchOrderRequest {
  shipment_ids: string[];
  street?: string;
  building_number?: string;
  city?: string;
  post_code?: string;
  name?: string;
  phone?: string;
  email?: string;
  comment?: string;
}

export interface DispatchOrderResponse {
  id: number;
  status: string;
}

// === Shipping Rate Shopping ===
export interface ShippingRate {
  carrier_name: string;
  carrier_code: string;
  service_name: string;
  price: number;
  currency: string;
  estimated_days: number;
  pickup_point: boolean;
  is_estimate: boolean;
}

export interface GetRatesRequest {
  from_postal_code: string;
  from_country: string;
  to_postal_code: string;
  to_country: string;
  weight: number;
  width: number;
  height: number;
  length: number;
  cod: number;
}

export interface GetRatesResponse {
  rates: ShippingRate[];
}

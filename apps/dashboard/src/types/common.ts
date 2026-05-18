// === Shared Models ===
export interface Address {
  name?: string;
  street?: string;
  city?: string;
  postal_code?: string;
  country?: string;
}

// === Pagination ===
export interface ListResponse<T> {
  items: T[];
  total: number;
  limit: number;
  offset: number;
}

export interface PaginationParams {
  limit?: number;
  offset?: number;
  sort_by?: string;
  sort_order?: "asc" | "desc";
}

// === API Error ===
export interface ApiError {
  error: string;
}

// === WebSocket Events ===
export interface WSEvent {
  type: string;
  tenant_id: string;
  payload?: Record<string, unknown>;
}

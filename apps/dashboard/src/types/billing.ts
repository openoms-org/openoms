// === Billing ===
export interface PublicPlanInfo {
  id: string;
  name: string;
  monthly_amount: number;
  yearly_amount: number;
  currency: string;
  trial_days: number;
  limits: PlanLimits;
  features: string[];
}

export interface PlanLimits {
  max_users: number;
  max_orders_monthly: number;
  max_integrations: number;
}

export interface CheckoutSessionRequest {
  plan_id: string;
  interval: "month" | "year";
}

export interface CheckoutSessionResponse {
  checkout_url: string;
  session_id: string;
}

export interface CheckoutSessionStatus {
  plan: string;
  interval: string;
  email: string;
  status: "pending" | "completed" | "registered";
  limits: PlanLimits;
}

export interface SubscriptionStatus {
  plan: string;
  status: "trialing" | "active" | "past_due" | "canceled" | "suspended";
  billing_interval?: "month" | "year";
  trial_end?: string;
  current_period_end?: string;
  canceled_at?: string;
  limits?: PlanLimits;
}

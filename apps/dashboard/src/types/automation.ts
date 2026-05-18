import type { PaginationParams } from "./common";

// === Automation Rules ===
export interface AutomationCondition {
  field: string;
  operator: string;
  value: unknown;
}

export interface AutomationAction {
  type: string;
  config: Record<string, unknown>;
  delay_seconds?: number;
}

export interface DelayedAction {
  id: string;
  tenant_id: string;
  rule_id: string;
  action_index: number;
  order_id?: string;
  execute_at: string;
  executed: boolean;
  executed_at?: string;
  error?: string;
  created_at: string;
  action_data: Record<string, unknown>;
  event_data: Record<string, unknown>;
}

export interface AutomationRule {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  enabled: boolean;
  priority: number;
  trigger_event: string;
  conditions: AutomationCondition[];
  actions: AutomationAction[];
  last_fired_at?: string;
  fire_count: number;
  created_at: string;
  updated_at: string;
}

export interface CreateAutomationRuleRequest {
  name: string;
  description?: string;
  enabled?: boolean;
  priority?: number;
  trigger_event: string;
  conditions: AutomationCondition[];
  actions: AutomationAction[];
}

export interface UpdateAutomationRuleRequest {
  name?: string;
  description?: string;
  enabled?: boolean;
  priority?: number;
  trigger_event?: string;
  conditions?: AutomationCondition[];
  actions?: AutomationAction[];
}

export interface AutomationRuleLog {
  id: string;
  tenant_id: string;
  rule_id: string;
  trigger_event: string;
  entity_type: string;
  entity_id: string;
  conditions_met: boolean;
  actions_executed: Record<string, unknown>[];
  error_message?: string;
  executed_at: string;
}

export interface AutomationRuleListParams extends PaginationParams {
  trigger_event?: string;
  enabled?: boolean;
}

export interface TestAutomationRuleRequest {
  data: Record<string, unknown>;
}

export interface TestAutomationRuleResponse {
  condition_results: ConditionResult[];
  all_conditions_met: boolean;
  actions_to_execute: AutomationAction[];
}

interface ConditionResult {
  condition: AutomationCondition;
  met: boolean;
}

// === Workflow Builder ===
export interface WorkflowPosition {
  x: number;
  y: number;
}

export interface WorkflowViewport {
  x: number;
  y: number;
  zoom: number;
}

export interface WorkflowNode {
  id: string;
  type: "trigger" | "condition" | "action";
  position: WorkflowPosition;
  data: Record<string, unknown>;
}

export interface WorkflowEdge {
  id: string;
  source: string;
  target: string;
  sourceHandle?: string;
  targetHandle?: string;
  label?: string;
}

export interface WorkflowDefinition {
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
  viewport: WorkflowViewport;
}

export interface WorkflowTemplate {
  id: string;
  name: string;
  description: string;
  definition: WorkflowDefinition;
}

export interface ValidateWorkflowResponse {
  valid: boolean;
  errors: string[];
}

export interface ConvertWorkflowResponse {
  trigger_event: string;
  conditions: AutomationCondition[];
  actions: AutomationAction[];
}

// === Message Templates ===
export interface MessageTemplate {
  id: string;
  tenant_id: string;
  name: string;
  channel: string; // "allegro" | "email" | "sms"
  subject: string;
  body: string;
  variables: string[];
  is_autoresponder: boolean;
  trigger_event?: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateMessageTemplateRequest {
  name: string;
  channel: string;
  subject?: string;
  body: string;
  variables?: string[];
  is_autoresponder?: boolean;
  trigger_event?: string;
  enabled?: boolean;
}

export interface UpdateMessageTemplateRequest {
  name?: string;
  channel?: string;
  subject?: string;
  body?: string;
  variables?: string[];
  is_autoresponder?: boolean;
  trigger_event?: string;
  enabled?: boolean;
}

export interface MessageTemplateListParams {
  channel?: string;
  enabled?: boolean;
}

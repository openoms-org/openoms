package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"

	"github.com/google/uuid"
)

// WorkflowNodeType represents the type of a workflow node.
type WorkflowNodeType string

const (
	WorkflowNodeTrigger   WorkflowNodeType = "trigger"
	WorkflowNodeCondition WorkflowNodeType = "condition"
	WorkflowNodeAction    WorkflowNodeType = "action"
)

// WorkflowPosition represents x/y coordinates on the canvas.
type WorkflowPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// WorkflowViewport stores pan/zoom state of the canvas.
type WorkflowViewport struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Zoom float64 `json:"zoom"`
}

// WorkflowNode represents a single node in the visual workflow.
type WorkflowNode struct {
	ID       string           `json:"id"`
	Type     WorkflowNodeType `json:"type"`
	Position WorkflowPosition `json:"position"`
	Data     map[string]any   `json:"data"`
}

// WorkflowEdge represents a connection between two nodes.
type WorkflowEdge struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	SourceHandle string `json:"sourceHandle,omitempty"`
	TargetHandle string `json:"targetHandle,omitempty"`
	Label        string `json:"label,omitempty"`
}

// WorkflowDefinition is the complete visual workflow graph.
type WorkflowDefinition struct {
	Nodes    []WorkflowNode   `json:"nodes"`
	Edges    []WorkflowEdge   `json:"edges"`
	Viewport WorkflowViewport `json:"viewport"`
}

// WorkflowTemplate is a pre-built workflow template.
type WorkflowTemplate struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Definition  WorkflowDefinition `json:"definition"`
}

// ValidateWorkflowRequest is the payload for validating a workflow definition.
type ValidateWorkflowRequest struct {
	Definition WorkflowDefinition `json:"definition"`
}

// ValidateWorkflowResponse is the result of workflow validation.
type ValidateWorkflowResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
}

// ConvertWorkflowRequest is the payload for converting a visual workflow to an automation rule.
type ConvertWorkflowRequest struct {
	Definition WorkflowDefinition `json:"definition"`
	Name       string             `json:"name"`
	Enabled    *bool              `json:"enabled,omitempty"`
	Priority   *int               `json:"priority,omitempty"`
}

// ConvertWorkflowResponse returns the generated automation rule fields.
type ConvertWorkflowResponse struct {
	TriggerEvent string                `json:"trigger_event"`
	Conditions   []AutomationCondition `json:"conditions"`
	Actions      []AutomationAction    `json:"actions"`
}

// ValidateWorkflowDefinition checks the workflow for structural errors.
func ValidateWorkflowDefinition(def WorkflowDefinition) []string {
	var errs []string

	if len(def.Nodes) == 0 {
		errs = append(errs, "Workflow musi zawierac co najmniej jeden wezel")
		return errs
	}

	// Count trigger nodes
	triggerCount := 0
	nodeIDs := make(map[string]bool)
	for _, n := range def.Nodes {
		if nodeIDs[n.ID] {
			errs = append(errs, fmt.Sprintf("Duplikat ID wezla: %s", n.ID))
		}
		nodeIDs[n.ID] = true

		if n.Type == WorkflowNodeTrigger {
			triggerCount++
		}

		switch n.Type {
		case WorkflowNodeTrigger, WorkflowNodeCondition, WorkflowNodeAction:
			// valid
		default:
			errs = append(errs, fmt.Sprintf("Nieznany typ wezla: %s", n.Type))
		}
	}

	if triggerCount == 0 {
		errs = append(errs, "Workflow musi zawierac co najmniej jeden wezel wyzwalajacy (trigger)")
	}
	if triggerCount > 1 {
		errs = append(errs, "Workflow moze zawierac tylko jeden wezel wyzwalajacy (trigger)")
	}

	// Validate edges reference existing nodes
	for _, e := range def.Edges {
		if !nodeIDs[e.Source] {
			errs = append(errs, fmt.Sprintf("Krawedz '%s' odnosi sie do nieistniejacego wezla zrodlowego: %s", e.ID, e.Source))
		}
		if !nodeIDs[e.Target] {
			errs = append(errs, fmt.Sprintf("Krawedz '%s' odnosi sie do nieistniejacego wezla docelowego: %s", e.ID, e.Target))
		}
	}

	// Check that trigger node has valid event
	for _, n := range def.Nodes {
		if n.Type == WorkflowNodeTrigger {
			event, _ := n.Data["event"].(string)
			if event == "" {
				errs = append(errs, "Wezel wyzwalajacy musi miec przypisane zdarzenie (event)")
			} else if !ValidTriggerEvents[event] {
				errs = append(errs, fmt.Sprintf("Nieprawidlowe zdarzenie wyzwalajace: %s", event))
			}
		}
		if n.Type == WorkflowNodeCondition {
			field, _ := n.Data["field"].(string)
			if field == "" {
				errs = append(errs, fmt.Sprintf("Wezel warunku '%s' musi miec przypisane pole (field)", n.ID))
			}
		}
		if n.Type == WorkflowNodeAction {
			actionType, _ := n.Data["actionType"].(string)
			if actionType == "" {
				errs = append(errs, fmt.Sprintf("Wezel akcji '%s' musi miec przypisany typ akcji (actionType)", n.ID))
			}
		}
	}

	return errs
}

// WorkflowToAutomationRule converts a visual WorkflowDefinition into flat AutomationRule fields.
// It traverses the graph from the trigger node, collecting conditions and actions along the way.
func WorkflowToAutomationRule(def WorkflowDefinition) (*ConvertWorkflowResponse, error) {
	// Build adjacency: source -> [targets]
	adj := make(map[string][]string)
	for _, e := range def.Edges {
		adj[e.Source] = append(adj[e.Source], e.Target)
	}

	// Find the trigger node
	nodeMap := make(map[string]WorkflowNode)
	var triggerNode *WorkflowNode
	for i, n := range def.Nodes {
		nodeMap[n.ID] = n
		if n.Type == WorkflowNodeTrigger {
			triggerNode = &def.Nodes[i]
		}
	}
	if triggerNode == nil {
		return nil, errors.New("no trigger node found")
	}

	event, _ := triggerNode.Data["event"].(string)
	if event == "" {
		return nil, errors.New("trigger node has no event")
	}

	// BFS from trigger
	var conditions []AutomationCondition
	var actions []AutomationAction

	visited := make(map[string]bool)
	queue := []string{triggerNode.ID}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true

		node := nodeMap[current]

		switch node.Type {
		case WorkflowNodeCondition:
			field, _ := node.Data["field"].(string)
			operator, _ := node.Data["operator"].(string)
			if operator == "" {
				operator = "eq"
			}
			value := node.Data["value"]
			conditions = append(conditions, AutomationCondition{
				Field:    field,
				Operator: operator,
				Value:    value,
			})

		case WorkflowNodeAction:
			actionType, _ := node.Data["actionType"].(string)
			config := make(map[string]any)
			for k, v := range node.Data {
				if k != "actionType" && k != "label" {
					config[k] = v
				}
			}
			delaySeconds := 0
			if ds, ok := node.Data["delay_seconds"]; ok {
				switch v := ds.(type) {
				case float64:
					delaySeconds = int(v)
				case json.Number:
					i, _ := v.Int64()
					delaySeconds = int(i)
				}
			}
			actions = append(actions, AutomationAction{
				Type:         actionType,
				Config:       config,
				DelaySeconds: delaySeconds,
			})
		}

		queue = append(queue, adj[current]...)
	}

	if conditions == nil {
		conditions = []AutomationCondition{}
	}
	if actions == nil {
		actions = []AutomationAction{}
	}

	return &ConvertWorkflowResponse{
		TriggerEvent: event,
		Conditions:   conditions,
		Actions:      actions,
	}, nil
}

// AutomationRuleToWorkflow converts an existing AutomationRule into a visual WorkflowDefinition
// for editing in the visual workflow builder.
func AutomationRuleToWorkflow(rule AutomationRule) (*WorkflowDefinition, error) {
	var nodes []WorkflowNode
	var edges []WorkflowEdge

	// Parse conditions
	var conditions []AutomationCondition
	if rule.Conditions != nil && string(rule.Conditions) != "[]" && string(rule.Conditions) != "null" {
		if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
			conditions = []AutomationCondition{}
		}
	}

	// Parse actions
	var actions []AutomationAction
	if rule.Actions != nil && string(rule.Actions) != "[]" && string(rule.Actions) != "null" {
		if err := json.Unmarshal(rule.Actions, &actions); err != nil {
			actions = []AutomationAction{}
		}
	}

	// Create trigger node
	triggerID := uuid.New().String()
	nodes = append(nodes, WorkflowNode{
		ID:       triggerID,
		Type:     WorkflowNodeTrigger,
		Position: WorkflowPosition{X: 250, Y: 50},
		Data: map[string]any{
			"event": rule.TriggerEvent,
			"label": rule.TriggerEvent,
		},
	})

	prevID := triggerID
	yPos := 200.0

	// Add condition nodes
	for _, c := range conditions {
		condID := uuid.New().String()
		nodes = append(nodes, WorkflowNode{
			ID:       condID,
			Type:     WorkflowNodeCondition,
			Position: WorkflowPosition{X: 250, Y: yPos},
			Data: map[string]any{
				"field":    c.Field,
				"operator": c.Operator,
				"value":    c.Value,
				"label":    fmt.Sprintf("%s %s %v", c.Field, c.Operator, c.Value),
			},
		})
		edges = append(edges, WorkflowEdge{
			ID:     uuid.New().String(),
			Source: prevID,
			Target: condID,
		})
		prevID = condID
		yPos += 150
	}

	// Add action nodes
	for _, a := range actions {
		actionID := uuid.New().String()
		data := map[string]any{
			"actionType": a.Type,
			"label":      a.Type,
		}
		maps.Copy(data, a.Config)
		if a.DelaySeconds > 0 {
			data["delay_seconds"] = a.DelaySeconds
		}
		nodes = append(nodes, WorkflowNode{
			ID:       actionID,
			Type:     WorkflowNodeAction,
			Position: WorkflowPosition{X: 250, Y: yPos},
			Data:     data,
		})
		edges = append(edges, WorkflowEdge{
			ID:     uuid.New().String(),
			Source: prevID,
			Target: actionID,
		})
		prevID = actionID
		yPos += 150
	}

	return &WorkflowDefinition{
		Nodes: nodes,
		Edges: edges,
		Viewport: WorkflowViewport{
			X:    0,
			Y:    0,
			Zoom: 1,
		},
	}, nil
}

// GetWorkflowTemplates returns pre-built workflow templates.
func GetWorkflowTemplates() []WorkflowTemplate {
	return []WorkflowTemplate{
		{
			ID:          "auto-invoice",
			Name:        "Auto-faktura",
			Description: "Automatycznie tworzy fakture po wyslaniu zamowienia",
			Definition: WorkflowDefinition{
				Nodes: []WorkflowNode{
					{
						ID:       "trigger-1",
						Type:     WorkflowNodeTrigger,
						Position: WorkflowPosition{X: 250, Y: 50},
						Data:     map[string]any{"event": "order.status_changed", "label": "Zmiana statusu zamowienia"},
					},
					{
						ID:       "condition-1",
						Type:     WorkflowNodeCondition,
						Position: WorkflowPosition{X: 250, Y: 200},
						Data:     map[string]any{"field": "status", "operator": "eq", "value": "shipped", "label": "Status = Wyslane"},
					},
					{
						ID:       "action-1",
						Type:     WorkflowNodeAction,
						Position: WorkflowPosition{X: 250, Y: 350},
						Data:     map[string]any{"actionType": "create_invoice", "invoice_type": "vat", "label": "Utworz fakture"},
					},
				},
				Edges: []WorkflowEdge{
					{ID: "edge-1", Source: "trigger-1", Target: "condition-1"},
					{ID: "edge-2", Source: "condition-1", Target: "action-1"},
				},
				Viewport: WorkflowViewport{X: 0, Y: 0, Zoom: 1},
			},
		},
		{
			ID:          "email-notification",
			Name:        "Powiadomienie email",
			Description: "Wysyla email po utworzeniu zamowienia",
			Definition: WorkflowDefinition{
				Nodes: []WorkflowNode{
					{
						ID:       "trigger-1",
						Type:     WorkflowNodeTrigger,
						Position: WorkflowPosition{X: 250, Y: 50},
						Data:     map[string]any{"event": "order.created", "label": "Zamowienie utworzone"},
					},
					{
						ID:       "action-1",
						Type:     WorkflowNodeAction,
						Position: WorkflowPosition{X: 250, Y: 200},
						Data:     map[string]any{"actionType": "send_email", "to": "admin@firma.pl", "label": "Wyslij email"},
					},
				},
				Edges: []WorkflowEdge{
					{ID: "edge-1", Source: "trigger-1", Target: "action-1"},
				},
				Viewport: WorkflowViewport{X: 0, Y: 0, Zoom: 1},
			},
		},
		{
			ID:          "auto-status-paid",
			Name:        "Auto-status po zaplacie",
			Description: "Automatycznie zmienia status zamowienia na 'przetwarzane' po zaplatie",
			Definition: WorkflowDefinition{
				Nodes: []WorkflowNode{
					{
						ID:       "trigger-1",
						Type:     WorkflowNodeTrigger,
						Position: WorkflowPosition{X: 250, Y: 50},
						Data:     map[string]any{"event": "order.created", "label": "Zamowienie utworzone"},
					},
					{
						ID:       "condition-1",
						Type:     WorkflowNodeCondition,
						Position: WorkflowPosition{X: 250, Y: 200},
						Data:     map[string]any{"field": "payment_status", "operator": "eq", "value": "paid", "label": "Platnosc = Oplacone"},
					},
					{
						ID:       "action-1",
						Type:     WorkflowNodeAction,
						Position: WorkflowPosition{X: 250, Y: 350},
						Data:     map[string]any{"actionType": "set_status", "status": "processing", "label": "Ustaw status: Przetwarzane"},
					},
				},
				Edges: []WorkflowEdge{
					{ID: "edge-1", Source: "trigger-1", Target: "condition-1"},
					{ID: "edge-2", Source: "condition-1", Target: "action-1"},
				},
				Viewport: WorkflowViewport{X: 0, Y: 0, Zoom: 1},
			},
		},
		{
			ID:          "vip-tagging",
			Name:        "Tagowanie VIP",
			Description: "Automatycznie dodaje tag VIP dla zamowien powyzej 500 zl",
			Definition: WorkflowDefinition{
				Nodes: []WorkflowNode{
					{
						ID:       "trigger-1",
						Type:     WorkflowNodeTrigger,
						Position: WorkflowPosition{X: 250, Y: 50},
						Data:     map[string]any{"event": "order.created", "label": "Zamowienie utworzone"},
					},
					{
						ID:       "condition-1",
						Type:     WorkflowNodeCondition,
						Position: WorkflowPosition{X: 250, Y: 200},
						Data:     map[string]any{"field": "total_amount", "operator": "gt", "value": 500, "label": "Kwota > 500 zl"},
					},
					{
						ID:       "action-1",
						Type:     WorkflowNodeAction,
						Position: WorkflowPosition{X: 250, Y: 350},
						Data:     map[string]any{"actionType": "add_tag", "tag": "VIP", "label": "Dodaj tag: VIP"},
					},
				},
				Edges: []WorkflowEdge{
					{ID: "edge-1", Source: "trigger-1", Target: "condition-1"},
					{ID: "edge-2", Source: "condition-1", Target: "action-1"},
				},
				Viewport: WorkflowViewport{X: 0, Y: 0, Zoom: 1},
			},
		},
	}
}

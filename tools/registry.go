package tools

import (
	"context"
	"fmt"
	"strings"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(ctx context.Context, params map[string]interface{}) (string, error)
}

type ToolRegistry struct {
	tools map[string]Tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

func (tr *ToolRegistry) Register(tool Tool) {
	tr.tools[tool.Name()] = tool
}

func (tr *ToolRegistry) Unregister(name string) {
	delete(tr.tools, name)
}

func (tr *ToolRegistry) Get(name string) Tool {
	return tr.tools[name]
}

func (tr *ToolRegistry) Has(name string) bool {
	_, exists := tr.tools[name]
	return exists
}

func (tr *ToolRegistry) GetDefinitions() []map[string]interface{} {
	definitions := make([]map[string]interface{}, 0, len(tr.tools))
	for _, tool := range tr.tools {
		definitions = append(definitions, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name(),
				"description": tool.Description(),
				"parameters":  tool.Parameters(),
			},
		})
	}
	return definitions
}

func (tr *ToolRegistry) Execute(ctx context.Context, name string, params map[string]interface{}) (string, error) {
	tool, exists := tr.tools[name]
	if !exists {
		return "", fmt.Errorf("tool '%s' not found", name)
	}

	if err := tr.validateParams(tool, params); err != nil {
		return "", fmt.Errorf("invalid parameters for tool '%s': %w", name, err)
	}

	return tool.Execute(ctx, params)
}

func (tr *ToolRegistry) validateParams(tool Tool, params map[string]interface{}) error {
	schema := tool.Parameters()
	if schemaType, ok := schema["type"].(string); !ok || schemaType != "object" {
		return fmt.Errorf("schema must be object type, got %v", schemaType)
	}
	if errs := tr.validate(params, schema, ""); len(errs) > 0 {
		return fmt.Errorf("invalid parameters: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (tr *ToolRegistry) validate(val interface{}, schema map[string]interface{}, path string) []string {
	var errors []string

	schemaType, _ := schema["type"].(string)
	switch schemaType {
	case "string":
		if _, ok := val.(string); !ok {
			errors = append(errors, fmt.Sprintf("%s should be string", path))
		}
	case "integer":
		if _, ok := val.(int); !ok {
			errors = append(errors, fmt.Sprintf("%s should be integer", path))
		}
	case "number":
		if _, ok := val.(float64); !ok {
			errors = append(errors, fmt.Sprintf("%s should be number", path))
		}
	case "boolean":
		if _, ok := val.(bool); !ok {
			errors = append(errors, fmt.Sprintf("%s should be boolean", path))
		}
	case "array":
		if _, ok := val.([]interface{}); !ok {
			errors = append(errors, fmt.Sprintf("%s should be array", path))
		}
	case "object":
		obj, ok := val.(map[string]interface{})
		if !ok {
			errors = append(errors, fmt.Sprintf("%s should be object", path))
			break
		}
		props, _ := schema["properties"].(map[string]interface{})
		required, _ := schema["required"].([]interface{})
		for _, r := range required {
			if req, ok := r.(string); ok {
				if _, exists := obj[req]; !exists {
					errors = append(errors, fmt.Sprintf("missing required %s", req))
				}
			}
		}
		for k, v := range obj {
			if propSchema, ok := props[k].(map[string]interface{}); ok {
				newPath := k
				if path != "" {
					newPath = path + "." + k
				}
				errors = append(errors, tr.validate(v, propSchema, newPath)...)
			}
		}
	}

	if enum, ok := schema["enum"].([]interface{}); ok {
		found := false
		for _, e := range enum {
			if e == val {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, fmt.Sprintf("%s must be one of %v", path, enum))
		}
	}

	return errors
}

func (tr *ToolRegistry) ToolNames() []string {
	names := make([]string, 0, len(tr.tools))
	for name := range tr.tools {
		names = append(names, name)
	}
	return names
}

func (tr *ToolRegistry) Len() int {
	return len(tr.tools)
}

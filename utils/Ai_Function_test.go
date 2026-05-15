package utils

import (
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func TestNewOpenAITool(t *testing.T) {
	params := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"city": {
				Type:        jsonschema.String,
				Description: "城市名称",
			},
		},
		Required: []string{"city"},
	}

	tool := NewOpenAITool(params, "getWeather", "查询指定城市的天气")

	if tool.Type != openai.ToolTypeFunction {
		t.Errorf("Type = %q, want %q", tool.Type, openai.ToolTypeFunction)
	}
	if tool.Function == nil {
		t.Fatal("Function is nil")
	}
	if tool.Function.Name != "getWeather" {
		t.Errorf("Name = %q, want getWeather", tool.Function.Name)
	}
	if tool.Function.Description != "查询指定城市的天气" {
		t.Errorf("Description = %q", tool.Function.Description)
	}

	// 验证 Parameters 被正确传递
	funcParams, ok := tool.Function.Parameters.(jsonschema.Definition)
	if !ok {
		t.Fatal("Parameters is not jsonschema.Definition")
	}
	if len(funcParams.Required) != 1 || funcParams.Required[0] != "city" {
		t.Errorf("Required = %v, want [city]", funcParams.Required)
	}
}

func TestNewOpenAITool_NoRequired(t *testing.T) {
	params := jsonschema.Definition{
		Type:       jsonschema.Object,
		Properties: map[string]jsonschema.Definition{},
	}

	tool := NewOpenAITool(params, "noArgs", "无参数工具")

	if tool.Function.Name != "noArgs" {
		t.Errorf("Name = %q, want noArgs", tool.Function.Name)
	}
	funcParams := tool.Function.Parameters.(jsonschema.Definition)
	if len(funcParams.Required) != 0 {
		t.Errorf("Required should be empty, got %v", funcParams.Required)
	}
}

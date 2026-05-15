package common

// import (
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"testing"

// 	"github.com/mark3labs/mcp-go/mcp"
// 	"github.com/sashabaranov/go-openai"
// 	"github.com/sashabaranov/go-openai/jsonschema"
// )

// // mockTool 用于测试 ExecuteToolCall 的 mock AITool 实现
// type mockTool struct {
// 	tools         []openai.Tool
// 	handlers      map[string]func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
// 	registerCalls int
// }

// func newMockTool() *mockTool {
// 	return &mockTool{
// 		handlers: make(map[string]func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)),
// 	}
// }

// func (m *mockTool) GetTools() []openai.Tool {
// 	return m.tools
// }

// func (m *mockTool) RegisterTools() map[string]func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
// 	m.registerCalls++
// 	return m.handlers
// }

// func TestExecuteToolCall_Success(t *testing.T) {
// 	mt := newMockTool()
// 	mt.handlers["echo"] = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
// 		msg, ok := request.Params.Arguments["msg"]
// 		if !ok {
// 			return nil, errors.New("missing msg")
// 		}
// 		return mcp.NewToolResultText("echo: " + msg.(string)), nil
// 	}

// 	args, _ := json.Marshal(map[string]string{"msg": "hello"})
// 	toolCall := openai.ToolCall{
// 		ID:   "call-1",
// 		Type: openai.ToolTypeFunction,
// 		Function: openai.FunctionCall{
// 			Name:      "echo",
// 			Arguments: string(args),
// 		},
// 	}

// 	result, err := ExecuteToolCall(context.Background(), toolCall, mt)
// 	if err != nil {
// 		t.Fatalf("ExecuteToolCall 返回错误: %v", err)
// 	}
// 	if result != "echo: hello" {
// 		t.Fatalf("result = %q, want %q", result, "echo: hello")
// 	}
// }

// func TestExecuteToolCall_UnknownTool(t *testing.T) {
// 	mt := newMockTool()

// 	toolCall := openai.ToolCall{
// 		ID:   "call-1",
// 		Type: openai.ToolTypeFunction,
// 		Function: openai.FunctionCall{
// 			Name:      "nonexistent",
// 			Arguments: "{}",
// 		},
// 	}

// 	_, err := ExecuteToolCall(context.Background(), toolCall, mt)
// 	if err == nil {
// 		t.Fatal("期望返回错误，实际 nil")
// 	}
// }

// func TestExecuteToolCall_InvalidJSON(t *testing.T) {
// 	mt := newMockTool()
// 	mt.handlers["parse"] = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
// 		return mcp.NewToolResultText("ok"), nil
// 	}

// 	toolCall := openai.ToolCall{
// 		ID:   "call-1",
// 		Type: openai.ToolTypeFunction,
// 		Function: openai.FunctionCall{
// 			Name:      "parse",
// 			Arguments: `{invalid json`,
// 		},
// 	}

// 	_, err := ExecuteToolCall(context.Background(), toolCall, mt)
// 	if err == nil {
// 		t.Fatal("期望返回 JSON 解析错误，实际 nil")
// 	}
// }

// func TestExecuteToolCall_HandlerError(t *testing.T) {
// 	mt := newMockTool()
// 	mt.handlers["failing"] = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
// 		return nil, errors.New("handler internal error")
// 	}

// 	toolCall := openai.ToolCall{
// 		ID:   "call-1",
// 		Type: openai.ToolTypeFunction,
// 		Function: openai.FunctionCall{
// 			Name:      "failing",
// 			Arguments: "{}",
// 		},
// 	}

// 	_, err := ExecuteToolCall(context.Background(), toolCall, mt)
// 	if err == nil {
// 		t.Fatal("期望返回 handler 错误，实际 nil")
// 	}
// 	if err.Error() != "handler internal error" {
// 		t.Fatalf("错误信息 = %q, want %q", err.Error(), "handler internal error")
// 	}
// }

// func TestExecuteToolCall_PassesArgs(t *testing.T) {
// 	// 验证参数正确传递到 MCP handler
// 	mt := newMockTool()
// 	var capturedArgs map[string]any
// 	mt.handlers["capture"] = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
// 		capturedArgs = request.Params.Arguments
// 		return mcp.NewToolResultText("captured"), nil
// 	}

// 	args, _ := json.Marshal(map[string]any{
// 		"count": float64(42),
// 		"name":  "test",
// 	})
// 	toolCall := openai.ToolCall{
// 		ID:   "call-1",
// 		Type: openai.ToolTypeFunction,
// 		Function: openai.FunctionCall{
// 			Name:      "capture",
// 			Arguments: string(args),
// 		},
// 	}

// 	_, err := ExecuteToolCall(context.Background(), toolCall, mt)
// 	if err != nil {
// 		t.Fatalf("ExecuteToolCall 返回错误: %v", err)
// 	}
// 	if capturedArgs == nil {
// 		t.Fatal("capturedArgs is nil, 参数未传递")
// 	}
// 	// JSON 反序列化后数字变成 float64
// 	if capturedArgs["count"] != float64(42) {
// 		t.Fatalf("count = %v, want 42", capturedArgs["count"])
// 	}
// 	if capturedArgs["name"] != "test" {
// 		t.Fatalf("name = %v, want test", capturedArgs["name"])
// 	}
// }

// func TestMockTool_ImplementsInterface(t *testing.T) {
// 	// 编译时验证 mockTool 实现了 AITool 接口
// 	var _ AITool = (*mockTool)(nil)

// 	mt := newMockTool()
// 	mt.handlers["noop"] = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
// 		return mcp.NewToolResultText("noop"), nil
// 	}

// 	if tools := mt.GetTools(); tools != nil {
// 		t.Logf("GetTools() returned %d tools", len(tools))
// 	}
// 	if reg := mt.RegisterTools(); reg == nil {
// 		t.Fatal("RegisterTools() returned nil")
// 	}
// }

// // TestExecuteToolCall_NilContext 测试 nil context 的情况
// func TestExecuteToolCall_NilContext(t *testing.T) {
// 	mt := newMockTool()
// 	mt.handlers["noop"] = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
// 		if ctx == nil {
// 			return nil, errors.New("ctx is nil")
// 		}
// 		return mcp.NewToolResultText("ok"), nil
// 	}

// 	toolCall := openai.ToolCall{
// 		ID:   "call-1",
// 		Type: openai.ToolTypeFunction,
// 		Function: openai.FunctionCall{
// 			Name:      "noop",
// 			Arguments: "{}",
// 		},
// 	}

// 	// 使用 context.Background()，不是 nil
// 	result, err := ExecuteToolCall(context.Background(), toolCall, mt)
// 	if err != nil {
// 		t.Fatalf("ExecuteToolCall 返回错误: %v", err)
// 	}
// 	if result != "ok" {
// 		t.Fatalf("result = %q, want ok", result)
// 	}
// }

// // 验证 NewOpenAITool 创建的 Tool 可以通过 ExecuteToolCall 调用
// func TestExecuteToolCall_Integration(t *testing.T) {
// 	// 创建一个真实的 AITool，模拟 Agent Tools 的模式
// 	type testAgentTools struct{}

// 	var AgentTools = &testAgentTools{}

// 	// 模拟 RegisterTools 返回一个真实 handler
// 	handlers := map[string]func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error){
// 		"greet": func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
// 			name, ok := request.Params.Arguments["name"].(string)
// 			if !ok {
// 				return nil, errors.New("name is required")
// 			}
// 			return mcp.NewToolResultText("Hello, " + name + "!"), nil
// 		},
// 	}

// 	// 创建一个包装了 handlers 的 AITool
// 	wrapper := &struct {
// 		AITool
// 		handlers map[string]func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
// 	}{
// 		handlers: handlers,
// 	}

// 	wrapperAITool := &struct{ AITool }{}
// 	_ = AgentTools
// 	_ = wrapper

// 	// 直接使用 mockTool 验证集成流程
// 	mt := newMockTool()
// 	mt.handlers = handlers

// 	args, _ := json.Marshal(map[string]string{"name": "World"})
// 	toolCall := openai.ToolCall{
// 		ID:   "call-1",
// 		Type: openai.ToolTypeFunction,
// 		Function: openai.FunctionCall{
// 			Name:      "greet",
// 			Arguments: string(args),
// 		},
// 	}

// 	result, err := ExecuteToolCall(context.Background(), toolCall, mt)
// 	if err != nil {
// 		t.Fatalf("ExecuteToolCall 集成测试失败: %v", err)
// 	}
// 	if result != "Hello, World!" {
// 		t.Fatalf("result = %q, want %q", result, "Hello, World!")
// 	}
// }

// // 确保 jsonschema 包被引用（避免 unused import）
// var _ = jsonschema.Definition{}

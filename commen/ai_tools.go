package common

import (
	"context"
	"encoding/json"
	"fmt"

	"AutoOps/utils"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
)

// AITool 预备的接口，实现了获取工具和注册工具的方法就可以使用AI工具
type AITool interface {
	GetTools() []openai.Tool
	RegisterTools() map[string]func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

// ExecuteToolCall 执行AI工具调用
// 参数:ctx - 上下文控制
// 参数:toolCall - OpenAI模型调用的工具调用对象
// 参数:tool - 实现了AITool接口的工具实例
// 返回:string - 工具执行结果的纯文本表示
// 返回:error - 执行过程中遇到的错误
// 说明:获取工具函数，解析JSON参数为map，构建MCP请求执行，最后调用utils转换为纯文本
func ExecuteToolCall(ctx context.Context, toolCall openai.ToolCall, tool AITool) (string, error) {
	registerTools := tool.RegisterTools()
	executeTool, ok := registerTools[toolCall.Function.Name] // 根据工具名称获取对应的函数
	if !ok {
		return "", fmt.Errorf("unexpected tool name: %s", toolCall.Function.Name)
	}

	// 1. 解析模型传来的JSON参数
	var args map[string]any
	// toolCall.Function.Arguments 是一个JSON字符串，例如 `{"city":"北京"}`
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		// 如果模型生成畸形JSON，则返回明确错误，上层会中断流程。
		return "", fmt.Errorf("decode tool arguments failed: %w", err)
	}

	// 2. 构建MCP标准请求结构
	request := mcp.CallToolRequest{}
	request.Params.Name = toolCall.Function.Name
	// 将解析后的参数重新包装为MCP期望的map[string]any格式
	request.Params.Arguments = args

	// 3. 调用实际的业务逻辑（executeTool）。该函数内部可能包含网络请求、缓存、降级等。
	result, err := executeTool(ctx, request)
	if err != nil {
		// 底层调用失败（例如网络超时、服务端5xx错误），作为严重错误返回。
		return "", err
	}

	// 4. 将MCP格式的响应简化为纯文本
	return utils.ToolResultToText(result), nil
}

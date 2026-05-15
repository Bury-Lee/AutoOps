package utils

import (
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func ToolResultToText(result *mcp.CallToolResult) string {
	if result == nil {
		return "" // 无结果时返回空字符串，避免panic
	}

	var parts []string
	// result.Content 是 []interface{} 类型，具体元素可能是 mcp.TextContent 或 *mcp.TextContent
	for _, item := range result.Content {
		switch content := item.(type) {
		case mcp.TextContent:
			parts = append(parts, content.Text)
		case *mcp.TextContent: // 处理指针类型，两种类型都覆盖
			parts = append(parts, content.Text)
		}
		// 如果未来有ImageContent等其他类型，可以在此扩展case，但目前忽略非文本内容
	}
	return strings.Join(parts, "\n")
}

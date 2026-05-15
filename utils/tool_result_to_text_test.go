package utils

// import (
// 	"testing"

// 	"github.com/mark3labs/mcp-go/mcp"
// )

// func TestToolResultToText(t *testing.T) {
// 	tests := []struct {
// 		name   string
// 		result *mcp.CallToolResult
// 		want   string
// 	}{
// 		{
// 			name:   "nil result",
// 			result: nil,
// 			want:   "",
// 		},
// 		{
// 			name: "empty content",
// 			result: &mcp.CallToolResult{
// 				Content: []interface{}{},
// 			},
// 			want: "",
// 		},
// 		{
// 			name: "single TextContent value",
// 			result: &mcp.CallToolResult{
// 				Content: []interface{}{
// 					mcp.TextContent{Text: "hello"},
// 				},
// 			},
// 			want: "hello",
// 		},
// 		{
// 			name: "single *TextContent pointer",
// 			result: &mcp.CallToolResult{
// 				Content: []interface{}{
// 					&mcp.TextContent{Text: "world"},
// 				},
// 			},
// 			want: "world",
// 		},
// 		{
// 			name: "mixed value and pointer",
// 			result: &mcp.CallToolResult{
// 				Content: []interface{}{
// 					mcp.TextContent{Text: "line1"},
// 					&mcp.TextContent{Text: "line2"},
// 				},
// 			},
// 			want: "line1\nline2",
// 		},
// 		{
// 			name: "multiple entries with newline join",
// 			result: &mcp.CallToolResult{
// 				Content: []interface{}{
// 					mcp.TextContent{Text: "a"},
// 					mcp.TextContent{Text: "b"},
// 					mcp.TextContent{Text: "c"},
// 				},
// 			},
// 			want: "a\nb\nc",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got := ToolResultToText(tt.result)
// 			if got != tt.want {
// 				t.Errorf("ToolResultToText() = %q, want %q", got, tt.want)
// 			}
// 		})
// 	}
// }

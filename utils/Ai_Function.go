package utils

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func NewOpenAITool(params jsonschema.Definition, name, Description string) openai.Tool {
	// 使用jsonschema.Definition构建参数结构，这是OpenAI官方库提供的便捷方式

	//参数示例
	// params := jsonschema.Definition{
	// 	Type: jsonschema.Object, // 参数本身是一个JSON对象
	// 	Properties: map[string]jsonschema.Definition{
	// 		"city": {
	// 			Type:        jsonschema.String,  // 属性类型：字符串
	// 			Description: "要查询天气的城市,例如北京或上海", // 给模型看的示例和说明
	// 		},
	// 	},
	// 	Required: []string{"city"}, // 约束模型：调用时必须提供city字段
	// }

	// 返回符合OpenAI规范的工具对象
	return openai.Tool{
		Type: openai.ToolTypeFunction, // 固定值，表示这是一个函数调用工具
		Function: &openai.FunctionDefinition{
			Name:        name,        // 工具名称，模型后续会通过此名称发起调用
			Description: Description, // 模型的决策依据之一，清晰的描述能提高调用准确率
			Parameters:  params,      // 附加的参数JSON Schema
		},
	}
}

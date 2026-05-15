package conf

import (
	"github.com/sashabaranov/go-openai"
)

type AI struct { //AI模型配置
	Enable      bool    `yaml:"enable"`                         //是否启用ai
	Model       string  `yaml:"model" json:"model"`             // AI模型名称,为local时使用本地模型
	Temperature float32 `yaml:"temperature" json:"temperature"` // 温度参数，控制生成文本的随机性
	MaxTokens   int     `yaml:"max_tokens" json:"max_tokens"`   // 最大生成令牌数
	Host        string  `yaml:"host" json:"host"`               // 模型地址API,默认http://localhost:1234/v1,当model为local时生效
	ApiKey      string  `yaml:"ApiKey" json:"-"`                // AI模型密钥
	APIType     string  `yaml:"apiType" json:"apiType"`         // AI模型平台
}

// 由于防止模型的并发恢复导致配置的竞态修改,所以一般只允许一个恢复模型在运行
type Agent struct {
	Enable      bool                            //是否启用,不启用的话当analysis调用MCP的话自动移交人工
	History     *[]openai.ChatCompletionMessage //使用指针提高速度
	LLM         *openai.Client
	ModelName   string
	MaxTokens   int
	Temperature float32
	Tools       []openai.Tool
}

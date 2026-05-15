package flags

import (
	"AutoOps/global"
	"context"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func ConnectTest() {
	tests := []struct {
		name string
		fn   func() error
	}{
		{name: "主数据库", fn: func() error { return pingDB(global.DB) }},
		{name: "AI数据库", fn: func() error { return pingDB(global.AiDB) }},
		{name: "分析模型", fn: func() error {
			return pingAI(global.AnalysAIClient, global.Config.AnalysAI.Model)
		}},
		{name: "恢复模型", fn: func() error {
			return pingAI(global.RecoveryAgent.LLM, global.RecoveryAgent.ModelName)
		}},
	}

	var failed []string
	logrus.Info("开始执行连通性测试")
	for _, test := range tests {
		if err := test.fn(); err != nil {
			logrus.Errorf("%s连通性测试失败: %v", test.name, err)
			failed = append(failed, fmt.Sprintf("%s: %v", test.name, err))
			continue
		}
		logrus.Infof("%s连通性测试成功", test.name)
	}

	if len(failed) > 0 {
		logrus.Errorf("连通性测试未通过，共失败 %d 项", len(failed))
		for _, item := range failed {
			logrus.Errorf("失败项: %s", item)
		}
		return
	}
	logrus.Info("所有连通性测试已通过")
}

func pingDB(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取底层连接失败: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("数据库不可达: %w", err)
	}
	return nil
}

func pingAI(client *openai.Client, model string) error {
	if client == nil {
		return fmt.Errorf("模型客户端未初始化")
	}
	if model == "" {
		return fmt.Errorf("模型名称为空")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "这是连通性测试,收到请回复1",
			},
		},
		MaxTokens:   1024,
		Temperature: 0,
	})
	if err != nil {
		return fmt.Errorf("模型接口不可达: %w", err)
	} else {
		fmt.Printf("模型回复: %v\n", resp.Choices[0].Message.Content)
	}
	return nil
}

package core

import (
	"AutoOps/conf"
	"AutoOps/global"
	"os"

	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

func InitAnalysAI() *openai.Client {
	conf := openai.DefaultConfig(global.Config.AnalysAI.ApiKey)
	conf.BaseURL = global.Config.AnalysAI.Host
	conf.APIType = openai.APIType(global.Config.AnalysAI.APIType)

	client := openai.NewClientWithConfig(conf)
	logrus.Info("模型已加载")
	//TODO: 进行模型连通性测试

	if client == nil {
		logrus.Panic("ai连接失败!")
	}
	return client
}

func InitAgent() conf.Agent {
	var result conf.Agent
	conf := openai.DefaultConfig(global.Config.AgentAI.ApiKey)
	conf.BaseURL = global.Config.AgentAI.Host
	conf.APIType = openai.APIType(global.Config.AgentAI.APIType)

	client := openai.NewClientWithConfig(conf)
	logrus.Info("模型已加载")
	//TODO: 进行模型连通性测试

	if client == nil {
		logrus.Panic("ai连接失败!")
	}
	result.LLM = client
	result.History = &[]openai.ChatCompletionMessage{}
	result.ModelName = global.Config.AgentAI.Model
	result.MaxTokens = global.Config.AgentAI.MaxTokens
	result.Temperature = global.Config.AgentAI.Temperature
	result.Enable = global.Config.AgentAI.Enable
	result.Tools = []openai.Tool{}
	return result
}

func InitSkill() {
	//确保skill文件夹存在
	if _, err := os.Stat("skills"); os.IsNotExist(err) {
		os.MkdirAll("skills", 0755)
	}
}

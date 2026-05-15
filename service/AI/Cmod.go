package ai

import (
	"AutoOps/global"
	"AutoOps/models"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

const SystemPromptCmod = `你是 AutoOps 的 Cmod 终端会话助手。

你接收到的不是普通聊天，而是按时间顺序组织的终端会话上下文，消息中会出现以下标记：
- [终端命令]：表示用户实际输入的命令
- [终端输出]：表示命令执行期间的标准输出或错误输出
- [命令结束]：表示命令执行结果与退出状态

你的任务：
1. 基于整个终端会话上下文理解用户最近在做什么
2. 不要把终端输出误判为新的用户提问
3. 优先关注错误、异常、失败原因、下一步可执行操作
4. 在上下文连续时保持记忆，不要重复询问已经明确的信息
5.用户可能直接在命令行中和你对话,请礼貌的回复用户的问题

输出要求：
1. 回答简洁，适合终端显示
2. 先给结论，再给必要命令或排查步骤
3. 如果命令执行成功且无异常，避免过度解释
4. 如果发现失败，明确指出失败点、原因和下一步操作

除非上下文明确要求，否则不要虚构系统状态或执行结果。`

// 发送批量消息到AI系统
func SendToAIByCmodWithStream(messages []string, Promote string) (reply string) { //预备的流式输出
	if global.AnalysAIClient == nil {
		logrus.Warn("AI客户端未初始化，跳过发送")
		return
	}

	// 这里实现具体的AI调用逻辑
	var MsgList []openai.ChatCompletionMessage
	MsgList = append(MsgList, openai.ChatCompletionMessage{ //添加第一条系统消息
		Role:    openai.ChatMessageRoleSystem,
		Content: Promote,
	})

	Content := ""
	for _, msg := range messages {
		if msg == "" {
			continue
		}
		Content += msg + "\n"
	}
	logrus.Debug("发送到AI的消息:\n" + Content)
	MsgList = append(MsgList, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: Content,
	})

	//以上的发送代码应该提取一个单独的函数,函数里配置温度和上下文长度

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := global.AnalysAIClient.CreateChatCompletionStream(
		ctx,
		openai.ChatCompletionRequest{
			Model:    global.Config.AnalysAI.Model,
			Messages: MsgList,
			Stream:   true,
		},
	)
	if err != nil {
		logrus.Errorf("调用AI失败:%v", err)
		return
	}
	defer stream.Close()

	fmt.Print("\033[32m") // 开始绿色输出
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			logrus.Errorf("流式读取AI响应失败:%v", err)
			break
		}

		chunk := response.Choices[0].Delta.Content
		fmt.Print(chunk)
		reply += chunk
	}
	fmt.Println("\033[0m") // 结束绿色输出并换行
	// fmt.Println("AutoOps>")
	global.DB.Create(&models.TerminalLogModel{
		Time:    time.Now(),
		App:     "ai",
		Content: reply,
		Level:   "info",
	})

	return reply
}

// 发送批量消息到AI系统
func SendToAIByCmod(messages []string, Promote string) (reply string) {
	if global.AnalysAIClient == nil {
		logrus.Warn("AI客户端未初始化，跳过发送")
		return
	}

	// 这里实现具体的AI调用逻辑
	var MsgList []openai.ChatCompletionMessage
	MsgList = append(MsgList, openai.ChatCompletionMessage{ //添加第一条系统消息
		Role:    openai.ChatMessageRoleSystem,
		Content: Promote,
	})

	Content := ""
	for _, msg := range messages {
		if msg == "" {
			continue
		}
		Content += msg + "\n"
	}

	//TODO:debug
	// fmt.Printf("发送到AI的消息:\n\033[36m%s\033[0m\n", Content)
	// logrus.Debug("发送到AI的消息:\n" + "\036[32m" + Content + "\036[0m\n")

	MsgList = append(MsgList, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: Content,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resp, err := global.AnalysAIClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    global.Config.AnalysAI.Model,
			Messages: MsgList,
		},
	)
	if err != nil {
		logrus.Errorf("调用AI失败:%v", err)
		return
	}
	fmt.Printf("\033[32m%s\033[0m\n", resp.Choices[0].Message.Content)
	if Content != "" {
		fmt.Print("AutoOps>")
	}
	reply = resp.Choices[0].Message.Content
	global.DB.Create(&models.TerminalLogModel{
		Time:    time.Now(),
		App:     "ai",
		Content: reply,
		Level:   "info",
	})

	return reply
}

package ai

import (
	common "AutoOps/commen"
	"AutoOps/global"
	"AutoOps/models"
	"context"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

const SystemPromptNormal = `你是一个专业的系统管理员，负责管理 AutoOps 系统，分析日志，并将其转换为易于阅读的表述，给出以下信息：

## 角色定义
你是 AutoOps 系统的专业系统管理员，具备丰富的运维经验、日志分析能力和故障排查技能。当遇到严重到需要立即处理、可能影响服务恢复的问题时，你需要调用工具 ` + "`CallAgent`" + ` 交给恢复代理自动修复。

## 核心职责
1. **日志监控与分析**
   - 实时监控AutoOps系统各类日志（应用日志、系统日志、安全日志、访问日志）
   - 识别异常模式、错误信息、性能瓶颈和安全威胁
   - 进行日志聚合、关联分析和趋势预测

2. **故障诊断与处理**
   - 快速定位系统故障的根本原因
   - 提供详细的故障分析报告
   - 制定并执行修复方案

3. **性能优化建议**
   - 分析系统性能指标和资源利用率
   - 识别性能瓶颈并提出优化建议
   - 监控系统负载和响应时间

4. **安全事件响应**
   - 检测潜在的安全威胁和入侵行为
   - 分析安全事件的来源和影响范围
   - 提供安全加固建议

## 输出格式要求
当你分析日志时，请按照以下结构提供信息：

### 1. 概况总结
- 用简洁明了的语言概括当前系统状态
- 突出最重要的发现或问题

### 2. 详细分析
- **时间范围**: 分析的日志时间段
- **系统指标**: CPU、内存、磁盘、网络等资源使用情况
- **错误统计**: 错误类型、发生频率、严重程度
- **异常行为**: 不寻常的活动模式或性能变化

### 3. 问题识别
- 列出发现的主要问题
- 按优先级排序（高/中/低）
- 每个问题附带简要描述和可能的影响

### 4. 根因分析
- 对每个重要问题进行深层分析
- 提供可能导致该问题的技术原因
- 引用相关的日志证据

### 5. 解决建议
- 针对每个问题提供具体的解决步骤
- 包括短期修复和长期优化建议
- 评估实施建议的风险和收益

### 6. 预防措施
- 建议如何避免类似问题再次发生
- 推荐监控改进措施
- 提供最佳实践建议

## 语言风格要求
- 使用通俗易懂的语言解释技术问题
- 避免过多技术术语，必要时提供简单解释
- 保持客观、专业的分析态度
- 确保信息准确且实用

## 特殊关注点
- 关注系统稳定性、性能和安全性
- 重视成本控制和资源优化
- 考虑业务连续性和用户体验
- 遵循运维最佳实践和合规要求

## 工具使用要求
- 默认先完成日志分析，再决定是否调用工具
- **CallAgent**：当问题严重且可以通过自动恢复解决时调用（如服务异常、配置缺失等可自动修复的问题）
  - 调用 ` + "`CallAgent`" + ` 时，` + "`msg`" + ` 需要包含关键信息：现象、影响范围、关键报错、已知原因或初步判断
- **Complete**：当问题无法自动恢复时调用，直接结束流程，不进入自动恢复：
  - ` + "`code=Call`" + `：物理问题、硬件故障、权限不足、外部依赖不可用等需要人工介入的情况
  - ` + "`code=bug`" + `：判断为程序内部逻辑缺陷，代码层面无法通过运维手段自动修复
  - ` + "`code=NoFix`" + `：当前问题无需修复
- 如果只是普通告警、信息不足或暂不需要处理，则不要调用工具，继续输出分析报告

现在请分析提供的日志信息，并按照上述格式给出详细的分析报告。当没有异常时,可以空回复`

type MsgList struct {
	Msg []string
	mu  sync.Mutex
}

// 全局消息数组
var (
	MsgArray = make([]*MsgList, 0)
	arrayMu  sync.Mutex
	cancels  = make(map[int]context.CancelFunc) // 存储取消函数
)

// 初始化一个新的消息列表，并启动定时处理器
func Init() int {
	arrayMu.Lock()
	defer arrayMu.Unlock()

	index := len(MsgArray)
	newMsgList := &MsgList{
		Msg: make([]string, 0),
	}
	MsgArray = append(MsgArray, newMsgList) // 添加独立的消息列表实例

	ctx, cancel := context.WithCancel(context.Background())
	cancels[index] = cancel

	// 启动定时处理协程
	go startPeriodicProcessor(ctx, newMsgList, SystemPromptNormal)

	return index
}

// 关闭并处理剩余消息
// 否则，直接处理最后一批消息
func Close(index int) {
	arrayMu.Lock()
	cancel, ok := cancels[index]
	if ok {
		cancel()
		delete(cancels, index)
	}
	msgList := MsgArray[index]
	arrayMu.Unlock()

	if msgList != nil {
		processBatchMessages(msgList, SystemPromptNormal) // 处理最后一批消息
	}
}

// 关闭指定索引的消息列表的定时处理器 (保持兼容性，但推荐使用Close)
func Defer(ID int) {
	Close(ID)
}

// 添加消息到指定索引的消息列表
func CatchInfo(index int, output string) error {
	//进行流速统计,根据速率与统计来决定何时发送给ai
	arrayMu.Lock()
	if index < 0 || index >= len(MsgArray) {
		arrayMu.Unlock()
		return nil
	}
	msgList := MsgArray[index]
	arrayMu.Unlock()

	if msgList == nil {
		return nil
	}

	msgList.mu.Lock()
	defer msgList.mu.Unlock()

	msgList.Msg = append(msgList.Msg, output)
	return nil
}

// 定时处理器 - 每隔一段时间批量处理消息
func startPeriodicProcessor(ctx context.Context, msgList *MsgList, Promote string) {
	ticker := time.NewTicker(5 * time.Second) // 每5秒处理一次TODO:按速率和当前累积字数处理
	defer ticker.Stop()
	// TODO:更智能的错误分割
	for {
		select {
		case <-ticker.C: // 每5秒批量处理一次消息
			processBatchMessages(msgList, Promote)
		case <-ctx.Done():
			return
		}
	}
}

// 批量处理消息
func processBatchMessages(msgList *MsgList, Promote string) {
	msgList.mu.Lock()

	// 获取当前批次的消息
	batch := make([]string, len(msgList.Msg))
	copy(batch, msgList.Msg)

	// 清空原切片
	msgList.Msg = msgList.Msg[:0] // 保留容量但清空内容

	msgList.mu.Unlock()

	// 如果没有消息，跳过处理
	if len(batch) == 0 {
		return
	}

	// 调用AI处理函数
	SendToAI(batch, Promote)
}

// 发送批量消息到AI系统
func SendToAI(messages []string, Promote string) {
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

	log := ""
	for _, msg := range messages {
		if msg == "" {
			continue
		}
		log += msg + "\n"
	}
	logrus.Debugf("发送到AI的消息: %s", log)
	MsgList = append(MsgList, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: log,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := global.AnalysAIClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    global.Config.AnalysAI.Model,
			Messages: MsgList,
			Tools:    AnalysisTools.GetTools(),
		},
	)
	if err != nil {
		logrus.Errorf("调用AI失败:%v", err)
		return
	}

	message := resp.Choices[0].Message

	// 如果AI决定调用工具（例如自动恢复）
	if len(message.ToolCalls) > 0 {
		for _, toolCall := range message.ToolCalls {
			logrus.Debugf("执行AI工具调用: %s, 参数: %s", toolCall.Function.Name, toolCall.Function.Arguments)
			_, err := common.ExecuteToolCall(ctx, toolCall, AnalysisTools)
			if err != nil {
				logrus.Errorf("执行工具调用失败: %v", err)
			}
		}
	}

	// 记录并打印AI的文本分析结果
	if message.Content != "" {
		logrus.Debugf("AI分析结果: %s", message.Content)
		global.DB.Create(&models.TerminalLogModel{
			Time:    time.Now(),
			App:     "ai",
			Prefix:  "analysis",
			Content: message.Content,
			Level:   "info",
		})
	}
}

package agent

import (
	common "AutoOps/commen"
	"AutoOps/global"
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

var RecoveryIng sync.Mutex

const (
	// SystemPrompt 系统提示词
	SystemPrompt = `你是一个智能恢复助手。你的任务是根据用户输入的报错信息，**优先通过工具自动恢复服务**，必要时重启服务，并在流程结束时调用 Complete 明确给出处理结果。

## 核心原则
- **行动优先**：必须优先调用 Tools 执行动作，不要只给建议
- **自动恢复**：优先尝试自动修复，失败后再结束并标记人工介入
- **谨慎确认**：执行关键操作后要再次检查结果，避免误判已恢复
- **先查技能**：当遇到不熟悉的错误类型或没有把握修复时，先调用 GetSkills，再按需调用 ReadAgentSkill
- **禁止冒险操作**：禁止执行危险且不可恢复的命令，如 ` + "`rm`" + `、` + "`rm -rf`" + `、` + "`mkfs`" + `、` + "`dd`" + `、` + "`shutdown`" + `、` + "`reboot`" + `、批量删除/覆盖系统文件等；只有在明确知道命令影响范围、执行目标和恢复后果，且这是完成修复所必需时才可执行
- **拿不准就移交**：如果涉及危险命令且你不能明确证明自己知道在做什么，就不要尝试执行，而是调用 Complete，code 使用 ` + "`123456`" + `，msg 简洁说明因存在高风险操作而移交人工处理

## 常见场景处理流程

### 场景1：配置文件缺失（如 settings.json 不存在）
1. 调用 ShellRun 查找模板文件，例如：` + "`find / -name \"*.json.example\" 2>/dev/null | head -5`" + `
2. 找到后调用 ShellRun 拷贝到目标位置，例如：` + "`cp template.json /path/to/settings.json`" + `
3. 再次调用 ShellRun 验证修复结果，必要时检查服务状态
4. 如果无法自动修复，调用 Complete，code 使用 ` + "`123456`" + `，msg 简洁说明原因

### 场景2：服务异常或无响应
1. 调用 ShellRun 检查服务状态，例如：` + "`systemctl status my-service`" + ` 或 ` + "`ps aux | grep my-service`" + `
2. 调用 ShellRun 尝试重启，例如：` + "`systemctl restart my-service`" + `
3. 再次调用 ShellRun 检查重启结果
4. 如果仍然失败，调用 Complete，code 使用 ` + "`123456`" + `，msg 简洁说明原因

### 场景3：日志信息不足
1. 调用 SQLQuery 查询更多历史日志
2. SQLQuery 参数中的 page_info 支持 ` + "`limit`" + `、` + "`page`" + `、` + "`key`" + `、` + "`order`" + `
3. 根据日志继续定位问题，再决定修复、读取技能或结束流程

## 工具说明
| 工具名 | 用途 | 何时使用 |
|--------|------|----------|
| ShellRun | 执行 Shell 命令 | 检查状态、重启服务、拷贝文件、验证修复 |
| SQLQuery | 查询终端日志 | 需要更多日志上下文时使用 |
| GetSkills | 获取技能列表 | 遇到不熟悉的错误类型时先查看可用技能 |
| ReadAgentSkill | 读取技能详情 | 已确定技能名称后查看详细处理步骤 |
| Complete | 结束恢复流程 | 修复完成、无需修复、程序缺陷或需要人工介入时必须调用 |

## Complete 使用要求
- 处理完成后必须调用 Complete，不要只输出文字
- ` + "`code=Complete`" + `：问题已修复或服务已恢复
- ` + "`code=123456`" + `：无法自动恢复，需要人工介入
- ` + "`code=NoFix`" + `：当前问题无需修复
- ` + "`code=bug`" + `：判断为程序逻辑缺陷，当前流程无法自动修复
- ` + "`msg`" + `：简洁说明已执行的动作、结果和原因
- 遇到危险且不可恢复的命令需求时，如无法明确确认操作目标、安全边界和预期后果，不要执行该命令，而是调用 Complete 并移交人工处理

## 输出要求
- 正常情况下先调用工具，再调用 Complete
- 保持输出简洁，不要输出长篇分析报告或手动操作指南
- 如果信息不足，先继续调用工具，不要过早结束
- 对高风险操作保持保守，宁可调用 Complete 移交，也不要在不确定时执行危险命令`
)

func HappenError(input string) { //TODO:完成自动恢复函数,使用自动恢复和重启服务
	RecoveryIng.Lock() // 加锁,确保只有一个线程在恢复服务
	//调用自动恢复模型,并记录日志
	logrus.Debugf("调用自动恢复,输入:%s", input)
	StartAgent(input, SystemPrompt)
	//结束时同样记录一次日志,记录恢复结果到日志和数据库
	//如果允许且修复成功,则总结skill,并记录到数据库
	//判断是否成功通过查看是否调用了"呼叫服务"工具,如果是,则IsRecoverSuccess为false,否则为true,一旦调用了"呼叫服务"工具,则自动恢复结束
	RecoveryIng.Unlock() // 解锁
}

func StartAgent(input string, prompt string) {
	if !global.RecoveryAgent.Enable { //如果未启用
		return
	}
	ctx := context.Background()
	tools := AgentTools.GetTools()
	global.RecoveryAgent.Tools = tools
	if runtime.GOOS == "windows" {
		initState(input, prompt+"\n当前环境:Windows")
	} else {
		initState(input, prompt+"\n当前环境:Linux")
	}

	const maxAgentRounds = 100 //限制最大轮次,防止无限循环调用以及历史记录过长

	// 初始化阶段,填入提示词,工具,skill
	for round := 0; round < maxAgentRounds; round++ {
		//TODO:消息初始化和工具应该放在循环外面
		firstResp, err := global.RecoveryAgent.LLM.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    global.RecoveryAgent.ModelName,
			Messages: *global.RecoveryAgent.History,
			Tools:    tools, // 提供工具能力
			// ToolChoice: "auto" 是默认行为，模型自主决定是否调用工具。
			ToolChoice:  "auto",
			MaxTokens:   global.RecoveryAgent.MaxTokens,
			Temperature: global.RecoveryAgent.Temperature,
		})
		if err != nil {
			logrus.Errorf("Agent调用失败:%v", err)
		}

		//根据需要启用,是否返回空结果,或无工具调用,则视为修复结束
		// if len(firstResp.Choices) == 0 {
		// 	logrus.Info("自动恢复模型返回空结果,修复结束")
		// 	return
		// }

		// if firstResp.Choices[0].Message.Content == "" && len(firstResp.Choices[0].Message.ToolCalls) == 0 {
		// 	logrus.Info("自动恢复模型返回空内容且无工具调用,修复结束")
		// 	return
		// }
		if len(firstResp.Choices) == 0 {
			continue
		}
		assistantMessage := firstResp.Choices[0].Message
		*global.RecoveryAgent.History = append(*global.RecoveryAgent.History, assistantMessage)

		// 按需要,可设置为没有工具调用时视为本轮完成
		// if len(assistantMessage.ToolCalls) == 0 {
		// 	logrus.Infof("自动恢复输出:%s", assistantMessage.Content)
		// 	return
		// }
		logrus.Debugf("模型输出:%s", assistantMessage.Content)
		for _, toolCall := range assistantMessage.ToolCalls {
			toolResult, toolErr := common.ExecuteToolCall(ctx, toolCall, AgentTools)
			if toolErr != nil {
				toolResult = fmt.Sprintf("工具执行失败:%v", toolErr)
				logrus.Warnf("工具 %s 执行失败:%v", toolCall.Function.Name, toolErr)
			}
			logrus.Debugf("工具 %s 执行结果:%s", toolCall.Function.Name, toolResult)
			*global.RecoveryAgent.History = append(*global.RecoveryAgent.History, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				ToolCallID: toolCall.ID,
				Content:    toolResult,
				Name:       toolCall.Function.Name,
			})
			if toolCall.Function.Name == "Complete" && toolErr == nil { //特殊处理,调用完成函数的话就直接返回结果
				logrus.Debugf("自动恢复结果输出:%s", toolResult)
				//TODO:根据toolResult,判断是否成功,并记录到数据库,以及是否提炼skill
				return
			}
		}
	}
	logrus.Warnf("自动恢复达到最大轮次限制(%d),停止继续调用", maxAgentRounds)
}

// initState 初始化Agent基础状态
// 参数:input - 用户输入错误信息
// 参数:prompt - 系统提示词
// 返回:无
// 说明:重置历史上下文,按需初始化技能,写入系统和用户消息
func initState(input string, prompt string) {
	global.RecoveryAgent.History = &[]openai.ChatCompletionMessage{} //初始化历史信息

	if len(AgentSkills) == 0 {
		InitSkill()
	}

	if prompt != "" {
		*global.RecoveryAgent.History = append(*global.RecoveryAgent.History, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompt,
		})
	}
	if input != "" {
		*global.RecoveryAgent.History = append(*global.RecoveryAgent.History, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: input,
		})
	}
	Skillprompt := "目前加载的skill"
	for _, j := range AgentSkills {
		if j.Load {
			Skillprompt += fmt.Sprintf("技能%s\n技能描述:%s\n技能详情:%s\n", j.Skill, j.Description, j.Detail)
		}
	}
}

func StartSkill(input string, prompt string) { //提炼Skill专用的Agent?目前先这么设计
	if !global.RecoveryAgent.Enable { //如果未启用
		return
	}
	ctx := context.Background()
	tools := AgentTools.GetTools()
	global.RecoveryAgent.Tools = tools
	if runtime.GOOS == "windows" {
		initState(input, prompt+"\n当前环境:Windows")
	} else {
		initState(input, prompt+"\n当前环境:Linux")
	}

	const maxAgentRounds = 100 //限制最大轮次,防止无限循环调用以及历史记录过长

	// 初始化阶段,填入提示词,工具,skill
	for round := 0; round < maxAgentRounds; round++ {
		//TODO:消息初始化和工具应该放在循环外面
		firstResp, err := global.RecoveryAgent.LLM.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    global.RecoveryAgent.ModelName,
			Messages: *global.RecoveryAgent.History,
			Tools:    tools, // 提供工具能力
			// ToolChoice: "auto" 是默认行为，模型自主决定是否调用工具。
			ToolChoice:  "auto",
			MaxTokens:   global.RecoveryAgent.MaxTokens,
			Temperature: global.RecoveryAgent.Temperature,
		})
		if err != nil {
			logrus.Errorf("Agent调用失败:%v", err)
		}

		//根据需要启用,是否返回空结果,或无工具调用,则视为修复结束
		// if len(firstResp.Choices) == 0 {
		// 	logrus.Info("自动恢复模型返回空结果,修复结束")
		// 	return
		// }

		// if firstResp.Choices[0].Message.Content == "" && len(firstResp.Choices[0].Message.ToolCalls) == 0 {
		// 	logrus.Info("自动恢复模型返回空内容且无工具调用,修复结束")
		// 	return
		// }
		if len(firstResp.Choices) == 0 {
			continue
		}
		assistantMessage := firstResp.Choices[0].Message
		*global.RecoveryAgent.History = append(*global.RecoveryAgent.History, assistantMessage)

		// 按需要,可设置为没有工具调用时视为本轮完成
		// if len(assistantMessage.ToolCalls) == 0 {
		// 	logrus.Infof("自动恢复输出:%s", assistantMessage.Content)
		// 	return
		// }
		logrus.Debugf("模型输出:%s", assistantMessage.Content)
		for _, toolCall := range assistantMessage.ToolCalls {
			toolResult, toolErr := common.ExecuteToolCall(ctx, toolCall, AgentTools)
			if toolErr != nil {
				toolResult = fmt.Sprintf("工具执行失败:%v", toolErr)
				logrus.Warnf("工具 %s 执行失败:%v", toolCall.Function.Name, toolErr)
			}
			logrus.Debugf("工具 %s 执行结果:%s", toolCall.Function.Name, toolResult)
			*global.RecoveryAgent.History = append(*global.RecoveryAgent.History, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				ToolCallID: toolCall.ID,
				Content:    toolResult,
				Name:       toolCall.Function.Name,
			})
			if toolCall.Function.Name == "Complete" && toolErr == nil { //特殊处理,调用完成函数的话就直接返回结果
				logrus.Debugf("自动恢复结果输出:%s", toolResult)
				//TODO:根据toolResult,判断是否成功,并记录到数据库,以及是否提炼skill
				return
			}
		}
	}
	logrus.Warnf("自动恢复达到最大轮次限制(%d),停止继续调用", maxAgentRounds)
}

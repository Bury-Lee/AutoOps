package ai

import (
	"AutoOps/global"
	agent "AutoOps/service/Agent"
	"AutoOps/utils"
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/sirupsen/logrus"
)

type analyzeTools struct {
}

var AnalysisTools = &analyzeTools{}

func (self *analyzeTools) GetTools() []openai.Tool {
	var tools []openai.Tool

	CallAgent := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"msg": {
				Type:        jsonschema.String,
				Description: "错误详细的情况，用于传递给代理进行分析和恢复",
			},
		},
		Required: []string{"msg"},
	}
	tools = append(tools, utils.NewOpenAITool(CallAgent, "CallAgent", "当认为有必要进行自动恢复时，调用该函数使代理恢复模型处理错误"))

	Complete := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"code": {
				Type:        jsonschema.String,
				Description: "结束恢复动作编码，可选值：Call（无法自动恢复，需要人工介入）、bug（判断为程序逻辑缺陷，需要人工介入）",
			},
			"msg": {
				Type:        jsonschema.String,
				Description: "简洁说明已执行的动作、结果和原因",
			},
		},
		Required: []string{"code", "msg"},
	}
	tools = append(tools, utils.NewOpenAITool(Complete, "Complete", "当问题无法自动恢复或判断为程序逻辑缺陷时调用，结束恢复流程。code=Call表示需要人工介入，code=bug表示程序逻辑缺陷并触发人工介入"))

	return tools
}

func (self *analyzeTools) RegisterTools() map[string]func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return map[string]func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error){
		"CallAgent": CallAgent,
		"Complete":  Complete,
	}
}

const (
	// SystemPrompt 系统提示词
	SystemPrompt = `你是一个智能恢复助手。你的任务是根据用户输入的报错信息，**优先通过工具自动恢复服务**，必要时重启服务，并在流程结束时调用 Complete 明确给出处理结果。

## 核心原则
- **行动优先**：必须优先调用 Tools 执行动作，不要只给建议
- **自动恢复**：优先尝试自动修复，失败后再结束并标记人工介入
- **谨慎确认**：执行关键操作后要再次检查结果，避免误判已恢复
- **先查技能**：当遇到不熟悉的错误类型或没有把握修复时，先调用 GetSkills，再按需调用 ReadAgentSkill

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

## 输出要求
- 正常情况下先调用工具，再调用 Complete
- 保持输出简洁，不要输出长篇分析报告或手动操作指南
- 如果信息不足，先继续调用工具，不要过早结束`
)

type CallAgentArgs struct {
	Msg string `json:"msg"` //这个是错误详细的情况
}

func CallAgent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logrus.Debugf("调用代理恢复函数%+v\n", request)
	// 当认为有必要进行自动恢复时,日志分析模型调用代理恢复模型
	//使用 JSON 序列化/反序列化
	if !global.RecoveryAgent.Enable {
		return mcp.NewToolResultText("未启用自动恢复"), nil
	}

	argsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %v", err)
	}

	var args CallAgentArgs
	if err := json.Unmarshal(argsBytes, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %v", err)
	}

	agent.StartAgent(args.Msg, SystemPrompt)
	return mcp.NewToolResultText("调用成功,调用代理"), nil
}

type CallServiceArgs struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

func Complete(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) { //结束恢复
	logrus.Debugf("调用呼叫人工介入函数%+v\n", request)
	argsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	var args CallServiceArgs
	if err := json.Unmarshal(argsBytes, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	switch args.Code {
	case "Call":
		//调用服务人工介入
		//结束恢复动作编码，可选值：Complete、Call、NoFix、bug,Complete表示完成修复，Call表示需要问题无法自动恢复，需要人工介入，bug表示为程序本身的逻辑问题,无法修复，需要NoFix表示当前问题无需修复
		result, err := Call(args.Msg)
		if err != nil {
			return nil, fmt.Errorf("call service failed: %w", err)
		}
		return mcp.NewToolResultText("调用成功,结束恢复:" + result), nil
	case "bug":
		result, err := Call(args.Msg)
		if err != nil {
			return nil, fmt.Errorf("call service failed: %w", err)
		}
		return mcp.NewToolResultText("调用成功,结束恢复:" + result), nil
	default:
		return nil, fmt.Errorf("invalid code: %s", args.Code)
	}
}

func Call(args string) (string, error) {
	//运行指定的人工呼叫命令
	result, err := utils.RunShellCommand(global.Config.System.Call_Command)
	return result, err
}

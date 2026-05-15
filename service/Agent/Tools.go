package agent

import (
	common "AutoOps/commen"
	"AutoOps/global"
	"AutoOps/models"
	"AutoOps/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/sirupsen/logrus"
)

type agentTools struct {
}

var AgentTools = &agentTools{}
var skillReviewRunning atomic.Bool

const SkillReviewPrompt = `你是 AutoOps 的修复复盘助手。你的任务是在一次自动恢复成功后，基于完整修复记录完成两件事：

1. 提炼本次修复中的稳定记忆，便于后续排障复用
2. 判断是否值得沉淀为新的 repair skill，并在合适时自动写入 skills 目录

## 工作原则
- 只根据本次修复记录行动，不要编造未执行过的命令和结论
- 如果当前问题只是偶发、信息不足、步骤不稳定，允许只总结记忆，不创建 skill
- 如果已有高度相似的 skill，优先复用，不要重复创建
- 创建 skill 前先调用 GetSkills，需要时再调用 ReadAgentSkill 查看已有内容
- 可以使用 ShellRun 创建目录、写入文件、检查文件是否生成成功
- 最后必须调用 Complete 结束本轮复盘

## 记忆提炼要求
- 提炼触发条件、根因线索、关键修复动作、验证方式
- 记忆内容以简洁结论为主，不要照搬整段对话
- 本次原始记录已经由系统落盘保存，可结合输入中的文件路径进行处理

## skill 生成要求
- 仅在“触发条件明确、修复步骤稳定、可跨场景复用”时创建
- skill 目录格式必须为 skills/<skill_name>/skill.json
- skill_name 使用小写英文和下划线
- JSON 结构必须包含 skill、description、detail 三个字段
- description 用一句话说明何时使用和核心修复动作
- detail 至少包含 Trigger、Recovery、Verify、Use on 四部分信息
- 写入后再次检查文件内容，确认 JSON 合法

## 结束要求
- 如果已成功沉淀 skill，调用 Complete，code=Complete
- 如果本次只保存记忆但不适合生成 skill，调用 Complete，code=NoFix
- msg 需要简洁说明是否生成了 skill，以及生成的 skill 名称或未生成原因`

//以后考虑使用新的设计
// type ToolDefinition struct {
//     Name        string
//     Description string
//     Parameters  jsonschema.Definition
//     Executor    func(ctx context.Context, args map[string]any) (string, error)
// }

// func (a *agentTools) ListTools() []ToolDefinition {
//     return []ToolDefinition{
//         {Name: "ShellRun", Executor: a.shellRun, ...},
//     }
// }

// // 自动生成 GetTools 和 RegisterTools
// func (a *agentTools) RegisterTools() map[string]func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
//     return map[string]func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error){
//         "ShellRun": a.shellRun,
//     }
// }

func (self *agentTools) GetTools() []openai.Tool {
	// 给在RegisterTools的函数一一写上注册的描述
	var tools []openai.Tool

	// SQLQuery: 查询终端日志（参数与 SQLQueryArgs 一致）
	sqlQueryParams := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"app": {
				Type:        jsonschema.String,
				Description: "应用名称，例如 api-gateway；与 prefix 至少传一个",
			},
			"prefix": {
				Type:        jsonschema.String,
				Description: "日志前缀，例如 analysis；与 app 至少传一个",
			},
			"page_info": {
				Type:        jsonschema.Object,
				Description: "分页与查询参数，对应 common.PageInfo，所有字段均可选",
				Properties: map[string]jsonschema.Definition{
					"endId": {
						Type:        jsonschema.Integer,
						Description: "某页末尾的游标 ID，用于游标翻页，可减轻数据库压力",
					},
					"limit": {
						Type:        jsonschema.Integer,
						Description: "每页数量，范围 1-40，默认 10",
					},
					"page": {
						Type:        jsonschema.Integer,
						Description: "页码，范围 1-20，默认 1",
					},
					"key": {
						Type:        jsonschema.String,
						Description: "模糊匹配关键字，会匹配 content字段",
					},
					"order": {
						Type:        jsonschema.String,
						Description: "排序字段，例如 time desc",
					},
				},
			},
		},
	}
	tools = append(tools, utils.NewOpenAITool(sqlQueryParams, "SQLQuery", "查询终端日志，参数支持 app、prefix、page_info（page_info: endId/limit/page/key/order，app 与 prefix 至少传一个）"))

	// ReadAgentSkill: 读取指定智能体技能信息
	readAgentSkillParams := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"skill": {
				Type:        jsonschema.String,
				Description: "要读取的技能名称",
			},
		},
		Required: []string{"skill"},
	}
	tools = append(tools, utils.NewOpenAITool(readAgentSkillParams, "ReadAgentSkill", "读取指定技能的详细信息"))

	// GetSkills: 获取全部技能列表
	getSkillsParams := jsonschema.Definition{
		Type:       jsonschema.Object,
		Properties: map[string]jsonschema.Definition{},
	}
	tools = append(tools, utils.NewOpenAITool(getSkillsParams, "GetSkills", "获取当前已加载的技能列表,当遇到没有把握修复的错误或者特定程序错误的时候请尝试使用skill"))

	// ShellRun: 执行Shell命令
	shellRunParams := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"command": {
				Type:        jsonschema.String,
				Description: "要执行的Shell命令",
			},
		},
		Required: []string{"command"},
	}
	tools = append(tools, utils.NewOpenAITool(shellRunParams, "ShellRun", "执行Shell命令并返回执行结果"))

	// Complete: 结束恢复流程
	completeParams := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"code": {
				Type:        jsonschema.String,
				Description: "结束恢复动作编码，可选值：Complete、Call、NoFix、bug,Complete表示完成修复，Call表示需要问题无法自动恢复，需要人工介入，bug表示为程序本身的逻辑问题,无法修复，需要NoFix表示当前问题无需修复",
			},
			"msg": {
				Type:        jsonschema.String,
				Description: "结束恢复时返回的说明信息",
			},
		},
		Required: []string{"code", "msg"},
	}

	tools = append(tools, utils.NewOpenAITool(completeParams, "Complete", "结束恢复流程，可选择完成修复、人工介入或标记为无需修复"))
	return tools
}

func (self *agentTools) RegisterTools() map[string]func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return map[string]func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error){
		"SQLQuery":       SQLQuery,
		"ReadAgentSkill": ReadAgentSkill,
		"GetSkills":      GetSkills,
		"ShellRun":       ShellRun,
		"Complete":       Complete,
	}
}

type SQLQueryArgs struct {
	common.PageInfo `json:"page_info"`
	App             string `json:"app"`
	Prefix          string `json:"prefix"`
}

// 目标:调用这个函数
func SQLQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logrus.Debugf("调用SQL查询工具%+v\n", request)

	// 通过 JSON 序列化/反序列化
	argsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	var sqlReq SQLQueryArgs
	if err := json.Unmarshal(argsBytes, &sqlReq); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// 验证必填字段
	if sqlReq.App == "" && sqlReq.Prefix == "" {
		return nil, errors.New("at least one of 'app' or 'prefix' is required")
	}

	// 构建查询
	var model models.TerminalLogModel
	model.App = sqlReq.App
	model.Prefix = sqlReq.Prefix

	var options common.Options
	options.PageInfo = sqlReq.PageInfo
	options.Likes = []string{"content", "level", "prefix"}

	// 注意：这里需要处理返回值
	result, _, err := common.ListQuery(model, options)
	if err != nil {
		return nil, fmt.Errorf("SQL查询失败: %w", err)
	}

	return mcp.NewToolResultText(fmt.Sprintf("SQL查询成功，结果: %+v", result)), nil
}

type ReadSkillArgs struct {
	Skill string `json:"skill"`
}

func ReadAgentSkill(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logrus.Debugf("调用读取智能技能工具: %+v", request)

	argsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	var parsedArgs ReadSkillArgs
	if err := json.Unmarshal(argsBytes, &parsedArgs); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if parsedArgs.Skill == "" {
		return nil, errors.New("skill parameter cannot be empty")
	}
	//热加载一次技能
	InitSkill()

	// 检查skill是否存在
	skillData, exists := AgentSkills[parsedArgs.Skill]
	if !exists {
		return nil, fmt.Errorf("skill '%s' not found", parsedArgs.Skill)
	}

	detail := skillData.Detail
	if detail == "" {
		detail = skillData.Description
	}

	return mcp.NewToolResultText(
		"skill名称: " + skillData.Skill + "\n" +
			"skill描述: " + skillData.Description + "\n" +
			"skill详细信息: " + detail,
	), nil
}

type ShellRunArgs struct {
	Command string `json:"command"`
}

func ShellRun(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logrus.Debugf("调用ShellRun工具%+v\n", request)
	argsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	var parsedArgs ShellRunArgs
	if err := json.Unmarshal(argsBytes, &parsedArgs); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if parsedArgs.Command == "" {
		return nil, errors.New("command is empty")
	}
	// result := global.RecoveryAgent.ShellRun(command)//运行shell脚本,捕获终端信息并回馈给模型
	result, err := utils.RunShellCommand(parsedArgs.Command)
	if err != nil {
		return nil, fmt.Errorf("shell run failed: %w", err)
	}
	return mcp.NewToolResultText("调用成功, 执行结果:" + result), nil
}

func GetSkills(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	//获取skill列表,不需要参数
	InitSkill()
	var result string
	for _, v := range AgentSkills {
		result += "{\n"
		result += "技能名称:" + v.Skill + "\n"
		result += "描述:" + v.Description + "\n"
		result += "}"
	}
	return mcp.NewToolResultText(result), nil
}

type CallServiceArgs struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

func Complete(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) { //结束恢复
	logrus.Debugf("调用结束函数%+v\n", request)
	argsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	var args CallServiceArgs
	if err := json.Unmarshal(argsBytes, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	switch args.Code {
	case "Complete":
		//完成修复
		Done()
		//TODO:完成修复,把历史记录保存下来,自动提炼为Skill
	case "Call":
		//调用服务人工介入
		//结束恢复动作编码，可选值：Complete、Call、NoFix、bug,Complete表示完成修复，Call表示需要问题无法自动恢复，需要人工介入，bug表示为程序本身的逻辑问题,无法修复，需要NoFix表示当前问题无需修复
		result, err := Call(args.Msg)
		if err != nil {
			return nil, fmt.Errorf("call service failed: %w", err)
		}
		return mcp.NewToolResultText("调用成功,结束恢复:" + result), nil
	case "NoFix":
		return mcp.NewToolResultText("调用成功,结束恢复:" + args.Msg), nil
	case "bug":
		result, err := Call(args.Msg)
		if err != nil {
			return nil, fmt.Errorf("call service failed: %w", err)
		}
		return mcp.NewToolResultText("调用成功,结束恢复:" + result), nil
	default:
		return nil, fmt.Errorf("invalid code: %s", args.Code)
	}

	return mcp.NewToolResultText("调用成功,结束恢复:" + args.Msg), nil
}

func Call(args string) (string, error) {
	//运行指定的人工呼叫命令
	result, err := utils.RunShellCommand(global.Config.System.Call_Command)
	return result, err
}

func Done() {
	if skillReviewRunning.Load() {
		logrus.Debug("当前处于复盘流程中,跳过再次触发 Done")
		return
	}

	historyText := buildRecoveryHistoryText()
	if historyText == "" {
		logrus.Warn("恢复完成后未找到可保存的历史记录")
		return
	}

	memoryPath, err := saveRecoveryMemory(historyText)
	if err != nil {
		logrus.Warnf("保存恢复记忆失败:%v", err)
	} else {
		logrus.Infof("恢复记忆已保存:%s", memoryPath)
	}

	skillReviewRunning.Store(true)
	defer skillReviewRunning.Store(false)
	StartSkill(buildSkillReviewInput(historyText, memoryPath), SkillReviewPrompt)
}

// buildRecoveryHistoryText 构建可保存的恢复记录文本
// 参数:无
// 返回:string - 格式化后的恢复记录
// 说明:读取当前恢复历史,保留角色、内容和工具调用,用于落盘和二次提炼
func buildRecoveryHistoryText() string {
	if global.RecoveryAgent.History == nil || len(*global.RecoveryAgent.History) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("# AutoOps 恢复记录\n\n")
	for index, msg := range *global.RecoveryAgent.History {
		builder.WriteString(fmt.Sprintf("## 消息 %d\n", index+1))
		builder.WriteString("角色: " + messageRole(msg) + "\n")
		if msg.Name != "" {
			builder.WriteString("名称: " + msg.Name + "\n")
		}
		if msg.ToolCallID != "" {
			builder.WriteString("工具调用ID: " + msg.ToolCallID + "\n")
		}
		if strings.TrimSpace(msg.Content) != "" {
			builder.WriteString("内容:\n")
			builder.WriteString(msg.Content)
			builder.WriteString("\n")
		}
		if len(msg.ToolCalls) > 0 {
			builder.WriteString("工具调用:\n")
			for _, toolCall := range msg.ToolCalls {
				builder.WriteString(fmt.Sprintf("- %s: %s\n", toolCall.Function.Name, toolCall.Function.Arguments))
			}
		}
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

// saveRecoveryMemory 保存本次恢复记忆
// 参数:historyText - 格式化后的恢复记录
// 返回:string - 保存后的文件路径
// 返回:error - 保存失败时的错误
// 说明:固定写入 memory/recovery 目录,便于后续回溯和二次提炼
func saveRecoveryMemory(historyText string) (string, error) {
	if strings.TrimSpace(historyText) == "" {
		return "", nil
	}

	dir := filepath.Join("memory", "recovery")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	filePath := filepath.Join(dir, "recovery_"+time.Now().Format("20060102_150405")+".md")
	if err := os.WriteFile(filePath, []byte(historyText+"\n"), 0o644); err != nil {
		return "", err
	}
	return filePath, nil
}

// buildSkillReviewInput 组装复盘输入
// 参数:historyText - 格式化后的恢复记录
// 参数:memoryPath - 已保存的记忆文件路径
// 返回:string - 提供给复盘Agent的输入文本
// 说明:同时传入记忆文件路径和原始记录,避免复盘阶段丢失上下文
func buildSkillReviewInput(historyText string, memoryPath string) string {
	var builder strings.Builder
	if memoryPath != "" {
		builder.WriteString("本次恢复记录已保存到: ")
		builder.WriteString(memoryPath)
		builder.WriteString("\n\n")
	}
	builder.WriteString("以下是本次自动恢复的完整记录,请先提炼关键记忆,再判断是否需要沉淀为新的修复 skill。\n\n")
	builder.WriteString(historyText)
	return builder.String()
}

// messageRole 返回消息角色文本
// 参数:msg - OpenAI聊天消息
// 返回:string - 角色名称
// 说明:为空时返回 unknown,避免保存记忆时出现空角色
func messageRole(msg openai.ChatCompletionMessage) string {
	if msg.Role == "" {
		return "unknown"
	}
	return msg.Role
}

type NoneArgs struct {
	Any any `json:"any"`
}

// 用于测试的无操作函数
func None(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) { //无操作
	logrus.Debugf("调用函数None%+v\n", request)
	argsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	var args NoneArgs
	if err := json.Unmarshal(argsBytes, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	//这里根据需要写返回值
	return mcp.NewToolResultText("无操作"), nil
}

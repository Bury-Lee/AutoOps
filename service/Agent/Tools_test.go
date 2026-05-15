package agent

import (
	"AutoOps/global"
	"AutoOps/models"
	"AutoOps/utils"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/glebarez/sqlite"
	"github.com/mark3labs/mcp-go/mcp"
	"gorm.io/gorm"
)

// newAgentTestDB 创建测试用内存数据库
func newAgentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.TerminalLogModel{}); err != nil {
		t.Fatalf("迁移日志表失败: %v", err)
	}
	return db
}

// seedAgentLogs 写入测试日志数据
func seedAgentLogs(t *testing.T, db *gorm.DB) {
	t.Helper()
	rows := []models.TerminalLogModel{
		{App: "api-gateway", Prefix: "ERROR", Content: "panic: nil pointer", Level: "ERROR"},
		{App: "api-gateway", Prefix: "INFO", Content: "request /api/users 200", Level: "INFO"},
	}
	for _, r := range rows {
		if err := db.Create(&r).Error; err != nil {
			t.Fatalf("写入测试日志失败: %v", err)
		}
	}
}

func TestSQLQuery_Success(t *testing.T) {
	db := newAgentTestDB(t)
	seedAgentLogs(t, db)

	restoreDB := global.DB
	global.DB = db
	defer func() { global.DB = restoreDB }()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"app":    "api-gateway",
		"prefix": "ERROR",
		"page_info": map[string]any{
			"page":  float64(1),
			"limit": float64(10),
		},
	}

	result, err := SQLQuery(context.Background(), req)
	if err != nil {
		t.Fatalf("SQLQuery 返回错误: %v", err)
	}

	text := utils.ToolResultToText(result)
	if text == "" {
		t.Fatal("SQLQuery 返回空结果")
	}
}

func TestSQLQuery_MissingAppAndPrefix(t *testing.T) {
	db := newAgentTestDB(t)

	restoreDB := global.DB
	global.DB = db
	defer func() { global.DB = restoreDB }()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	_, err := SQLQuery(context.Background(), req)
	if err == nil {
		t.Fatal("期望返回错误（缺少 app 和 prefix），实际 nil")
	}
}

func TestGetSkills(t *testing.T) {
	// 在临时目录创建 skills 结构
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills", "test_skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("创建测试技能目录失败: %v", err)
	}

	skillData := `{
  "skill": "test_skill",
  "description": "测试技能简介",
  "detail": "测试步骤: step1 && step2"
}`
	if err := os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(skillData), 0o644); err != nil {
		t.Fatalf("写入测试技能文件失败: %v", err)
	}

	// 保存并切换工作目录
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("切换测试工作目录失败: %v", err)
	}

	result, err := GetSkills(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("GetSkills 返回错误: %v", err)
	}

	text := utils.ToolResultToText(result)
	if text == "" {
		t.Fatal("GetSkills 返回空结果")
	}
}

func TestShellRun_Success(t *testing.T) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"command": "echo hello-from-test",
	}

	result, err := ShellRun(context.Background(), req)
	if err != nil {
		t.Fatalf("ShellRun 返回错误: %v", err)
	}

	text := utils.ToolResultToText(result)
	if text == "" {
		t.Fatal("ShellRun 返回空结果")
	}
}

func TestShellRun_EmptyCommand(t *testing.T) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"command": "",
	}

	_, err := ShellRun(context.Background(), req)
	if err == nil {
		t.Fatal("期望返回错误（空命令），实际 nil")
	}
}

func TestComplete_NoFix(t *testing.T) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"code": "NoFix",
		"msg":  "无需修复",
	}

	result, err := Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete NoFix 返回错误: %v", err)
	}

	text := utils.ToolResultToText(result)
	if text == "" {
		t.Fatal("Complete NoFix 返回空结果")
	}
}

func TestComplete_InvalidCode(t *testing.T) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"code": "invalid",
		"msg":  "test",
	}

	_, err := Complete(context.Background(), req)
	if err == nil {
		t.Fatal("期望返回错误（无效 code），实际 nil")
	}
}

func TestComplete_CompleteCallsDone(t *testing.T) {
	// Complete 的 code=Complete 路径调用 Done()（当前为空实现），不应 panic 或报错
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"code": "Complete",
		"msg":  "修复完成",
	}

	result, err := Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete code=Complete 返回错误: %v", err)
	}

	text := utils.ToolResultToText(result)
	if text == "" {
		t.Fatal("Complete code=Complete 返回空结果")
	}
}

func TestAgentTools_GetTools(t *testing.T) {
	tools := AgentTools.GetTools()
	if len(tools) == 0 {
		t.Fatal("AgentTools.GetTools() 返回空列表")
	}

	// 验证所有期望的工具都存在
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Function.Name] = true
	}

	expected := []string{"SQLQuery", "ReadAgentSkill", "GetSkills", "ShellRun", "Complete"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("AgentTools 缺少工具: %s", name)
		}
	}
}

func TestAgentTools_RegisterTools(t *testing.T) {
	registered := AgentTools.RegisterTools()
	if len(registered) == 0 {
		t.Fatal("AgentTools.RegisterTools() 返回空")
	}

	expected := []string{"SQLQuery", "ReadAgentSkill", "GetSkills", "ShellRun", "Complete"}
	for _, name := range expected {
		if _, ok := registered[name]; !ok {
			t.Errorf("RegisterTools 缺少: %s", name)
		}
	}
}

func TestBuildRecoveryHistoryText(t *testing.T) {
	restoreHistory := global.RecoveryAgent.History
	defer func() { global.RecoveryAgent.History = restoreHistory }()

	history := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "nginx 启动失败",
		},
		{
			Role:    openai.ChatMessageRoleAssistant,
			Content: "先检查 nginx 状态",
			ToolCalls: []openai.ToolCall{
				{
					Function: openai.FunctionCall{
						Name:      "ShellRun",
						Arguments: `{"command":"systemctl status nginx"}`,
					},
				},
			},
		},
		{
			Role:       openai.ChatMessageRoleTool,
			Name:       "ShellRun",
			ToolCallID: "call_1",
			Content:    "nginx is active",
		},
	}
	global.RecoveryAgent.History = &history

	text := buildRecoveryHistoryText()
	for _, want := range []string{
		"# AutoOps 恢复记录",
		"角色: user",
		"角色: assistant",
		"工具调用:",
		"ShellRun",
		"nginx is active",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("恢复记录缺少内容 %q, got=%s", want, text)
		}
	}
}

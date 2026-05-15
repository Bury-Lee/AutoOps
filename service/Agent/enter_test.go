package agent

//TODO:更新测试用例
import (
	"AutoOps/conf"
	"AutoOps/models"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// 测试自动恢复,查看是否会自动调用
const (
	ApiKey    = "your-api-key-here"
	Model     = "qwen/qwen3.5-9b"
	Host      = "http://localhost:1234/v1"
	MaxTokens = 2000
	Temp      = 0.7
	TestInput = "请调用智能技能,读取skill:read_skill"
)

var log []string
var Agents conf.Agent
var Skills map[string]AgentSkill

func init() {
	log = []string{
		"[ERROR] order-api panic: nil pointer dereference",
	} // 模拟输入给模型的错误日志
	Agents = conf.Agent{
		ModelName:   Model,
		MaxTokens:   MaxTokens,
		Temperature: float32(Temp),
	}
	// Skills := map[string]AgentSkill{
	// 	"read_skill": {
	// 		Skill:       "read_skill",
	// 		Description: "读取技能详情",
	// 		Detail:      "用于检查并读取技能说明",
	// 	},
	// }
}

// newTestDB 创建测试数据库连接
// 参数:t - 测试上下文
// 返回:*gorm.DB - 测试数据库
// 说明:使用内存sqlite,迁移终端日志表,保证SQLQuery可走成功路径
func newTestDB(t *testing.T) *gorm.DB {
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

// seedTerminalLogs 初始化终端日志数据
// 参数:t - 测试上下文
// 参数:db - 测试数据库
// 返回:无
// 说明:写入一条可查询日志,用于SQLQuery成功测试
func seedTerminalLogs(t *testing.T, db *gorm.DB) {
	t.Helper()
	row := models.TerminalLogModel{
		Time:    time.Now(),
		App:     "Terminal",
		Prefix:  "ERROR",
		Content: "panic: nil pointer dereference",
		Level:   "ERROR",
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("插入日志数据失败: %v", err)
	}
}

// func TestRegisterTools_EachFunc(t *testing.T) {
// 	backupAgent := global.RecoveryAgent
// 	backupDB := global.DB
// 	defer func() {
// 		global.RecoveryAgent = backupAgent
// 		global.DB = backupDB
// 	}()

// 	global.RecoveryAgent = Agents
// 	global.DB = newTestDB(t)
// 	seedTerminalLogs(t, global.DB)

// 	tools := RegisterTools()

// 	t.Run("SQLQuery request 返回正确结构", func(t *testing.T) {
// 		fn, ok := tools["SQLQuery"]
// 		if !ok {
// 			t.Fatal("RegisterTools 缺少 SQLQuery")
// 		}
// 		req := mcp.CallToolRequest{}
// 		req.Params.Arguments = map[string]any{
// 			"app":       "Terminal",
// 			"prefix":    "ERROR",
// 			"page_info": map[string]any{"page": 1, "limit": 10},
// 		}
// 		result, err := fn(context.Background(), req)
// 		if err != nil {
// 			t.Fatalf("调用 SQLQuery 失败: %v", err)
// 		}
// 		if result == nil || len(result.Content) == 0 {
// 			t.Fatal("SQLQuery 返回结构为空")
// 		}
// 		text := utils.ToolResultToText(result)
// 		if !strings.Contains(text, "SQL查询成功") {
// 			t.Fatalf("返回文本不符合预期, got=%s", text)
// 		}
// 	})

// 	t.Run("ReadAgentSkill request 返回正确结构", func(t *testing.T) {
// 		fn, ok := tools["ReadAgentSkill"]
// 		if !ok {
// 			t.Fatal("RegisterTools 缺少 ReadAgentSkill")
// 		}
// 		req := mcp.CallToolRequest{}
// 		req.Params.Arguments = map[string]any{"skill": "read_skill"}
// 		result, err := fn(context.Background(), req)
// 		if err != nil {
// 			t.Fatalf("调用 ReadAgentSkill 失败: %v", err)
// 		}
// 		if result == nil || len(result.Content) == 0 {
// 			t.Fatal("ReadAgentSkill 返回结构为空")
// 		}
// 		text := utils.ToolResultToText(result)
// 		if !strings.Contains(text, "读取技能详情") {
// 			t.Fatalf("返回内容不符合预期, got=%s", text)
// 		}
// 	})

// 	t.Run("ShellRun request 返回正确结构", func(t *testing.T) {
// 		fn, ok := tools["ShellRun"]
// 		if !ok {
// 			t.Fatal("RegisterTools 缺少 ShellRun")
// 		}
// 		req := mcp.CallToolRequest{}
// 		req.Params.Arguments = map[string]any{"command": "echo hello"}
// 		result, err := fn(context.Background(), req)
// 		if err != nil {
// 			t.Fatalf("调用 ShellRun 失败: %v", err)
// 		}
// 		if result == nil || len(result.Content) == 0 {
// 			t.Fatal("ShellRun 返回结构为空")
// 		}
// 		text := utils.ToolResultToText(result)
// 		if !strings.Contains(text, "调用成功,ShellRun:echo hello") {
// 			t.Fatalf("返回内容不符合预期, got=%s", text)
// 		}
// 	})

// 	t.Run("CallService request 返回正确结构", func(t *testing.T) {
// 		fn, ok := tools["CallService"]
// 		if !ok {
// 			t.Fatal("RegisterTools 缺少 CallService")
// 		}
// 		req := mcp.CallToolRequest{}
// 		req.Params.Arguments = CallServiceArgs{Code: "123"}
// 		result, err := fn(context.Background(), req)
// 		if err != nil {
// 			t.Fatalf("调用 CallService 失败: %v", err)
// 		}
// 		if result == nil || len(result.Content) == 0 {
// 			t.Fatal("CallService 返回结构为空")
// 		}
// 		text := utils.ToolResultToText(result)
// 		if !strings.Contains(text, "调用成功,呼叫服务") {
// 			t.Fatalf("返回内容不符合预期, got=%s", text)
// 		}
// 	})
// }

// func TestStartAgent_AICall_WithMockLLM(t *testing.T) {
// 	backupAgent := global.RecoveryAgent
// 	defer func() {
// 		global.RecoveryAgent = backupAgent
// 	}()

// 	var round int32
// 	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
// 			http.NotFound(w, r)
// 			return
// 		}
// 		body, _ := io.ReadAll(r.Body)
// 		defer r.Body.Close()

// 		var req map[string]any
// 		_ = json.Unmarshal(body, &req)
// 		if req["model"] != Model {
// 			t.Errorf("模型参数不正确, got=%v", req["model"])
// 		}

// 		if atomic.LoadInt32(&round) == 0 {
// 			msgs, _ := req["messages"].([]any)
// 			if len(msgs) < 2 {
// 				t.Errorf("消息数量不符合预期, got=%d", len(msgs))
// 			}
// 			tools, _ := req["tools"].([]any)
// 			if len(tools) == 0 {
// 				t.Error("首轮请求未携带工具定义")
// 			}
// 			first := map[string]any{
// 				"id":      "chatcmpl-1",
// 				"object":  "chat.completion",
// 				"created": 1,
// 				"model":   Model,
// 				"choices": []any{
// 					map[string]any{
// 						"index": 0,
// 						"message": map[string]any{
// 							"role":    "assistant",
// 							"content": "",
// 							"tool_calls": []any{
// 								map[string]any{
// 									"id":   "call-1",
// 									"type": "function",
// 									"function": map[string]any{
// 										"name":      "ReadAgentSkill",
// 										"arguments": `{"skill":"read_skill"}`,
// 									},
// 								},
// 							},
// 						},
// 						"finish_reason": "tool_calls",
// 					},
// 				},
// 			}
// 			atomic.StoreInt32(&round, 1)
// 			_ = json.NewEncoder(w).Encode(first)
// 			return
// 		}

// 		second := map[string]any{
// 			"id":      "chatcmpl-2",
// 			"object":  "chat.completion",
// 			"created": 2,
// 			"model":   Model,
// 			"choices": []any{
// 				map[string]any{
// 					"index": 0,
// 					"message": map[string]any{
// 						"role":    "assistant",
// 						"content": "已完成恢复建议",
// 					},
// 					"finish_reason": "stop",
// 				},
// 			},
// 		}
// 		_ = json.NewEncoder(w).Encode(second)
// 	}))
// 	defer mockLLM.Close()

// 	cfg := openai.DefaultConfig(ApiKey)
// 	cfg.BaseURL = mockLLM.URL + "/v1"

// 	Agents.LLM = openai.NewClientWithConfig(cfg)
// 	global.RecoveryAgent = Agents

// 	StartAgent(log[0], SystemPrompt)

// 	if atomic.LoadInt32(&round) != 1 {
// 		t.Fatalf("模型未完成两轮对话调用, round=%d", atomic.LoadInt32(&round))
// 	}
// 	if global.RecoveryAgent.History == nil {
// 		t.Fatal("历史记录为空")
// 	}
// 	if len(*global.RecoveryAgent.History) < 5 {
// 		t.Fatalf("历史记录长度不符合预期, got=%d", len(*global.RecoveryAgent.History))
// 	}

// 	foundToolMessage := false
// 	for _, msg := range *global.RecoveryAgent.History {
// 		if msg.Role == openai.ChatMessageRoleTool &&
// 			msg.Name == "ReadAgentSkill" &&
// 			strings.Contains(msg.Content, "skill详细信息") {
// 			foundToolMessage = true
// 			break
// 		}
// 	}
// 	if !foundToolMessage {
// 		t.Fatal("未找到工具调用结果消息,未验证到skill调用成功")
// 	}
// }

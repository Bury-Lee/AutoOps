package command

import (
	"AutoOps/conf"
	"AutoOps/global"
	"AutoOps/models"
	"AutoOps/utils"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestProcessLogQueue 测试消费日志队列循环
// 参数: t - 测试上下文
// 返回: 无
// 说明: 使用内存数据库测试日志队列消费，验证日志等级匹配和入库逻辑
func TestProcessLogQueue(t *testing.T) {
	// Setup in-memory sqlite
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect database: %v", err)
	}
	db.AutoMigrate(&models.TerminalLogModel{})

	// Mock global.DB
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	// 准备测试数据
	sub := make(chan *utils.LogEntry, 10)
	sub <- &utils.LogEntry{Content: "This is an info message"}
	sub <- &utils.LogEntry{Content: "Error: something went wrong"}
	sub <- &utils.LogEntry{Content: "Fatal crash occurred"}
	sub <- &utils.LogEntry{Content: "Random text"}
	close(sub)

	// 配置日志选项
	option := &models.TerminalOption{
		App:    "TestApp",
		Prefix: "TestPrefix",
		Level: []conf.LevelRule{
			{Level: "info", Pattern: "(?i)info"},
			{Level: "error", Pattern: "(?i)error"},
			{Level: "fatal", Pattern: "(?i)fatal"},
		},
		AlertLevel: []string{"fatal"}, // 触发 HappenError (Goroutine 异步执行)
	}

	// 执行目标函数，使用 -1 作为 id 避免 ai.CatchInfo 的副作用
	processLogQueue(sub, option, -1)

	// 验证数据库入库情况
	var logs []models.TerminalLogModel
	if err := db.Order("id asc").Find(&logs).Error; err != nil {
		t.Fatalf("Query logs failed: %v", err)
	}

	if len(logs) != 4 {
		t.Fatalf("Expected 4 logs, got %d", len(logs))
	}

	// 验证第一条日志
	if logs[0].Level != "info" {
		t.Errorf("Expected first log level to be 'info', got '%s'", logs[0].Level)
	}
	if logs[0].Content != "This is an info message" {
		t.Errorf("Unexpected content: %s", logs[0].Content)
	}
	if logs[0].App != "TestApp" || logs[0].Prefix != "TestPrefix" {
		t.Errorf("Unexpected App or Prefix")
	}

	// 验证第二条日志
	if logs[1].Level != "error" {
		t.Errorf("Expected second log level to be 'error', got '%s'", logs[1].Level)
	}
	if logs[1].Content != "Error: something went wrong" {
		t.Errorf("Unexpected content: %s", logs[1].Content)
	}

	// 验证第三条日志 (匹配 fatal，并触发 Alert)
	if logs[2].Level != "fatal" {
		t.Errorf("Expected third log level to be 'fatal', got '%s'", logs[2].Level)
	}
	if logs[2].Content != "Fatal crash occurred" {
		t.Errorf("Unexpected content: %s", logs[2].Content)
	}

	// 验证第四条日志 (未匹配任何规则)
	if logs[3].Level != "unknown" {
		t.Errorf("Expected fourth log level to be 'unknown', got '%s'", logs[3].Level)
	}
	if logs[3].Content != "Random text" {
		t.Errorf("Unexpected content: %s", logs[3].Content)
	}
}

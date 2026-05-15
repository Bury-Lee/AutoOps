package core

import (
	"AutoOps/conf"
	"AutoOps/global"
	"AutoOps/models"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func TestReadConf(t *testing.T) {
	restoreWD := chdirTemp(t)
	defer restoreWD()

	content := `run_mode: "release"
log:
  app: "tester"
  dir: "logs"
  log_level: "info"
db:
  sql_name: "sqlite"
  db_name: "main.db"
ai_db:
  sql_name: "sqlite"
  db_name: "ai.db"
analys_ai:
  model: "analysis-model"
  temperature: 0.2
  max_tokens: 128
  host: "http://analysis.example/v1"
  ApiKey: "analysis-key"
  apiType: "openai"
agent_ai:
  model: "agent-model"
  temperature: 0.5
  max_tokens: 256
  host: "http://agent.example/v1"
  ApiKey: "agent-key"
  apiType: "azure"
system:
  ip: "127.0.0.1"
  port: "8080"
  allow_rpg: true
  allow_remote: false
`
	writeFile(t, filepath.Join(mustGetwd(t), fileName), content)

	cfg := ReadConf()

	if cfg.RunMode != "release" {
		t.Fatalf("RunMode = %q, want release", cfg.RunMode)
	}
	if cfg.Log.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want info", cfg.Log.LogLevel)
	}
	if cfg.DB.SqlName != conf.DBSqliteMode {
		t.Fatalf("DB.SqlName = %q, want %q", cfg.DB.SqlName, conf.DBSqliteMode)
	}
	if cfg.AnalysAI.Host != "http://analysis.example/v1" {
		t.Fatalf("AnalysAI.Host = %q, want analysis host", cfg.AnalysAI.Host)
	}
	if !cfg.System.AllowRPG {
		t.Fatal("AllowRPG = false, want true")
	}
}

func TestReadConfPanicsOnInvalidYAML(t *testing.T) {
	restoreWD := chdirTemp(t)
	defer restoreWD()

	writeFile(t, filepath.Join(mustGetwd(t), fileName), "log:\n  log_level: [broken")

	defer func() {
		if recover() == nil {
			t.Fatal("ReadConf() did not panic for invalid yaml")
		}
	}()

	ReadConf()
}

func TestInitAnalysAI(t *testing.T) {
	restoreGlobal := snapshotGlobalState()
	defer restoreGlobal()

	global.Config.AnalysAI = conf.AI{
		Host:    "http://analysis.example/v1",
		ApiKey:  "analysis-key",
		APIType: "azure",
	}

	client := InitAnalysAI()
	if client == nil {
		t.Fatal("InitAnalysAI() returned nil client")
	}

	clientConfig := extractClientConfig(t, client)
	if clientConfig.BaseURL != global.Config.AnalysAI.Host {
		t.Fatalf("BaseURL = %q, want %q", clientConfig.BaseURL, global.Config.AnalysAI.Host)
	}
	if clientConfig.APIType != openai.APIType(global.Config.AnalysAI.APIType) {
		t.Fatalf("APIType = %q, want %q", clientConfig.APIType, global.Config.AnalysAI.APIType)
	}
	if extractClientAuthToken(t, clientConfig) != global.Config.AnalysAI.ApiKey {
		t.Fatal("auth token does not match config")
	}
}

func TestInitAgent(t *testing.T) {
	restoreGlobal := snapshotGlobalState()
	defer restoreGlobal()

	global.Config.AgentAI = conf.AI{
		Model:       "agent-model",
		Temperature: 0.6,
		MaxTokens:   512,
		Host:        "http://agent.example/v1",
		ApiKey:      "agent-key",
		APIType:     "openai",
	}

	agent := InitAgent()

	if agent.LLM == nil {
		t.Fatal("InitAgent() returned nil LLM")
	}
	if agent.History == nil {
		t.Fatal("History = nil, want empty slice pointer")
	}
	if len(*agent.History) != 0 {
		t.Fatalf("History length = %d, want 0", len(*agent.History))
	}
	if agent.ModelName != global.Config.AgentAI.Model {
		t.Fatalf("ModelName = %q, want %q", agent.ModelName, global.Config.AgentAI.Model)
	}
	if agent.MaxTokens != global.Config.AgentAI.MaxTokens {
		t.Fatalf("MaxTokens = %d, want %d", agent.MaxTokens, global.Config.AgentAI.MaxTokens)
	}
	if agent.Temperature != global.Config.AgentAI.Temperature {
		t.Fatalf("Temperature = %v, want %v", agent.Temperature, global.Config.AgentAI.Temperature)
	}
	if len(agent.Tools) != 0 {
		t.Fatalf("Tools length = %d, want 0", len(agent.Tools))
	}

	clientConfig := extractClientConfig(t, agent.LLM)
	if clientConfig.BaseURL != global.Config.AgentAI.Host {
		t.Fatalf("BaseURL = %q, want %q", clientConfig.BaseURL, global.Config.AgentAI.Host)
	}
}

func TestInitSkillCreatesDirectory(t *testing.T) {
	restoreWD := chdirTemp(t)
	defer restoreWD()

	InitSkill()

	info, err := os.Stat(filepath.Join(mustGetwd(t), "skills"))
	if err != nil {
		t.Fatalf("skills dir stat failed: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("skills exists but is not a directory")
	}
}

func TestInitDB(t *testing.T) {
	restoreGlobal := snapshotGlobalState()
	defer restoreGlobal()

	global.Config.RunMode = "release"
	global.Config.DB = conf.DB{
		SqlName: conf.DBSqliteMode,
		DBName:  filepath.Join(t.TempDir(), "main.db"),
	}

	db := InitDB()
	defer closeGormDB(t, db)

	if db == nil {
		t.Fatal("InitDB() returned nil")
	}
	if err := db.Exec("SELECT 1").Error; err != nil {
		t.Fatalf("database ping query failed: %v", err)
	}
}

func TestInitAiDB(t *testing.T) {
	restoreGlobal := snapshotGlobalState()
	defer restoreGlobal()

	global.Config.RunMode = "release"
	global.Config.AiDB = conf.DB{
		SqlName: conf.DBSqliteMode,
		DBName:  filepath.Join(t.TempDir(), "ai.db"),
	}

	db := InitAiDB()
	defer closeGormDB(t, db)

	if db == nil {
		t.Fatal("InitAiDB() returned nil")
	}
	if err := db.Exec("SELECT 1").Error; err != nil {
		t.Fatalf("ai database ping query failed: %v", err)
	}
}

func TestConsoleFormatterFormat(t *testing.T) {
	formatter := &ConsoleFormatter{}
	entry := &logrus.Entry{
		Logger:  logrus.New(),
		Level:   logrus.WarnLevel,
		Message: "console-message",
		Time:    time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
		Caller: &runtime.Frame{
			Function: "AutoOps/core.TestConsoleFormatterFormat",
			File:     "C:/workspace/core/enter_test.go",
			Line:     88,
		},
	}

	got, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	text := string(got)
	if !strings.Contains(text, "\x1b[") {
		t.Fatal("console formatter output should contain ANSI color")
	}
	if !strings.Contains(text, "enter_test.go:88") {
		t.Fatal("console formatter output missing caller file")
	}
	if !strings.Contains(text, "console-message") {
		t.Fatal("console formatter output missing message")
	}
}

func TestFileFormatterFormat(t *testing.T) {
	formatter := &FileFormatter{}
	entry := &logrus.Entry{
		Logger:  logrus.New(),
		Level:   logrus.InfoLevel,
		Message: "file-message",
		Time:    time.Date(2026, 5, 9, 12, 30, 0, 0, time.UTC),
		Caller: &runtime.Frame{
			Function: "AutoOps/core.TestFileFormatterFormat",
			File:     "C:/workspace/core/enter_test.go",
			Line:     120,
		},
	}

	got, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	text := string(got)
	if strings.Contains(text, "\x1b[") {
		t.Fatal("file formatter output should not contain ANSI color")
	}
	if !strings.Contains(text, "[info]") {
		t.Fatal("file formatter output missing level")
	}
	if !strings.Contains(text, "file-message") {
		t.Fatal("file formatter output missing message")
	}
}

func TestInitFileWritesLog(t *testing.T) {
	restoreLogger := snapshotLoggerState()
	defer restoreLogger()

	logDir := t.TempDir()
	nowDate := time.Now().Format("2006-01-02")

	logger := logrus.StandardLogger()
	logger.SetOutput(io.Discard)
	logger.SetLevel(logrus.InfoLevel)

	InitFile(logDir, "autoops-test")
	logrus.WithTime(time.Now()).Info("write-log-file")

	logFile := filepath.Join(logDir, nowDate, "autoops-test.log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file failed: %v", err)
	}
	if !strings.Contains(string(data), "write-log-file") {
		t.Fatal("log file does not contain written message")
	}
}

func TestInitLogrus(t *testing.T) {
	restoreGlobal := snapshotGlobalState()
	defer restoreGlobal()
	restoreLogger := snapshotLoggerState()
	defer restoreLogger()

	logDir := t.TempDir()
	global.Config.RunMode = "develop"
	global.Config.Log = conf.Log{
		App:      "autoops-test",
		Dir:      logDir,
		LogLevel: "warn",
	}

	InitLogrus()

	logger := logrus.StandardLogger()
	if logger.Level != logrus.WarnLevel {
		t.Fatalf("logger level = %v, want %v", logger.Level, logrus.WarnLevel)
	}
	if !logger.ReportCaller {
		t.Fatal("ReportCaller = false, want true in develop mode")
	}
	if _, ok := logger.Formatter.(*ConsoleFormatter); !ok {
		t.Fatalf("formatter type = %T, want *ConsoleFormatter", logger.Formatter)
	}

	logFile := filepath.Join(logDir, time.Now().Format("2006-01-02"), "autoops-test.log")
	if _, err := os.Stat(logFile); err != nil {
		t.Fatalf("expected log file to be created: %v", err)
	}
}

func TestInitLogrusPanicsOnMissingLogLevel(t *testing.T) {
	restoreGlobal := snapshotGlobalState()
	defer restoreGlobal()
	restoreLogger := snapshotLoggerState()
	defer restoreLogger()

	global.Config.RunMode = "release"
	global.Config.Log = conf.Log{
		App: "autoops-test",
		Dir: t.TempDir(),
	}

	defer func() {
		if recover() == nil {
			t.Fatal("InitLogrus() did not panic when log level was empty")
		}
	}()

	InitLogrus()
}

func TestInitRouterRoutes(t *testing.T) {
	restoreGlobal := snapshotGlobalState()
	defer restoreGlobal()

	originalMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	defer gin.SetMode(originalMode)

	global.Config.System = conf.System{
		AllowRPG:    true,
		AllowRemote: true,
	}

	router := InitRouter()
	routeSet := make(map[string]bool)
	for _, route := range router.Routes() {
		routeSet[route.Method+" "+route.Path] = true
	}

	expected := []string{
		"POST /command",
		"GET /log",
		"GET /log/:id",
		"GET /logList",
	}
	for _, item := range expected {
		if !routeSet[item] {
			t.Fatalf("route %q not registered", item)
		}
	}
}

func TestCommandRejectsInvalidJSON(t *testing.T) {
	originalMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	defer gin.SetMode(originalMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest("POST", "/command", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	Command(ctx)

	if recorder.Code != 400 {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "message") {
		t.Fatal("response body missing error message")
	}
}

func TestLogDetail(t *testing.T) {
	restoreGlobal := snapshotGlobalState()
	defer restoreGlobal()

	db := newTestDB(t, "log-detail.db")
	global.DB = db

	record := models.TerminalLogModel{
		Time:    time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
		App:     "autoops",
		Prefix:  "INFO",
		Content: "detail-log",
		Level:   "INFO",
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create log record failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/log/1", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}

	LogDetail(ctx)

	if recorder.Code != 200 {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "detail-log") {
		t.Fatal("response body missing log content")
	}
}

func TestLogList(t *testing.T) {
	restoreGlobal := snapshotGlobalState()
	defer restoreGlobal()

	db := newTestDB(t, "log-list.db")
	global.DB = db

	record := models.TerminalLogModel{
		Time:    time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
		App:     "autoops",
		Prefix:  "INFO",
		Content: "list-log",
		Level:   "INFO",
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create log record failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/logList?page=1&limit=10", nil)

	LogList(ctx)

	if recorder.Code != 200 {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var resp struct {
		Message string                    `json:"message"`
		Data    []models.TerminalLogModel `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data length = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].Content != "list-log" {
		t.Fatalf("content = %q, want list-log", resp.Data[0].Content)
	}
}

func snapshotGlobalState() func() {
	oldConfig := global.Config
	oldDB := global.DB
	oldAiDB := global.AiDB
	oldAnalysAIClient := global.AnalysAIClient
	oldRecoveryAgent := global.RecoveryAgent

	return func() {
		global.Config = oldConfig
		global.DB = oldDB
		global.AiDB = oldAiDB
		global.AnalysAIClient = oldAnalysAIClient
		global.RecoveryAgent = oldRecoveryAgent
	}
}

func snapshotLoggerState() func() {
	logger := logrus.StandardLogger()
	oldOut := logger.Out
	oldFormatter := logger.Formatter
	oldLevel := logger.Level
	oldReportCaller := logger.ReportCaller
	oldHooks := logger.ReplaceHooks(make(logrus.LevelHooks))

	return func() {
		logger.SetOutput(oldOut)
		logger.SetFormatter(oldFormatter)
		logger.SetLevel(oldLevel)
		logger.SetReportCaller(oldReportCaller)
		logger.ReplaceHooks(oldHooks)
	}
}

func chdirTemp(t *testing.T) func() {
	t.Helper()

	oldWD := mustGetwd(t)
	newWD := t.TempDir()
	if err := os.Chdir(newWD); err != nil {
		t.Fatalf("chdir temp dir failed: %v", err)
	}

	return func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore workdir failed: %v", err)
		}
	}
}

func mustGetwd(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	return wd
}

func writeFile(t *testing.T, filePath, content string) {
	t.Helper()

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
}

func newTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), name)
	db, err := gorm.Open((&conf.DB{
		SqlName: conf.DBSqliteMode,
		DBName:  dbPath,
	}).DSN(), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db failed: %v", err)
	}
	if err := db.AutoMigrate(&models.TerminalLogModel{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	t.Cleanup(func() {
		closeGormDB(t, db)
	})
	return db
}

func closeGormDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db failed: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db failed: %v", err)
	}
}

func extractClientConfig(t *testing.T, client *openai.Client) openai.ClientConfig {
	t.Helper()

	value := reflect.ValueOf(client)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		t.Fatal("client is nil")
	}

	field := value.Elem().FieldByName("config")
	if !field.IsValid() {
		t.Fatal("client config field not found")
	}

	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(openai.ClientConfig)
}

func extractClientAuthToken(t *testing.T, cfg openai.ClientConfig) string {
	t.Helper()

	value := reflect.ValueOf(&cfg).Elem().FieldByName("authToken")
	if !value.IsValid() {
		t.Fatal("client authToken field not found")
	}

	return reflect.NewAt(value.Type(), unsafe.Pointer(value.UnsafeAddr())).Elem().String()
}

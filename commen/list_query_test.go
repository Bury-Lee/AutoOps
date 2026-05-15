package common

import (
	"AutoOps/global"
	"AutoOps/models"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// newListQueryDB 创建测试用内存数据库并迁移 TerminalLogModel
func newListQueryDB(t *testing.T) *gorm.DB {
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

// seedLogs 写入多条测试日志，每条日志 Content 不同以区分
func seedLogs(t *testing.T, db *gorm.DB) {
	t.Helper()
	logs := []models.TerminalLogModel{
		{App: "svc-a", Prefix: "INFO", Content: "startup complete", Level: "INFO"},
		{App: "svc-a", Prefix: "ERROR", Content: "connection refused", Level: "ERROR"},
		{App: "svc-b", Prefix: "INFO", Content: "health check ok", Level: "INFO"},
		{App: "svc-c", Prefix: "WARN", Content: "disk usage 85%", Level: "WARN"},
		{App: "svc-a", Prefix: "ERROR", Content: "timeout on /api/users", Level: "ERROR"},
	}
	for _, l := range logs {
		if err := db.Create(&l).Error; err != nil {
			t.Fatalf("写入日志失败: %v", err)
		}
	}
}
func TestListQuery_Pagination(t *testing.T) {
	db := newListQueryDB(t)
	seedLogs(t, db)

	restoreDB := global.DB
	global.DB = db
	defer func() { global.DB = restoreDB }()

	// 分页第1页，每页2条
	list, count, err := ListQuery(models.TerminalLogModel{}, Options{
		PageInfo: PageInfo{Page: 1, Limit: 2},
	})
	if err != nil {
		t.Fatalf("ListQuery 失败: %v", err)
	}
	if count != 5 {
		t.Fatalf("count = %d, want 5", count)
	}
	if len(list) != 2 {
		t.Fatalf("page1 len = %d, want 2", len(list))
	}

	// 分页第2页
	list2, count2, err := ListQuery(models.TerminalLogModel{}, Options{
		PageInfo: PageInfo{Page: 2, Limit: 2},
	})
	if err != nil {
		t.Fatalf("ListQuery page2 失败: %v", err)
	}
	if count2 != 5 {
		t.Fatalf("count2 = %d, want 5", count2)
	}
	if len(list2) != 2 {
		t.Fatalf("page2 len = %d, want 2", len(list2))
	}

	// 分页第3页，只剩1条
	list3, _, err := ListQuery(models.TerminalLogModel{}, Options{
		PageInfo: PageInfo{Page: 3, Limit: 2},
	})
	if err != nil {
		t.Fatalf("ListQuery page3 失败: %v", err)
	}
	if len(list3) != 1 {
		t.Fatalf("page3 len = %d, want 1", len(list3))
	}
}
func TestListQuery_FuzzyMatch(t *testing.T) {
	db := newListQueryDB(t)
	seedLogs(t, db)

	restoreDB := global.DB
	global.DB = db
	defer func() { global.DB = restoreDB }()

	// 模糊匹配 "connection"
	list, count, err := ListQuery(models.TerminalLogModel{}, Options{
		PageInfo: PageInfo{Key: "connection"},
		Likes:    []string{"content"},
	})
	if err != nil {
		t.Fatalf("ListQuery fuzzy 失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("fuzzy connection count = %d, want 1", count)
	}
	if len(list) != 1 || list[0].Content != "connection refused" {
		t.Fatalf("fuzzy connection result mismatch: %+v", list)
	}

	// 模糊匹配 "health"
	list2, count2, err := ListQuery(models.TerminalLogModel{}, Options{
		PageInfo: PageInfo{Key: "health"},
		Likes:    []string{"content"},
	})
	if err != nil {
		t.Fatalf("ListQuery fuzzy health 失败: %v", err)
	}
	if count2 != 1 {
		t.Fatalf("fuzzy health count = %d, want 1", count2)
	}
	if list2[0].Content != "health check ok" {
		t.Fatalf("fuzzy health result mismatch: %+v", list2)
	}
}
func TestListQuery_FuzzyMatchNoHit(t *testing.T) {
	db := newListQueryDB(t)
	seedLogs(t, db)

	restoreDB := global.DB
	global.DB = db
	defer func() { global.DB = restoreDB }()

	list, count, err := ListQuery(models.TerminalLogModel{}, Options{
		PageInfo: PageInfo{Key: "nonexistent"},
		Likes:    []string{"content"},
	})
	if err != nil {
		t.Fatalf("ListQuery no-hit 失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("no-hit count = %d, want 0", count)
	}
	if len(list) != 0 {
		t.Fatalf("no-hit list len = %d, want 0", len(list))
	}
}
func TestListQuery_CursorPagination(t *testing.T) {
	db := newListQueryDB(t)
	seedLogs(t, db)

	restoreDB := global.DB
	global.DB = db
	defer func() { global.DB = restoreDB }()

	// 先用普通分页获取最后一页的末尾 ID
	listAll, _, err := ListQuery(models.TerminalLogModel{}, Options{
		PageInfo: PageInfo{Page: 1, Limit: 10},
	})
	if err != nil {
		t.Fatalf("ListQuery all 失败: %v", err)
	}
	if len(listAll) != 5 {
		t.Fatalf("listAll len = %d, want 5", len(listAll))
	}
	lastID := listAll[4].ID // 第5条的 ID

	// 游标分页：从 lastID 之前取 2 条
	list, count, err := ListQuery(models.TerminalLogModel{}, Options{
		PageInfo: PageInfo{EndId: lastID, Limit: 2},
	})
	if err != nil {
		t.Fatalf("ListQuery cursor 失败: %v", err)
	}
	if count != 5 {
		t.Fatalf("cursor count = %d, want 5", count)
	}
	if len(list) != 2 {
		t.Fatalf("cursor len = %d, want 2", len(list))
	}
	// 游标结果应该都是 ID < lastID
	for _, item := range list {
		if item.ID >= lastID {
			t.Fatalf("cursor result ID %d >= lastID %d", item.ID, lastID)
		}
	}
}
func TestListQuery_WhereFilter(t *testing.T) {
	db := newListQueryDB(t)
	seedLogs(t, db)

	restoreDB := global.DB
	global.DB = db
	defer func() { global.DB = restoreDB }()

	// 仅查询 svc-a 的日志
	list, count, err := ListQuery(models.TerminalLogModel{
		App: "svc-a",
	}, Options{
		PageInfo: PageInfo{Page: 1, Limit: 10},
	})
	if err != nil {
		t.Fatalf("ListQuery where 失败: %v", err)
	}
	if count != 3 {
		t.Fatalf("where svc-a count = %d, want 3", count)
	}
	for _, item := range list {
		if item.App != "svc-a" {
			t.Fatalf("where result has unexpected App: %s", item.App)
		}
	}
}
func TestListQuery_DefaultOrder(t *testing.T) {
	db := newListQueryDB(t)
	seedLogs(t, db)

	restoreDB := global.DB
	global.DB = db
	defer func() { global.DB = restoreDB }()

	list, _, err := ListQuery(models.TerminalLogModel{}, Options{
		PageInfo:     PageInfo{Page: 1, Limit: 10},
		DefaultOrder: "created_at desc",
	})
	if err != nil {
		t.Fatalf("ListQuery defaultOrder 失败: %v", err)
	}
	if len(list) != 5 {
		t.Fatalf("defaultOrder len = %d, want 5", len(list))
	}
}


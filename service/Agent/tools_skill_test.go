package agent

import (
	"AutoOps/utils"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestReadAgentSkillReturnsDetail 验证读取技能时会返回 detail 内容
// 参数:t - 测试上下文
// 返回:无
// 说明:在临时目录构造 skills 文件,切换工作目录后调用工具
func TestReadAgentSkillReturnsDetail(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills", "test_skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("创建测试技能目录失败: %v", err)
	}

	skillData := `{
  "skill": "test_skill",
  "description": "测试技能简介",
  "detail": "测试技能详细步骤: step1 && step2"
}`
	if err := os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(skillData), 0o644); err != nil {
		t.Fatalf("写入测试技能文件失败: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("切换测试工作目录失败: %v", err)
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"skill": "test_skill",
	}

	result, err := ReadAgentSkill(context.Background(), req)
	if err != nil {
		t.Fatalf("调用 ReadAgentSkill 失败: %v", err)
	}

	text := utils.ToolResultToText(result)
	if !strings.Contains(text, "skill描述: 测试技能简介") {
		t.Fatalf("返回结果缺少技能描述: %s", text)
	}
	if !strings.Contains(text, "skill详细信息: 测试技能详细步骤: step1 && step2") {
		t.Fatalf("返回结果缺少技能详细信息: %s", text)
	}
}

package flags

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type skillConfig struct {
	Skill       string `json:"skill"`
	Description string `json:"description"`
	Detail      string `json:"detail"`
}

// NewSkill 创建技能目录和默认配置文件
// 参数:无
// 返回:无
// 说明:交互式读取技能信息,校验技能名,已存在时不覆盖
func NewSkill() {
	//TODO:改进一下,把skillName,description,detail都写到skill.json文件里
	// 确保skills根目录存在
	if err := os.MkdirAll("skills", 0755); err != nil {
		fmt.Printf("创建skills目录失败: %v\n", err)
		return
	}
	reader := bufio.NewReader(os.Stdin)

	// 引导用户输入技能名
	fmt.Print("请输入 skill 名称: ")
	skillName, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("读取 skill 名称失败: %v\n", err)
		return
	}
	skillName = strings.TrimSpace(skillName)
	if skillName == "" {
		fmt.Println("skill 名称不能为空")
		return
	}

	// 引导用户输入技能描述
	fmt.Print("请输入 skill 描述: ")
	description, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("读取 skill 描述失败: %v\n", err)
		return
	}
	description = strings.TrimSpace(description)

	// 引导用户输入技能详细说明
	fmt.Print("请输入 skill 详细说明: ")
	detail, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("读取 skill 详细说明失败: %v\n", err)
		return
	}
	detail = strings.TrimSpace(detail)

	// 确保skills根目录存在
	if err := os.MkdirAll("skills", 0755); err != nil {
		fmt.Printf("创建skills目录失败: %v\n", err)
		return
	}

	skillDir := filepath.Join("skills", skillName)
	if _, err := os.Stat(skillDir); err == nil {
		fmt.Printf("技能 %s 已存在: %s\n", skillName, skillDir)
		return
	} else if !os.IsNotExist(err) {
		fmt.Printf("检查技能目录失败: %v\n", err)
		return
	}

	// 创建技能目录
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		fmt.Printf("创建技能目录失败: %v\n", err)
		return
	}

	if description == "" {
		description = "请填写技能描述"
	}
	if detail == "" {
		detail = "请填写技能的详细描述"
	}

	config := skillConfig{
		Skill:       skillName,
		Description: description,
		Detail:      detail,
	}

	data, err := json.Marshal(config)
	if err != nil {
		fmt.Printf("生成skill.json失败: %v\n", err)
		return
	}

	skillFile := filepath.Join(skillDir, "skill.json")
	// 创建默认skill.json配置
	if err := os.WriteFile(skillFile, data, 0644); err != nil {
		fmt.Printf("写入skill.json失败: %v\n", err)
		return
	}

	fmt.Printf("技能创建成功: %s\n", skillFile)
}

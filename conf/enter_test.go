package conf

//TODO:更新测试用例

// 测试InitSkill方法
// func TestInitSkill(t *testing.T) {
// 	// 1. 准备测试数据：创建临时的skills目录和skill.json文件
// 	err := os.MkdirAll("skills/test_skill", 0755)
// 	if err != nil {
// 		t.Fatalf("创建测试目录失败: %v", err)
// 	}

// 	// 2. 创建测试用的skill.json文件
// 	testSkill := AgentSkill{
// 		Skill:       "test_skill",
// 		Description: "这是一个测试技能",
// 		Detail:      "测试技能的详细描述",
// 	}
// 	skillData, err := json.Marshal(testSkill)
// 	if err != nil {
// 		t.Fatalf("序列化测试数据失败: %v", err)
// 	}

// 	err = os.WriteFile("skills/test_skill/skill.json", skillData, 0644)
// 	if err != nil {
// 		t.Fatalf("写入测试文件失败: %v", err)
// 	}

// 	// 3. 创建Agent实例
// 	agent := &Agent{
// 		Skill: make(map[string]AgentSkill),
// 	}

// 	// 4. 调用InitSkill方法
// 	agent.InitSkill()

// 	// 5. 输出结果并验证
// 	t.Logf("InitSkill执行后的内容: %+v", agent.Skill)

// 	// 6. 验证结果
// 	if len(agent.Skill) == 0 {
// 		t.Errorf("期望Skill map不为空，实际为空")
// 	}

// 	if skill, exists := agent.Skill["test_skill"]; !exists {
// 		t.Errorf("期望找到key为'test_skill'的技能，但未找到")
// 	} else {
// 		if skill.Description != "这是一个测试技能" {
// 			t.Errorf("期望Description为'这是一个测试技能'，实际为'%s'", skill.Description)
// 		}
// 	}

// 	t.Log("测试完成")
// }

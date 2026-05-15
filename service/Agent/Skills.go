package agent

import (
	"encoding/json"
	"os"

	"github.com/sirupsen/logrus"
)

var AgentSkills map[string]AgentSkill

type AgentSkill struct {
	Load        bool   `json:"load"` //启用时,会在第一次就强制加载
	Skill       string `json:"skill"`
	Description string `json:"description"`
	Detail      string `json:"detail"`
}

// 把Skill独立出来作为工具,变成两层结构,工具1:返回所有skill+描述,工具2:返回某个skill详细信息
func InitSkill() {
	AgentSkills = make(map[string]AgentSkill)
	//扫描Skill文件夹,加载所有技能
	dirs, err := os.ReadDir("skills")
	for _, dir := range dirs {
		if dir.IsDir() {
			//如果是文件夹,就扫描内部的skill.json文件,把内容赋值到AgentSkills中
			data, err := os.ReadFile("skills/" + dir.Name() + "/skill.json")
			if err != nil {
				logrus.Warnf("读取skills文件夹失败:%v", err)
				return
			}
			var Skill AgentSkill
			err = json.Unmarshal(data, &Skill)
			if err != nil {
				logrus.Warnf("解析skills文件夹失败:%v", err)
				return
			}
			AgentSkills[Skill.Skill] = Skill // 把skill名作为key,把技能内容作为value
		}
	}
	if err != nil {
		logrus.Warnf("读取skills文件夹失败:%v", err)
		return
	}
}

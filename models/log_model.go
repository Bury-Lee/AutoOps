// models/log_model.go
// 日志模型定义
// 记录系统操作日志、用户登录日志等信息
package models

import (
	"AutoOps/conf"
	"time"
)

type TerminalLogModel struct { //原始终端日志模型
	Model
	Time    time.Time `json:"time"`    // 日志时间戳
	App     string    `json:"app"`     //来自哪个服务
	Prefix  string    `json:"prefix"`  //日志前缀
	Content string    `json:"content"` // 日志内容
	Level   string    `json:"level"`   // 日志级别枚举
}

type TerminalOption struct { //默认终端日志配置
	App        string   `json:"app"`         //来自哪个服务
	MaxSize    int      `json:"max_size"`    //单条日志截断长度
	Prefix     string   `json:"prefix"`      //日志前缀，默认INFO
	BufferSize int      `json:"buffer_size"` //缓冲区大小
	AlertLevel []string `json:"alert_level"` //触发警报的等级,默认ERROR

	Level []conf.LevelRule `json:"level"` //各等级的正则表达式,表示为[等级]=正则表达式

	AdditionalFields map[string]string `json:"additional_fields"` //额外字段,用于存储自定义信息,比如环境信息,yaml配置等,如果要启用自动修复,强烈建议把环境信息填入,可以让自动恢复ai快速定位到问题
}

func (self *TerminalOption) IsSetLogLevel() bool { //是否设置了信息日志等级判定
	if self.Level == nil { //表示未启用任何等级判定
		return false
	}
	return true
}

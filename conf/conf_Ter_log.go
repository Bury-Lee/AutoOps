package conf

type TerminalLog struct {
	App        string      `json:"app" yaml:"app"`
	MaxSize    int         `json:"max_size" yaml:"max_size"`
	Truncate   string      `json:"truncate" yaml:"truncate"`
	Prefix     string      `json:"prefix" yaml:"prefix"`
	BufferSize int         `json:"buffer_size" yaml:"buffer_size"`
	AlertLevel []string    `json:"alert_level" yaml:"alert_level"`
	Level      []LevelRule `json:"level" yaml:"level"`
}
type LevelRule struct {
	Level   string `yaml:"level"`
	Pattern string `yaml:"pattern"`
}

func (self *TerminalLog) IsSetInfoLevel() bool { //是否设置了信息日志等级判定
	if self.Level == nil { //表示未启用任何等级判定
		return false
	}
	return true
}

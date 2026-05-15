package conf

type Config struct {
	RunMode     string      `yaml:"run_mode"` //运行模式
	Log         Log         `yaml:"log"`
	DB          DB          `yaml:"db"`
	AiDB        DB          `yaml:"ai_db"`
	TerminalLog TerminalLog `yaml:"terminal_log"`
	AnalysAI    AI          `yaml:"analys_ai"`
	AgentAI     AI          `yaml:"agent_ai"`
	System      System      `yaml:"system"`
}

const Version = "1.0.0"

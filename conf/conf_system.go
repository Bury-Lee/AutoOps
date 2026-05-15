package conf

type System struct {
	IP   string `yaml:"ip"`
	Port string `yaml:"port"`
	//鉴权的工作交由鉴权中间件处理
	AllowRPG     bool   `yaml:"allow_rpg"`    //是否允许远程调用
	AllowRemote  bool   `yaml:"allow_remote"` //是否允许远程查询日志
	Call_Command string `yaml:"call_command"` //呼叫人工服务的命令
}

func (s *System) Addr() string {
	return s.IP + ":" + s.Port
}

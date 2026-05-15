package flags

import "fmt"

func ShowHelp() { //先这样吧,之后再看看怎么优化,实现自动注册选项
	helpText := `
AutoOps - 自动化运维工具

USAGE:
    ./AutoOps [选项]

OPTIONS:
    -h, -help              显示此帮助信息
    -v                     显示版本信息
    -db                    执行数据库迁移
    
    -t <类型>              指定操作类型
                           可选值: skill, test, init, r
    
    -s <文件路径>          指定配置文件路径 (支持JSON/YAML格式)
    
    -c "<命令>"            直接执行指定的原始命令
    
    -tmod                  启用终端模式 (实验性功能，体验类似普通终端)
    
    -cmod                  启用终端聊天模式 (实验性功能，AI助手模式)

EXAMPLES:
    # 基础操作
    ./AutoOps -v                              # 查看版本
    ./AutoOps -db                             # 数据库迁移
    ./AutoOps -h                              # 显示帮助
    
    # 操作类型示例
    ./AutoOps -t init                         # 初始化配置
    ./AutoOps -t test                         # 测试连接
    ./AutoOps -t skill                        # 运行技能
    
    # 执行命令
    ./AutoOps -c "ls -la"                     # 执行系统命令
    ./AutoOps -c "deploy" -s config.json      # 使用配置执行命令
    ./AutoOps -t r -s /path/to/config.json    # 通过JSON文件运行
    
    # 实验性功能
    ./AutoOps -tmod                           # 进入终端模式
    ./AutoOps -cmod                           # 进入聊天模式

NOTES:
    • 多个主命令不能同时使用，请一次只选择一种运行方式
    • 使用 -t r 时必须同时指定 -s 参数
    • init 操作和查看版本/帮助不需要启动完整服务
    • 配置文件支持JSON和YAML两种格式

了解更多信息请访问项目文档或联系维护者。
`
	fmt.Print(helpText)
}

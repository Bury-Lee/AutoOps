package main

import (
	"AutoOps/core"
	"AutoOps/flags"
	"AutoOps/global"
	cron_service "AutoOps/service/cron"
	"encoding/json"

	"github.com/sirupsen/logrus"
)

func main() {
	flags.Parse()                               //解析命令行参数
	if !flags.FlagOptions.RequiresBootstrap() { //如果是进行初始化,就不执行其他配置行为
		flags.Run()
		return
	}
	global.Config = *core.ReadConf()            //读取文件
	core.InitLogrus()                           //初始化日志
	core.InitSkill()                            //初始化技能文件夹
	global.DB = core.InitDB()                   //初始化数据库
	global.AiDB = core.InitAiDB()               //初始化AI数据库
	global.AnalysAIClient = core.InitAnalysAI() //初始化分析模型
	global.RecoveryAgent = core.InitAgent()     //初始化自愈AI模型

	//debug模式下打印配置
	if global.Config.RunMode == "develop" {
		configDebug, err := json.MarshalIndent(global.Config, "", "  ")
		if err != nil {
			logrus.Error("配置文件反序列化失败:", err)
			return
		}
		logrus.Debug(string(configDebug))
	}

	// 启动定时任务
	go cron_service.CronArticle()

	if global.Config.System.AllowRemote || global.Config.System.AllowRPG { //只有启用远程查看日志或远程调用时才启动监控服务
		router := core.InitRouter()
		router.Run(global.Config.System.Addr())
	}
	flags.Run() //运行命令行参数
}

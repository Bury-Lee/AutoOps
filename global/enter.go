package global

import (
	"AutoOps/conf"

	"github.com/sashabaranov/go-openai"
	"gorm.io/gorm"
)

var Config conf.Config
var DB *gorm.DB
var AnalysAIClient *openai.Client
var RecoveryAgent conf.Agent

var AiDB *gorm.DB //给ai用的数据库,和DB分开,以免ai的操作影响到主数据库

// 其他全局变量...

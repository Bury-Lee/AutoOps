package flags

import (
	"AutoOps/global"
	"AutoOps/models"

	"github.com/sirupsen/logrus"
)

func FlagDB() { //数据库迁移
	err := global.DB.AutoMigrate(
		&models.TerminalLogModel{},
	)
	if err != nil {
		logrus.Errorf("数据库迁移失败 %s", err)
	} else {
		logrus.Info("数据库已迁移")
	}
}

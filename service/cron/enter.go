package cron_service

import (
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

func CronArticle() {
	var crontab *cron.Cron
	timezone, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		logrus.Warnf("无法设置时区,已使用UTC时区:%v", err)
		crontab = cron.New(cron.WithSeconds(), cron.WithLocation(time.UTC))
	} else {
		crontab = cron.New(cron.WithSeconds(), cron.WithLocation(timezone))
	}
	crontab.AddFunc("0 */10 * * * *", cleanLog) //每10分钟一次检查
	crontab.Start()
}

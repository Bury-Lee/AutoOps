package cron_service

import (
	"AutoOps/global"
	"AutoOps/models"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// 加锁执行,一次只允许一个线程进行清理

var Lock sync.Mutex

func cleanLog() {
	// 计算过期时间（30天前）
	expireTime := time.Now().AddDate(0, 0, -30)
	//也许最好再带个游标?
	if !Lock.TryLock() {
		return
	} else {
		defer Lock.Unlock()
		for {
			logrus.Debugf("开始清理超过一个月的浏览记录")

			// 执行删除（每次最多删除50条）
			tx := global.DB.
				Where("created_at < ?", expireTime).
				Limit(50).
				Delete(&models.TerminalLogModel{})

			if tx.Error != nil {
				logrus.Errorf("清理失败: %v", tx.Error)
				return
			}

			affected := tx.RowsAffected

			// 如果本次一条都没删，说明已经清理完了
			if affected == 0 {
				logrus.Infof("浏览记录清理完成")
				return
			}

			logrus.Debugf("本次清理 %d 条记录", affected)

			// 每批之间休眠，避免数据库压力过大
			time.Sleep(5 * time.Second)
		}
	}

}

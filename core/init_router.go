package core

import (
	common "AutoOps/commen"
	"AutoOps/global"
	"AutoOps/models"
	"AutoOps/service/command"
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func InitRouter() *gin.Engine {
	// 初始化路由
	router := gin.Default()
	if global.Config.System.AllowRPG {
		//执行指令路由
		router.POST("/command", Command)
	}
	if global.Config.System.AllowRemote {
		//远程监控日志路由
		router.GET("/log") //预留:通过websocket远程推送日志流
		//查询单条日志路由
		router.GET("/log/:id", LogDetail)
		//查询多条日志路由
		router.GET("/logList", LogList)
	}
	return router
}

type CommandRequest struct {
	Command string                 `json:"command"`
	Options *models.TerminalOption `json:"options"`
}

func Command(c *gin.Context) {
	// 处理指令路由
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"message": err.Error(),
		})
		return
	}
	logrus.Infof("收到指令:%v", req)

	ctx := context.Background()
	// 执行指令
	command.StartCommand(ctx, req.Command, req.Options)
	// 返回成功
	response := ""
	if req.Options != nil {
		response = fmt.Sprintf("已加入操作队列:%s,操作选项:%v", req.Command, req.Options)
	} else {
		response = fmt.Sprintf("已加入操作队列:%s", req.Command)
	}
	c.JSON(200, gin.H{
		"message": response,
	})
}

type LogListRequest struct {
	common.PageInfo
}

func LogList(c *gin.Context) {
	var req LogListRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, gin.H{
			"message": err.Error(),
		})
		return
	}

	Options := common.Options{
		PageInfo:     req.PageInfo,
		DefaultOrder: "created_at desc", //默认按创建时间降序排序
	}
	List, count, err := common.ListQuery(models.TerminalLogModel{}, Options)
	if err != nil {
		c.JSON(500, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"message": fmt.Sprintf("查询到%d条日志", count),
		"data":    List,
	})
	// 处理查询多条日志路由
}

type LogDetailRequest struct {
	ID int `uri:"id"`
}

func LogDetail(c *gin.Context) {
	// 处理查询单条日志路由
	var req LogDetailRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(400, gin.H{
			"message": err.Error(),
		})
		return
	}
	var log models.TerminalLogModel
	if err := global.DB.Where("id = ?", req.ID).First(&log).Error; err != nil {
		c.JSON(404, gin.H{
			"message": "日志不存在",
		})
		return
	}

	c.JSON(200, gin.H{
		"data": log,
	})
}

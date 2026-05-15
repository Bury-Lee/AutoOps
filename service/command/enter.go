package command

import (
	"AutoOps/global"
	"AutoOps/models"
	ai "AutoOps/service/AI"
	agent "AutoOps/service/Agent"
	"io"
	"regexp"

	"AutoOps/utils"
	"bufio"
	"context"
	"os/exec"
	"runtime"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
)

// 消费日志队列循环
func processLogQueue(sub <-chan *utils.LogEntry, option *models.TerminalOption, id int) {
	Level := make([]*regexp.Regexp, 0, len(option.Level))
	// 预编译日志等级匹配规则，避免循环内重复编译
	if option.IsSetLogLevel() {
		for _, matcher := range option.Level {
			compiled, err := utils.NewMatcher(matcher.Pattern)
			if err != nil {
				logrus.Warn(err)
			}
			Level = append(Level, compiled)
		}
	}

	for entry := range sub {
		level := "unknown"
		for i, matcher := range Level {
			if matcher != nil && matcher.MatchString(entry.Content) {
				level = option.Level[i].Level
				break
			}
		}

		global.DB.Create(&models.TerminalLogModel{
			Time:    time.Now(),
			App:     option.App,
			Prefix:  option.Prefix,
			Level:   level,
			Content: entry.Content,
		})
		if utils.ContainsString(option.AlertLevel, level) {
			go agent.HappenError(entry.Content) //逐个匹配
			//提示ai当前问题已经移交给ai,不必再次调用服务
			entry.Content = "=====\n本条日志已启用自动恢复,不必再次调用自动恢复服务:" + entry.Content + "\n====="
		}
		//TODO:在这里考虑进行option.AdditionalFields的填入衔接
		ai.CatchInfo(id, entry.Content)
	}
}

// 扫描终端输出
func scanCommandOutput(reader io.Reader, option *models.TerminalOption, logBuffer *utils.LogBuffer, isError bool) {
	scanner := bufio.NewScanner(reader)
	maxCapacity := option.BufferSize
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		bytes := scanner.Bytes()
		line := string(bytes)
		if runtime.GOOS == "windows" && !utf8.Valid(bytes) {
			if utf8Line, err := utils.ConvertGBKToUTF8(bytes); err == nil {
				line = utf8Line
			}
		}

		logrus.Debugf("命令行输出: %s", line)
		if isError {
			logrus.Errorf("%s", line)
		}

		logBuffer.Push(line, "来自服务:"+option.App+",来自前缀:"+option.Prefix)
		if isError {
			//目前存在一个问题,扫描过快导致触发非常频繁,需要优化
			// agent.HappenError(fmt.Sprintf("命令执行错误: %s,来自: %s", line, option.App+":"+option.Prefix))
		}
	}
}

// 扫描stdout的封装
func scanStdout(stdout io.Reader, option *models.TerminalOption, logBuffer *utils.LogBuffer) {
	scanCommandOutput(stdout, option, logBuffer, false)
}

// 扫描stderr的封装
func scanStderr(stderr io.Reader, option *models.TerminalOption, logBuffer *utils.LogBuffer) {
	scanCommandOutput(stderr, option, logBuffer, true)
}

// 启动命令
func StartCommand(ctx context.Context, Content string, option *models.TerminalOption) { //TODO:以后加入上下文取消功能,允许运行时中止命令
	if option == nil { //启用默认配置
		option = &models.TerminalOption{
			App:        global.Config.TerminalLog.App,
			MaxSize:    global.Config.TerminalLog.MaxSize,
			Prefix:     global.Config.TerminalLog.Prefix,
			BufferSize: global.Config.TerminalLog.BufferSize,
			Level:      global.Config.TerminalLog.Level,
			AlertLevel: global.Config.TerminalLog.AlertLevel,
		}
	}
	if option.BufferSize == 0 {
		option.BufferSize = 1024 * 512 //默认512KB
	}

	logrus.Debugf("以配置 %+v 执行命令: %s", option, Content)

	//这里调用管道和服务
	// 启动一个子进程，例如运行 "ping" 命令
	// cmd := exec.Command(rawContent)
	// 原封不动地执行用户输入的命令
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		logrus.Debugf("开始执行: %s", Content)
		cmd = exec.CommandContext(ctx, "cmd", "/C", Content)
	default:
		logrus.Debugf("开始执行: %s", Content)
		cmd = exec.CommandContext(ctx, "sh", "-c", Content)
	}

	// 获取命令的stdout管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logrus.Error(err)
		return
	}

	// 获取命令的stderr管道
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logrus.Error(err)
		return
	}

	//初始化消息队列
	var logBuffer utils.LogBuffer
	var wg sync.WaitGroup

	// 订阅日志并处理（在启动命令前订阅，防止丢失日志）
	sub := logBuffer.Subscribe(option.BufferSize)

	id := ai.Init()
	logBuffer.Push("[执行命令] "+Content, option.App+":"+option.Prefix)

	var dbWg sync.WaitGroup
	dbWg.Add(1)

	go func() {
		defer dbWg.Done()
		processLogQueue(sub, option, id)
	}()

	// 启动命令
	if err := cmd.Start(); err != nil {
		logrus.Error(err)
		// 如果启动失败，需要清理资源
		logBuffer.Drop()
		dbWg.Wait()
		return
	}

	// 使用goroutine处理stdout,当有消息时,放入消息队列
	wg.Add(1)

	go func() {
		defer wg.Done()
		scanStdout(stdout, option, &logBuffer)
		//后期加入远程推送功能,利用管道消费,可以把stdout的输出推送到远程服务器
	}()

	// 使用goroutine处理stderr,当有消息时,放入消息队列
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanStderr(stderr, option, &logBuffer)
	}()

	// 等待扫描器完成（即命令输出结束）
	wg.Wait()

	// 等待命令执行结束并清理资源
	err = cmd.Wait()
	if err != nil {
		logrus.Errorf("命令执行结束，退出状态: %v", err)
	}

	// 关闭日志缓冲区，通知订阅者结束
	logBuffer.Drop()

	// 等待数据库写入完成
	dbWg.Wait()

	// 关闭AI处理并处理最后一批消息
	ai.Close(id)
}

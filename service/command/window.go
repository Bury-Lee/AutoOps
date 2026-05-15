package command

//实验性功能: 终端助手模式,用户可以在终端中输入命令,并实时收到ai助手回复,目前无法实现伪终端功能,暂时搁置
import (
	"AutoOps/global"
	"AutoOps/models"
	ai "AutoOps/service/AI"
	"AutoOps/utils"
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
)

const (
	defaultCmodWindowTurns = 50
)

type CmodWindow struct {
	sync.Mutex          // 使用原子锁
	Items      []string // 存储命令历史记录
	HasNew     bool     // 是否有新消息
	IsEnd      bool     // 是否结束
}

func (this *CmodWindow) StartListener() {
	for {
		if this.IsEnd {
			return
		}
		if this.HasNew {
			this.SendToAI()
			this.HasNew = false
		} else {
			time.Sleep(time.Second)
			continue
		}
	}
}

func (this *CmodWindow) AddNew(cmd string) {
	this.Lock()
	logrus.Debugf("addNew获取到锁: %s", cmd)
	defer this.Unlock()
	if len(this.Items) > defaultCmodWindowTurns {
		this.Items = this.Items[1:] // 移除最早命令
	}
	this.Items = append(this.Items, cmd)
	this.HasNew = true
	logrus.Debugf("addNew解锁: %s", cmd)
}
func (this *CmodWindow) Add(msg string) {
	this.Lock()
	logrus.Debugf("Add获取到锁: %s", msg)
	defer this.Unlock()
	if len(this.Items) > defaultCmodWindowTurns {
		this.Items = this.Items[1:] // 移除最早命令
	}
	this.Items = append(this.Items, msg)
	logrus.Debugf("Add解锁: %s", msg)
}

// 问题一定出在这里,每次运行都导致发送给ai的api调用次数+1
// 需要锁的调用另一个需要锁的函数就百分百死锁
func (this *CmodWindow) SendToAI() { //发送给ai助手并将回复添加到历史记录
	this.Lock()
	defer this.Unlock()
	logrus.Debugf("sendToAI获取到锁")
	reply := ai.SendToAIByCmod(this.Items, ai.SystemPromptCmod)
	if len(this.Items) > defaultCmodWindowTurns {
		this.Items = this.Items[1:] // 移除最早命令
	}
	this.Items = append(this.Items, reply)
	logrus.Debugf("sendToAI解锁: %s", reply)
}

func (this *CmodWindow) End() {
	this.Lock()
	defer this.Unlock()
	this.IsEnd = true
	logrus.Debug("退出Cmod模式")
}

// 在Cmod模式下执行命令
func StartServerByCmod(ctx context.Context, window *CmodWindow, Content string, option *models.TerminalOption) { //TODO:以后加入上下文取消功能,允许运行时中止命令
	if option == nil { //启用默认配置
		option = &models.TerminalOption{
			App:        global.Config.TerminalLog.App,
			MaxSize:    global.Config.TerminalLog.MaxSize,
			Prefix:     global.Config.TerminalLog.Prefix,
			AlertLevel: global.Config.TerminalLog.AlertLevel,
			BufferSize: global.Config.TerminalLog.BufferSize,
			Level:      global.Config.TerminalLog.Level,
		}
	}
	if option.BufferSize == 0 {
		option.BufferSize = 1024 * 512 //默认512KB
	}

	logrus.Debugf("以配置 %+v 执行命令: %s", option, Content)

	if window == nil {
		logrus.Error("窗口为nil")
	}
	//这里调用管道和服务
	// 启动一个子进程，例如运行 "ping" 命令
	// cmd := exec.Command(rawContent)
	// 原封不动地执行用户输入的命令
	var cmd *exec.Cmd

	if Content[0] == '?' { //?的前缀直接发送给ai
		window.Add("[用户发送给ai助手]" + Content[1:]) // 移除?前缀
		window.SendToAI()
		return
	}

	switch runtime.GOOS {
	case "windows": //Windows系统使用cmd执行
		// fmt.Print("开始执行:", Content, "\n")
		cmd = exec.CommandContext(ctx, "cmd", "/C", Content)
		window.AddNew("[用户Windows下执行命令]" + Content)

	default:
		// fmt.Print("开始执行:", Content, "\n")
		cmd = exec.CommandContext(ctx, "sh", "-c", Content)
		window.AddNew("[用户Linux下执行命令]" + Content)
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

	var dbWg sync.WaitGroup
	dbWg.Add(1)
	go func() { // 处理生产的日志队列
		defer dbWg.Done()
		for entry := range sub {
			global.DB.Create(&models.TerminalLogModel{
				Time:    time.Now(),
				App:     option.App,
				Prefix:  option.Prefix,
				Level:   "unknown", //等正则表达式完成了再确定等级
				Content: entry.Content,
			})
			// 这里可以实现WebSocket推送、HTTP请求等方式
			window.AddNew("[命令行输出]" + entry.Content + "\n")
		}
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
		scanner := bufio.NewScanner(stdout)
		maxCapacity := option.BufferSize // 使用更多内存
		buf := make([]byte, maxCapacity)
		scanner.Buffer(buf, maxCapacity)
		for scanner.Scan() {
			bytes := scanner.Bytes()
			line := string(bytes)
			// Windows 环境下，如果不是 UTF-8 编码，则尝试从 GBK 转换为 UTF-8
			if runtime.GOOS == "windows" && !utf8.Valid(bytes) {
				if utf8Line, err := utils.ConvertGBKToUTF8(bytes); err == nil {
					line = utf8Line
				}
			}
			if global.Config.RunMode == "develop" {
				logrus.Debugf("命令行输出: %s", line)
			} else {
				fmt.Println(line) // 换行打印
			}
			// 发送消息
			logBuffer.Push("[命令行输出]"+line, option.App+":"+option.Prefix)
		}
	}()

	// 使用goroutine处理stderr,当有消息时,放入消息队列
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		maxCapacity := option.BufferSize // 使用更多内存
		buf := make([]byte, maxCapacity)
		scanner.Buffer(buf, maxCapacity)
		for scanner.Scan() {
			bytes := scanner.Bytes()
			line := string(bytes)
			// Windows 环境下，如果不是 UTF-8 编码，则尝试从 GBK 转换为 UTF-8
			if runtime.GOOS == "windows" && !utf8.Valid(bytes) {
				if utf8Line, err := utils.ConvertGBKToUTF8(bytes); err == nil {
					line = utf8Line
				}
			}
			if global.Config.RunMode == "develop" {
				logrus.Errorf("命令行输出: %s", line)
			} else {
				fmt.Println(line) // 换行打印
			}
			// 加入队列
			logBuffer.Push(line, option.App+":"+option.Prefix)
		}
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
	//这里不用处理了,反正这里结束说明用户主动退出了,直接返回即可
	logrus.Debug("执行完成")
}

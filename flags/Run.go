package flags

import (
	"AutoOps/models"
	"AutoOps/service/command"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

type CommandJson struct {
	Command string                 `json:"command"`
	Option  *models.TerminalOption `json:"option"`
}

// 按配置运行
func RunByJson(Path string) {
	// 读取配置文件
	data, err := os.ReadFile(Path)
	if err != nil {
		fmt.Printf("读取Json文件失败: %v\n", err)
		return
	}

	// 解析命令配置
	var commands []CommandJson
	if err = json.Unmarshal(data, &commands); err != nil {
		fmt.Printf("解析Json文件失败: %v\n", err)
		return
	}

	if len(commands) == 0 {
		fmt.Println("Json文件中未找到可执行命令")
		return
	}

	// 按配置并发运行
	var wait sync.WaitGroup
	for _, item := range commands {
		if strings.TrimSpace(item.Command) == "" {
			continue
		}

		wait.Add(1)
		go func(command CommandJson) {
			defer wait.Done()
			StartServer(command.Command, command.Option)
		}(item)
	}

	wait.Wait()
}

// 接收函数，拿到的是原始字符串
func StartServer(rawContent string, option *models.TerminalOption) {
	//按照回车分割命令,然后启用多个管道,每个管道执行一个任务
	commands := strings.Split(rawContent, "\n")
	var wg sync.WaitGroup
	for _, cmd := range commands {
		if cmd == "" {
			continue
		}
		wg.Add(1)
		go func(c string) {
			defer wg.Done()
			ctx := context.Background()
			command.StartCommand(ctx, c, option)
		}(cmd)
	}
	wg.Wait()
}

// 接收函数，拿到的是原始字符串(带主动取消上下文)
func StartServerWithContext(ctx context.Context, rawContent string, option *models.TerminalOption) { //这个基本上就是专门用于终端模式的
	//按照回车分割命令,然后启用多个管道,每个管道执行一个任务
	commands := strings.Split(rawContent, "\n")
	var wg sync.WaitGroup
	for _, cmd := range commands {
		if cmd == "" {
			continue
		}
		wg.Add(1)
		go func(c string) {
			defer wg.Done()
			command.StartCommand(ctx, c, option)
		}(cmd)
	}
	wg.Wait()
}

// 解析Json文件
func ParseJson(Path string) *models.TerminalOption {
	// 读取配置文件
	data, err := os.ReadFile(Path)
	if err != nil {
		fmt.Printf("读取Json文件失败: %v\n", err)
		return nil
	}
	// 解析Json文件
	var option models.TerminalOption
	if err = json.Unmarshal(data, &option); err != nil {
		fmt.Printf("解析Json文件失败: %v\n", err)
		return nil
	}
	return &option
}

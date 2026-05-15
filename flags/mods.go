package flags

import (
	"AutoOps/service/command"
	"context"
	"fmt"
	"os"
	"strings"
)

func Cmod() {
	fmt.Println("进入终端助手模式，输入命令按回车执行。命令行第一个字符为\"?\"时将不执行命令并把命令移交给ai助手")
	fmt.Println("执行期间可使用快捷键 Ctrl+S并回车 中断当前任务，输入 'exitcmod' 退出")
	fmt.Print("AutoOps> ")

	var window command.CmodWindow
	defer window.End()
	go window.StartListener() // 监听协程仅启动一次，避免重复触发 SendToAI

	// 开启单一协程读取标准输入，避免多个 reader 冲突
	inputChan := make(chan byte)
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				close(inputChan)
				return
			}
			inputChan <- buf[0]
		}
	}()

	var cmdBuf []byte
	for {
		cmdBuf = cmdBuf[:0] // 清空缓存

		// 读取命令行输入
		for b := range inputChan {
			if b == '\n' {
				break
			}
			if b == '\r' {
				continue
			}
			cmdBuf = append(cmdBuf, b)
		}

		cmdStr := strings.TrimSpace(string(cmdBuf))
		if cmdStr == "" {
			continue
		}
		if strings.ToLower(cmdStr) == "exitcmod" {
			fmt.Println("退出终端模式。")
			break
		}

		// 创建带取消功能的上下文
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		defer cancel()

		//问题出现在执行程序的地方
		// 开启协程执行命令
		go func(c string) {
			command.StartServerByCmod(ctx, &window, c, nil)
			close(done) //执行完成后关闭channel
		}(cmdStr)

		// 阻塞等待命令完成或接收到 ctrl+s+enter
	waitLoop:
		for {
			select {
			case <-done:
				break waitLoop
			case b, ok := <-inputChan:
				if !ok {
					return // 输入流关闭，退出
				}
				if b == 19 { // 19 是 Ctrl+S 的 ASCII 码
					fmt.Println("\n[收到 Ctrl+S，正在中断当前任务...]")
					cancel()
				}
				// 任务执行期间的其他字符输入暂时丢弃，避免干扰下一次输入
			}
		}
		fmt.Print("AutoOps> ")
	}
}

func Tmod() {
	fmt.Println("进入终端模式，输入命令按回车执行。")
	fmt.Println("执行期间可使用快捷键 Ctrl+S (或输入 Ctrl+S 并回车) 中断当前任务，输入 'exit' 退出。")

	// 开启单一协程读取标准输入，避免多个 reader 冲突
	inputChan := make(chan byte)
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				close(inputChan)
				return
			}
			inputChan <- buf[0]
		}
	}()

	var cmdBuf []byte
	for {

		cmdBuf = cmdBuf[:0] // 清空缓存

		// 读取命令行输入
		for b := range inputChan {
			if b == '\n' {
				break
			}
			if b == '\r' {
				continue
			}
			cmdBuf = append(cmdBuf, b)
		}

		cmdStr := strings.TrimSpace(string(cmdBuf))
		if cmdStr == "" {
			continue
		}
		if strings.ToLower(cmdStr) == "exit" {
			fmt.Println("退出终端模式。")
			break
		}

		// 创建带取消功能的上下文
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		defer cancel()

		// 开启协程执行命令
		go func(c string) {
			StartServerWithContext(ctx, c, nil)
			close(done) //执行完成后关闭channel
		}(cmdStr)

		// 阻塞等待命令完成或接收到 ctrl+s+enter

		//当前bug: ctrl+s+enter 中断后直接就无法执行别的命令了
	waitLoop:
		for {
			select {
			case <-done:
				break waitLoop
			case b, ok := <-inputChan:
				if !ok {
					return // 输入流关闭，退出
				}
				if b == 19 { // 19 是 Ctrl+S 的 ASCII 码
					fmt.Println("\n[收到 Ctrl+S，正在中断当前任务...]")
					cancel()
				}
				// 任务执行期间的其他字符输入暂时丢弃，避免干扰下一次输入
			}
		}
		fmt.Print("AutoOps> ")
	}
}

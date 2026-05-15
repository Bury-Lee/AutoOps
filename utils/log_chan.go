package utils

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// LogEntry 表示日志内容
type LogEntry struct {
	Content string // 日志内容
	Source  string // 日志来源标识
}

// LogBuffer 是一个线程安全的日志广播器
// 支持管道订阅广播模式，不进行日志持久化存储
type LogBuffer struct {
	mu   sync.Mutex       // 保护所有字段的并发访问
	subs []chan *LogEntry // 订阅者列表，用于广播新日志
}

// Push 向所有订阅者广播一条日志
// 参数:content - 日志内容, source - 日志来源标识
// 返回:无
// 说明:线程安全地创建日志对象并分发给所有活跃订阅者,非阻塞模式以防死锁
func (this *LogBuffer) Push(content string, source string) {
	this.mu.Lock()
	defer this.mu.Unlock()

	newEntry := &LogEntry{
		Content: content,
		Source:  source,
	}

	// 广播给所有订阅者
	for _, sub := range this.subs {
		select {
		case sub <- newEntry:
		default: // 防止订阅者阻塞导致死锁
			logrus.Warn("LogBuffer: Push: 订阅者管道已满,无法发送日志")
		}
	}
}

// Subscribe 订阅日志管道
// 参数:size - 管道缓冲区大小
// 返回:ch - 返回一个只读的日志管道
// 说明:将新创建的管道加入订阅列表,返回给调用方进行监听
func (this *LogBuffer) Subscribe(size int) <-chan *LogEntry {
	this.mu.Lock()
	defer this.mu.Unlock()

	ch := make(chan *LogEntry, size) // 使用带缓冲的管道
	this.subs = append(this.subs, ch)
	return ch
}

// Unsubscribe 取消订阅日志管道
// 参数:ch - 需要取消订阅的管道
// 返回:无
// 说明:从订阅列表中移除指定的管道,并关闭该管道
func (this *LogBuffer) Unsubscribe(ch <-chan *LogEntry) {
	this.mu.Lock()
	defer this.mu.Unlock()

	for i, sub := range this.subs {
		if sub == ch {
			this.subs = append(this.subs[:i], this.subs[i+1:]...)
			close(sub)
			break
		}
	}
}

// FromChan 监听管道消息并直接广播
// 参数:ch - 输入消息管道
// 返回:无
// 说明:持续监听传入管道,将获取到的消息广播给所有订阅者
func (this *LogBuffer) FromChan(msg LogEntry) {
	this.Push(msg.Content, msg.Source)
}

// Drop 清理函数,清理资源并关闭所有订阅管道
// 参数:无
// 返回:无
// 说明:关闭所有订阅管道并清空订阅列表
func (this *LogBuffer) Drop() {
	this.mu.Lock()
	defer this.mu.Unlock()
	for _, sub := range this.subs {
		close(sub)
	}
	this.subs = []chan *LogEntry{}
}

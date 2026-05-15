package utils

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestLogBuffer(t *testing.T) {
	lb := &LogBuffer{}

	ch1 := lb.Subscribe(10)
	ch2 := lb.Subscribe(10)

	lb.Push("test message", "source1")

	select {
	case msg := <-ch1:
		if msg.Content != "test message" || msg.Source != "source1" {
			t.Errorf("ch1 received unexpected msg: %+v", msg)
		}
	case <-time.After(time.Second):
		t.Error("ch1 did not receive message")
	}

	select {
	case msg := <-ch2:
		if msg.Content != "test message" || msg.Source != "source1" {
			t.Errorf("ch2 received unexpected msg: %+v", msg)
		}
	case <-time.After(time.Second):
		t.Error("ch2 did not receive message")
	}

	lb.Unsubscribe(ch1)

	lb.Push("test message 2", "source2")

	select {
	case msg, ok := <-ch1:
		if ok {
			t.Errorf("ch1 should be closed, got msg: %+v", msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("ch1 was not closed immediately")
	}

	select {
	case msg := <-ch2:
		if msg.Content != "test message 2" || msg.Source != "source2" {
			t.Errorf("ch2 received unexpected msg: %+v", msg)
		}
	case <-time.After(time.Second):
		t.Error("ch2 did not receive message 2")
	}

	lb.FromChan(LogEntry{Content: "from chan", Source: "source3"})

	select {
	case msg := <-ch2:
		if msg.Content != "from chan" || msg.Source != "source3" {
			t.Errorf("ch2 received unexpected msg: %+v", msg)
		}
	case <-time.After(time.Second):
		t.Error("ch2 did not receive from chan message")
	}

	lb.Drop()

	select {
	case _, ok := <-ch2:
		if ok {
			t.Error("ch2 should be closed after Drop")
		}
	case <-time.After(time.Second):
		t.Error("ch2 was not closed after Drop")
	}
}

// TestLogBuffer_Stress 进行压力和性能测试
// 说明: 启动多个生产者和消费者,进行数十轮随机数据广播测试,统计处理性能
func TestLogBuffer_Stress(t *testing.T) {
	// 设置日志级别，减少测试时的终端输出干扰
	logrus.SetLevel(logrus.ErrorLevel)

	buffer := &LogBuffer{}
	rounds := 50         // 测试轮数
	producers := 10      // 生产者数量
	consumers := 5       // 消费者数量
	msgsPerRound := 1000 // 每轮每生产者发送的消息数

	t.Logf("开始性能测试: %d 轮, %d 生产者, %d 消费者, 每轮总计 %d 条消息\n",
		rounds, producers, consumers, producers*msgsPerRound)

	// 订阅测试
	var wg sync.WaitGroup
	consumerChans := make([]<-chan *LogEntry, consumers)
	for i := 0; i < consumers; i++ {
		consumerChans[i] = buffer.Subscribe(1000)
		wg.Add(1)
		go func(id int, ch <-chan *LogEntry) {
			defer wg.Done()
			count := 0
			expected := rounds * producers * msgsPerRound
			for range ch {
				count++
				if count >= expected {
					// 理论上由于 Push 是非阻塞的，且管道有容量限制，实际收到的可能少于预期
					// 但在这里我们主要关注广播过程是否稳定
				}
			}
		}(i, consumerChans[i])
	}

	startTime := time.Now()

	// 循环进行数十轮测试
	for r := 1; r <= rounds; r++ {
		roundStart := time.Now()
		var roundWg sync.WaitGroup

		for p := 0; p < producers; p++ {
			roundWg.Add(1)
			go func(pid int) {
				defer roundWg.Done()
				source := fmt.Sprintf("Producer-%d", pid)
				for i := 0; i < msgsPerRound; i++ {
					// 生成随机内容进行测试
					content := fmt.Sprintf("Round %d: Random Data %d", r, rand.Intn(1000000))
					buffer.Push(content, source)
				}
			}(p)
		}

		roundWg.Wait()
		t.Logf("第 %d 轮测试完成, 耗时: %v\n", r, time.Since(roundStart))
	}

	totalTime := time.Since(startTime)
	totalMsgs := rounds * producers * msgsPerRound

	// 清理资源
	buffer.Drop()
	wg.Wait()

	t.Logf("-------------------------------------------")
	t.Logf("测试结束!")
	t.Logf("总消息数: %d", totalMsgs)
	t.Logf("总耗时: %v", totalTime)
	t.Logf("平均吞吐量: %.2f msgs/sec", float64(totalMsgs)/totalTime.Seconds())
	t.Logf("-------------------------------------------")
}

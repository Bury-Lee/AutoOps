package utils

import (
	"fmt"
	"regexp"
	"time"

	"github.com/sirupsen/logrus"
)

// Result 结果结构体，用于传递信息
type Result struct {
	Match bool        // 是否匹配
	Data  interface{} // 其他数据
	Err   error       // 错误信息
}

// CheckPattern 检查文本是否匹配给定的正则表达式规则
// pattern: 自定义的正则表达式规则字符串
// text: 待检测的文本
// 返回值: bool - 是否匹配, error - 错误信息
func CheckPattern(pattern, text string) (bool, error) {
	// 编译正则表达式
	re, err := regexp.Compile(pattern)
	if err != nil {
		logrus.Errorf("正则表达式格式错误: %v", err)
		return false, fmt.Errorf("正则表达式格式错误: %v", err)
	}

	// 检查是否匹配
	return re.MatchString(text), nil
}

// CheckPatternWithTimeout 检查文本是否匹配给定的正则表达式规则（带超时控制）
// 参数:pattern - 自定义的正则表达式规则字符串, text - 待检测的文本, timeout - 超时时间
// 返回:bool - 是否匹配, error - 错误信息
// 说明:在独立的goroutine中执行匹配,如果超时则返回错误
func CheckPatternWithTimeout(pattern, text string, timeout time.Duration) (bool, error) {
	// 编译正则表达式
	re, err := regexp.Compile(pattern)
	if err != nil {
		logrus.Errorf("正则表达式格式错误: %v", err)
		return false, fmt.Errorf("正则表达式格式错误: %v", err)
	}

	done := make(chan bool, 1)
	go func() {
		done <- re.MatchString(text)
	}()

	select {
	case match := <-done:
		return match, nil
	case <-time.After(timeout):
		return false, fmt.Errorf("正则匹配超时: %s", pattern)
	}
}

// NewMatcher 创建一个新的正则表达式匹配器
// pattern: 自定义的正则表达式规则字符串
// 返回值: *regexp.Regexp - 编译后的正则表达式对象, error - 编译错误信息
func NewMatcher(pattern string) (*regexp.Regexp, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return re, nil

}

// Match 检查文本是否匹配给定的正则表达式规则
// pattern: 自定义的正则表达式规则字符串
// target: 待检测的文本
// 返回值: bool - 是否匹配
func Match(pattern, target string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		// 无效的正则表达式：返回 false
		return false
	}
	return re.MatchString(target)
}

package utils

// ContainsString 检查字符串数组是否包含指定字符串
// 参数: str - 待检查的字符串, strings - 字符串数组
// 返回: bool - 是否包含
func ContainsString(strings []string, str string) bool {
	for _, s := range strings {
		if s == str {
			return true
		}
	}
	return false
}

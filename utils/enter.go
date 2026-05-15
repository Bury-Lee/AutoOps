package utils

import (
	"bytes"
	"io"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// ConvertGBKToUTF8 将 GBK 编码的字节数组转换为 UTF-8 编码的字符串
// 参数:gbkBytes - GBK 编码的字节数组
// 返回:string - 转换后的 UTF-8 字符串, error - 错误信息
// 说明:使用 simplifiedchinese 包进行解码转换
func ConvertGBKToUTF8(gbkBytes []byte) (string, error) {
	reader := transform.NewReader(bytes.NewReader(gbkBytes), simplifiedchinese.GBK.NewDecoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(d), nil
}

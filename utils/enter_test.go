package utils

import (
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestConvertGBKToUTF8(t *testing.T) {
	// Create some GBK encoded bytes
	utf8Str := "测试转换"
	encoder := simplifiedchinese.GBK.NewEncoder()
	gbkBytes, err := encoder.Bytes([]byte(utf8Str))
	if err != nil {
		t.Fatalf("Failed to encode to GBK: %v", err)
	}

	result, err := ConvertGBKToUTF8(gbkBytes)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result != utf8Str {
		t.Errorf("Expected %q, got %q", utf8Str, result)
	}
}

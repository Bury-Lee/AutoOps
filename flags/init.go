package flags

import (
	_ "embed"
	"fmt"
	"os"
)

//go:embed example.yaml
var defaultSettingsYAML []byte

// InitYaml 创建默认配置文件
// 参数:无
// 返回:无
// 说明:仅在 settings.yaml 不存在时写入,避免覆盖用户已有配置
func InitYaml() {
	const fileName = "settings.yaml"

	_, err := os.Stat(fileName)
	if err == nil {
		fmt.Printf("%s 已存在，跳过初始化\n", fileName)
		return
	}
	if !os.IsNotExist(err) {
		fmt.Printf("检查 %s 失败: %v\n", fileName, err)
		return
	}
	if err := os.WriteFile(fileName, defaultSettingsYAML, 0644); err != nil {
		fmt.Printf("写入 %s 失败: %v\n", fileName, err)
		return
	}
	fmt.Printf("已生成默认配置文件: %s\n", fileName)
}

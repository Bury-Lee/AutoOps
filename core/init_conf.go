package core

import (
	"AutoOps/conf"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

const fileName = "settings.yaml"

func ReadConf() *conf.Config {
	byteData, err := os.ReadFile(fileName) //读取文件
	if err != nil {
		panic(err) //以后试试改为"无法读取配置文件"+err.Error()
	}
	var c = new(conf.Config)
	err = yaml.Unmarshal(byteData, c) //结构绑定
	if err != nil {
		panic(fmt.Sprintf("yaml文件格式错误 %s", err))
	}
	//设置版本号
	return c
}

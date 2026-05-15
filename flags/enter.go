package flags

import (
	"AutoOps/models"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

type Option struct {
	RunDBMigration bool
	ShowVersion    bool
	Action         string
	ConfigPath     string
	RawCommand     string
	TerminalMode   bool
	ChatMode       bool
}

const (
	actionSkill = "skill"
	actionTest  = "test"
	actionInit  = "init"
	actionRun   = "r"
)

type runAction string

const (
	runActionNone     runAction = ""
	runActionDB       runAction = "db"
	runActionVersion  runAction = "version"
	runActionCommand  runAction = "command"
	runActionType     runAction = "type"
	runActionTerminal runAction = "tmod"
	runActionChat     runAction = "cmod"
)

var FlagOptions = new(Option)

func Parse() {
	flag.BoolVar(&FlagOptions.RunDBMigration, "db", false, "数据库迁移")
	flag.BoolVar(&FlagOptions.ShowVersion, "v", false, "版本")
	flag.StringVar(&FlagOptions.Action, "t", "", "操作类型")
	flag.StringVar(&FlagOptions.ConfigPath, "s", "", "配置路径")
	flag.StringVar(&FlagOptions.RawCommand, "c", "", "原始命令内容")
	flag.BoolVar(&FlagOptions.TerminalMode, "tmod", false, "是否启用终端模式(体验上就像是普通终端)")
	flag.BoolVar(&FlagOptions.ChatMode, "cmod", false, "是否启用终端聊天模式(体验上就像是普通终端)")
	flag.Parse()
	FlagOptions.normalize()
}

func (o *Option) normalize() {
	o.Action = strings.ToLower(strings.TrimSpace(o.Action))
	o.ConfigPath = strings.TrimSpace(o.ConfigPath)
	o.RawCommand = strings.TrimSpace(o.RawCommand)
}

func (o *Option) RequiresBootstrap() bool {
	return !o.IsInitOnly() && !o.ShowVersion
}

func (o *Option) IsInitOnly() bool {
	return o.Action == actionInit
}

func (o *Option) resolveRunAction() (runAction, error) {
	var actions []runAction

	if o.RunDBMigration {
		actions = append(actions, runActionDB)
	}
	if o.ShowVersion {
		actions = append(actions, runActionVersion)
	}
	if o.RawCommand != "" {
		actions = append(actions, runActionCommand)
	}
	if o.Action != "" {
		switch o.Action {
		case actionSkill, actionTest, actionInit, actionRun:
			actions = append(actions, runActionType)
		default:
			return runActionNone, fmt.Errorf("不支持的操作类型: %s", o.Action)
		}
	}
	if o.TerminalMode {
		actions = append(actions, runActionTerminal)
	}
	if o.ChatMode {
		actions = append(actions, runActionChat)
	}

	if len(actions) == 0 {
		return runActionNone, nil
	}
	if len(actions) > 1 {
		return runActionNone, fmt.Errorf("检测到多个主命令,请一次只使用一种运行方式")
	}
	return actions[0], nil
}

func (o *Option) runCommand() {
	var option *models.TerminalOption
	if o.ConfigPath != "" {
		option = ParseJson(o.ConfigPath)
	}
	StartServer(o.RawCommand, option)
}

func (o *Option) runTypeAction() {
	switch o.Action {
	case actionSkill:
		NewSkill()
	case actionTest:
		ConnectTest()
	case actionInit:
		InitYaml()
	case actionRun:
		if o.ConfigPath == "" {
			fmt.Println("未指定Json文件!")
			os.Exit(1)
		}
		RunByJson(o.ConfigPath)
	}

	os.Exit(0)
}

func Run() {
	action, err := FlagOptions.resolveRunAction()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	switch action {
	case runActionNone:
		return
	case runActionDB:
		FlagDB()
		os.Exit(0)
	case runActionVersion:
		fmt.Println("当前版本为: 1")
		os.Exit(0)
	case runActionCommand:
		FlagOptions.runCommand()
	case runActionType:
		FlagOptions.runTypeAction()
	case runActionTerminal:
		logrus.Debugf("进入终端模式(目前为实验性功能)")
		Tmod()
	case runActionChat:
		logrus.Debugf("进入终端助手模式(目前为实验性功能)") //debug模式下打印日志
		Cmod()
	}
}

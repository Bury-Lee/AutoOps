package flags

import "testing"

func TestOptionNormalize(t *testing.T) {
	option := &Option{
		Action:     "  INIT  ",
		ConfigPath: "  demo.json  ",
		RawCommand: "  echo hello  ",
	}

	option.normalize()

	if option.Action != actionInit {
		t.Fatalf("Action 归一化失败: %q", option.Action)
	}
	if option.ConfigPath != "demo.json" {
		t.Fatalf("ConfigPath 归一化失败: %q", option.ConfigPath)
	}
	if option.RawCommand != "echo hello" {
		t.Fatalf("RawCommand 归一化失败: %q", option.RawCommand)
	}
}

func TestOptionResolveRunAction(t *testing.T) {
	tests := []struct {
		name    string
		option  Option
		want    runAction
		wantErr bool
	}{
		{
			name:   "无动作",
			option: Option{},
			want:   runActionNone,
		},
		{
			name: "原始命令",
			option: Option{
				RawCommand: "echo hello",
			},
			want: runActionCommand,
		},
		{
			name: "类型命令",
			option: Option{
				Action: actionRun,
			},
			want: runActionType,
		},
		{
			name: "未知类型命令",
			option: Option{
				Action: "unknown",
			},
			wantErr: true,
		},
		{
			name: "冲突动作",
			option: Option{
				ShowVersion:  true,
				TerminalMode: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.option.resolveRunAction()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("期望返回错误, 实际无错误")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveRunAction 返回错误: %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveRunAction = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOptionRequiresBootstrap(t *testing.T) {
	tests := []struct {
		name   string
		option Option
		want   bool
	}{
		{
			name: "初始化跳过启动",
			option: Option{
				Action: actionInit,
			},
			want: false,
		},
		{
			name: "版本跳过启动",
			option: Option{
				ShowVersion: true,
			},
			want: false,
		},
		{
			name: "普通命令需要启动",
			option: Option{
				Action: actionRun,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.option.RequiresBootstrap(); got != tt.want {
				t.Fatalf("RequiresBootstrap = %v, want %v", got, tt.want)
			}
		})
	}
}

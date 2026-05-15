package utils

import (
	"os/exec"
	"runtime"
)

// RunShellCommand 在对应系统的 shell 中执行 command 字符串，返回合并的标准输出+标准错误，及执行错误。
func RunShellCommand(command string) (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Windows 使用 cmd.exe，/C 表示执行字符串指定的命令然后终止
		cmd = exec.Command("cmd", "/C", command)
	default:
		// Linux / macOS / BSD 等类 Unix 系统使用 /bin/sh，-c 表示执行后面的命令字符串
		cmd = exec.Command("/bin/sh", "-c", command)
	}

	// 合并 stdout 和 stderr，按原始顺序返回
	output, err := cmd.CombinedOutput()
	result := string(output)
	if runtime.GOOS == "windows" {
		result, _ = ConvertGBKToUTF8(output) //TODO:处理错误
	}
	// 如果不需要去掉尾部换行，可以直接返回 string(output)，但通常外面会自行 trim
	return result, err
}

package dockerlib

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"golang.org/x/net/context"
)

func Test(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "d:\\vmshare\\blockchain\\code-v2.0\\bcchain\\bin\\test.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Stdout = os.Stdout
	cmd.Start()

	time.Sleep(5 * time.Second)
	fmt.Println("退出程序中...", cmd.Process.Pid)
	cancel()

	cmd.Wait()

}

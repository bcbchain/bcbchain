package dockerlib

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/tendermint/tmlibs/log"
	xsha3 "golang.org/x/crypto/sha3"
	"golang.org/x/net/context"
)

type container struct {
	cancel context.CancelFunc
}

// DockerLib 是我們自定義的 Docker API 的 Wrapper
type DockerLib struct {
	logger     log.Logger
	containers map[string]container
	prefix     string
}

const dockerHubIP = "127.0.0.1"

// GetMyIntranetIP 獲得本機局網網卡 IP，如有多個，取第一個
func (l *DockerLib) GetMyIntranetIP() string {
	return dockerHubIP
}

// GetDockerHubIP 獲得本機 Docker 的網卡地址，如果有服務需要 Docker 容器內部訪問，就可以訪問這個地址
func (l *DockerLib) GetDockerHubIP() string {

	return dockerHubIP
}

// Run 運行 Docker 容器，執行某個功能
func (l *DockerLib) Run(dockerImageName, containerName string, params *DockerRunParams) (bool, error) {
	if l.containers == nil {
		l.containers = make(map[string]container)
	}
	l.logger.Debug("DockerLib Run", "image", dockerImageName, "containerName", containerName, "params", params)

	container, found := l.containers[containerName]
	cmdName := params.Cmd[0]
	cmdParams := params.Cmd[1:]

	if found {
		container.cancel()
	}
	container.cancel = nil

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, cmdName, cmdParams...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	cmd.Dir = params.WorkDir
	cmd.Env = params.Env

	var out bytes.Buffer
	cmd.Stdout = &out

	if params.NeedWait {
		var out bytes.Buffer
		cmd.Stderr = &out
		err := cmd.Run()
		params.FirstOutput = out.String()
		if err != nil {
			return false, err
		}
		return true, nil
	}
	go func() {
		err := cmd.Start()
		if err != nil {
			params.FirstOutput = err.Error()
		}

		container.cancel = cancel
		err = cmd.Wait()
		if err != nil {
			params.FirstOutput = err.Error()
		}
	}()

	//等cancel被赋值，说明程序已经启动
	beginTime := time.Now()
	timeOut := 10.00 // wait 10s smcrunsvc start
	for {
		time.Sleep(time.Duration(time.Millisecond * 200))
		if container.cancel != nil {
			break
		}
		n := time.Now()
		sub := n.Sub(beginTime).Seconds()
		if sub > timeOut {
			return false, errors.New("can not run file " + containerName)
		}
		continue
	}
	l.containers[containerName] = container
	if params.FirstOutput == "" {
		params.FirstOutput = out.String()
	}

	return true, nil
}

// Kill 殺死一個 Docker 容器，並且清理現場
func (l *DockerLib) Kill(containerName string) bool {
	container, ok := l.containers[containerName]
	if ok {
		container.cancel()
	}
	return true
}

// Status 查詢一個容器的狀態
func (l *DockerLib) Status(containerName string) bool {
	_, ok := l.containers[containerName]
	if ok {
		return true
	}
	return false
}

// Reset 殺掉所有自己啓動的容器(以特定字冠命名的)
func (l *DockerLib) Reset(prefix string) bool {
	for k, v := range l.containers {
		if strings.HasPrefix(k, prefix) {
			v.cancel()
		}
	}
	return true
}

// GetDockerIP 通過容器的名字獲取容器 IP 地址
func (l *DockerLib) GetDockerContainerIP(containerName string) string {
	return dockerHubIP
}

func Sum256(datas ...[]byte) []byte {

	hasher := xsha3.New256()
	for _, data := range datas {
		hasher.Write(data)
	}
	return hasher.Sum(nil)
}

// SetPrefix set container name's prefix
func (l *DockerLib) SetPrefix(p string) {
	l.prefix = strings.ReplaceAll(strings.ReplaceAll(p, "[", ""), "]", "")
}

func (l *DockerLib) Exec(config ExecConfig, startConfig ExecStartCheck, container string) error {

	return nil
}

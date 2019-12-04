package dockerlib

import (
	"sync"

	"github.com/tendermint/tmlibs/log"
)

var (
	myLib        *DockerLib
	instanceOnce sync.Once
	initOnce     sync.Once
)

// Mounts DockerRun所需目錄映射
type Mounts struct {
	Source      string
	Destination string
	ReadOnly    bool
}

// HostPort DockerRun 需要映射到的本機 IP 和 端口
type HostPort struct {
	Port string
	Host string
}

// DockerRunParams 運行 Docker 容器需要的參數，避免調用者還依賴 Docker API
type DockerRunParams struct {
	Env         []string
	Cmd         []string
	WorkDir     string
	Mounts      []Mounts
	PortMap     map[string]HostPort
	NeedPull    bool   // 因为有些主机没有因特网，你懂的原因，可能会导致一些未知错误，我们就手工load，然后不pull了吧
	FirstOutput string // 回寫
	NeedOut     bool   // 需要拿到控制臺輸出（只拿開始的內容，不能一直等，有些進程會一直輸出）
	NeedRemove  bool   // 需要手工清理掉屍體
	AutoRemove  bool   // 給 daemon 設置一下，如果它們掛了，就自己打掃戰場，不留垃圾
	NeedWait    bool   // 等它執行結束（需要注意 daemon 不會結束）
}

// GetDockerLib 初始化得到 DockerLib 對象指針
func GetDockerLib() *DockerLib {
	instanceOnce.Do(func() {
		myLib = &DockerLib{}
	})
	return myLib
}

// Init 傳入日志對象，不能不傳
func (l *DockerLib) Init(log log.Logger) {
	initOnce.Do(func() {
		l.logger = log
	})
}

// ExecConfig is a small subset of the Config struct that holds the configuration
// for the exec feature of docker.
type ExecConfig struct {
	User         string   // User that will run the command
	Privileged   bool     // Is the container in privileged mode
	Tty          bool     // Attach standard streams to a tty.
	AttachStdin  bool     // Attach the standard input, makes possible user interaction
	AttachStderr bool     // Attach the standard error
	AttachStdout bool     // Attach the standard output
	Detach       bool     // Execute in detach mode
	DetachKeys   string   // Escape keys for detach
	Env          []string // Environment variables
	Cmd          []string // Execution commands and args
}

// ExecStartCheck is a temp struct used by execStart
// Config fields is part of ExecConfig in runconfig package
type ExecStartCheck struct {
	// ExecStart will first check if it's detached
	Detach bool
	// Check if there's a tty
	Tty bool
}

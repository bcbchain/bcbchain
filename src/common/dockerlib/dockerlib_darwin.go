package dockerlib

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/docker/docker/api/types/container"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/tendermint/tmlibs/log"
	"golang.org/x/net/context"
)

// DockerLib 是我們自定義的 Docker API 的 Wrapper
type DockerLib struct {
	logger log.Logger
	prefix string
}

// GetMyIntranetIP 獲得本機局網網卡 IP，如有多個，取第一個
func (l *DockerLib) GetMyIntranetIP() string {
	addrArray, err := net.InterfaceAddrs()
	if err != nil {
		l.logger.Warn("DockerLib GetMyIntranetIP cause ERROR", "err", err)
		return ""
	}
	for _, addr := range addrArray {
		if ip, ok := addr.(*net.IPNet); ok && !ip.IP.IsLoopback() {
			if ip.IP.To4() != nil {
				return ip.IP.String()
			}
		}
	}
	return ""
}

// GetDockerHubIP 獲得本機 Docker 的網卡地址，如果有服務需要 Docker 容器內部訪問，就可以訪問這個地址
func (l *DockerLib) GetDockerHubIP() string {
	params := DockerRunParams{
		Cmd:        []string{"ip", "r"},
		NeedOut:    true,
		NeedWait:   true,
		NeedRemove: true,
	}
	ok, _ := l.Run("alpine:latest", "", &params)
	if !ok {
		return ""
	}
	listStr := strings.Split(params.FirstOutput, " ")
	if len(listStr) < 5 {
		l.logger.Warn("GetDockerHubIP got strange output:", "stdout", params.FirstOutput)
	}
	return listStr[2] // this is the result
}

// Run 運行 Docker 容器，執行某個功能。由於無法直接獲知Docker內Service的啓動狀態，請參考test文件中的處理辦法，或者在Service啓動的時候主動回調
func (l *DockerLib) Run(dockerImageName, containerName string, params *DockerRunParams) (bool, error) {
	containerName = l.generalContainerName(containerName)
	l.logger.Debug("DockerLib Run", "image", dockerImageName, "containerName", containerName, "params", params)
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		l.logger.Warn("DockerLib Run NewEnvClient Error:", "err", err)
		return false, errors.New("DockerLib Run NewEnvClient Error:" + err.Error())
	}
	defer cli.Close()

	if params.NeedPull {
		// pull image，三次机会，还不成功可以手动获取
		imageOK := false
		for i := 0; i < 3; i++ {
			imageOK, err = l.ensureImage(ctx, cli, dockerImageName)
			if imageOK {
				break
			} else {
				continue
			}
		}
		if !imageOK {
			return false, err
		}
	}
	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image:        dockerImageName,
			Cmd:          params.Cmd,
			Tty:          params.NeedOut,
			Env:          params.Env,
			WorkingDir:   params.WorkDir,
			ExposedPorts: assemblePortSet(params),
		}, &container.HostConfig{
			Mounts:       assembleMounts(params),
			PortBindings: assemblePortMap(params),
			AutoRemove:   params.AutoRemove,
		}, nil, containerName)
	if err != nil {
		l.logger.Warn("DockerLib Run ContainerCreate Error:", "err", err)
		return false, errors.New("DockerLib Run ContainerCreate Error:" + err.Error())
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		l.logger.Warn("DockerLib Run ContainerStart Error:", "err", err)
		return false, errors.New("DockerLib Run ContainerStart Error:" + err.Error())
	}

	if params.NeedWait {
		if _, err = cli.ContainerWait(ctx, resp.ID); err != nil {
			l.logger.Warn("DockerLib Run ContainerWait Error:", "err", err)
			return false, errors.New("DockerLib Run ContainerWait Error:" + err.Error())
		}
	}

	if !l.feedBack(ctx, cli, resp.ID, params) {
		return false, errors.New("DockerLib Run feedBack Error")
	}

	if params.NeedRemove {
		err = cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
		if err != nil {
			l.logger.Warn("DockerLib Run remove cause ERROR:", "err", err, "Please remove manually", containerName)
		}
	}

	return true, nil
}

func (l *DockerLib) feedBack(ctx context.Context, cli *client.Client, containerID string, params *DockerRunParams) bool {
	if params.NeedOut {
		out, err := cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{ShowStdout: true})
		if err != nil {
			l.logger.Warn("DockerLib Run ContainerLogs cause ERROR:", "err", err)
			return false
		}

		byt := make([]byte, 3000)
		n, err := out.Read(byt)
		if err != nil && err != io.EOF {
			l.logger.Warn("DockerLib Run Read From ContainerLogs cause ERROR:", "err", err)
		}
		if n < 0 {
			l.logger.Warn("DockerLib Run Read From ContainerLogs cause ERROR: output is zero length")
		} else if n == 0 {
			params.FirstOutput = ""
		} else {
			params.FirstOutput = string(byt[:n])
		}
	}

	return true
}

func assemblePortSet(params *DockerRunParams) nat.PortSet {
	portSet := make(map[nat.Port]struct{}, 0)
	for k := range params.PortMap {
		p := nat.Port(k)
		portSet[p] = struct{}{}
	}
	return portSet
}

func assemblePortMap(params *DockerRunParams) nat.PortMap {
	portMap := make(map[nat.Port][]nat.PortBinding, 0)
	for k, v := range params.PortMap {
		p := nat.Port(k)
		bindings := make([]nat.PortBinding, 1)
		bindings[0] = nat.PortBinding{
			HostIP:   v.Host,
			HostPort: v.Port,
		}
		portMap[p] = bindings
	}
	return portMap
}

func assembleMounts(params *DockerRunParams) []mount.Mount {
	mounts := make([]mount.Mount, 0)
	for _, m := range params.Mounts {
		mt := mount.Mount{Type: mount.TypeBind,
			Source:   m.Source,
			Target:   m.Destination,
			ReadOnly: m.ReadOnly,
		}
		mounts = append(mounts, mt)
	}
	return mounts
}

func (l *DockerLib) ensureImage(ctx context.Context, cli *client.Client, imageName string) (bool, error) {
	images, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		l.logger.Warn("DockerLib Run ImageList Error:", "err", err)
		return false, errors.New("DockerLib Run ImageList Error:" + err.Error())
	}

	if notExists(images, imageName) {
		p, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
		defer func() {
			if p != nil {
				if e := p.Close(); e != nil {
					l.logger.Warn(e.Error())
				}
			}
		}()
		if err != nil {
			l.logger.Warn("DockerLib Run ImagePull Error:", "err", err)
			return false, errors.New("DockerLib Run ImagePull Error:" + err.Error())
		}

		byt := make([]byte, 500)
		for {
			n, err := p.Read(byt)
			if err != nil && err != io.EOF {
				l.logger.Info("DockerLib ImagePull can't Read output", "err", err)
				break
			} else {
				if strings.Contains(string(byt), "Downloaded") || strings.Contains(string(byt), "up to date") {
					l.logger.Debug("DockerLib ImagePull:", "result", string(byt[:n]))
					break
				}
			}
		}
	}
	return true, nil
}

func notExists(images []types.ImageSummary, imageName string) bool {
	exists := false
	for _, image := range images {
		// fmt.Println(image.RepoTags)
		for _, tag := range image.RepoTags {
			if tag == imageName {
				exists = true
				break
			}
		}
		if exists {
			break
		}
	}
	return !exists
}

// Kill 殺死一個 Docker 容器，並且清理現場
func (l *DockerLib) Kill(containerName string) bool {
	containerName = l.generalContainerName(containerName)
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		l.logger.Warn("DockerLib Kill NewEnvClient cause ERROR:", "err", err)
		return false
	}
	defer cli.Close()

	containerID := l.getContainerIDByName(ctx, cli, containerName)
	if containerID == "" {
		l.logger.Debug("No such containerName:", "name", containerName)
		return true // 木有的情況也返回 true 吧，就省了 remove 了
	}

	return l.killByID(ctx, cli, containerID)
}

func (l *DockerLib) killByID(ctx context.Context, cli *client.Client, containerID string) bool {
	err := cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		l.logger.Warn("DockerLib Kill remove cause ERROR:", "err", err, "Please remove manually", containerID)
		return false
	}
	return true
}

func (l *DockerLib) getContainerIDByName(ctx context.Context, cli *client.Client, containerName string) string {
	containerID := ""
	list, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		l.logger.Warn("DockerLib getContainerIDByName cause ERROR:", "err", err)
		return ""
	}
	for _, con := range list {
		for _, name := range con.Names {
			if name[1:] == containerName {
				containerID = con.ID
				break
			}
		}
		if containerID != "" {
			break
		}
	}
	return containerID
}

// Status 查詢一個容器的狀態
func (l *DockerLib) Status(containerName string) bool {
	containerName = l.generalContainerName(containerName)
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		l.logger.Warn("DockerLib Status NewEnvClient cause ERROR:", "err", err)
		return false
	}
	defer cli.Close()

	containerID := l.getContainerIDByName(ctx, cli, containerName)
	if containerID == "" {
		l.logger.Warn("No such containerName:", "name", containerName)
		return false
	}
	stat, err := cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		l.logger.Warn("DockerLib Status ContainerStats cause ERROR:", "err", err)
		return false
	}
	if stat.Body == nil {
		return false
	}
	err = stat.Body.Close()
	if err != nil {
		l.logger.Debug("DockerLib Status Close ContainerStats response:", "err", err)
	}

	return true
}

// Reset 殺掉所有自己啓動的容器(以特定字冠命名的)
func (l *DockerLib) Reset(prefix string) bool {
	prefix = formatContainerName(prefix)
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		l.logger.Warn("DockerLib Reset NewEnvClient cause ERROR:", "err", err)
		return false
	}
	defer cli.Close()

	containerList, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		l.logger.Warn("DockerLib Reset ContainerList cause ERROR:", "err", err)
		return false
	}

	var idList []string
	for _, c := range containerList {
		for _, name := range c.Names {
			if strings.HasPrefix(name[1:], prefix) {
				idList = append(idList, c.ID)
				break
			}
		}
	}
	for _, id := range idList {
		if ok := l.killByID(ctx, cli, id); !ok {
			return false
		}
	}
	return true
}

// GetDockerContainerIP 通過容器的名字獲取容器 IP 地址
func (l *DockerLib) GetDockerContainerIP(containerName string) string {
	return "127.0.0.1"
	// 以下代码可以拿到容器的IP，但在苹果系统上无法访问到，必须通过本地地址才能通过
	// 网络访问容器内部
	/*
		ctx := context.Background()
		cli, err := client.NewEnvClient()
		if err != nil {
			l.logger.Warn("DockerLib Reset NewEnvClient cause ERROR:", "err", err)
		}

		containerID := l.getContainerIDByName(ctx, cli, containerName)
		if containerID == "" {
			l.logger.Warn("DockerLib GetDockerContainerIP no such container ERROR:", "name", containerName)
			return ""
		}
		resp, err := cli.ContainerInspect(ctx, containerID)
		if err != nil {
			l.logger.Warn("DockerLib GetDockerContainerIP ContainerInspect cause ERROR:", "name", containerName, "err", err)
			return ""
		}

		return resp.NetworkSettings.IPAddress
	*/
}

func mapIP(s string) string {
	return strings.Map(func(r rune) rune {
		if r != 46 && (r < 48 || r > 57) { // 只留 [.0-9]
			return -1
		}
		return r
	}, s)
}

// SetPrefix set container name's prefix
func (l *DockerLib) SetPrefix(p string) {
	l.prefix = strings.ReplaceAll(strings.ReplaceAll(p, "[", ""), "]", "")
}

func (l *DockerLib) generalContainerName(name string) string {
	if name != "" {
		return l.prefix + name
	} else {
		n := l.prefix + generateID(cryptorand.Reader)
		return n
	}
}
func generateID(r io.Reader) string {
	b := make([]byte, 32)
	for {
		if _, err := io.ReadFull(r, b); err != nil {
			panic(err) // This shouldn't happen
		}
		id := hex.EncodeToString(b)
		// if we try to parse the truncated for as an int and we don't have
		// an error then the value is all numeric and causes issues when
		// used as a hostname. ref #3869
		if _, err := strconv.ParseInt(TruncateID(id), 10, 64); err == nil {
			continue
		}
		return id
	}
}

func TruncateID(id string) string {
	if i := strings.IndexRune(id, ':'); i >= 0 {
		id = id[i+1:]
	}
	if len(id) > 12 {
		id = id[:12]
	}
	return id
}

func (l *DockerLib) Exec(config ExecConfig, startConfig ExecStartCheck, container string) error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	execConfig := types.ExecConfig{
		User:         config.User,
		Privileged:   config.Privileged,
		Tty:          config.Tty,
		AttachStdin:  config.AttachStdin,
		AttachStderr: config.AttachStderr,
		AttachStdout: config.AttachStdout,
		Detach:       config.Detach,
		DetachKeys:   config.DetachKeys,
		Env:          config.Env,
		Cmd:          config.Cmd,
	}

	container = l.prefix + container
	res, err := cli.ContainerExecCreate(ctx, container, execConfig)
	if err != nil {
		return err
	}
	execID := res.ID
	err = cli.ContainerExecStart(ctx, execID, types.ExecStartCheck{
		Detach: startConfig.Detach,
		Tty:    startConfig.Tty,
	})
	if err != nil {
		return err
	}

	return nil
}

func formatContainerName(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, "[", ""), "]", "")
}

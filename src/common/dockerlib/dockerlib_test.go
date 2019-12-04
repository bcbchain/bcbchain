package dockerlib

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/tendermint/tmlibs/log"
)

func TestDockerLib_GetDockerHubIP(t *testing.T) {
	logger := log.NewOldTMLogger(os.Stdout)
	lib := GetDockerLib()
	lib.Init(logger)
	ip := lib.GetDockerHubIP()
	println(ip)
}

func TestDockerLib_Run(t *testing.T) {
	logger := log.NewOldTMLogger(os.Stdout)
	lib := GetDockerLib()
	lib.Init(logger)
	lib.Kill("my8080")
	params := DockerRunParams{
		PortMap: map[string]HostPort{
			"8000": {
				Port: "8080",
				Host: "0.0.0.0",
			},
		},
		Mounts:     []Mounts{{"/tmp", "/a", true}},
		WorkDir:    "/",
		AutoRemove: true,
		Cmd:        []string{"sh", "-c", "python3 -m http.server"},
	}
	ret, err := lib.Run("python:3-alpine", "my8080", &params)
	assert.Equal(t, ret, true)
	assert.Equal(t, err, nil)

	timer := time.NewTimer(10 * time.Second)
	checkTimer := time.NewTicker(20 * time.Millisecond)
	defer func() { checkTimer.Stop() }()
	count := 0
	for {
		select {
		case <-checkTimer.C:
			resp, err := http.Get("http://localhost:8080/")
			if err == nil && resp.StatusCode == 200 {
				goto GOTOEND
			}
			fmt.Println("err =", err, "; resp =", resp)
			count++
			continue
		case <-timer.C:
			assert.Error(t, fmt.Errorf("啓動時間過長"), "")
			goto GOTOEND
		}
	}
GOTOEND:
	fmt.Println("count=", count)
	resp, err := http.Get("http://localhost:8080/")
	assert.Equal(t, err, nil)
	body, _ := ioutil.ReadAll(resp.Body)

	assert.Equal(t, resp.StatusCode, 200)
	fmt.Println(resp.Header.Get("Content-Type"))
	fmt.Println(string(body))
}

func TestDockerLib_Run2(t *testing.T) {
	logger := log.NewOldTMLogger(os.Stdout)
	lib := GetDockerLib()
	lib.Init(logger)
	lib.Kill("my8000")
	params := DockerRunParams{
		WorkDir: "/",
		Cmd:     []string{"sh", "-c", "python3 -m http.server"},
	}
	ret, err := lib.Run("python:3-alpine", "my8000", &params)
	assert.Equal(t, ret, true)
	assert.Equal(t, err, nil)

	ip := lib.GetDockerContainerIP("my8000")
	fmt.Println("ip=", ip)
	timer := time.NewTimer(10 * time.Second)
	checkTimer := time.NewTicker(100 * time.Millisecond)
	defer func() { checkTimer.Stop() }()
	count := 0
	for {
		select {
		case <-checkTimer.C:
			resp, err := http.Get("http://" + ip + ":8000/")
			if err == nil && resp.StatusCode == 200 {
				goto GOTOEND
			}
			fmt.Println("err =", err)
			if resp != nil {
				fmt.Println("statusCode =", resp.StatusCode)
			}
			count++
			continue
		case <-timer.C:
			assert.Error(t, fmt.Errorf("啓動時間過長"), "")
			goto GOTOEND
		}
	}
GOTOEND:
	fmt.Println("count=", count)
	resp, err := http.Get("http://" + ip + ":8000/")
	assert.Equal(t, err, nil)
	body, _ := ioutil.ReadAll(resp.Body)

	assert.Equal(t, resp.StatusCode, 200)
	fmt.Println(resp.Header.Get("Content-Type"))
	fmt.Println(string(body))
}

func TestDockerLib_Run3(t *testing.T) { // 测试 build
	logger := log.NewOldTMLogger(os.Stdout)
	lib := GetDockerLib()
	lib.Init(logger)

	params := DockerRunParams{
		Cmd:     []string{"/bin/ash", "-c", "./go-install.sh"},
		Env:     []string{"GOPATH=/build:/blockchain/sdk:/blockchain/thirdparty", "CGO_ENABLED=0"},
		WorkDir: "/build/src",
		Mounts: []Mounts{
			{
				Source:      "/home/rustic/ddd/bin",
				Destination: "/build/bin",
			},
			{
				Source:      "/home/rustic/ddd/thirdparty",
				Destination: "/blockchain/thirdparty",
				ReadOnly:    true,
			},
			{
				Source:      "/home/rustic/ddd/sdk",
				Destination: "/blockchain/sdk",
				ReadOnly:    true,
			},
			{
				Source:      "/home/rustic/ddd/build",
				Destination: "/build",
			},
		},
		NeedRemove: true,
		NeedOut:    false,
		NeedWait:   true,
	}
	ok, err := lib.Run("golang:alpine", "", &params)
	assert.Equal(t, ok, true)

	byt, ef := ioutil.ReadFile("/home/rustic/ddd/build/src/ff")
	assert.Equal(t, ef, nil)
	fmt.Println("|", string(byt), "|")

	assert.Equal(t, err, nil)
}

func TestDockerLib_Run5(t *testing.T) { // 测试 Docker run 会挂起无响应？
	logger := log.NewOldTMLogger(os.Stdout)
	lib := GetDockerLib()
	lib.Init(logger)

	params := DockerRunParams{
		Cmd: []string{"/smcrunsvc", "start", "-p", "6094", "-c", "tcp://192.168.41.148:32333"},
		Mounts: []Mounts{
			{
				Source:      "/home/rustic/ddd/smcrunsvc",
				Destination: "/smcrunsvc",
			},
			{
				Source:      "/home/rustic/ddd/log",
				Destination: "/log",
				ReadOnly:    true,
			},
		},
		PortMap: map[string]HostPort{
			"6094": {
				Port: "6094",
				Host: "0.0.0.0",
			},
		},
		NeedRemove: false,
		NeedOut:    false,
		NeedWait:   false,
	}
	ok, err := lib.Run("alpine", "orgJgaGConUyK81zibntUBjQ33PKctpk1K1G", &params)
	assert.Equal(t, ok, true)
	assert.Equal(t, err, nil)
}

func TestDockerLib_GetDockerIP(t *testing.T) {
	logger := log.NewOldTMLogger(os.Stdout)
	lib := GetDockerLib()
	lib.Init(logger)
	ip := lib.GetDockerContainerIP("my8000")
	fmt.Println(ip)
	assert.Equal(t, ip, "172.17.0.3")
}

func TestDockerLib_Kill(t *testing.T) {
	logger := log.NewOldTMLogger(os.Stdout)
	lib := GetDockerLib()
	lib.Init(logger)
	result := lib.Kill("my8000")
	assert.Equal(t, result, true)
}

func TestDockerLib_GetMyIntranetIP(t *testing.T) {
	logger := log.NewOldTMLogger(os.Stdout)
	lib := GetDockerLib()
	lib.Init(logger)
	ip := lib.GetMyIntranetIP()
	assert.Equal(t, ip, "192.168.1.4")
}

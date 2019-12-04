package smcdocker

import (
	"blockchain/abciapp/common"
	"blockchain/algorithm"
	"blockchain/smcbuilder"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdkimpl/helper"
	"common/nu"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"blockchain/common/statedbhelper"
	"blockchain/types"
	"common/dockerlib"

	"github.com/tendermint/tmlibs/log"
)

//SMCDocker delcare object
type SMCDocker struct {
	logger          log.Logger
	orgNameToURL    sync.Map //维护组织ID与对应的调用URL
	callbackURL     string
	orgIdToLastTime sync.Map // key: orgID, value: the time of the last call
	RunDocker       chan RunDocker
}

type RunDocker struct {
	TransID      int64
	TxID         int64
	ContractAddr string
	OrgID        string
	c            chan RunDockerRes
}

type RunDockerRes struct {
	Url   string
	Error string
}

var (
	mgr          *SMCDocker
	instanceOnce sync.Once
	initOnce     sync.Once
	fc           func(url string)
)

//GetInstance get instance
func GetInstance() *SMCDocker {
	instanceOnce.Do(func() {
		mgr = &SMCDocker{}
	})
	return mgr
}

//Init init and starting monitor
func (sd *SMCDocker) Init(log log.Logger, callbackURL string, f func(url string)) {
	initOnce.Do(func() {
		sd.logger = log
		sd.callbackURL = callbackURL
		sd.RunDocker = make(chan RunDocker)
		go sd.runDockerSever()
		//go maintainDocker(im) //暂时不用维护
		fc = f
	})
}

//GetContractInvokeURL get contract invoke URL
func (sd *SMCDocker) GetContractInvokeURL(transID, txID int64, contractAddr types.Address) (string, string, error) {
	sd.logger.Trace("smcdocker GetContractInvokeURL", "transID", transID, "contract", contractAddr)
	//根据合约地址，查询组织ID
	var orgID string
	if contractAddr == smcbuilder.ThirdPartyContract {
		orgID = smcbuilder.ThirdPartyContract
	} else if contractAddr == std.GetGenesisContractAddr(statedbhelper.GetChainID()) {
		blh := helper.BlockChainHelper{}
		orgID = blh.CalcOrgID("genesis")
	} else {
		split := strings.Split(contractAddr, ".")
		if len(split) == 2 {
			orgID = split[0]
			contractAddr = split[1]
		} else {
			orgID = statedbhelper.GetOrgID(transID, txID, contractAddr)
		}
		sd.logger.Debug("smcdocker GetContractInvokeURL", "transID", transID, "orgID", orgID, "contract", contractAddr)
		if orgID == "" {
			return "", "", errors.New("invalid contractAddr: " + contractAddr)
		}
		orgCodeHash := statedbhelper.GetOrgCodeHash(transID, txID, orgID)

		blh := helper.BlockChainHelper{}
		genesisOrgID := blh.CalcOrgID("genesis")

		var genesisOrgHashStr string
		if orgID != genesisOrgID {
			genesisOrgHashStr = string(statedbhelper.GetOrgCodeHash(transID, txID, genesisOrgID))
		}

		smcsvcFilePath := algorithm.CalcCodeHash(genesisOrgHashStr + string(orgCodeHash))
		smcsvcFilePathStr := hex.EncodeToString(smcsvcFilePath)
		targetBin := filepath.Join(smcbuilder.GetInstance().WorkDir, "bin", orgID, smcsvcFilePathStr, "smcrunsvc")
		_, err := os.Stat(targetBin)
		if err != nil {
			sd.logger.Debug("genesis org contract has updated, rebuild current org's contracts")
			if v, ok := sd.orgNameToURL.Load(orgID); ok {
				fc(v.(string))
			}
			sd.orgNameToURL.Delete(orgID)
		}
	}

	v, ok := sd.orgNameToURL.Load(orgID)
	if ok {
		url := v.(string)
		sd.orgIdToLastTime.Store(orgID, time.Now())
		sd.logger.Debug("smcdocker GetContractInvokeURL map exist ", "transID", transID, "url", url)
		return contractAddr, url, nil
	} else {
		sd.logger.Debug("smcdocker GetContractInvokeURL map not exist",
			"transID", transID, "orgID", orgID, "contract", contractAddr)
		c := make(chan RunDockerRes)
		sd.RunDocker <- RunDocker{
			TransID:      transID,
			TxID:         txID,
			ContractAddr: contractAddr,
			OrgID:        orgID,
			c:            c,
		}
		res := <-c
		if res.Error != "" {
			return "", "", errors.New(res.Error)
		}
		return contractAddr, res.Url, nil
	}
}

//DirtyContractInvokeURL dirty ContractInvokeURL, next invoke the contract will build a new image
func (sd *SMCDocker) DirtyContractInvokeURL(transID, txID int64, contractAddr types.Address) {
	orgID := ""
	if contractAddr == std.GetGenesisContractAddr(statedbhelper.GetChainID()) {
		blh := helper.BlockChainHelper{}
		orgID = blh.CalcOrgID("genesis")
	} else {
		orgID = statedbhelper.GetOrgID(transID, txID, contractAddr)
	}

	if v, ok := sd.orgNameToURL.Load(orgID); ok {
		fc(v.(string))
	}
	sd.orgNameToURL.Delete(orgID)
	d := dockerlib.GetDockerLib()
	sd.logger.Debug("DirtyContractInvokeURL", "orgID", orgID)
	isKilled := d.Kill(orgID)
	sd.logger.Debug("DirtyContractInvokeURL", "orgID", orgID, "killResult", isKilled)
	if !isKilled {
		panic(fmt.Sprintf("kill docker for %v fail!", orgID))
	}
}

// DirtyAllURL dirty all containers URL and kill all containers
func (sd *SMCDocker) DirtyAllURL() {
	sd.orgNameToURL.Range(func(key, value interface{}) bool {
		k := key.(string)
		sd.orgNameToURL.Delete(k)
		d := dockerlib.GetDockerLib()
		sd.logger.Debug("DirtyAllURL", "orgID", k)
		isKilled := d.Kill(k)
		sd.logger.Debug("DirtyAllURL", "orgID", k, "killResult", isKilled)
		if !isKilled {
			panic(fmt.Sprintf("kill docker for %v fail!", k))
		}

		return true
	})
}

// CheckDockerLiveTime 检查 docker 上一次发生交易的时间，超过一定时间就杀掉。
func (sd *SMCDocker) CheckDockerLiveTime() {
	sd.orgIdToLastTime.Range(func(key, value interface{}) bool {
		k := key.(string)
		v := value.(time.Time)
		timeout := common.GlobalConfig.ContainerTimeout
		if timeout == 0 {
			timeout = 30
		}
		if time.Since(v) > time.Duration(timeout*int64(time.Minute)) {
			v, ok := sd.orgNameToURL.Load(k)
			if ok {
				d := dockerlib.GetDockerLib()
				sd.logger.Debug("CheckDockerLiveTime kill", "orgID", k)
				isKilled := d.Kill(k)
				sd.logger.Debug("CheckDockerLiveTime kill", "orgID", k, "killResult", isKilled)
				if !isKilled {
					panic("")
				}
				fc(v.(string))
				sd.orgNameToURL.Delete(k)
				sd.orgIdToLastTime.Delete(k)
			}
		}
		return true
	})
}

func (sd *SMCDocker) runDockerSever() {
	var startingDocker sync.Map // orgID => url
	imageName := "alpine:latest"
	for {
		rd := <-sd.RunDocker
		var v []chan RunDockerRes
		if value, ok := startingDocker.Load(rd.OrgID); ok {
			v = value.([]chan RunDockerRes)
			v = append(v, rd.c)
			startingDocker.Store(rd.OrgID, v)
			continue
		}
		v = append(v, rd.c)
		startingDocker.Store(rd.OrgID, v)
		go func(rd RunDocker) {
			sd.logger.Debug("smcdocker GetContractInvokeURL map not exist,begin builder.GetContractDllPath ", "transID", rd.TransID)

			portStr := strconv.Itoa(int(nu.GetIdlePort()))
			runParam := dockerlib.DockerRunParams{}
			builder := smcbuilder.GetInstance()

			callBackUrl := sd.callbackURL
			if rd.OrgID == smcbuilder.ThirdPartyContract {
				callBackUrl = strings.Replace(callBackUrl, "32333", "32332", 1)
			}

			dllPath, err := builder.GetContractDllPath(rd.TransID, rd.TxID, rd.OrgID)
			if err != nil {
				if value, ok := startingDocker.Load(rd.OrgID); ok {
					res := value.([]chan RunDockerRes)
					for _, v := range res {
						v <- RunDockerRes{
							Url:   "",
							Error: err.Error(),
						}
					}
					startingDocker.Delete(rd.OrgID)
				}
				return
			}
			sd.logger.Debug("Contract dll path:" + dllPath)

			if runtime.GOOS == "windows" {
				runParam.Cmd = []string{
					".\\smcrunsvc.exe",
					"start",
					"-p",
					portStr,
					"-c",
					callBackUrl,
				}
				dllPath = strings.Replace(dllPath, "\\smcrunsvc.exe", "", 1)
				runParam.WorkDir = dllPath
			} else {
				logPath := filepath.Join(builder.WorkDir, "log", rd.OrgID)
				err := os.MkdirAll(logPath, 0750)
				if err != nil {
					startingDocker.Delete(rd.OrgID)
					startingDocker.Range(func(key, value interface{}) bool {
						v := value.(chan RunDockerRes)
						v <- RunDockerRes{
							Url:   "",
							Error: err.Error(),
						}
						return true
					})
					return
				}
				runParam.Cmd = []string{
					"/smcrunsvc",
					"start",
					"-p",
					portStr,
					"-c",
					callBackUrl,
				}
				workDirDocker := "/log/" + rd.OrgID
				runParam.WorkDir = workDirDocker

				runParam.AutoRemove = false
				runParam.Mounts = []dockerlib.Mounts{
					{
						Source:      dllPath,
						Destination: "/smcrunsvc",
					},
					{
						Source:      builder.WorkDir + "/log",
						Destination: "/log",
					},
				}

				hostPort := dockerlib.HostPort{Port: portStr, Host: "0.0.0.0"}
				runParam.PortMap = make(map[string]dockerlib.HostPort)
				runParam.PortMap[portStr+"/tcp"] = hostPort
			}

			d := dockerlib.GetDockerLib()
			sd.logger.Debug("runDockerSever kill", "killOrgID", rd.OrgID)
			if v, ok := sd.orgNameToURL.Load(rd.OrgID); ok {
				fc(v.(string))
			}
			isKill := d.Kill(rd.OrgID)
			sd.logger.Debug("runDockerSever kill", "killOrgID", rd.OrgID, "killResult", isKill)
			if !isKill {
				panic(fmt.Sprintf("kill docker for %v fail!", rd.OrgID))
			}
			sd.logger.Debug("Run docker", "imageName", imageName, "orgID", rd.OrgID, "param", runParam)

			ok, err := d.Run(imageName, rd.OrgID, &runParam)
			sd.logger.Debug("Run docker result", "orgID", rd.OrgID, "result", ok)
			if !ok {
				if value, ok := startingDocker.Load(rd.OrgID); ok {
					res := value.([]chan RunDockerRes)
					for _, v := range res {
						v <- RunDockerRes{
							Url:   "",
							Error: err.Error(),
						}
					}
					startingDocker.Delete(rd.OrgID)
				}
				return
			}

			dockerURL := "tcp://" + d.GetDockerContainerIP(rd.OrgID) + ":" + portStr
			sd.logger.Debug("waitSmcRunSvcReady begin", "dockerURL", dockerURL)
			if waitSmcRunSvcReady(dockerURL, sd.logger) {
				sd.logger.Info("smcdocker GetContractInvokeURL run docker ok ", "transID", rd.TransID, "URL", dockerURL)
			} else {
				panic(fmt.Sprintf("smcdocker GetContractInvokeURL run docker for %v failed", rd.OrgID))
			}
			sd.logger.Debug("put url to orgNameToURL map", "dockerURL", dockerURL)

			sd.orgNameToURL.Store(rd.OrgID, dockerURL)
			sd.orgIdToLastTime.Store(rd.OrgID, time.Now())

			sd.logger.Debug("write response to run docker channel", "orgID", rd.OrgID, "dockerURL", dockerURL)
			if value, ok := startingDocker.Load(rd.OrgID); ok {
				res := value.([]chan RunDockerRes)
				for _, v := range res {
					v <- RunDockerRes{
						Url:   dockerURL,
						Error: "",
					}
				}
				startingDocker.Delete(rd.OrgID)
			}
		}(rd)
	}
}

/*
func maintainDocker(im *SMCDocker) {

	d := dockerlib.GetDockerLib()
	for {
		time.Sleep(time.Duration(1 * time.Second))
		for orgID := range im.orgNameToURL {
			ok := d.Status(orgID)
			if !ok {
				m.Lock()
				fc(im.orgNameToURL[orgID])
				delete(im.orgNameToURL, orgID)
				im.logger.Debug("maintainDocker kill", "orgID", orgID)
				isKilled := d.Kill(orgID)
				im.logger.Debug("maintainDocker", "orgID", orgID, "killResult", isKilled)
				if !isKilled {
					panic(fmt.Sprintf("kill docker for %v fail!", orgID))
				}
				m.Unlock()
			}
		}
	}
}*/

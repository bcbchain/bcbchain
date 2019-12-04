package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/tendermint/go-crypto"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/types"
	"github.com/tendermint/tendermint/types/priv_validator"
)

const (
	// PATH - the url path for exchange pubKey & nodeID
	PATH = "/init-special-request-path-for-exchange-pub-key-and-node-id"
	// PATH2 - the url path for exchange notify (gotten)
	PATH2 = "/init-special-request-path-for-notify-i-got-it"
	// ReportPeriod - continue to notify
	ReportPeriod = 1500 * time.Millisecond
)

// PORT - default rpc port
var PORT = ":46657"

// NodeDef - define a node
type NodeDef struct {
	Name       string   `json:"name"`
	Power      int64    `json:"power"`
	Reward     string   `json:"reward_addr"`
	ListenPort int      `json:"listen_port"`
	Announce   string   `json:"announce"`
	IPIn       string   `json:"ip_in"`
	IPOut      string   `json:"ip_out"`
	IPPriv     string   `json:"ip_priv"`
	Apps       []string `json:"apps"`
}

var (
	serverList  []NodeDef
	serverTable map[string]NodeDef
)

// ReqJSON - request json
type ReqJSON struct {
	PubKey crypto.PubKey `json:"pubKey,omitempty"`
	NodeID string        `json:"nodeId"`
}

// ProcessP2P - process p2p node info exchange
// nolint cyclomatic
func ProcessP2P(genesisDoc types.GenesisDoc, nodeFile string, proxyApp string) {
	content, err := ioutil.ReadFile(nodeFile)
	if err != nil {
		fmt.Println("Node List file read error!!!")
		return
	}

	serverList = make([]NodeDef, 0)
	err = json.Unmarshal(content, &serverList)
	if err != nil {
		fmt.Printf("node List file Unmarshal err: %v\n", err)
		return
	}
	if serverList[0].ListenPort != 0 {
		PORT = fmt.Sprintf(":%d", serverList[0].ListenPort)
	}

	nodeKeyFile := config.NodeKeyFile()
	var nodeKey *p2p.NodeKey
	if nodeKey, err = p2p.LoadNodeKey(nodeKeyFile); err != nil {
		fmt.Println("Processing P2P: Load or Gen Node Key error")
		return
	}

	myNodeID := fmt.Sprintf("%v", nodeKey.ID())
	privValidator := privval.LoadOrGenFilePV(config.PrivValidatorFile())
	myPost := ReqJSON{
		PubKey: privValidator.GetPubKey(),
		NodeID: myNodeID,
	}

	jsonByte, err := cdc.MarshalJSON(myPost)
	if err != nil {
		fmt.Printf("can't marshal my PubKey+NodeId to json: %v\n", err)
	}
	// fmt.Println(string(jsonByte))

	// use map @ server side, for safe delete
	serverTable = make(map[string]NodeDef)
	for _, v := range serverList {
		serverTable[v.Name] = v
	}

	serverChan := make(chan bool, 2)
	go runSimpleServer(serverTable, genesisDoc, myNodeID, serverChan, proxyApp)

	collectedResult := make(map[string]map[string]bool)
	servResultChan := make(chan map[string]map[string]bool, 0)
	go func() {

		reportTicker := time.NewTicker(ReportPeriod)
		defer func() {
			reportTicker.Stop()
		}()
		for {
			select {
			case resp := <-servResultChan:
				for k, v := range resp {
					if val, ok := collectedResult[k]; ok {
						for kk, vv := range v {
							val[kk] = vv
						}
					} else {
						collectedResult[k] = make(map[string]bool)
						for kk, vv := range v {
							collectedResult[k][kk] = vv
						}
					}
				}
			case <-reportTicker.C:
				fmt.Print("sendResult: ")
				keys := make([]string, len(collectedResult))
				i := 0
				for k := range collectedResult {
					keys[i] = k
					i++
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Printf("%s:{", k)
					for kk, vv := range collectedResult[k] {
						fmt.Printf("%s:%v,", kk, vv)
					}
					fmt.Print("}, ")
				}
				fmt.Printf("\n")
			}
		}
	}()
	go func() {
		for {
			for _, n := range serverList {
				if ret, err := dialPeer(n.IPIn, PATH, jsonByte); err == nil && ret == "OK" {
					servResultChan <- map[string]map[string]bool{n.Name: {"pub": true}}
				}
				if ret, err := dialPeer(n.IPPriv, PATH, jsonByte); err == nil && ret == "OK" {
					servResultChan <- map[string]map[string]bool{n.Name: {"pri": true}}
				}
			}
			time.Sleep(time.Second)
		}
	}()

	<-serverChan
	<-serverChan
	fmt.Println("all finished ,gracefully quit")
	for i := 10; i > 0; i-- {
		fmt.Print(i)
		fmt.Print("..")
		time.Sleep(time.Second)
	}

}

// nolint cyclomatic
func runSimpleServer(nodeTable map[string]NodeDef, genesisDoc types.GenesisDoc, myNodeId string, done chan bool, proxyApp string) {
	nodeTableFlag := make(map[string]bool)
	var nodeTableFlagLock sync.Mutex
	for k := range nodeTable {
		nodeTableFlag[k] = true
	}

	conf := cfg.DefaultConfig()
	tmPath := os.Getenv("TMHOME")
	if tmPath == "" {
		home := os.Getenv("HOME")
		if home != "" {
			tmPath = filepath.Join(home, cfg.DefaultTendermintDir)
		}
	}
	if tmPath == "" {
		tmPath = "/" + cfg.DefaultTendermintDir
	}
	config.SetRoot(tmPath)
	configFilePath := filepath.Join(tmPath, "config", "config.toml")

	_ = viper.Unmarshal(conf)

	validators := types.ValidatorsFromFile(genesisDoc, conf.ValidatorsFile())
	validatorMap := make(map[string]types.GenesisValidator)
	var validatorLock sync.Mutex

	iGotIt := false
	startNotifyIGotIt := false
	allReceived := false

	parseRemoteIP := func(r *http.Request) string {
		var remoteIP string

		xRealIP := r.Header.Get("X-Real-Ip")
		xForwardedFor := r.Header.Get("X-Forwarded-For")
		if xRealIP == "" && xForwardedFor == "" {
			// If there are colon in remote address, remove the port number
			// otherwise, return remote address as is
			if strings.ContainsRune(r.RemoteAddr, ':') {
				remoteIP, _, _ = net.SplitHostPort(r.RemoteAddr)
			} else {
				remoteIP = r.RemoteAddr
			}
		}
		return remoteIP
	}

	parseRequest := func(w http.ResponseWriter, r *http.Request) ReqJSON {
		var reqJSON ReqJSON
		jsonByte, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println("request=", string(jsonByte))
			fmt.Printf("decode err: %v\n", err)
			_, _ = fmt.Fprintf(w, "What?")
		}
		err = cdc.UnmarshalJSON(jsonByte, &reqJSON)
		if err != nil {
			fmt.Println("request=", string(jsonByte))
			fmt.Printf("decode err: %v\n", err)
			_, _ = fmt.Fprintf(w, "What?")
		}
		return reqJSON
	}

	notifiedMap := make(map[string]bool)
	var notifiedLock sync.Mutex
	var sentLock sync.Mutex
	http.HandleFunc(PATH2, func(w http.ResponseWriter, r *http.Request) {

		reqJson := parseRequest(w, r)

		inTable := false
		remoteIP := parseRemoteIP(r)
		for name, v := range nodeTable {
			if v.IPOut == remoteIP {
				_, _ = fmt.Fprintf(w, "ACK")
				notifiedLock.Lock()
				notifiedMap[name] = true
				notifiedLock.Unlock()
				inTable = true
				break
			}
		}
		if !inTable {
			ip := net.ParseIP(remoteIP)
			if ip == nil {
				fmt.Printf("I don't know the client: %v\n", remoteIP)
				_, _ = fmt.Fprintf(w, "Who r u?")
				return
			} else {
				if reqJson.NodeID == myNodeId && len(nodeTable)-len(notifiedMap) == 1 {
					// it can be identified when it's the last one
					for name := range nodeTable {
						// I don't know how to get the just ONE k,v pair from the table nicely, so loop it. disgusting.
						if !notifiedMap[name] && nodeTable[name].IPPriv == remoteIP {
							_, _ = fmt.Fprintf(w, "ACK")
							notifiedLock.Lock()
							notifiedMap[name] = true
							notifiedLock.Unlock()
							break
						}
					}
				} else {
					fmt.Printf("unknown client: %v\n", remoteIP)
					_, _ = fmt.Fprintf(w, "continue")
				}
			}
		}

		notifiedLock.Lock()
		if len(notifiedMap) == len(nodeTable) && !allReceived {
			fmt.Printf("notifiedMap: %v\n", notifiedMap)
			allReceived = true
			done <- true
		}
		notifiedLock.Unlock()
	})

	http.HandleFunc(PATH, func(w http.ResponseWriter, r *http.Request) {
		reqJson := parseRequest(w, r)

		remoteIP := parseRemoteIP(r)

		addPeer := func(name string, v NodeDef) {
			// fmt.Println("peer is connected:", v.Domain)

			// add pub key
			for _, val := range *validators {
				if val.Name == name {
					// fmt.Printf("pubkey address => %v\n", reqJson.PubKey.Address())
					pubKey, _ := crypto.PubKeyFromBytes(reqJson.PubKey.Bytes())
					validator := types.GenesisValidator{
						Name:       val.Name,
						Power:      val.Power,
						RewardAddr: val.RewardAddr,
						PubKey:     pubKey,
					}
					validatorLock.Lock()
					validatorMap[name] = validator
					validatorLock.Unlock()
					break
				}
			}

			// add proxy app
			if reqJson.NodeID == myNodeId {
				if proxyApp == "" {
					conf.ProxyApp = v.Apps
				} else {
					conf.ProxyApp = []string{"tcp://" + proxyApp + ":46658"}
				}
				conf.P2P.AAddress = v.Announce
				conf.P2P.ListenAddress = "tcp://0.0.0.0:" + strconv.Itoa(v.ListenPort)
				conf.RPC.ListenAddress = "tcp://0.0.0.0:" + strconv.Itoa(v.ListenPort+1)
			} else {
				// add peer nodeId to persistentPeers
				if !strings.Contains(conf.P2P.PersistentPeers, reqJson.NodeID) {
					if conf.P2P.PersistentPeers == "" {
						conf.P2P.PersistentPeers = fmt.Sprintf("%s@%s", reqJson.NodeID, v.Announce)
					} else {
						conf.P2P.PersistentPeers = fmt.Sprintf("%s,%s@%s",
							conf.P2P.PersistentPeers, reqJson.NodeID, v.Announce)
					}
				}
			}

			// and delete node from table
			nodeTableFlagLock.Lock()
			delete(nodeTableFlag, name)
			nodeTableFlagLock.Unlock()
			_, _ = fmt.Fprintf(w, "OK")
		}

		inTable := false
		for name, v := range nodeTable {
			if v.IPOut == remoteIP {
				addPeer(name, v)
				inTable = true
				break
			}
		}

		if !inTable {
			ip := net.ParseIP(remoteIP)
			if ip == nil {
				fmt.Printf("I don't know the client: %v\n", remoteIP)
				_, _ = fmt.Fprintf(w, "Who r u?")
				return
			} else {
				if reqJson.NodeID == myNodeId && len(nodeTableFlag) == 1 {
					// it can be identified when it's the last one
					for name := range nodeTableFlag {
						if nodeTable[name].IPPriv != remoteIP {
							_, _ = fmt.Fprintf(w, "continue")
							break
						}
						// I don't know how to get the just ONE k,v pair from the table nicely, so loop it. disgusting.
						addPeer(name, nodeTable[name])
						nodeTableFlagLock.Lock()
						nodeTableFlag[name] = true
						nodeTableFlagLock.Unlock()
						iGotIt = true
					}
				} else {
					_, _ = fmt.Fprintf(w, "continue")
				}
			}
		}

		if len(nodeTableFlag) == 0 {
			iGotIt = true
		}

		if iGotIt {
			home := os.Getenv("TMHOME")
			if strings.HasPrefix(home, "/etc") {
				paths := strings.Split(home, "/")
				var myHome string
				if len(paths) > 2 {
					myHome = "/home/" + paths[2]
				} else {
					myHome = "/home/tmcore"
				}
				conf.DBPath = myHome + "/data"
				conf.LogPath = myHome + "/log"
				conf.Mempool.WalPath = myHome + "/data/mempool.wal"
				conf.Consensus.WalPath = myHome + "/data/cs.wal/wal"
			}
			cfg.WriteConfigFile(configFilePath, conf)

			validatorsResult := make([]types.GenesisValidator, 0)
			//validatorLock.Lock()
			for _, v := range validatorMap {
				validatorsResult = append(validatorsResult, v)
			}
			//validatorLock.Unlock()
			outByte, err := cdc.MarshalJSONIndent(validatorsResult, "", "  ")
			if err != nil {
				fmt.Printf("last step,marshal validators err: %v\n", err)
				_, _ = fmt.Fprintf(w, "What?")
				return
			}
			_ = ioutil.WriteFile(conf.ValidatorsFile(), outByte, 0600)

			sentLock.Lock()
			if !startNotifyIGotIt {
				startNotifyIGotIt = true
				go notifyIGotIt(nodeTable, myNodeId, done)
			}
			sentLock.Unlock()
		}
	})

	fmt.Println("Listen @", PORT)
	_ = http.ListenAndServe(PORT, nil)
}

func notifyIGotIt(nodeTable map[string]NodeDef, myNodeID string, done chan bool) {

	myPost := ReqJSON{NodeID: myNodeID}

	jsonByte, err := cdc.MarshalJSON(myPost)
	if err != nil {
		fmt.Printf("can't marshal my PubKey+NodeId to json: %v\n", err)
	}

	myTable := make(map[string]bool)
	doneSent := false
	for {
		for _, n := range nodeTable {
			if respStr, err := dialPeer(n.IPPriv, PATH2, jsonByte); respStr == "ACK" && err == nil {
				myTable[n.IPIn] = true
			}
			if respStr, err := dialPeer(n.IPIn, PATH2, jsonByte); respStr == "ACK" && err == nil {
				myTable[n.IPIn] = true
			}
		}

		if len(myTable) == len(nodeTable) && !doneSent {
			fmt.Printf("doneSent: %v\n", myTable)
			doneSent = true
			done <- true
		}
	}
}

func dialPeer(ip string, path string, jsonByte []byte) (string, error) {
	url := "http://" + ip + PORT + path
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonByte))
	if err != nil {
		// fmt.Printf("Dial to", ip, "cause error1:", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: time.Duration(time.Second),
	}
	resp, err := client.Do(req)
	if err != nil {
		// fmt.Printf("Never mind -> Dial to", ip, "not ready")
		return "", err
	}
	defer func() { _ = resp.Body.Close() }() // nolint unhandled

	// fmt.Println("response Status:", resp.Status)
	// fmt.Println("response Headers:", resp.Header)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// fmt.Printf("Dial to", ip, "cause error3:", err)
		return "", err
	}
	// fmt.Println("response Body:", string(body))
	return string(body), nil
}

// RFC1918 - private network definition, but only the ipv4 we concerned.
// https://en.wikipedia.org/wiki/Private_network
//func isPrivateAddress(ipAddress net.IP) bool {
//	privateNets := []*net.IPNet{ // in long-term server, privateNets should be initialized in init function,
//		str2Net("127.0.0.1/8"), //  but I don't wanna to to so
//		str2Net("10.0.0.0/8"),
//		str2Net("172.16.0.0/12"),
//		str2Net("192.168.0.0/16"),
//	}
//	for i := range privateNets {
//		if privateNets[i].Contains(ipAddress) {
//			return true
//		}
//	}
//	return false
//}

// we discard the error, it's impossible
//func str2Net(str string) *net.IPNet {
//	_, ipNet, _ := net.ParseCIDR(str)
//	return ipNet
//}

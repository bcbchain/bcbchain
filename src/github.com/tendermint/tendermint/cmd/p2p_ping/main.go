package main

import (
	"flag"
	"fmt"
	"github.com/tendermint/go-crypto"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/p2p"
	cmn "github.com/tendermint/tmlibs/common"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var (
	config *cfg.P2PConfig
)

func init() {
	config = cfg.DefaultP2PConfig()
	config.PexReactor = true
}

func MakeSwitch(cfg *cfg.P2PConfig) *p2p.Switch {
	nodeKey := &p2p.NodeKey{
		PrivKey: crypto.GenPrivKeyEd25519(),
	}
	sw := p2p.NewSwitch(cfg)
	ni := p2p.NodeInfo{
		ID:         nodeKey.ID(),
		Moniker:    "testing",
		Network:    "testing",
		Version:    "1.0.2",
		ListenAddr: cmn.Fmt("testing:%v", cmn.RandIntn(64512)+1023),
	}

	sw.SetNodeInfo(ni)
	sw.SetNodeKey(nodeKey)
	return sw
}

func RegSplit(text string, regDelimiter string) []string {
	reg := regexp.MustCompile(regDelimiter)
	indexes := reg.FindAllStringIndex(text, -1)
	lastStart := 0
	result := make([]string, len(indexes)+1)
	for i, element := range indexes {
		result[i] = text[lastStart:element[0]]
		lastStart = element[1]
	}
	result[len(indexes)] = text[lastStart:]
	return result
}

func main() {
	cidPtr := flag.String("c", "devtest", "chainID")
	flag.Parse()
	var wg sync.WaitGroup
	crypto.SetChainId(*cidPtr)
	s := MakeSwitch(config)

	t := func(addr string) {
		defer wg.Done()
		ap := strings.Split(addr, ":")
		if len(ap) != 2 {
			fmt.Println(addr, "in bad format")
			return
		}
		ip := net.ParseIP(ap[0])
		if ip == nil {
			if len(ap[0]) > 0 {
				ips, err := net.LookupIP(ap[0])
				if err != nil {
					fmt.Println(addr, "failed")
					return
				}
				ip = ips[0]
			}
		}
		port := ap[1]
		portN, err := strconv.Atoi(port)
		if err != nil {
			fmt.Println(addr, "in bad format")
			return
		}
		netAddr := p2p.NewNetAddressIPPort(ip, uint16(portN))
		netAddr.ID = "a"
		err = s.DialPeerWithAddress(netAddr, true)
		if err != nil {
			et := err.Error()
			if et[:20] == "Failed to authentica" {
				let := RegSplit(et, " ")
				fmt.Println(addr, "is ok. peerID->", let[len(let)-1])
				return
			}
		}
		fmt.Println(addr, "failed")
	}

	args := os.Args[1:]
	if args[0] == "-c" {
		args = args[2:]
	}
	if len(args[0]) > 3 && args[0][:3] == "-c=" {
		args = args[1:]
	}
	for _, arg := range args {
		lArg := RegSplit(arg, "[,;]")
		for _, ipPort := range lArg {
			wg.Add(1)
			go t(ipPort)
		}
	}
	wg.Wait()
}

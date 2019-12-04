package main

import (
	"blockchain/types"
	"common/rpc/lib/server"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"

	"github.com/tendermint/go-amino"
	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
)

var routes = map[string]*rpcserver.RPCFunc{
	"hello_world": rpcserver.NewRPCFunc(HelloWorld, "name,num"),
}

// MyStruct - ..
type MyStruct struct {
	Name   string `json:"name"`
	Gender string `json:"gender"`
	Age    string `json:"age"`
}

// HelloWorld - hello world
func HelloWorld(name MyStruct, num types.PubKey) (Result, error) {
	return Result{fmt.Sprintf("hi %#v %s", name, hex.EncodeToString(num))}, nil
}

// Result - result
type Result struct {
	Result string
}

func main() {
	mux := http.NewServeMux()
	cdc := amino.NewCodec()
	logger := log.NewTMLogger("test.log", "rpc.lib.test")
	rpcserver.RegisterRPCFuncs(mux, routes, cdc, logger)
	_, err := rpcserver.StartHTTPServer("tcp://0.0.0.0:8008", mux, logger)
	if err != nil {
		cmn.Exit(err.Error())
	}

	// Wait forever
	cmn.TrapSignal(func(sig os.Signal) {
	})

}

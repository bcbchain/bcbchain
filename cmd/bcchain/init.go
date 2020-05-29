package main

import (
	"github.com/bcbchain/bcbchain/abciapp/common"
	"bytes"
	"github.com/bcbchain/bclib/fs"
	"github.com/bcbchain/bclib/jsoniter"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/bcbchain/bclib/tendermint/go-amino"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type BcchainGenesisFile struct {
	Name string          `json:"name"`
	F    json.RawMessage `json:"f"`
}

func initFiles(cmd *cobra.Command, args []string) {
	byzantium, err := cmd.Flags().GetString("follow")
	if err != nil {
		fmt.Printf("init tendermint parse follow err: %s\n", err)
		return
	}
	if byzantium == "" {
		fmt.Printf("init bcchain must use flag \"--follow\", url list split by \",\"\n")
		return
	}

	voters := strings.Split(byzantium, ",")
	if len(voters) == 0 {
		fmt.Println("invalid url list")
		return
	}

	for _, v := range voters {
		pkg := getPkgFromNode(v)
		if pkg != nil {
			if len(pkg) == 0 {
				// genesis from v1.
				return
			}

			configPath := common.GlobalConfig.Path
			if err = fs.UnTarGz(configPath, bytes.NewReader(pkg), nil); err != nil {
				fmt.Printf("UnTar bcchain genesis files failed: %s\n", err)
			}
			return
		}
	}
	fmt.Printf("can not get genesis files from %s \n", byzantium)
	return
}

func getPkgFromNode(node string) []byte {
	url := "https://" + node + "/genesis_pkg?tag=\"bcchain\""
	url2 := "http://" + node + "/genesis_pkg?tag=\"bcchain\""
	pkg := getPkgFromURL(url)
	if pkg != nil {
		return pkg
	}
	return getPkgFromURL(url2)
}

func getPkgFromURL(url string) []byte {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("get pkg from %s cause err1: %v\n", url, err)
		return nil
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("get pkg from %s cause err2: %v\n", url, err)
		return nil
	}
	defer func() { _ = resp.Body.Close() }() // nolint unhandled

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("get pkg from %s cause err3: %v\n", url, err)
		return nil
	}

	type pkgResponse struct {
		Result BcchainGenesisFile `json:"result"`
	}

	var peerResponse pkgResponse
	err = cdc.UnmarshalJSON(body, &peerResponse)
	if err != nil {
		fmt.Printf("get pkg from %s parse err4: %v\n", url, err)
		return nil
	}

	if len(peerResponse.Result.F) == 0 {
		return []byte{}
	}

	var pkg []byte
	err = jsoniter.Unmarshal(peerResponse.Result.F, &pkg)
	if err != nil {
		fmt.Printf("get pkg from %s unmarshal err5: %v\n", url, err)
		return nil
	}
	return pkg
}

var cdc = amino.NewCodec()

func init() {
	crypto.RegisterAmino(cdc)
}

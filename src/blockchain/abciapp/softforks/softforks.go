package softforks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"common/sig"

	"github.com/pkg/errors"
)

var TagToForkInfo map[string]ForkInfo

//具体含义请参考 bcchain.yaml
type ForkInfo struct {
	Tag               string `json:"tag,omitempty"`               //Tag, contains the former released version
	BugBlockHeight    int64  `json:"bugblockheight,omitempty"`    // bug block height
	EffectBlockHeight int64  `json:"effectblockheight,omitempty"` // Effect Block Height
	Description       string `json:"description,omitempty"`       // Description for the fork
}

// explicit call
func Init() {

	if len(TagToForkInfo) == 0 {
		TagToForkInfo = make(map[string]ForkInfo)
	} else {
		return
	}

	ex, err := os.Executable()
	if err != nil {
		fmt.Println(err)
		return
	}

	dir := filepath.Dir(ex)
	if dir == "" {
		panic(errors.New("Failed to get path of forks file"))
	}

	forksFile := dir + "/abci-forks.json"
	if _, err = os.Stat(forksFile); err != nil {
		//File doesn't exist, terminate the process
		//panic(err.Error())
		return
	}
	sigFile := dir + "/abci-forks.json.sig"
	if _, err = os.Stat(forksFile); err != nil {
		//File doesn't exist, terminate the process
		//panic(err.Error())
		return
	}
	// Verify Fork.json
	_, err = sig.VerifyTextFile(forksFile, sigFile)
	if err != nil {
		// Failed to verify, terminate the process
		panic(err.Error())
	}

	// Notes: be careful of permission of the file, should be 444 or 644
	data, err := ioutil.ReadFile(forksFile)
	if err != nil {
		panic(err.Error())
	}

	AppForkInfo := make([]ForkInfo, 0)
	err = json.Unmarshal(data, &AppForkInfo)
	if err != nil {
		panic(err.Error())
	}

	for _, v := range AppForkInfo {
		TagToForkInfo[v.Tag] = v
	}
}

// Fixs bug #2092, only the last reward be shown in block.
// Adds the softfork to show all of rewards in block
func V1_0_2_3233(blockHeight int64) bool {
	if forkInfo, ok := TagToForkInfo["fork-abci#1.0.2.3233"]; ok {
		return blockHeight < forkInfo.EffectBlockHeight
	}

	return false
}

// Fixs bug #4281, sdk block hash not equal tendermint block hahs.
// Adds the softfork to reset sdk block hash
func V2_0_1_13780(blockHeight int64) bool {
	if forkInfo, ok := TagToForkInfo["fork-abci#2.0.1.13780"]; ok {
		return blockHeight < forkInfo.EffectBlockHeight
	}

	return false
}

// Fixs bug #4251, gas_used showed be sum of all messages in block.
// Adds the softfork to show all of gas_used in block
func IsForkForV2_0_2_14654(blockHeight int64) bool {
	if forkInfo, ok := TagToForkInfo["fork-abci#2.0.2.14654"]; ok {
		if blockHeight < forkInfo.EffectBlockHeight &&
			blockHeight > forkInfo.BugBlockHeight {
			return true
		}
	}
	return false
}

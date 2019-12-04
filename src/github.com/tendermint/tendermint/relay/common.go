package relay

import (
	"blockchain/tx2"
	"blockchain/types"
	"errors"
	"github.com/tendermint/go-crypto"
	cfg "github.com/tendermint/tendermint/config"
	pvm "github.com/tendermint/tendermint/types/priv_validator"
	"strings"
)

func blockQuery(url string, height int64) (resultBlock *ResultBlock, err error) {
	resultBlock = new(ResultBlock)
	client := getClient(url)
	_, err = client.Call("block", map[string]interface{}{"height": height}, resultBlock)
	return
}

func blockResultQuery(url string, height int64) (resultBlockResults *ResultBlockResults, err error) {
	resultBlockResults = new(ResultBlockResults)
	client := getClient(url)
	_, err = client.Call("block_results", map[string]interface{}{"height": height}, resultBlockResults)
	return
}

func abciInfoQuery(url string) (resultABCIInfo *ResultABCIInfo, err error) {
	resultABCIInfo = new(ResultABCIInfo)
	client := getClient(url)
	_, err = client.Call("abci_info", nil, resultABCIInfo)
	return
}

func generateTx(contract types.Address, method uint32, params []interface{}, nonce uint64, gaslimit int64, note, privKey, toChainID string) string {
	items := tx2.WrapInvokeParams(params...)
	message := types.Message{
		Contract: contract,
		MethodID: method,
		Items:    items,
	}
	payload := tx2.WrapPayload(nonce, gaslimit, note, message)

	return tx2.WrapTxEx(toChainID, payload, privKey)
}

func queryIBCContract(url, orgID string) (*Contract, error) {
	versionList, err := queryVersionList(url, "ibc", orgID)
	if err != nil {
		return nil, errors.New("query ibc version list failed：" + err.Error())
	}

	if len(versionList.ContractAddrList) == 0 {
		return nil, errors.New("can not get ibc contract version list")
	}

	remoteBlockHeight, err := queryCurrentHeight(url)
	if err != nil {
		return nil, err
	}

	for i := len(versionList.ContractAddrList) - 1; i >= 0; i-- {
		contract, err := queryContract(url, versionList.ContractAddrList[i])
		if err != nil {
			return nil, errors.New("query ibc contract list failed：" + err.Error())
		}
		if contract.EffectHeight <= remoteBlockHeight {
			return contract, nil
		}
	}

	return nil, errors.New("can not get valid ibc contract address")
}

func getCurrentNodePrivKey(config *cfg.Config) crypto.PrivKey {
	privValidatorFile := config.PrivValidatorFile()
	return pvm.LoadFilePV(privValidatorFile).PrivKey
}

func getOtherOrgID(pktsProofs []*PktsProof, genesisOrgID string) (otherOrgID string) {
	for _, pktProof := range pktsProofs {
		for _, p := range pktProof.Packets {
			if p.OrgID != genesisOrgID {
				otherOrgID = p.OrgID
				return
			}
		}
	}
	return
}

func splitQueueID(queueID string) (fromChainID, toChainID string) {
	idList := strings.Split(queueID, "->")
	return idList[0], idList[1]
}

func makeQueueID(fromChainID, toChainID string) string {
	return fromChainID + "->" + toChainID
}

func getCurrentNodeAddress(config *cfg.Config) string {
	privValidatorFile := config.PrivValidatorFile()
	return pvm.LoadFilePV(privValidatorFile).GetAddress()
}

func getMainChaidID(chainID string) string {
	if strings.Contains(chainID, "[") {
		return chainID[:strings.Index(chainID, "[")]
	}
	return chainID
}
func replaceChainID(address, queueID string) string {
	localChainID, remoteChainID := splitQueueID(queueID)
	return strings.Replace(address, localChainID, remoteChainID, 1)
}

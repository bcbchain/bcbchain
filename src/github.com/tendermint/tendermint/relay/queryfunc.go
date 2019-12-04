package relay

import jsoniter "github.com/json-iterator/go"

func queryContract(url, address string) (*Contract, error) {
	contract := new(Contract)
	err := abciQueryAndParse(url, keyOfContract(address), contract)
	return contract, err

}

func queryVersionList(url, contractName, orgID string) (*ContractVersionList, error) {
	versionList := new(ContractVersionList)
	err := abciQueryAndParse(url, keyOfVersionList(contractName, orgID), versionList)
	return versionList, err
}

func queryCurrentHeight(url string) (int64, error) {
	result, err := abciInfoQuery(url)
	if err != nil {
		return 0, err
	}
	return result.Response.LastBlockHeight, nil
}

func queryIBCMsgIndex(url, queueID string, seq uint64) (*MessageIndex, error) {
	msgIndex := new(MessageIndex)
	err := abciQueryAndParse(url, keyOfMessageIndex(queueID, seq), msgIndex)
	return msgIndex, err
}

func querySequence(url, queueID string) (sequence uint64, err error) {
	resultQuery := new(ResultABCIQuery)
	resultQuery, err = abciQuery(url, keyOfSequence(queueID))
	if err != nil {
		return
	}

	if len(resultQuery.Response.GetValue()) == 0 {
		return 0, nil
	}

	err = jsoniter.Unmarshal(resultQuery.Response.GetValue(), &sequence)
	return
}

func queryAccountNonce(url, address string) (uint64, error) {
	type account struct {
		Nonce uint64 `json:"nonce"`
	}

	result, err := abciQuery(url, keyOfAccountNonce(address))
	if err != nil {
		return 0, err
	}
	if len(result.Response.GetValue()) == 0 {
		return 0, nil
	}

	acc := new(account)
	if err := jsoniter.Unmarshal(result.Response.GetValue(), acc); err != nil {
		return 0, err
	}
	return acc.Nonce, nil
}

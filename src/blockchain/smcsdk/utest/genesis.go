//unittestplatform
// genesis.go 实现测试链的创世功能。
//另： 如果采用在线测试链，则需要支持发送https消息。

package utest

import (
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"

	"github.com/tendermint/tmlibs/log"
)

//GetGenesisToken Get Genesis Token
func GetGenesisToken() std.Token {
	return UTP.g.AppStateJSON.GnsToken
}

//GetGenesisContracts Get Genesis Contracts
func GetGenesisContracts() []std.Contract {
	return UTP.g.AppStateJSON.GnsContracts
}

// GenesisDocFromFile reads JSON data from a file and unmarshalls it into a GenesisDoc.
func readGenesisFile() (*genesis, error) {
	genDoc := genesis{}
	err := jsoniter.Unmarshal([]byte(genesisStr), &genDoc)
	if err != nil {
		panic(err.Error())
	}

	return &genDoc, nil
}

func initGenesis(g *genesis) (smcError types.Error) {
	initStateDB()

	sdkToken := std.Token{
		Name:             g.AppStateJSON.GnsToken.Name,
		Address:          g.AppStateJSON.GnsToken.Address,
		Owner:            g.AppStateJSON.GnsToken.Owner,
		Symbol:           g.AppStateJSON.GnsToken.Symbol,
		TotalSupply:      g.AppStateJSON.GnsToken.TotalSupply,
		AddSupplyEnabled: g.AppStateJSON.GnsToken.AddSupplyEnabled,
		BurnEnabled:      g.AppStateJSON.GnsToken.BurnEnabled,
		GasPrice:         g.AppStateJSON.GnsToken.GasPrice,
	}
	saveToken(&sdkToken)

	addrList := make([]types.Address, len(g.AppStateJSON.GnsContracts))
	for i, c := range g.AppStateJSON.GnsContracts {
		marBytes, err := jsoniter.Marshal(c)
		if err != nil {
			panic(err.Error())
		}
		setToDB(std.KeyOfGenesisContract(c.Address), marBytes)
		setToDB(std.KeyOfContract(c.Address), marBytes)

		cnt := std.ContractVersionList{}
		cnt.ContractAddrList = make([]types.Address, 1)
		cnt.EffectHeights = make([]int64, 1)
		cnt.Name = c.Name
		cnt.ContractAddrList[0] = c.Address
		cnt.EffectHeights[0] = c.EffectHeight
		cntBytes, err := jsoniter.Marshal(cnt)
		if err != nil {
			panic(err.Error())
		}
		setToDB(std.KeyOfContractsWithName(utOrgID, c.Name), cntBytes)

		addrList[i] = c.Address
	}
	gcs, err := jsoniter.Marshal(addrList)
	if err != nil {
		panic(err.Error())
	}

	setToDB(std.KeyOfGenesisContractAddrList(), gcs)

	var gasprice int64 = 2500
	gasbyte, err := jsoniter.Marshal(gasprice)
	if err != nil {
		panic(err.Error())
	}
	setToDB(std.KeyOfTokenBaseGasPrice(), gasbyte)

	Commit()

	return
}

func saveToken(token *std.Token) {
	resBytes, err := jsoniter.Marshal(token)
	if err != nil {
		panic(err.Error())
	}
	setToDB(std.KeyOfGenesisToken(), resBytes)
	setToDB(std.KeyOfToken(token.Address), resBytes)

	bAddr, err := jsoniter.Marshal(token.Address)
	if err != nil {
		panic(err.Error())
	}
	setToDB(std.KeyOfTokenWithName(token.Name), bAddr)
	setToDB(std.KeyOfTokenWithSymbol(token.Symbol), bAddr)

	// set balance of token
	acctInfo := std.AccountInfo{Address: token.Address, Balance: token.TotalSupply}
	resBytes, err = jsoniter.Marshal(acctInfo)
	if err != nil {
		panic(err.Error())
	}
	setToDB(std.KeyOfAccountToken(token.Owner, token.Address), resBytes)
}

//InitLog init log
func InitLog(moduleName string) log.Loggerf {
	l := log.NewTMLogger("./log", moduleName)
	l.SetOutputToFile(true)
	l.SetOutputToScreen(false)
	l.AllowLevel("info")

	return l
}

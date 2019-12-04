/*
 * blockchain.go 实现区块链相关功能，包括：
 * 部署测试合约，设置测试区块链的区块高度，生成下一区块，设置、获取区块链ID，以及重置区块链数据库等。
 */

package utest

import (
	"blockchain/algorithm"
	"blockchain/smcsdk/common/gls"
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
	"blockchain/smcsdk/sdkimpl/object"
	"blockchain/smcsdk/sdkimpl/sdkhelper"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/tmlibs/log"
)

func init() {
	crypto.SetChainId("test")
}

func upgradeContract(contractName, orgID string, methods, interfaces []string) sdk.IAccount {

	bc := sdbGet(0, 0, std.KeyOfContractsWithName(orgID, contractName))
	if bc == nil {
		return nil
	}
	bc = data(std.KeyOfContractsWithName(orgID, contractName), bc)

	caddr := make([]types.Address, 0)
	a := new(std.ContractVersionList)
	err := jsoniter.Unmarshal(bc, a)
	if err != nil {
		panic(err)
	}
	caddr = append(caddr, a.ContractAddrList...)

	// Get last one
	bc = sdbGet(0, 0, std.KeyOfContract(caddr[len(caddr)-1]))
	if bc == nil {
		return nil
	}

	contract := std.Contract{}
	r := std.GetResult{}
	err = jsoniter.Unmarshal(bc, &r)
	if err != nil {
		panic(err.Error())
	}
	err = jsoniter.Unmarshal(r.Data, &contract)
	if err != nil {
		panic(err.Error())
	}
	contract.LoseHeight = UTP.Block().Height() + 1
	cons := make(map[string][]byte)
	byteCon1, err := jsoniter.Marshal(contract)
	if err != nil {
		panic(err.Error())
	}
	cons[std.KeyOfContract(contract.Address)] = byteCon1

	newcontract := std.Contract{
		Address:      UTP.Helper().BlockChainHelper().CalcContractAddress(contractName, "2.0", orgID),
		Name:         contractName,
		Account:      contract.Account,
		Owner:        contract.Owner,
		Version:      "2.0",
		CodeHash:     []byte(contractName + "Hash"),
		EffectHeight: UTP.Block().Height() + 1,
		LoseHeight:   0,
		KeyPrefix:    contract.KeyPrefix,
		Methods:      make([]std.Method, 0),
		Token:        contract.Token,
		OrgID:        orgID,
	}
	byteCon2, err := jsoniter.Marshal(newcontract)
	if err != nil {
		panic(err.Error())
	}
	cons[std.KeyOfContract(newcontract.Address)] = byteCon2

	caddr = append(caddr, newcontract.Address)
	byteCon3, err := jsoniter.Marshal(caddr)
	if err != nil {
		panic(err.Error())
	}
	cons[std.KeyOfContractsWithName(orgID, contractName)] = byteCon3
	sdbSet(0, 0, cons)

	_contract := object.NewContractFromAddress(UTP.ISmartContract, newcontract.Address)
	api := UTP.ISmartContract.(*sdkimpl.SmartContract)
	api.Message().(*object.Message).SetContract(_contract)

	UTP.setTxSender(newcontract.Owner)

	Commit()

	return object.NewAccount(UTP.ISmartContract, newcontract.Owner)
}

func deployContract(contractName, orgID string, methods, interfaces []string, logger log.Loggerf) sdk.IAccount {

	sdkhelper.Init(TransferFunc, build, sdbSet, sdbGet, getBlock, nil, &logger)
	UTP.ISmartContract = sdkhelper.New(
		1,
		0,
		UTP.g.AppStateJSON.GnsToken.Owner,
		UTP.g.AppStateJSON.GnsToken.Owner,
		1000,
		1000,
		"note",
		[]byte(""),
		UTP.g.AppStateJSON.GnsToken.Address,
		"45676666",
		tx,
		nil)

	var owner sdk.IAccount
	gls.Mgr.SetValues(gls.Values{gls.SDKKey: UTP.ISmartContract}, func() {
		bc := sdbGet(0, 0, std.KeyOfContractsWithName(orgID, contractName))
		bc = data(std.KeyOfContractsWithName(orgID, contractName), bc)
		if bc != nil {
			owner = upgradeContract(contractName, orgID, methods, interfaces)
			return
		}

		owner = NewAccount(UTP.g.AppStateJSON.GnsToken.Name, bn.N(0))

		pc := std.Contract{
			Address:      UTP.Helper().BlockChainHelper().CalcContractAddress(contractName, "1.0", orgID),
			Name:         contractName,
			Account:      UTP.Helper().BlockChainHelper().CalcAccountFromName(contractName, orgID),
			Owner:        owner.Address(),
			Version:      "1.0",
			CodeHash:     []byte(contractName + "Hash"),
			EffectHeight: 1,
			LoseHeight:   0,
			KeyPrefix:    "/" + contractName,
			Methods:      make([]std.Method, 0),
			Token:        "",
			OrgID:        orgID,
		}
		prefix = pc.KeyPrefix

		for _, m := range methods {
			md := std.Method{
				MethodID:  algorithm.ConvertMethodID(algorithm.CalcMethodId(m)),
				Gas:       200,
				ProtoType: m,
			}
			pc.Methods = append(pc.Methods, md)
		}
		for _, m := range interfaces {
			md := std.Method{
				MethodID:  algorithm.ConvertMethodID(algorithm.CalcMethodId(m)),
				Gas:       100,
				ProtoType: m,
			}
			pc.Interfaces = append(pc.Interfaces, md)
		}

		addrs := make([]types.Address, 0)
		addrs = append(addrs, pc.Address)
		addrsBytes, err := jsoniter.Marshal(addrs)
		if err != nil {
			panic(err.Error())
		}
		resBytes, err := jsoniter.Marshal(pc)
		if err != nil {
			panic(err.Error())
		}
		setToDB(std.KeyOfContractsWithName(orgID, contractName), addrsBytes)
		setToDB(std.KeyOfContract(pc.Address), resBytes)
		setToDB(std.KeyOfAccountContracts(pc.Owner), addrsBytes)

		addrList := std.ContractVersionList{
			Name:             contractName,
			ContractAddrList: addrs,
			EffectHeights:    []int64{1},
		}
		resBytes, err = jsoniter.Marshal(addrList)
		if err != nil {
			panic(err.Error())
		}
		setToDB(std.KeyOfContractsWithName(orgID, contractName), resBytes)

		_contract := object.NewContractFromSTD(UTP.ISmartContract, &pc)
		api := UTP.ISmartContract.(*sdkimpl.SmartContract)
		api.Message().(*object.Message).SetContract(_contract)

		UTP.setTxSender(owner.Address())
	})

	Commit()
	return owner
}

//SetBlockHeight set block height
func SetBlockHeight(height int64) {
	BlockHeight = height
}

func setChainID(chain string) {
	utChainID = chain
	crypto.SetChainId(chain)
}

//GetChainID get chainID
func GetChainID() string {
	return utChainID
}

// CalcAccountFromPubKey calculate account address from pubKey
func CalcAccountFromPubKey(pubKey types.PubKey) types.Address {
	sdk.Require(pubKey != nil && len(pubKey) == 32,
		types.ErrInvalidParameter, "invalid pubKey")

	pk := crypto.PubKeyEd25519FromBytes(pubKey)
	return pk.Address(utChainID)
}

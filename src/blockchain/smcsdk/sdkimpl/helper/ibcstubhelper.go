package helper

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/rlp"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
	"blockchain/smcsdk/sdkimpl/object"
	"errors"
)

// ReceiptHelper receipt helper information
type IBCStubHelper struct {
	smc sdk.ISmartContract //指向智能合约API对象指针
}

var _ sdk.IIBCStubHelper = (*IBCStubHelper)(nil)
var _ sdkimpl.IAcquireSMC = (*IBCStubHelper)(nil)

// SMC get smart contract object
func (ish *IBCStubHelper) SMC() sdk.ISmartContract { return ish.smc }

// SetSMC set smart contract object
func (ish *IBCStubHelper) SetSMC(smc sdk.ISmartContract) { ish.smc = smc }

// Recast invoke by ibc contract for recast asset
func (ish *IBCStubHelper) Recast(ibcHash types.HexBytes, orgID, contractName string, inReceipts []types.KVPair) (br bool, outReceipts []types.KVPair, err types.Error) {
	defer ish.deferFunc(&err)

	llState := ish.smc.(*sdkimpl.SmartContract).LlState()
	oldCache := llState.GetCache()
	oldMsg := ish.smc.Message()
	methodID := "d12ee281" // Recast(types.Hash)bool

	// reset message for sdk
	e := ish.resetMessageForSDK(orgID, contractName, methodID, inReceipts, ibcHash)
	if e != nil {
		return false, nil, types.Error{
			ErrorCode: types.ErrInvalidParameter,
			ErrorDesc: e.Error(),
		}
	}

	data, outReceipts, err := sdkimpl.IBCInvokeFunc(ish.smc)
	if err.ErrorCode != types.CodeOK {
		llState.SetCache(oldCache)
		sdkimpl.McInst.DirtyTransTx(llState.TransID(), llState.TxID())
		br = false
	} else {
		var ds []interface{}
		e := jsoniter.Unmarshal([]byte(data), &ds)
		if e != nil {
			panic(e)
		}
		br = ds[0].(bool)
	}

	ish.smc.(*sdkimpl.SmartContract).SetMessage(oldMsg)
	ish.smc.Message().(*object.Message).AppendOutput(outReceipts)

	return
}

// Confirm invoke by ibc contract for confirm transaction
func (ish *IBCStubHelper) Confirm(ibcHash types.HexBytes, orgID, contractName string, inReceipts []types.KVPair) (outReceipts []types.KVPair, err types.Error) {
	defer ish.deferFunc(&err)

	oldMsg := ish.smc.Message()
	methodID := "a73649e6" // Confirm(types.Hash)

	// reset message for sdk
	e := ish.resetMessageForSDK(orgID, contractName, methodID, inReceipts, ibcHash)
	if e != nil {
		return nil, types.Error{
			ErrorCode: types.ErrInvalidParameter,
			ErrorDesc: e.Error(),
		}
	}

	_, outReceipts, err = sdkimpl.IBCInvokeFunc(ish.smc)

	ish.smc.(*sdkimpl.SmartContract).SetMessage(oldMsg)
	ish.smc.Message().(*object.Message).AppendOutput(outReceipts)

	return
}

// Cancel invoke by ibc contract for cancel transaction
func (ish *IBCStubHelper) Cancel(ibcHash types.HexBytes, orgID, contractName string, inReceipts []types.KVPair) (outReceipts []types.KVPair, err types.Error) {
	defer ish.deferFunc(&err)

	oldMsg := ish.smc.Message()
	methodID := "1a3dd2f" // Cancel(types.Hash)

	// reset message for sdk
	e := ish.resetMessageForSDK(orgID, contractName, methodID, inReceipts, ibcHash)
	if e != nil {
		return nil, types.Error{
			ErrorCode: types.ErrInvalidParameter,
			ErrorDesc: e.Error(),
		}
	}

	_, outReceipts, err = sdkimpl.IBCInvokeFunc(ish.smc)

	ish.smc.(*sdkimpl.SmartContract).SetMessage(oldMsg)
	ish.smc.Message().(*object.Message).AppendOutput(outReceipts)

	return
}

// TryHub invoke by ibc contract for try hub transaction
func (ish *IBCStubHelper) TryRecast(ibcHash types.HexBytes, orgID, contractName string, inReceipts []types.KVPair) (br bool, outReceipts []types.KVPair, err types.Error) {
	defer ish.deferFunc(&err)

	llState := ish.smc.(*sdkimpl.SmartContract).LlState()
	oldCache := llState.GetCache()
	oldMsg := ish.smc.Message()
	methodID := "39906762" // TryRecast(types.Hash)bool

	// reset message for sdk
	e := ish.resetMessageForSDK(orgID, contractName, methodID, inReceipts, ibcHash)
	if e != nil {
		return false, nil, types.Error{
			ErrorCode: types.ErrInvalidParameter,
			ErrorDesc: e.Error(),
		}
	}

	data, outReceipts, err := sdkimpl.IBCInvokeFunc(ish.smc)
	if err.ErrorCode != types.CodeOK {
		llState.SetCache(oldCache)
		sdkimpl.McInst.DirtyTransTx(llState.TransID(), llState.TxID())
	} else {
		var ds []interface{}
		e := jsoniter.Unmarshal([]byte(data), &ds)
		if e != nil {
			panic(e)
		}
		br = ds[0].(bool)
	}

	ish.smc.(*sdkimpl.SmartContract).SetMessage(oldMsg)
	ish.smc.Message().(*object.Message).AppendOutput(outReceipts)

	return
}

// ConfirmHub invoke by ibc contract for confirm hub transaction
func (ish *IBCStubHelper) ConfirmRecast(ibcHash types.HexBytes, orgID, contractName string, inReceipts []types.KVPair) (outReceipts []types.KVPair, err types.Error) {
	defer ish.deferFunc(&err)

	oldMsg := ish.smc.Message()
	methodID := "80b168c2" // ConfirmRecast(types.Hash)

	// reset message for sdk
	e := ish.resetMessageForSDK(orgID, contractName, methodID, inReceipts, ibcHash)
	if e != nil {
		return nil, types.Error{
			ErrorCode: types.ErrInvalidParameter,
			ErrorDesc: e.Error(),
		}
	}

	_, outReceipts, err = sdkimpl.IBCInvokeFunc(ish.smc)

	ish.smc.(*sdkimpl.SmartContract).SetMessage(oldMsg)
	ish.smc.Message().(*object.Message).AppendOutput(outReceipts)

	return
}

// CancelHub invoke by ibc contract for cancel hub transaction
func (ish *IBCStubHelper) CancelRecast(ibcHash types.HexBytes, orgID, contractName string, inReceipts []types.KVPair) (outReceipts []types.KVPair, err types.Error) {
	defer ish.deferFunc(&err)

	oldMsg := ish.smc.Message()
	methodID := "6ec2ef98" // CancelRecast(types.Hash)

	// reset message for sdk
	e := ish.resetMessageForSDK(orgID, contractName, methodID, inReceipts, ibcHash)
	if e != nil {
		return nil, types.Error{
			ErrorCode: types.ErrInvalidParameter,
			ErrorDesc: e.Error(),
		}
	}

	_, outReceipts, err = sdkimpl.IBCInvokeFunc(ish.smc)

	ish.smc.(*sdkimpl.SmartContract).SetMessage(oldMsg)
	ish.smc.Message().(*object.Message).AppendOutput(outReceipts)

	return
}

// Notify invoke by ibc contract for notify transaction
func (ish *IBCStubHelper) Notify(ibcHash types.HexBytes, orgID, contractName string, inReceipts []types.KVPair) (outReceipts []types.KVPair, err types.Error) {
	defer ish.deferFunc(&err)

	llState := ish.smc.(*sdkimpl.SmartContract).LlState()
	oldCache := llState.GetCache()
	oldMsg := ish.smc.Message()
	methodID := "36b9f7af" // Notify(types.Hash)

	// reset message for sdk
	e := ish.resetMessageForSDK(orgID, contractName, methodID, inReceipts, ibcHash)
	if e != nil {
		return nil, types.Error{
			ErrorCode: types.ErrInvalidParameter,
			ErrorDesc: e.Error(),
		}
	}

	_, outReceipts, err = sdkimpl.IBCInvokeFunc(ish.smc)
	if err.ErrorCode != types.CodeOK {
		llState.SetCache(oldCache)
		sdkimpl.McInst.DirtyTransTx(llState.TransID(), llState.TxID())
	}

	ish.smc.(*sdkimpl.SmartContract).SetMessage(oldMsg)
	ish.smc.Message().(*object.Message).AppendOutput(outReceipts)

	return
}

// resetMessageForSDK create a new message and the reset it
func (ish *IBCStubHelper) resetMessageForSDK(orgID, contractName, mID string, receipts []types.KVPair, params ...interface{}) error {
	contract, err := ish.contractOfNameEx(orgID, contractName)
	if err != nil {
		return err
	}

	originMessage := ish.smc.Message()

	originList := ish.smc.Message().Origins()
	originList = append(originList, ish.smc.Message().Contract().Address())

	items := ish.pack(params...)

	newmsg := object.NewMessage(ish.smc, contract, mID, items, originMessage.Contract().Account().Address(),
		originMessage.Payer().Address(), originList, receipts)
	ish.smc.(*sdkimpl.SmartContract).SetMessage(newmsg)

	return nil
}

// pack pack params with rlp one by one
func (ish *IBCStubHelper) pack(params ...interface{}) []types.HexBytes {
	paramsRlp := make([]types.HexBytes, len(params))
	for i, param := range params {

		paramRlp, err := rlp.EncodeToBytes(param)
		if err != nil {
			panic(err)
		}
		paramsRlp[i] = paramRlp
	}

	return paramsRlp
}

// ContractOfName get contract object with name
func (ish *IBCStubHelper) contractOfNameEx(orgID, name string) (sdk.IContract, error) {

	key := std.KeyOfContractsWithName(orgID, name)
	versionList := ish.smc.(*sdkimpl.SmartContract).LlState().McGet(key, &std.ContractVersionList{})
	if versionList == nil {
		return nil, errors.New("deploy " + name + " contract first")
	}

	var contract sdk.IContract
	vs := versionList.(*std.ContractVersionList)
	for i := len(vs.ContractAddrList) - 1; i >= 0; i-- {
		if ish.smc.Block().Height() >= vs.EffectHeights[i] {
			contract = ish.smc.Helper().ContractHelper().ContractOfAddress(vs.ContractAddrList[i])
			if contract.LoseHeight() != 0 && contract.LoseHeight() <= ish.smc.Block().Height() {
				return nil, errors.New("never effective " + name + " contract")
			}
			break
		}
	}

	return contract, nil
}

// deferFunc set sequence and recover panic
func (ish *IBCStubHelper) deferFunc(errPtr *types.Error) {
	if err := recover(); err != nil {
		if errInfo, ok := err.(types.Error); ok {
			errPtr.ErrorDesc = errInfo.ErrorDesc
			errPtr.ErrorCode = errInfo.ErrorCode
		}
	}
}

package smcrunctl

import (
	"github.com/bcbchain/bcbchain/abciapp_v1.0/bcerrors"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/contract/stubapi"
	"github.com/bcbchain/bcbchain/abciapp_v1.0/statedb"
	"github.com/bcbchain/bcbchain/smcbuilder"
	"github.com/bcbchain/bcbchain/smcdocker"
	"github.com/bcbchain/bclib/jsoniter"
	"github.com/bcbchain/bclib/socket"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
	"sync"
)

type InvokeMgr struct {
	invokeItems sync.Map // map[transID]invokeParams
	logger      log.Logger
}

var (
	doOnce    sync.Once
	invokeMgr *InvokeMgr
	p         *socket.ConnectionPool
)

func GetInstance() *InvokeMgr {
	doOnce.Do(func() {
		invokeMgr = &InvokeMgr{}
	})

	return invokeMgr
}

func (im *InvokeMgr) SetLogger(logger log.Logger) {
	im.logger = logger
}

func (im *InvokeMgr) Invoke(items *stubapi.InvokeParams, transID int64) (response stubapi.Response, bcErr bcerrors.BCError) {
	im.invokeItems.Store(transID, items)
	defer im.invokeItems.Delete(transID)

	itemsEx := newInvokeParamsEx(items, transID)

	client, err := im.pool(items.Ctx.TxState.StateDB).GetClient()
	if err != nil {
		panic(err)
	}
	defer im.pool(items.Ctx.TxState.StateDB).ReleaseClient(client)

	resp, err := client.Call("Invoke", map[string]interface{}{"callParam": itemsEx}, 60)
	if err != nil {
		panic(err)
	}

	err = jsoniter.Unmarshal([]byte(resp.(string)), &response)
	if err != nil {
		panic(err)
	}

	bcErr.ErrorCode = bcerrors.ErrCodeOK
	if response.Code != bcerrors.ErrCodeOK {
		bcErr.ErrorCode = response.Code
		bcErr.ErrorDesc = response.Log
	}

	return
}

func (im *InvokeMgr) InitContractDocker(stateDB *statedb.StateDB) bcerrors.BCError {
	im.invokeItems.Store(int64(0), &stubapi.InvokeParams{
		Ctx: &stubapi.InvokeContext{
			TxState: &statedb.TxState{
				StateDB: stateDB,
			},
		},
	})
	defer im.invokeItems.Delete(int64(0))

	client, err := im.pool(stateDB).GetClient()
	if err != nil {
		panic(err)
	}
	defer im.pool(stateDB).ReleaseClient(client)

	return bcerrors.BCError{
		ErrorCode: bcerrors.ErrCodeOK,
		ErrorDesc: "",
	}
}

func (im *InvokeMgr) RegisterIntoContractDocker(respData, nameVersion string, stateDb *statedb.StateDB) bcerrors.BCError {
	client, err := im.pool(stateDb).GetClient()
	if err != nil {
		panic(err)
	}
	defer im.pool(stateDb).ReleaseClient(client)

	resp, err := client.Call("RegisterContractDocker", map[string]interface{}{"respData": respData, "nameVersion": nameVersion}, 10)
	if err != nil {
		panic(err)
	}

	var bcErr bcerrors.BCError
	err = jsoniter.Unmarshal([]byte(resp.(string)), &bcErr)
	if err != nil {
		panic(err)
	}

	return bcErr
}

func (im *InvokeMgr) Softforks_2_0_2_14654(orgID, contractName string) bool {
	client, err := im.pool(nil).GetClient()
	if err != nil {
		panic(err)
	}
	defer im.pool(nil).ReleaseClient(client)

	resp, err := client.Call("Softforks_2_0_2_14654", map[string]interface{}{"orgID": orgID, "contractName": contractName}, 10)
	if err != nil {
		panic(err)
	}

	var bForks bool
	err = jsoniter.Unmarshal([]byte(resp.(string)), &bForks)
	if err != nil {
		panic(err)
	}

	return bForks
}

func (im *InvokeMgr) GetInvokeItems(transID int64) *stubapi.InvokeParams {
	item, ok := im.invokeItems.Load(transID)
	if !ok {
		return nil
	}

	invokeItems := item.(*stubapi.InvokeParams)
	return invokeItems
}

func (im *InvokeMgr) pool(stateDB *statedb.StateDB) *socket.ConnectionPool {
	_, url, err := smcdocker.GetInstance().GetContractInvokeURL(0, 0, smcbuilder.ThirdPartyContract)
	if err != nil {
		panic(err)
	}

	if p == nil {
		var err error
		p, err = socket.NewConnectionPool(url, 2, im.logger)
		if err != nil {
			panic(err)
		}

		if stateDB != nil {
			initDocker(stateDB)
		}

	} else if p.SvrAddr != url {
		p.Close()
		p = nil
		var err error
		p, err = socket.NewConnectionPool(url, 2, im.logger)
		if err != nil {
			panic(err)
		}

		if stateDB != nil {
			initDocker(stateDB)
		}
	}

	return p
}

func initDocker(stateDB *statedb.StateDB) {

	allContract, err := stateDB.GetContractAddrList()
	if err != nil {
		panic(err)
	}

	client, err := p.GetClient()
	if err != nil {
		panic(err)
	}
	resp, err := client.Call("InitContractDocker", map[string]interface{}{"allContract": allContract}, 60)
	if err != nil {
		panic(err)
	}

	var bcErr bcerrors.BCError
	err = jsoniter.Unmarshal([]byte(resp.(string)), &bcErr)
	if err != nil {
		panic(err)
	}
}

func newInvokeParamsEx(items *stubapi.InvokeParams, transID int64) *stubapi.InvokeParamsEx {
	itemsEx := &stubapi.InvokeParamsEx{
		TransID:         transID,
		Sender:          items.Ctx.Sender.Addr,
		Owner:           items.Ctx.Owner.Addr,
		BlockHash:       items.Ctx.BlockHash,
		BlockHeader:     items.Ctx.BlockHeader,
		GasLimit:        items.Ctx.GasLimit,
		Note:            items.Ctx.Note,
		ContractAddress: items.Ctx.TxState.ContractAddress,
		Params:          items.Params,
	}
	if items.Ctx.Proposer != nil {
		itemsEx.Proposer = items.Ctx.Proposer.Addr
	}
	if items.Ctx.Rewarder != nil {
		itemsEx.Rewarder = items.Ctx.Rewarder.Addr
	}

	return itemsEx
}

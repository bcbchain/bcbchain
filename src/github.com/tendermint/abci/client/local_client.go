package abcicli

import (
	"sync"

	"github.com/tendermint/abci/types"
	cmn "github.com/tendermint/tmlibs/common"
)

var _ Client = (*localClient)(nil)

type localClient struct {
	cmn.BaseService
	types.Application
	Callback
}

func NewLocalClient(mtx *sync.Mutex, app types.Application) *localClient {
	cli := &localClient{
		Application: app,
	}
	cli.BaseService = *cmn.NewBaseService(nil, "localClient", cli)
	return cli
}

func (app *localClient) SetResponseCallback(cb Callback) {
	app.Callback = cb
}

// TODO: change types.Application to include Error()?
func (app *localClient) Error() error {
	return nil
}

func (app *localClient) FlushAsync() *ReqRes {
	// Do nothing
	return newLocalReqRes(types.ToRequestFlush(), nil)
}

func (app *localClient) EchoAsync(msg string) *ReqRes {
	return app.callback(
		types.ToRequestEcho(msg),
		types.ToResponseEcho(msg),
	)
}

func (app *localClient) InfoAsync(req types.RequestInfo) *ReqRes {
	res := app.Application.Info(req)
	return app.callback(
		types.ToRequestInfo(req),
		types.ToResponseInfo(res),
	)
}

func (app *localClient) SetOptionAsync(req types.RequestSetOption) *ReqRes {
	res := app.Application.SetOption(req)
	return app.callback(
		types.ToRequestSetOption(req),
		types.ToResponseSetOption(res),
	)
}

func (app *localClient) DeliverTxAsync(tx []byte) *ReqRes {
	res := app.Application.DeliverTx(tx)
	return app.callback(
		types.ToRequestDeliverTx(tx),
		types.ToResponseDeliverTx(res),
	)
}

func (app *localClient) CheckTxAsync(tx []byte) *ReqRes {
	res := app.Application.CheckTx(tx)
	return app.callback(
		types.ToRequestCheckTx(tx),
		types.ToResponseCheckTx(res),
	)
}

func (app *localClient) QueryAsync(req types.RequestQuery) *ReqRes {
	res := app.Application.Query(req)
	return app.callback(
		types.ToRequestQuery(req),
		types.ToResponseQuery(res),
	)
}

func (app *localClient) QueryExAsync(req types.RequestQueryEx) *ReqRes {
	res := app.Application.QueryEx(req)
	return app.callback(
		types.ToRequestQueryEx(req),
		types.ToResponseQueryEx(res),
	)
}

func (app *localClient) CommitAsync() *ReqRes {
	res := app.Application.Commit()
	return app.callback(
		types.ToRequestCommit(),
		types.ToResponseCommit(res),
	)
}

func (app *localClient) InitChainAsync(req types.RequestInitChain) *ReqRes {
	res := app.Application.InitChain(req)
	reqRes := app.callback(
		types.ToRequestInitChain(req),
		types.ToResponseInitChain(res),
	)
	return reqRes
}

func (app *localClient) BeginBlockAsync(req types.RequestBeginBlock) *ReqRes {
	res := app.Application.BeginBlock(req)
	return app.callback(
		types.ToRequestBeginBlock(req),
		types.ToResponseBeginBlock(res),
	)
}

func (app *localClient) EndBlockAsync(req types.RequestEndBlock) *ReqRes {
	res := app.Application.EndBlock(req)
	return app.callback(
		types.ToRequestEndBlock(req),
		types.ToResponseEndBlock(res),
	)
}

func (app *localClient) CleanDataAsync() *ReqRes {
	res := app.Application.CleanData()
	return app.callback(
		types.ToRequestCleanData(),
		types.ToResponseCleanData(res),
	)
}

//-------------------------------------------------------

func (app *localClient) FlushSync() error {
	return nil
}

func (app *localClient) EchoSync(msg string) (*types.ResponseEcho, error) {
	return &types.ResponseEcho{Message: msg}, nil
}

func (app *localClient) InfoSync(req types.RequestInfo) (*types.ResponseInfo, error) {
	res := app.Application.Info(req)
	return &res, nil
}

func (app *localClient) SetOptionSync(req types.RequestSetOption) (*types.ResponseSetOption, error) {
	res := app.Application.SetOption(req)
	return &res, nil
}

func (app *localClient) DeliverTxSync(tx []byte) (*types.ResponseDeliverTx, error) {
	res := app.Application.DeliverTx(tx)
	return &res, nil
}

func (app *localClient) CheckTxSync(tx []byte) (*types.ResponseCheckTx, error) {
	res := app.Application.CheckTx(tx)
	return &res, nil
}

func (app *localClient) QuerySync(req types.RequestQuery) (*types.ResponseQuery, error) {
	res := app.Application.Query(req)
	return &res, nil
}

func (app *localClient) QueryExSync(req types.RequestQueryEx) (*types.ResponseQueryEx, error) {
	res := app.Application.QueryEx(req)
	return &res, nil
}

func (app *localClient) CommitSync() (*types.ResponseCommit, error) {
	res := app.Application.Commit()
	return &res, nil
}

func (app *localClient) InitChainSync(req types.RequestInitChain) (*types.ResponseInitChain, error) {
	res := app.Application.InitChain(req)
	return &res, nil
}

func (app *localClient) BeginBlockSync(req types.RequestBeginBlock) (*types.ResponseBeginBlock, error) {
	res := app.Application.BeginBlock(req)
	return &res, nil
}

func (app *localClient) EndBlockSync(req types.RequestEndBlock) (*types.ResponseEndBlock, error) {
	res := app.Application.EndBlock(req)
	return &res, nil
}

func (app *localClient) CleanDataSync() (*types.ResponseCleanData, error) {
	res := app.Application.CleanData()
	return &res, nil
}

//-------------------------------------------------------

func (app *localClient) callback(req *types.Request, res *types.Response) *ReqRes {
	app.Callback(req, res)
	return newLocalReqRes(req, res)
}

func newLocalReqRes(req *types.Request, res *types.Response) *ReqRes {
	reqRes := NewReqRes(req)
	reqRes.Response = res
	reqRes.SetDone()
	return reqRes
}

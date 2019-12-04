package mock

import (
	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
	"github.com/tendermint/tendermint/version"
	cmn "github.com/tendermint/tmlibs/common"
)

// ABCIApp will send all abci related request to the named app,
// so you can test app behavior from a client without needing
// an entire tendermint node
type ABCIApp struct {
	App abci.Application
}

var (
	_ client.ABCIClient = ABCIApp{}
	_ client.ABCIClient = ABCIMock{}
	_ client.ABCIClient = (*ABCIRecorder)(nil)
)

func (a ABCIApp) ABCIInfo() (*ctypes.ResultABCIInfo, error) {
	return &ctypes.ResultABCIInfo{Response: a.App.Info(abci.RequestInfo{Version: version.Version})}, nil
}

func (a ABCIApp) ABCIQuery(path string, data cmn.HexBytes) (*ctypes.ResultABCIQuery, error) {
	return a.ABCIQueryWithOptions(path, data, client.DefaultABCIQueryOptions)
}

func (a ABCIApp) ABCIQueryWithOptions(path string, data cmn.HexBytes, opts client.ABCIQueryOptions) (*ctypes.ResultABCIQuery, error) {
	q := a.App.Query(abci.RequestQuery{Data: data, Path: path, Height: opts.Height, Prove: opts.Trusted})
	return &ctypes.ResultABCIQuery{Response: q}, nil
}

func (a ABCIApp) ABCIQueryEx(path string) (*ctypes.ResultABCIQueryEx, error) {
	return a.ABCIQueryExWithOptions(path)
}

func (a ABCIApp) ABCIQueryExWithOptions(path string) (*ctypes.ResultABCIQueryEx, error) {
	q := a.App.QueryEx(abci.RequestQueryEx{Path: path})
	return &ctypes.ResultABCIQueryEx{Response: q}, nil
}

func (a ABCIApp) BroadcastTxCommit(tx types.Tx) (*ctypes.ResultBroadcastTxCommit, error) {
	res := ctypes.ResultBroadcastTxCommit{}
	res.CheckTx = a.App.CheckTx(tx)
	if res.CheckTx.IsErr() {
		return &res, nil
	}
	res.DeliverTx = a.App.DeliverTx(tx)
	return &res, nil
}

func (a ABCIApp) BroadcastTxAsync(tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	c := a.App.CheckTx(tx)
	// and this gets written in a background thread...
	if !c.IsErr() {
		go func() { a.App.DeliverTx(tx) }() // nolint: errcheck
	}
	return &ctypes.ResultBroadcastTx{Code: c.Code, Data: []byte(c.Data), Log: c.Log, Hash: tx.Hash()}, nil
}

func (a ABCIApp) BroadcastTxSync(tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	c := a.App.CheckTx(tx)
	// and this gets written in a background thread...
	if !c.IsErr() {
		go func() { a.App.DeliverTx(tx) }() // nolint: errcheck
	}
	return &ctypes.ResultBroadcastTx{Code: c.Code, Data: []byte(c.Data), Log: c.Log, Hash: tx.Hash()}, nil
}

// ABCIMock will send all abci related request to the named app,
// so you can test app behavior from a client without needing
// an entire tendermint node
type ABCIMock struct {
	Info            Call
	Query           Call
	QueryEx         Call
	BroadcastCommit Call
	Broadcast       Call
}

func (m ABCIMock) ABCIInfo() (*ctypes.ResultABCIInfo, error) {
	res, err := m.Info.GetResponse(nil)
	if err != nil {
		return nil, err
	}
	return &ctypes.ResultABCIInfo{Response: res.(abci.ResponseInfo)}, nil
}

func (m ABCIMock) ABCIQuery(path string, data cmn.HexBytes) (*ctypes.ResultABCIQuery, error) {
	return m.ABCIQueryWithOptions(path, data, client.DefaultABCIQueryOptions)
}

func (m ABCIMock) ABCIQueryWithOptions(path string, data cmn.HexBytes, opts client.ABCIQueryOptions) (*ctypes.ResultABCIQuery, error) {
	res, err := m.Query.GetResponse(QueryArgs{path, data, opts.Height, opts.Trusted})
	if err != nil {
		return nil, err
	}
	resQuery := res.(abci.ResponseQuery)
	return &ctypes.ResultABCIQuery{Response: resQuery}, nil
}

func (m ABCIMock) ABCIQueryEx(path string) (*ctypes.ResultABCIQueryEx, error) {
	return m.ABCIQueryExWithOptions(path)
}

func (m ABCIMock) ABCIQueryExWithOptions(path string) (*ctypes.ResultABCIQueryEx, error) {
	res, err := m.QueryEx.GetResponse(QueryExArgs{path})
	if err != nil {
		return nil, err
	}
	resQuery := res.(abci.ResponseQueryEx)
	return &ctypes.ResultABCIQueryEx{Response: resQuery}, nil
}

func (m ABCIMock) BroadcastTxCommit(tx types.Tx) (*ctypes.ResultBroadcastTxCommit, error) {
	res, err := m.BroadcastCommit.GetResponse(tx)
	if err != nil {
		return nil, err
	}
	return res.(*ctypes.ResultBroadcastTxCommit), nil
}

func (m ABCIMock) BroadcastTxAsync(tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	res, err := m.Broadcast.GetResponse(tx)
	if err != nil {
		return nil, err
	}
	return res.(*ctypes.ResultBroadcastTx), nil
}

func (m ABCIMock) BroadcastTxSync(tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	res, err := m.Broadcast.GetResponse(tx)
	if err != nil {
		return nil, err
	}
	return res.(*ctypes.ResultBroadcastTx), nil
}

// ABCIRecorder can wrap another type (ABCIApp, ABCIMock, or Client)
// and record all ABCI related calls.
type ABCIRecorder struct {
	Client client.ABCIClient
	Calls  []Call
}

func NewABCIRecorder(client client.ABCIClient) *ABCIRecorder {
	return &ABCIRecorder{
		Client: client,
		Calls:  []Call{},
	}
}

type QueryArgs struct {
	Path    string
	Data    cmn.HexBytes
	Height  int64
	Trusted bool
}

type QueryExArgs struct {
	Path string
}

func (r *ABCIRecorder) addCall(call Call) {
	r.Calls = append(r.Calls, call)
}

func (r *ABCIRecorder) ABCIInfo() (*ctypes.ResultABCIInfo, error) {
	res, err := r.Client.ABCIInfo()
	r.addCall(Call{
		Name:     "abci_info",
		Response: res,
		Error:    err,
	})
	return res, err
}

func (r *ABCIRecorder) ABCIQuery(path string, data cmn.HexBytes) (*ctypes.ResultABCIQuery, error) {
	return r.ABCIQueryWithOptions(path, data, client.DefaultABCIQueryOptions)
}

func (r *ABCIRecorder) ABCIQueryWithOptions(path string, data cmn.HexBytes, opts client.ABCIQueryOptions) (*ctypes.ResultABCIQuery, error) {
	res, err := r.Client.ABCIQueryWithOptions(path, data, opts)
	r.addCall(Call{
		Name:     "abci_query",
		Args:     QueryArgs{path, data, opts.Height, opts.Trusted},
		Response: res,
		Error:    err,
	})
	return res, err
}

func (r *ABCIRecorder) ABCIQueryEx(path string) (*ctypes.ResultABCIQueryEx, error) {
	return r.ABCIQueryExWithOptions(path)
}

func (r *ABCIRecorder) ABCIQueryExWithOptions(path string) (*ctypes.ResultABCIQueryEx, error) {
	res, err := r.Client.ABCIQueryExWithOptions(path)
	r.addCall(Call{
		Name:     "abci_query_ex",
		Args:     QueryExArgs{path},
		Response: res,
		Error:    err,
	})
	return res, err
}

func (r *ABCIRecorder) BroadcastTxCommit(tx types.Tx) (*ctypes.ResultBroadcastTxCommit, error) {
	res, err := r.Client.BroadcastTxCommit(tx)
	r.addCall(Call{
		Name:     "broadcast_tx_commit",
		Args:     tx,
		Response: res,
		Error:    err,
	})
	return res, err
}

func (r *ABCIRecorder) BroadcastTxAsync(tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	res, err := r.Client.BroadcastTxAsync(tx)
	r.addCall(Call{
		Name:     "broadcast_tx_async",
		Args:     tx,
		Response: res,
		Error:    err,
	})
	return res, err
}

func (r *ABCIRecorder) BroadcastTxSync(tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	res, err := r.Client.BroadcastTxSync(tx)
	r.addCall(Call{
		Name:     "broadcast_tx_sync",
		Args:     tx,
		Response: res,
		Error:    err,
	})
	return res, err
}

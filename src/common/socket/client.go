package socket

import (
	"blockchain/types"
	"bufio"
	"container/list"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/json-iterator/go"
	"github.com/tendermint/tmlibs/log"
)

// Client client information about socket
type Client struct {
	// request send queue
	reqSent  *list.List
	queueMtx sync.Mutex

	addr             string
	conn             net.Conn
	w                *bufio.Writer
	r                *bufio.Reader
	disableKeepAlive bool

	counter uint64
	mtx     sync.Mutex
	logger  log.Logger

	isRunning bool

	// close callback function
	closeCB func(*Client)
}

// NewClient newClient to create socket client object and connect to server
func NewClient(addr string, disableKeepAlive bool, logger log.Logger) (cli *Client, err error) {

	logger.Debug(fmt.Sprintf("New connect to %s, disableKeepAlive=%t", addr, disableKeepAlive))

	cli = &Client{
		reqSent:          list.New(),
		addr:             addr,
		disableKeepAlive: disableKeepAlive,
		counter:          0,
		logger:           logger}

	err = cli.connect()
	if err != nil {
		return
	}

	cli.isRunning = true
	go cli.recvResponseRoutine()

	return
}

// SetCloseCB set close callback function
func (cli *Client) SetCloseCB(cb func(*Client)) {
	cli.closeCB = cb
}

// Call call service with method and data
func (cli *Client) Call(method string, data map[string]interface{}, timeout time.Duration) (value interface{}, err error) {
	if !cli.isRunning {
		return nil, errors.New("client is invalid")
	}

	req := Request{Method: method, Data: data, Index: cli.index()}
	if cli.disableKeepAlive {
		defer cli.conn.Close()
	}
	cli.logger.Debug(fmt.Sprintf("to %s have a new request, method=%s, index=%d", cli.addr, method, req.Index))

	// wait response
	respChan := make(chan *Response, 1)
	closeChan := make(chan error, 1)
	cli.sentReq(req.Index, respChan, closeChan)
	defer cli.removeReq(req.Index)

	// send request
	cli.mtx.Lock()
	err = writeMessage(req, cli.w)
	if err != nil {
		cli.mtx.Unlock()
		cli.logger.Error(fmt.Sprintf("index=%d request error=%s", req.Index, err.Error()))
		return
	}
	err = cli.w.Flush()
	if err != nil {
		cli.mtx.Unlock()
		cli.logger.Error(fmt.Sprintf("index=%d request error=%s", req.Index, err.Error()))
		return
	}
	cli.mtx.Unlock()

	cli.logger.Debug(fmt.Sprintf("index=%d request wait response, timeout=%d", req.Index, timeout))
	select {
	case <-time.After(timeout * time.Second):
		return nil, fmt.Errorf("recv time out, index=%d", req.Index)
	case resp := <-respChan:
		//resp := <-respChan
		if resp.Code == types.CodeOK {
			return resp.Result.Data, nil
		} else {
			return nil, errors.New(resp.Log)
		}
	case err := <-closeChan:
		if err == io.EOF {
			return nil, errors.New("connection closed")
		} else {
			return nil, fmt.Errorf("connection error=%v", err)
		}
	}
}

// Close close connection
func (cli *Client) Close() (err error) {
	err = cli.conn.Close()
	if err != nil {
		return
	}
	cli.conn = nil

	return
}

func (cli *Client) connect() (err error) {
	proto, address := "tcp", cli.addr
	parts := strings.SplitN(cli.addr, "://", 2)
	if len(parts) == 2 {
		proto, address = parts[0], parts[1]
	}

	var keepAlive time.Duration
	if !cli.disableKeepAlive {
		keepAlive = 5 * time.Second
	}

	dialer := net.Dialer{Timeout: 60 * time.Second, KeepAlive: keepAlive}
	cli.conn, err = dialer.Dial(proto, address)
	if err != nil {
		return err
	}
	cli.w = bufio.NewWriter(cli.conn)
	cli.r = bufio.NewReader(cli.conn)

	return
}

func (cli *Client) recvResponseRoutine() {

	// if disableKeepAlive is true,then loop one time
	recvCount := 1
	if !cli.disableKeepAlive {
		recvCount = -1
	}
	for {
		if recvCount == 0 {
			break
		}

		value, err := readMessage(cli.r)
		if err != nil {
			cli.isRunning = false
			cli.logger.Fatal("readMessage error", "error", err)
			cli.sendCloseChan(err)
			return
		}

		var resp Response
		err = jsoniter.Unmarshal(value, &resp)
		if err != nil {
			cli.isRunning = false
			cli.logger.Fatal(fmt.Sprintf("value=%v cannot unmarshal to response", value), "error", err)
			cli.sendCloseChan(err)
			return
		}

		go cli.didRecvResponse(resp)
		if recvCount > 0 {
			recvCount--
		}
	}
}

func (cli *Client) didRecvResponse(resp Response) {
	next := cli.reqSent.Front()
	for next != nil {
		if next.Value.(ReqResp).Index == resp.Result.Index {
			break
		}

		next = next.Next()
	}

	if next != nil {
		next.Value.(ReqResp).RespChan <- &resp
	} else {
		//cli.logger.Error("didRecvResponse", "response index", resp.Result.Index, "reqSent", cli.reqSent)
		time.Sleep(time.Second)
	}
}

func (cli *Client) sendCloseChan(err error) {
	//cli.queueMtx.Lock()
	//defer cli.queueMtx.Unlock()

	next := cli.reqSent.Front()
	for next != nil {
		next.Value.(ReqResp).CloseChan <- err

		next = next.Next()
	}

	// invoke close callback function
	if cli.closeCB != nil {
		cli.closeCB(cli)
	}
}

func (cli *Client) sentReq(index uint64, respChan chan *Response, closeChan chan error) {
	cli.queueMtx.Lock()
	defer cli.queueMtx.Unlock()

	cli.reqSent.PushBack(ReqResp{Index: index, RespChan: respChan, CloseChan: closeChan})
}

func (cli *Client) removeReq(index uint64) {
	//cli.queueMtx.Lock()
	//defer cli.queueMtx.Unlock()

	next := cli.reqSent.Front()
	for next != nil {
		if next.Value.(ReqResp).Index == index {
			close(next.Value.(ReqResp).CloseChan)
			close(next.Value.(ReqResp).RespChan)
			cli.queueMtx.Lock()
			cli.reqSent.Remove(next)
			cli.queueMtx.Unlock()
			break
		}

		next = next.Next()
	}
}

func (cli *Client) index() uint64 {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	cli.counter += 1
	return cli.counter % math.MaxUint64
}

package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/tendermint/abci/types"
	cmn "github.com/tendermint/tmlibs/common"
)

// var maxNumberConnections = 2

type SocketServer struct {
	cmn.BaseService

	proto    string
	addr     string
	listener net.Listener

	connsMtx   sync.Mutex
	conns      map[int]net.Conn
	nextConnID int

	// appMtx sync.Mutex
	app types.Application
}

func NewSocketServer(protoAddr string, app types.Application) cmn.Service {
	proto, addr := cmn.ProtocolAndAddress(protoAddr)
	s := &SocketServer{
		proto:    proto,
		addr:     addr,
		listener: nil,
		app:      app,
		conns:    make(map[int]net.Conn),
	}
	s.BaseService = *cmn.NewBaseService(nil, "ABCIServer", s)
	return s
}

func (s *SocketServer) OnStart() error {
	if err := s.BaseService.OnStart(); err != nil {
		return err
	}
	ln, err := net.Listen(s.proto, s.addr)
	if err != nil {
		return err
	}
	s.listener = ln
	go s.acceptConnectionsRoutine()
	return nil
}

func (s *SocketServer) OnStop() {
	s.BaseService.OnStop()
	if err := s.listener.Close(); err != nil {
		s.Logger.Error("Error closing listener", "err", err)
	}

	s.connsMtx.Lock()
	defer s.connsMtx.Unlock()
	for id, conn := range s.conns {
		delete(s.conns, id)
		if err := conn.Close(); err != nil {
			s.Logger.Error("Error closing connection", "id", id, "conn", conn, "err", err)
		}
	}
}

func (s *SocketServer) addConn(conn net.Conn) int {
	s.connsMtx.Lock()
	defer s.connsMtx.Unlock()

	connID := s.nextConnID
	s.nextConnID++
	s.conns[connID] = conn

	return connID
}

// deletes conn even if close errs
func (s *SocketServer) rmConn(connID int) error {
	s.connsMtx.Lock()
	defer s.connsMtx.Unlock()

	conn, ok := s.conns[connID]
	if !ok {
		return fmt.Errorf("Connection %d does not exist", connID)
	}

	delete(s.conns, connID)
	return conn.Close()
}

func (s *SocketServer) acceptConnectionsRoutine() {
	var remoteIp string
	var connID int
	dataChan := make(chan bool)
	for {
		// Accept a connection
		s.Logger.Info("Waiting for new connection...")
		conn, err := s.listener.Accept()
		if err != nil {
			if !s.IsRunning() {
				return // Ignore error from listener closing.
			}
			s.Logger.Error("Failed to accept connection: " + err.Error())
			continue
		}

		addr := conn.RemoteAddr().String()
		currentRemoteIp := strings.Split(addr, ":")[0]

		if remoteIp == "" {
			//起go协程，设置select，select一个case接收channel数据，接收到数据后重置计时器，一个case接收超时，超时后重启bcchain
			go s.checkReqTimeOutInfo(dataChan)
			remoteIp = currentRemoteIp
		} else {
			if currentRemoteIp != remoteIp {
				//拒绝连接
				s.Logger.Error("Connection refused because client ip is invalid", "client ip", currentRemoteIp)
				conn.Close()
				continue
			}
		}

		//此处限制，当链接数大于3时，阻止进行连接
		connAmount := len(s.conns)
		if connAmount == 3 {
			s.Logger.Error("There are four connections from same IP.")
			s.killBcchain()
			return
		}

		connID = s.addConn(conn)
		s.Logger.Info("Accepted a new connection", "connID", connID)

		closeConn := make(chan error, 5)              // Push to signal connection closed
		responses := make(chan *types.Response, 1000) // A channel to buffer responses

		// Read requests from conn and deal with them
		go s.handleRequests(closeConn, conn, responses, dataChan)
		// Pull responses from 'responses' and write them to conn.
		go s.handleResponses(closeConn, conn, responses)

		// Wait until signal to close connection
		go s.waitForClose(closeConn, connID)
	}
}

func (s *SocketServer) checkReqTimeOutInfo(dataChan chan bool) {
	//var timer *time.Timer
	timer := time.NewTimer(600 * time.Second)
	for {
		select {
		case <-dataChan: //Timer Reset
			timer.Reset(600 * time.Second)
			s.Logger.Debug("Timer Reset")
		case <-timer.C:
			s.Logger.Warn("no request 600 Seconds, chain is committing suicide")
			s.Logger.Flush()
			s.killBcchain()
		}
	}
}

func (s *SocketServer) waitForClose(closeConn chan error, connID int) {
	err := <-closeConn
	if err == io.EOF {
		s.Logger.Error("Connection was closed by client", "connID", connID)
	} else if err != nil {
		s.Logger.Error("Connection error", "error", err)
	} else {
		// never happens
		s.Logger.Error("Connection was closed.")
	}

	// Close the connection
	if err := s.rmConn(connID); err != nil {
		s.Logger.Error("Error in closing connection", "error", err)
	}
	//杀死bcchain进程
	s.killBcchain()
}

func (s *SocketServer) killBcchain() {
	pid := os.Getpid()
	pstat, err := os.FindProcess(pid)
	if err != nil {
		panic(err.Error())
	}
	err = pstat.Signal(os.Kill) //kill process
	if err != nil {
		panic(err.Error())
	}
}

// Read requests from conn and deal with them
func (s *SocketServer) handleRequests(closeConn chan error, conn net.Conn, responses chan<- *types.Response, dataChan chan bool) {
	var bufReader = bufio.NewReader(conn)
	for {
		var req = &types.Request{}
		err := types.ReadMessage(bufReader, req)
		if err != nil {
			if err == io.EOF {
				closeConn <- err
			} else {
				closeConn <- fmt.Errorf("Error reading message: %v", err.Error())
			}
			return
		}
		dataChan <- true
		s.handleRequest(conn, req, responses)
	}
}

func (s *SocketServer) handleRequest(conn net.Conn, req *types.Request, responses chan<- *types.Response) {

	switch r := req.Value.(type) {
	case *types.Request_Echo:
		responses <- types.ToResponseEcho(r.Echo.Message)
	case *types.Request_Flush:
		responses <- types.ToResponseFlush()
	case *types.Request_Info:
		addr := conn.RemoteAddr().String()
		spl := strings.Split(addr, ":")
		r.Info.Host = spl[0]
		res := s.app.Info(*r.Info)
		responses <- types.ToResponseInfo(res)
	case *types.Request_SetOption:
		res := s.app.SetOption(*r.SetOption)
		responses <- types.ToResponseSetOption(res)
	case *types.Request_DeliverTx:
		res := s.app.DeliverTx(r.DeliverTx.Tx)
		responses <- types.ToResponseDeliverTx(res)
	case *types.Request_CheckTx:
		res := s.app.CheckTx(r.CheckTx.Tx)
		responses <- types.ToResponseCheckTx(res)
	case *types.Request_Commit:
		res := s.app.Commit()
		responses <- types.ToResponseCommit(res)
	case *types.Request_Query:
		res := s.app.Query(*r.Query)
		responses <- types.ToResponseQuery(res)
	case *types.Request_QueryEx:
		res := s.app.QueryEx(*r.QueryEx)
		responses <- types.ToResponseQueryEx(res)
	case *types.Request_InitChain:
		res := s.app.InitChain(*r.InitChain)
		responses <- types.ToResponseInitChain(res)
	case *types.Request_BeginBlock:
		res := s.app.BeginBlock(*r.BeginBlock)
		responses <- types.ToResponseBeginBlock(res)
	case *types.Request_EndBlock:
		res := s.app.EndBlock(*r.EndBlock)
		responses <- types.ToResponseEndBlock(res)
	case *types.Request_CleanData:
		res := s.app.CleanData()
		responses <- types.ToResponseCleanData(res)
	default:
		responses <- types.ToResponseException("Unknown request")
	}
}

// Pull responses from 'responses' and write them to conn.
func (s *SocketServer) handleResponses(closeConn chan error, conn net.Conn, responses <-chan *types.Response) {
	//var count int
	var bufWriter = bufio.NewWriter(conn)
	for {
		var res = <-responses
		err := types.WriteMessage(res, bufWriter)
		if err != nil {
			closeConn <- fmt.Errorf("Error writing message: %v", err.Error())
			return
		}
		if _, ok := res.Value.(*types.Response_Flush); ok {
			err = bufWriter.Flush()
			if err != nil {
				closeConn <- fmt.Errorf("Error flushing write buffer: %v", err.Error())
				return
			}
		}
		//count++
	}
}

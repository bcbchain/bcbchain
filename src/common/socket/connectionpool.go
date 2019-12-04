package socket

import (
	"container/list"
	"errors"
	"fmt"
	"github.com/tendermint/tmlibs/log"
	"sync"
	"time"
)

// ConnectionPool pool for connections
type ConnectionPool struct {
	SvrAddr string // 服务器地址

	busyConnS map[*Client]struct{} // 已被使用的连接
	idleConnS *list.List           // 空闲连接
	mtx       sync.Mutex

	maxCap int // 连接总数
	curCap int // 当前数

	logger log.Logger
}

// NewConnectionPool create pool for connections, init with cap, increment with incr
func NewConnectionPool(svrAddr string, curCap int, logger log.Logger) (pool *ConnectionPool, err error) {

	if curCap < 0 {
		curCap = 0
	}
	if curCap > 10 {
		curCap = 10
	}

	pool = &ConnectionPool{
		SvrAddr: svrAddr,
		maxCap:  10,

		idleConnS: list.New(),
		busyConnS: make(map[*Client]struct{}),

		logger: logger,
	}

	// init capacity
	for pool.idleConnS.Len() < curCap {
		err = pool.incrCapacity()
		if err != nil {
			return
		}
	}

	return
}

// Close close all connection
func (pool *ConnectionPool) Close() {
	pool.mtx.Lock()
	defer pool.mtx.Unlock()

	for len(pool.busyConnS) != 0 {
		time.Sleep(100 * time.Millisecond)
	}

	next := pool.idleConnS.Front()
	for next != nil {
		next.Value.(*Client).Close()

		next = next.Next()
	}
}

// getClient return an idle client
func (pool *ConnectionPool) GetClient() (cli *Client, err error) {

	pool.mtx.Lock()
	defer pool.mtx.Unlock()
	if pool.idleConnS.Len() <= 0 {
		err = pool.incrCapacity()
		if err != nil {
			return
		}
	}

	elm := pool.idleConnS.Front()
	cli = elm.Value.(*Client)

	pool.idleConnS.Remove(elm)
	pool.busyConnS[cli] = struct{}{}

	return
}

// releaseClient make client idle when it not used by invoker
func (pool *ConnectionPool) ReleaseClient(cli *Client) {
	pool.mtx.Lock()
	defer pool.mtx.Unlock()

	pool.idleConnS.PushBack(cli)
	delete(pool.busyConnS, cli)
}

// addClient new an idle client and then add it to pool
func (pool *ConnectionPool) addClient() (err error) {

	var cli *Client
	cli, err = NewClient(pool.SvrAddr, false, pool.logger)
	if err != nil {
		cli.logger.Fatal("new client err", "error", err)
		return
	}
	cli.SetCloseCB(pool.closeCB)

	pool.idleConnS.PushBack(cli)
	pool.curCap++

	return
}

// incrCapacity incremental capacity of pool with incr
func (pool *ConnectionPool) incrCapacity() (err error) {

	if pool.curCap >= pool.maxCap {
		return errors.New(fmt.Sprintf("no more connection, curCap=%d, maxCap=%d", pool.curCap, pool.maxCap))
	}

	err = pool.addClient()
	if err != nil {
		return
	}

	return
}

// closeCB remove client when it closed
func (pool *ConnectionPool) closeCB(cli *Client) {
	pool.mtx.Lock()
	defer pool.mtx.Unlock()

	delete(pool.busyConnS, cli)

	next := pool.idleConnS.Front()
	for next != nil {
		if next.Value.(*Client) == cli {
			pool.idleConnS.Remove(next)
			break
		}

		next = next.Next()
	}
	pool.curCap--
}

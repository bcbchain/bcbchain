/*bcchain v2.0重大问题和修订方案1.1解决方案3 客户端测试连接数*/
package main

import (
	"fmt"
	cli "github.com/tendermint/abci/client"
	"sync"
)
const (
	LINUX_C2  ="tcp://192.168.80.150:8080"
	WINDOWS_C2 = "tcp://192.168.1.177:8080"
)

func client2(w *sync.WaitGroup,i int)  {
	c:=cli.NewSocketClient(LINUX_C2,true)
	err:=c.OnStart()
	if err!=nil{
		fmt.Println("err:",err.Error())
	}
	fmt.Printf("start client :%v\n",i)

	w.Done()
}

func main()  {
	a:=make(chan bool)
	w:= &sync.WaitGroup{}
	for i:=0;i<1;i++{
		w.Add(1)
		go func(i int) {
			//fmt.Println(string(i+1))
			client2(w,i)
		}(i)
	}
	w.Wait()
	<-a
}

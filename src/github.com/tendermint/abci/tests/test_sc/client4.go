/*bcchain v2.0重大问题和修订方案1.1解决方案3 客户端测试正常数据*/
package main

import (
	"fmt"
	"github.com/tendermint/abci/types"
	"github.com/tendermint/tendermint/proxy"
)
const (
	LINUX_C4  ="tcp://192.168.80.150:46658"
	WINDOWS_C4 = "tcp://192.168.1.177:8080"
)

func main()  {
	a:=make(chan bool)
	clientCreator := proxy.NewRemoteClientCreator(LINUX_C4,"socket",true)

	cli,err:=clientCreator.NewABCIClient()
	if err!=nil{
		fmt.Println("err",err.Error())
	}
	err=cli.OnStart()
	if err!=nil{
		fmt.Println("serr",err.Error())
	}
	app := proxy.NewAppConnQuery(cli)
	//测试正常数据
	var reqQuery types.RequestQuery
	reqQuery.Path="/genesis/chainid"
	resp,err:=app.QuerySync(reqQuery)
	if err!=nil{
		fmt.Println("serr",err.Error())
	}
	fmt.Println(resp.Code)
	<-a
}

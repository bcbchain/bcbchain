package gls

import jgls "github.com/jtolds/gls"

var (
	Mgr    = jgls.NewContextManager()
	SDKKey = jgls.GenSym()
)

type Values = jgls.Values

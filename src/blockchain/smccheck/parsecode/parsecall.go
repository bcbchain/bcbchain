package parsecode

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// Callee 被調用的方法
type Callee struct {
	Select     []string
	LastImport Import
}

var (
	lastImport       = Import{Name: "", Path: "blockchain/smcsdk/sdk"}
	getTransferToMe1 = []string{"GetTransferToMe", "Message", "ISmartContract"}
	getTransferToMe2 = []string{"GetTransferToMe", "Message", "GetSdk"}
	getTransferToMe3 = []string{"GetTransferToMe", "IMessage"}

	// ITx interface about variable
	txLevel = [][]string{
		{"Tx", "ISmartContract"},
		{"Tx", "GetSdk"},
		{"ITx"},
	}
	txInters = []string{
		"Note",
		"GasLimit",
		"GasLeft",
		"Signer",
	}

	// IMessage interface about variable
	msgLevel = [][]string{
		{"Message", "ISmartContract"},
		{"Message", "GetSdk"},
		{"IMessage"},
	}
	msgInters = []string{
		"MethodID",
		"Items",
		"GasPrice",
		"Sender",
		"Payer",
		"Origins",
		"InputReceipts",
		"GetTransferToMe",
	}

	// IMessage interface about variable
	conLevel = [][]string{
		{"Contract", "Message", "ISmartContract"},
		{"Contract", "Message", "GetSdk"},
		{"Contract", "IMessage"},
		{"IContract"},
	}
	conInters = []string{
		"Address",
		"Account",
		"Owner",
		"Name",
		"Version",
		"CodeHash",
		"EffectHeight",
		"LoseHeight",
		"KeyPrefix",
		"Methods",
		"Interfaces",
		"Mine",
		"Token",
		"OrgID",
		"SetOwner",
	}

	txAndMsgCallee = make([]Callee, 0)
	transferCallee = make([]Callee, 0)
	variableCallee = make(map[string]struct{})
)

func (v *visitor) initTxAndMsgCallee() {
	// GetTransferToMe
	transferCallee = []Callee{
		{Select: getTransferToMe1},
		{Select: getTransferToMe2},
		{Select: getTransferToMe3},
	}

	// IMessage/ITx
	for _, item := range txInters {
		txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{item, txLevel[0][0], txLevel[0][1]}})
		txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{item, txLevel[1][0], txLevel[1][1]}})
		txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{item, txLevel[2][0]}})
	}
	//txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{txLevel[0][0], txLevel[0][1]}})
	//txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{txLevel[1][0], txLevel[1][1]}})

	for _, item := range msgInters {
		txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{item, msgLevel[0][0], msgLevel[0][1]}})
		txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{item, msgLevel[1][0], msgLevel[1][1]}})
		txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{item, msgLevel[2][0]}})
	}
	//txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{msgLevel[0][0], msgLevel[0][1]}})
	//txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{msgLevel[1][0], msgLevel[1][1]}})

	for _, item := range conInters {
		txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{item, conLevel[0][0], conLevel[0][1], conLevel[0][2]}})
		txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{item, conLevel[1][0], conLevel[1][1], conLevel[1][2]}})
		txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{item, conLevel[2][0], conLevel[2][1]}})
		txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{item, conLevel[3][0]}})
	}
	//txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{conLevel[0][0], conLevel[0][1], conLevel[0][2]}})
	//txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{conLevel[1][0], conLevel[1][1], conLevel[1][2]}})
	//txAndMsgCallee = append(txAndMsgCallee, Callee{Select: []string{conLevel[2][0], conLevel[2][1]}})
}

func (v *visitor) parseCall(calleeList []Callee, fileMap map[string]*ast.File) {
	for _, f := range fileMap {
		ast.Inspect(f, func(node ast.Node) bool {
			switch node.(type) {
			case *ast.FuncDecl:
				fun := node.(*ast.FuncDecl)
				if exists, _, _ := v.isCalling(calleeList, fun); exists {
					newCallee := v.parseFunction(fun)
					for idx, contractFunction := range v.res.Functions {
						if newCallee.Name == contractFunction.Name &&
							newCallee.Receiver.FieldType == contractFunction.Receiver.FieldType {
							v.res.Functions[idx].GetTransferToMe = true
							return false
						}
					}
					caller := v.parseCaller(fun)
					v.parseCall(caller, fileMap)
				}
			}
			return true
		})
	}
}

func (v *visitor) parseCallEx(calleeList []Callee, fileMap map[string]*ast.File) {
	for _, f := range fileMap {
		ast.Inspect(f, func(node ast.Node) bool {
			switch node.(type) {
			case *ast.FuncDecl:
				fun := node.(*ast.FuncDecl)
				if exists, pos, _ := v.isCalling(calleeList, fun); exists {
					newCallee := v.parseFunction(fun)
					if newCallee.Name != v.res.InitChain.Name &&
						newCallee.Name != v.res.UpdateChain.Name &&
						newCallee.Name != v.res.Mine.Name {
						return true
					}

					if newCallee.Name == v.res.InitChain.Name &&
						newCallee.Receiver.FieldType == v.res.InitChain.Receiver.FieldType {
						v.reportErr("ITx/IMessage interface is forbidden in InitChain", pos)
						return false
					}
					if newCallee.Name == v.res.UpdateChain.Name &&
						newCallee.Receiver.FieldType == v.res.UpdateChain.Receiver.FieldType {
						v.reportErr("ITx/IMessage interface is forbidden in UpdateChain", pos)
						return false
					}
					if newCallee.Name == v.res.Mine.Name &&
						newCallee.Receiver.FieldType == v.res.Mine.Receiver.FieldType {
						v.reportErr("ITx/IMessage interface is forbidden in Mine", pos)
						return false
					}

					caller := v.parseCaller(fun)
					v.parseCallEx(caller, fileMap)
				}
			}
			return true
		})
	}
}

func (v *visitor) parseEmitCall(fileMap map[string]*ast.File) {
	for _, f := range fileMap {
		ast.Inspect(f, func(node ast.Node) bool {
			switch node.(type) {
			case *ast.FuncDecl:
				fun := node.(*ast.FuncDecl)
				v.checkCallerExp(fun)
			}
			return true
		})
	}
}

func (v *visitor) parseCaller(d *ast.FuncDecl) []Callee {
	names := make([]string, 0)
	names = append(names, d.Name.Name)
	if d.Recv != nil && len(d.Recv.List) > 0 {
		receiverType := d.Recv.List[0].Type
		typ := ExpandTypeNoStar(Field{FieldType: receiverType})
		names = append(names, typ)
	}

	return []Callee{{Select: names}}
}

// nolint cyclomatic
func (v *visitor) isCalling(calleeList []Callee, d *ast.FuncDecl) (exists bool, pos token.Pos, node ast.Node) {
	ast.Inspect(d, func(n ast.Node) bool {
		if _, ok0 := n.(*ast.CallExpr); !ok0 {
			return true
		}

		target := n
		for _, callee := range calleeList {
			for _, sel := range callee.Select {
				if call, ok1 := target.(*ast.CallExpr); ok1 {
					if id, okID := call.Fun.(*ast.Ident); okID {
						if id.Name == sel {
							exists = true
							pos = target.Pos()
							node = target
							return false
						}
					} else if fun, ok2 := call.Fun.(*ast.SelectorExpr); !ok2 || (fun.Sel.Name != sel) {
						return true
					} else {
						target = fun.X
					}
				} else if id, ok3 := target.(*ast.Ident); ok3 {
					if assign, ok4 := id.Obj.Decl.(*ast.AssignStmt); ok4 {
						for i := 0; i < len(assign.Lhs); i++ {
							aid, ok5 := assign.Lhs[i].(*ast.Ident)
							if !ok5 {
								return true
							}
							if id.Name == aid.Name {
								if assignF, okf := assign.Rhs[i].(*ast.CallExpr); okf {
									if assignSel, okSel := assignF.Fun.(*ast.SelectorExpr); okSel {
										target = assignSel.X
									}
								}
							}
						}
					} else if field, ok6 := id.Obj.Decl.(*ast.Field); ok6 {
						if sel2, ok7 := field.Type.(*ast.SelectorExpr); ok7 {
							if sel != sel2.Sel.Name {
								return true
							}
							target = sel2.X
						} else if sel3, okSel3 := field.Type.(*ast.StarExpr); okSel3 {
							if id, okID := sel3.X.(*ast.Ident); okID && id.Name == sel {
								exists = true
								pos = target.Pos()
								node = target
								return false
							}
						} else if id, okID := field.Type.(*ast.Ident); okID && id.Name == sel {
							exists = true
							pos = target.Pos()
							node = target
							return false
						}
					} else if val, okVal := id.Obj.Decl.(*ast.ValueSpec); okVal {
						if idt, okt := val.Type.(*ast.Ident); okt && idt.Name == sel {
							exists = true
							pos = target.Pos()
							node = target
							return false
						}
					}
				} else if sel2, okT := target.(*ast.SelectorExpr); okT {
					if id, okID := sel2.X.(*ast.Ident); okID {
						if d.Recv != nil && id.Name != d.Recv.List[0].Names[0].Name {
							return true
						}
						target = sel2.Sel
					}
				}
			}

			pkg, ok8 := target.(*ast.Ident)
			if !ok8 {
				return true
			}
			paths := strings.Split(lastImport.Path, "/")
			pkgName := paths[len(paths)-1]
			if pkg.Name == callee.LastImport.Name || pkg.Name == pkgName || pkg.Name == d.Recv.List[0].Names[0].Name {
				exists = true
				if pos == 0 {
					pos = target.Pos()
				}
				return false
			}
		}

		return true
	})

	return
}

// nolint cyclomatic
func (v *visitor) checkCallerExp(d *ast.FuncDecl) {
	exp := ""

	ast.Inspect(d, func(n ast.Node) bool {
		if _, ok0 := n.(*ast.CallExpr); !ok0 {
			return true
		} else {
			target := n.(*ast.CallExpr)
			if caller, ok := target.Fun.(*ast.SelectorExpr); ok {
				if len(target.Args) == 1 {
					if arg, ok := target.Args[0].(*ast.CompositeLit); ok {
						if argType, ok := arg.Type.(*ast.SelectorExpr); ok {
							x := argType.X.(*ast.Ident)
							exp += caller.Sel.Name + "(" + x.Name + "." + argType.Sel.Name + ")"

							for _, receiptName := range receipts {
								if _, ok := basicContracts[v.res.ContractName]; !ok {
									if strings.Contains(exp, "Emit("+receiptName+")") {
										v.reportErr("cannot emit standard/ibc receipt in contract", target.Pos())
										exp = ""
										break
									}
								}
							}
						}
					}
				}
			}
		}

		return true
	})

	return
}

// The basic type of float64/float32 is forbidden in contract
// The expression of panic is forbidden in contract
// The expression of for range is forbidden in contract
func (v *visitor) check(fileMap map[string]*ast.File) {
	for _, f := range fileMap {
		ast.Inspect(f, func(node ast.Node) bool {
			switch node.(type) {
			case *ast.FuncDecl:
				fun := node.(*ast.FuncDecl)
				if exists, varName, pos := v.isCallingEx(fun); exists {
					v.reportErr(fmt.Sprintf("cannot use %s directly", varName), pos)
				}
			case *ast.ForStmt:
				v.reportErr("Cannot use for expression", node.Pos())
			case *ast.RangeStmt:
				v.reportErr("Cannot use range expression", node.Pos())
			case *ast.CallExpr:
				b := node.(*ast.CallExpr)
				if c, ok := b.Fun.(*ast.Ident); ok {
					if c.Name == "panic" {
						v.reportErr("Cannot use panic expression", b.Pos())
					}
				}
			case *ast.Ident:
				ti := node.(*ast.Ident)
				if ti.Name == "float64" {
					v.reportErr("Cannot use float type", ti.Pos())
				} else if ti.Name == "float32" {
					v.reportErr("Cannot use float type", ti.Pos())
				}
			case *ast.BasicLit:
				b := node.(*ast.BasicLit)
				if b.Kind.String() == "FLOAT" &&
					!strings.Contains(b.Value, "E") &&
					!strings.Contains(b.Value, "e") {
					v.reportErr("Cannot use float type", b.Pos())
				}
			}
			return true
		})
	}
}

// nolint cyclomatic
func (v *visitor) isCallingEx(d *ast.FuncDecl) (exists bool, varName string, pos token.Pos) {
	ast.Inspect(d, func(n ast.Node) bool {
		if _, ok0 := n.(*ast.SelectorExpr); !ok0 {
			return true
		}

		target := n
		if selector, ok1 := target.(*ast.SelectorExpr); ok1 {
			if _, ok := variableCallee[selector.Sel.Name]; ok {
				if x, ok := selector.X.(*ast.Ident); ok {
					if a, ok := x.Obj.Decl.(*ast.AssignStmt); ok {
						if r, ok := a.Rhs[0].(*ast.Ident); ok {
							x = r
						}
					}

					if a, ok := x.Obj.Decl.(*ast.Field); ok {
						if t, ok := a.Type.(*ast.StarExpr); ok {
							if i, ok := t.X.(*ast.Ident); ok {
								if i.Name == v.res.ContractStructure {
									exists = true
									if pos == 0 {
										pos = target.Pos()
									}
									varName = selector.Sel.Name
									return false
								}
							}
						}
					}
				}
			}
		}

		return true
	})

	return
}

type callingList []CallNode

type CallNode struct {
	RecvName string // receiver struct name
	FuncName string // func name
}

func (c *CallNode) equals(n *CallNode) bool {
	if c.FuncName == n.FuncName {
		return true
	}
	return false
}

// check cycle call and recursive
func (v *visitor) checkCycleCall(fileMap map[string]*ast.File) {

	callingMap := map[CallNode]callingList{}

	for _, f := range fileMap {
		ast.Inspect(f, func(node ast.Node) bool {
			switch node.(type) {
			case *ast.FuncDecl:
				d := node.(*ast.FuncDecl)
				callList := new(callingList)
				fcn := new(CallNode)
				fcn.FuncName = d.Name.Name

				if d.Recv != nil && len(d.Recv.List) > 0 && len(d.Recv.List[0].Names) > 0 {
					fcn.RecvName = d.Recv.List[0].Names[0].Name
				}
				callingMap[*fcn] = v.checkRecursive(node.(*ast.FuncDecl), *callList, callingMap)
			}
			return true
		})
	}
}

func (v *visitor) checkRecursive(d *ast.FuncDecl, callList callingList, cm map[CallNode]callingList) callingList {
	hasRecv := false
	if d.Recv != nil && len(d.Recv.List) > 0 {
		hasRecv = true
	}

	ast.Inspect(d, func(n ast.Node) bool {
		if _, ok0 := n.(*ast.CallExpr); !ok0 {
			return true
		}
		fcn := new(CallNode)
		fcn.FuncName = d.Name.Name

		if d.Recv != nil && len(d.Recv.List) > 0 && len(d.Recv.List[0].Names) > 0 {
			fcn.RecvName = d.Recv.List[0].Names[0].Name
		}

		if call, ok := n.(*ast.CallExpr); ok {
			callNode := new(CallNode)
			if id, okID := call.Fun.(*ast.Ident); okID {
				callNode.FuncName = id.Name
				if id.Name == d.Name.Name && !hasRecv {
					v.reportErr("contract cannot use recursive", n.Pos())
					return false
				}
			} else if fun, ok2 := call.Fun.(*ast.SelectorExpr); ok2 {
				if id, okID := fun.X.(*ast.Ident); okID && fun.Sel != nil {
					callNode.FuncName = fun.Sel.Name
					//callNode.RecvName = id.Name
					if hasRecv && d.Recv != nil && len(d.Recv.List[0].Names) > 0 && d.Recv.List[0].Names[0] != nil &&
						id.Name == d.Recv.List[0].Names[0].Name && fun.Sel.Name == d.Name.Name {
						v.reportErr("contract cannot use recursive", n.Pos())
						return false
					}
				}
			}
			if hasCycleCall(fcn, callNode, cm) {
				v.reportErr("can not cycle call", n.Pos())
				return false
			}
			callList = append(callList, *callNode)

		}
		return true
	})
	return callList
}

func (v *visitor) checkIBC() string {
	errStr := ""
	if v.res.ContractName == "ibc" {
		return errStr
	}

	ibcFuncCount := len(v.res.TFunctions)

	recastProto := "Recast(types.Hash)bool"
	confirmProto := "Confirm(types.Hash)"
	cancelProto := "Cancel(types.Hash)"
	tryRecastProto := "TryRecast(types.Hash)bool"
	confirmRecastProto := "ConfirmRecast(types.Hash)"
	cancelRecastProto := "CancelRecast(types.Hash)"
	notifyProto := "Notify(types.Hash)"
	switch ibcFuncCount {
	case 0:
		return errStr
	case 1:
		if isExist(v, notifyProto) {
			return errStr
		} else {
			errStr += "contract: " + v.res.ContractName + "\n"
			errStr += "invalid ibc functions, must be: " + notifyProto
		}
	case 3:
		if isExist(v, recastProto) && isExist(v, confirmProto) && isExist(v, cancelProto) {
			return errStr
		} else {
			errStr += "contract: " + v.res.ContractName + "\n"
			errStr += "invalid ibc functions, must be: \n" + recastProto + "\n" + confirmProto + "\n" + cancelProto
		}
	case 4:
		if isExist(v, recastProto) && isExist(v, confirmProto) && isExist(v, cancelProto) && isExist(v, notifyProto) {
			return errStr
		} else {
			errStr += "contract: " + v.res.ContractName + "\n"
			errStr += "invalid ibc functions, must be: \n" + recastProto + "\n" + confirmProto + "\n" + cancelProto + "\n" + notifyProto
		}
	case 6:
		if isExist(v, recastProto) && isExist(v, confirmProto) && isExist(v, cancelProto) &&
			isExist(v, tryRecastProto) && isExist(v, confirmRecastProto) && isExist(v, cancelRecastProto) {
			return errStr
		} else {
			errStr += "contract: " + v.res.ContractName + "\n"
			errStr += "invalid ibc functions, must be: \n" + recastProto + "\n" + confirmProto + "\n" + cancelProto + "\n" +
				tryRecastProto + "\n" + confirmRecastProto + "\n" + cancelRecastProto
		}
	case 7:
		if isExist(v, recastProto) && isExist(v, confirmProto) && isExist(v, cancelProto) &&
			isExist(v, tryRecastProto) && isExist(v, confirmRecastProto) && isExist(v, cancelRecastProto) &&
			isExist(v, notifyProto) {
			return errStr
		} else {
			errStr += "contract: " + v.res.ContractName + "\n"
			errStr += "invalid ibc functions, must be: \n" + recastProto + "\n" + confirmProto + "\n" + cancelProto + "\n" +
				tryRecastProto + "\n" + confirmRecastProto + "\n" + cancelRecastProto + "\n" + notifyProto
		}
	default:
		errStr += "contract: " + v.res.ContractName + "\n"
		errStr += "invalid ibc function's count"
	}

	return errStr
}

func isExist(v *visitor, ibcFuncProto string) bool {
	for _, ibcFunc := range v.res.TFunctions {
		if CreatePrototype(ibcFunc.Method) == ibcFuncProto {
			return true
		}
	}

	return false
}

func hasCycleCall(mainNode, currNode *CallNode, cm map[CallNode]callingList) bool {
	cn := &[]CallNode{}
	tempMap := cm
	cn = getCalledNodes(mainNode, tempMap, cn)
	for _, an := range *cn {
		if an.equals(currNode) {
			return true
		}
	}
	return false
}

func getCalledNodes(node *CallNode, cm map[CallNode]callingList, cns *[]CallNode) *[]CallNode {
	for k1, v1 := range cm {
		for _, v2 := range v1 {
			if node.equals(&v2) {
				*cns = append(*cns, k1)
				if node.equals(&k1) {
					continue
				}

				getCalledNodes(&k1, cm, cns)
			}
		}
	}

	return cns
}

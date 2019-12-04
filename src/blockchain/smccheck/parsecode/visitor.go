package parsecode

/*
 * pay attention to the receivers have star or not.. interesting ?
 */
import (
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"
)

const (
	methodGasPrefix    = "@:public:method:gas["
	interfaceGasPrefix = "@:public:interface:gas["
	ibcGasPrefix       = "@:public:ibc:gas["
)

// Import - ..
type Import struct {
	Name string
	Path string
}

// Field - describe the Field of go ast
type Field struct {
	Names         []string // names have 0 to n member(s)
	FieldType     ast.Expr
	RelatedImport map[Import]struct{} // the field type imported package
}

// Method - the interface's method member has no receiver
type Method struct {
	Name    string
	Params  []Field
	Results []Field
}

// Function - describe the function in go ast
type Function struct {
	Method
	Comments string
	Receiver Field // go Function's receiver is an array, but I doubt how it works.
	MGas     int64
	IGas     int64
	TGas     int64
	pos      token.Pos

	GetTransferToMe bool
}

type ImportContract struct {
	Name       string
	Interfaces []Method
	Pos        token.Pos
}

// Result is the parse result
type Result struct {
	DirectionName string
	PackageName   string
	Imports       map[Import]struct{} // current file parsed,
	AllImports    map[Import]struct{} // imports of all files, see function importsCollector

	ContractName      string
	OrgID             string
	Version           string
	Versions          []string
	Author            string
	ContractStructure string

	Stores      []Field
	StoreCaches []Field

	InitChain          Function
	IsExistInitChain   bool
	UpdateChain        Function
	IsExistUpdateChain bool
	Mine               Function
	IsExistMine        bool
	Functions          []Function // all function
	MFunctions         []Function // methods
	IFunctions         []Function // interfaces
	TFunctions         []Function // ibcs

	Receipts        []Method
	ImportContracts []ImportContract

	UserStruct map[string]ast.GenDecl

	ErrFlag   bool
	ErrorDesc []string
	ErrorPos  []token.Pos
}

type visitor struct {
	res     *Result
	depth   int
	inClass bool // walk in the contract structure, it's a flag (parse store)
}

func newVisitor() visitor {
	res := Result{
		Imports:         make(map[Import]struct{}),
		AllImports:      make(map[Import]struct{}),
		Functions:       make([]Function, 0),
		MFunctions:      make([]Function, 0),
		IFunctions:      make([]Function, 0),
		TFunctions:      make([]Function, 0),
		ImportContracts: make([]ImportContract, 0),
		Stores:          make([]Field, 0),
		StoreCaches:     make([]Field, 0),
		UserStruct:      make(map[string]ast.GenDecl),
		ErrorDesc:       make([]string, 0),
		ErrorPos:        make([]token.Pos, 0),
	}
	depth := 0

	return visitor{
		res:   &res,
		depth: depth,
	}
}

// Visit is a visitor method walk through the AST node in depth-first order, same to inspect
func (v visitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	switch d := n.(type) {
	case *ast.Ident: // parse package name
		v.parsePackageName(d)
	case *ast.FuncDecl: // parse function annotation (gas,constructor)
		v.parseAllFunc(d)
	case *ast.Field: // parse function annotation (store)
		v.parseStoreField(d)
	case *ast.GenDecl:
		v.parseGenDeclare(d)
	}

	v.depth++
	return v
}

func (v *visitor) parseField(d *ast.Field) Field {
	f := Field{}
	names := make([]string, 0)
	for _, id := range d.Names {
		names = append(names, id.Name)
	}
	f.Names = names

	f.FieldType = d.Type

	imports := make(map[Import]struct{})
	getFieldImports(d.Type, v.res.Imports, imports)

	f.RelatedImport = imports

	return f
}

func getFieldImports(t ast.Expr, imports map[Import]struct{}, filtered map[Import]struct{}) {
	getImport := func(pkg string) Import {
		for imp := range imports {
			if pkg == imp.Name || strings.HasSuffix(imp.Path, "/"+pkg+"\"") {
				return imp
			}
		}
		return Import{}
	}
	doSel := func(sel *ast.SelectorExpr) {
		if id, okID := sel.X.(*ast.Ident); okID {
			pkg := id.Name
			imp := getImport(pkg)
			if imp.Path != "" {
				filtered[imp] = struct{}{}
			}
		}
	}

CONTINUE:
	if sel, okSel := t.(*ast.SelectorExpr); okSel {
		doSel(sel)
	} else if star, okStar := t.(*ast.StarExpr); okStar {
		t = star.X
		goto CONTINUE
	} else if arr, okArr := t.(*ast.ArrayType); okArr {
		if sel, okSel := arr.Elt.(*ast.SelectorExpr); okSel {
			doSel(sel)
		} else if arr, okArr := arr.Elt.(*ast.ArrayType); okArr {
			t = arr
			goto CONTINUE
		}
	} else if mt, okM := t.(*ast.MapType); okM {
		if sel, okSel := mt.Key.(*ast.SelectorExpr); okSel {
			doSel(sel)
		} else if sel, okSel := mt.Key.(*ast.StarExpr); okSel {
			if sel, okSel := sel.X.(*ast.SelectorExpr); okSel {
				doSel(sel)
			}
		}
		if vt, okV := mt.Value.(*ast.SelectorExpr); okV {
			doSel(vt)
		} else if vt, okV := mt.Value.(*ast.MapType); okV {
			t = vt
			goto CONTINUE
		} else if vs, okVS := mt.Value.(*ast.StarExpr); okVS {
			if sel, okSel := vs.X.(*ast.SelectorExpr); okSel {
				doSel(sel)
			}
		} else if ar, okA := mt.Value.(*ast.ArrayType); okA {
			if sel, okS := ar.Elt.(*ast.SelectorExpr); okS {
				doSel(sel)
			} else if arr, okArr := ar.Elt.(*ast.ArrayType); okArr {
				t = arr
				goto CONTINUE
			}
		}
	}
}

func (v *visitor) parseStoreField(d *ast.Field) {
	if v.inClass && d.Doc != nil {
		list := strings.Split(d.Doc.Text(), "\n")
		for _, doc := range list {
			doc = strings.TrimSpace(doc)
			if doc == "@:public:store:cache" {
				cacheField := v.parseField(d)

				if e, ok := d.Type.(*ast.MapType); ok {
					if e, ok := e.Value.(*ast.MapType); ok {
						if _, ok := e.Value.(*ast.MapType); ok {
							v.reportErr("contract cannot support more than two level map", d.Pos())
						}
					}
				}

				v.res.StoreCaches = append(v.res.StoreCaches, cacheField)
			} else if doc == "@:public:store" {
				storeField := v.parseField(d)

				if e, ok := d.Type.(*ast.MapType); ok {
					if e, ok := e.Value.(*ast.MapType); ok {
						if _, ok := e.Value.(*ast.MapType); ok {
							v.reportErr("contract cannot support more than two level map", d.Pos())
						}
					}
				}

				v.res.Stores = append(v.res.Stores, storeField)
			}
		}

		v.initVariableCallee()
	}
}

func (v *visitor) initVariableCallee() {
	for _, s := range v.res.Stores {
		variableCallee[s.Names[0]] = struct{}{}
	}

	for _, sc := range v.res.StoreCaches {
		variableCallee[sc.Names[0]] = struct{}{}
	}
}

func (v *visitor) parseAllFunc(d *ast.FuncDecl) {

	f := v.parseFunction(d)
	if d.Doc != nil {
		// fmt.Println("FUNCTION::: name(", d.Name.Name, ") =>[[", d.Doc.Text(), "]]")
		if v.hasConstructorInComments(d) {
			if d.Name.Name == "InitChain" {
				if v.res.IsExistInitChain == true {
					v.reportErr("the InitChain function must be only one", d.Type.Pos())
				}
				v.res.InitChain = v.parseInitChain(f, d)
				v.res.IsExistInitChain = true
			} else if d.Name.Name == "UpdateChain" {
				if v.res.IsExistUpdateChain == true {
					v.reportErr("the UpdateChain function must be only one", d.Type.Pos())
				}
				v.res.UpdateChain = v.parseUpdateChain(f, d)
				v.res.IsExistUpdateChain = true
			}
		} else if v.hasMineInComments(d) {
			if d.Name.Name == "Mine" &&
				d.Type.Results != nil &&
				len(d.Type.Results.List) == 1 &&
				d.Type.Results.List[0].Type.(*ast.Ident).Name == "int64" {
				//if d.Name.Name == "Mine" {
				if v.res.IsExistMine == true {
					v.reportErr("the Mine function must be only one", d.Type.Pos())
				}
				v.res.Mine = v.parseMine(f, d)
				v.res.IsExistMine = true
			}
		}
		mGas, mB := v.getGasFromComments(d, methodGasPrefix)
		iGas, iB := v.getGasFromComments(d, interfaceGasPrefix)
		tGas, tB := v.getGasFromComments(d, ibcGasPrefix)
		if iGas < 0 {
			v.reportErr("interface gas must greater than zero", d.Pos())
		}
		if tGas != 0 && v.res.ContractName != "ibc" {
			v.reportErr("ibc interface gas must be zero", d.Pos())
		}

		if mB || iB || tB {

			v.res.Functions = append(v.res.Functions, f)
			if mB {
				f.MGas = mGas
				v.res.MFunctions = append(v.res.MFunctions, f)
			}
			if iB {
				if HaveUserDefinedStruct(f.Method) {
					v.reportErr("The method params/results type cannot use struct", f.pos)
				}

				f.IGas = iGas
				v.res.IFunctions = append(v.res.IFunctions, f)
			}
			if tB {
				f.TGas = tGas
				v.res.TFunctions = append(v.res.TFunctions, f)
			}
		}
	}
}

func (v *visitor) parseFunction(d *ast.FuncDecl) Function {
	// check
	f := Function{}

	if d.Recv != nil {
		f.Receiver = v.parseField(d.Recv.List[0])
	}
	f.Name = d.Name.Name
	f.pos = d.Pos()
	f.Params = make([]Field, 0)
	for _, param := range d.Type.Params.List {
		f.Params = append(f.Params, v.parseField(param))
	}
	f.Results = make([]Field, 0)
	if d.Type.Results != nil {
		for _, res := range d.Type.Results.List {
			f.Results = append(f.Results, v.parseField(res))
		}
	}
	f.Comments = d.Doc.Text()

	return f
}

func (v *visitor) parseInitChain(f Function, d *ast.FuncDecl) Function {

	if len(d.Type.Params.List) > 0 {
		v.reportErr("InitChain must have no params", d.Pos())
	}
	if d.Type.Results != nil && len(d.Type.Results.List) > 0 {
		v.reportErr("InitChain must have no results", d.Pos())
	}
	if d.Recv == nil || len(d.Recv.List) != 1 {
		v.reportErr("InitChain has wrong receiver", d.Pos())
	}
	if d.Recv != nil {
		f.Receiver = v.parseField(d.Recv.List[0])
		if f.Receiver.FieldType.(*ast.StarExpr).X.(*ast.Ident).Name != v.res.ContractStructure {
			v.reportErr("InitChain has wrong receiver", d.Pos())
		}
	}
	f.pos = d.Pos()

	return f
}

func (v *visitor) parseUpdateChain(f Function, d *ast.FuncDecl) Function {

	if len(d.Type.Params.List) > 0 {
		v.reportErr("UpdateChain must have no params", d.Pos())
	}
	if d.Type.Results != nil && len(d.Type.Results.List) > 0 {
		v.reportErr("UpdateChain must have no results", d.Pos())
	}
	if d.Recv == nil || len(d.Recv.List) != 1 {
		v.reportErr("UpdateChain has wrong receiver", d.Pos())
	}
	if d.Recv != nil {
		f.Receiver = v.parseField(d.Recv.List[0])
		if f.Receiver.FieldType.(*ast.StarExpr).X.(*ast.Ident).Name != v.res.ContractStructure {
			v.reportErr("UpdateChain has wrong receiver", d.Pos())
		}
	}
	f.pos = d.Pos()

	return f
}

func (v *visitor) parseMine(f Function, d *ast.FuncDecl) Function {

	if len(d.Type.Params.List) > 0 {
		v.reportErr("Mine must have no params", d.Pos())
	}
	if len(d.Type.Results.List) != 1 {
		v.reportErr("Mine must have one result", d.Pos())
	}
	if d.Recv == nil || len(d.Recv.List) != 1 {
		v.reportErr("Mine has wrong receiver", d.Pos())
	}
	if d.Recv != nil {
		f.Receiver = v.parseField(d.Recv.List[0])
		if f.Receiver.FieldType.(*ast.StarExpr).X.(*ast.Ident).Name != v.res.ContractStructure {
			v.reportErr("Mine has wrong receiver", d.Pos())
		}
	}
	f.pos = d.Pos()

	return f
}

func (v *visitor) hasConstructorInComments(d *ast.FuncDecl) bool {
	l := strings.Split(d.Doc.Text(), "\n")
	for _, c := range l {
		c = strings.TrimSpace(c)
		if strings.HasPrefix(c, "@:constructor") {
			return true
		}
	}
	return false
}

func (v *visitor) hasMineInComments(d *ast.FuncDecl) bool {
	l := strings.Split(d.Doc.Text(), "\n")
	for _, c := range l {
		c = strings.TrimSpace(c)
		if strings.HasPrefix(c, "@:public:mine") {
			return true
		}
	}
	return false
}

func (v *visitor) getGasFromComments(d *ast.FuncDecl, prefix string) (int64, bool) {
	l := strings.Split(d.Doc.Text(), "\n")
	for _, c := range l {
		c = strings.TrimSpace(c)
		if strings.HasPrefix(c, prefix) {
			if !d.Name.IsExported() {
				v.reportErr("func name invalid", d.Pos())
				return 0, false
			}

			if d.Recv == nil {
				v.reportErr("receiver required", d.Pos())
				return 0, false
			}

			if len(d.Recv.List) != 1 {
				v.reportErr("no receiver", d.Pos())
				return 0, false
			}
			if v.res.ContractStructure != d.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).String() {
				v.reportErr("receiver incorrect", d.Pos())
				return 0, false
			}
			gas := c[len(prefix) : len(c)-1]
			i, e := strconv.ParseInt(gas, 10, 64)
			if e != nil {
				v.reportErr("method and interface gas must be a number", d.Pos())
			}
			return i, true
		}
	}
	return 0, false
}

// Id @ level 1 is package name
func (v *visitor) parsePackageName(d *ast.Ident) {
	if v.depth == 1 {
		if d.Name == "std" {
			v.reportErr("Contract package name cannot use std", d.Pos())
		} else if d.Name == "ibc" {
			v.reportErr("Contract package name cannot use ibc", d.Pos())
		} else {
			v.res.PackageName = d.Name
		}
	}
}

func (v *visitor) parseGenDeclare(d *ast.GenDecl) {
	if d.Tok == token.IMPORT {
		v.parseImport(d)
	} else if d.Tok == token.VAR {
		v.parseVar(d)
	} else {
		v.parseStructsNInterface(d)
	}
}

func (v *visitor) parseImport(d *ast.GenDecl) {
	for _, spec := range d.Specs {
		if imp, ok := spec.(*ast.ImportSpec); ok {
			path := imp.Path.Value
			if _, okPKG := WhiteListPKG[path]; !okPKG && !isWhitePrefix(path) {
				v.res.ErrFlag = true
				v.res.ErrorDesc = append(v.res.ErrorDesc, "INVALID IMPORT: "+path)
				v.res.ErrorPos = append(v.res.ErrorPos, d.Pos())
			} else {
				im := Import{Path: path}
				if imp.Name != nil {
					im.Name = imp.Name.Name
				}
				v.res.Imports[im] = struct{}{}
			}
		}
	}
}

func isWhitePrefix(path string) bool {
	for _, pre := range WhiteListPkgPrefix {
		if strings.HasPrefix(path, pre) {
			return true
		}
	}
	return false
}

func (v *visitor) parseVar(d *ast.GenDecl) {
	for _, spec := range d.Specs {
		if value, ok := spec.(*ast.ValueSpec); ok {
			for _, name := range value.Names {
				if name.Name == "_" {
					continue
				}
				if v.depth <= 1 {
					v.reportErr("GLOBAL VAR:"+name.Name, d.Pos())
				} else if strings.HasPrefix(name.Obj.Name, "float") {
					v.reportErr("FLOAT VAR:"+name.Name, d.Pos())
				}
			}
		}
	}
}

func (v *visitor) parseStructsNInterface(d *ast.GenDecl) {
	for _, spec := range d.Specs {
		if typ, ok := spec.(*ast.TypeSpec); ok {
			if v.depth == 1 && d.Doc != nil {
				if _, ok := typ.Type.(*ast.InterfaceType); ok {
					v.parseInterface(d, typ)
					v.parseImportInterface(d, typ)
				}
				if _, ok := typ.Type.(*ast.StructType); ok {
					v.parseStructs(d, typ)
				}
			}
		}
	}
}

// parse interface annotation (receipt)
func (v *visitor) parseInterface(d *ast.GenDecl, typ *ast.TypeSpec) {
	// fmt.Println("INTERFACE::: name(", typ.Name, ") =>[[", d.Doc.Text(), "]]")
	if v.isReceipt(d) {
		if v.res.Receipts == nil {
			v.res.Receipts = make([]Method, 0)
		}
		it, _ := typ.Type.(*ast.InterfaceType)
		for _, am := range it.Methods.List {
			if am.Names[0].Name[:4] != "emit" {
				v.reportErr("Method os receipt interface must start with 'emit'", typ.Pos())
			}
			if m, ok := am.Type.(*ast.FuncType); ok {
				params := make([]Field, 0)
				for _, p := range m.Params.List {
					params = append(params, v.parseField(p))
				}
				results := make([]Field, 0)
				if m.Results != nil {
					for _, r := range m.Results.List {
						results = append(results, v.parseField(r))
					}
				}
				v.res.Receipts = append(v.res.Receipts, Method{
					Name:    am.Names[0].Name,
					Params:  params,
					Results: results,
				})
			}
		}
	}
}

func (v *visitor) isReceipt(d *ast.GenDecl) bool {
	for _, l := range d.Doc.List {
		doc := strings.TrimSpace(l.Text)
		if strings.ToLower(doc) == "//@:public:receipt" {
			return true
		}
	}
	return false
}

// parse import interface annotation (import)
func (v *visitor) parseImportInterface(d *ast.GenDecl, typ *ast.TypeSpec) {
	// fmt.Println("INTERFACE::: name(", typ.Name, ") =>[[", d.Doc.Text(), "]]")
	isImport, importContract := v.isImport(d)
	if isImport {
		if v.isExist(importContract) {
			v.reportErr("import contract must unique", d.Pos())
		}
		if importContract == typ.Name.Name {
			v.reportErr("import flag name cannot same to type name", typ.Pos())
		}
		imCon := ImportContract{Name: importContract, Interfaces: make([]Method, 0), Pos: d.Pos()}
		it, _ := typ.Type.(*ast.InterfaceType)
		for _, am := range it.Methods.List {
			if m, ok := am.Type.(*ast.FuncType); ok {
				params := make([]Field, 0)
				for _, p := range m.Params.List {
					params = append(params, v.parseField(p))
				}
				results := make([]Field, 0)
				if m.Results != nil {
					for _, r := range m.Results.List {
						results = append(results, v.parseField(r))
					}
				}
				imCon.Interfaces = append(imCon.Interfaces, Method{
					Name:    am.Names[0].Name,
					Params:  params,
					Results: results,
				})

				if HaveUserDefinedStruct(imCon.Interfaces[len(imCon.Interfaces)-1]) {
					v.reportErr("The method params/results type cannot use struct", d.Pos())
				}
			}
		}
		v.res.ImportContracts = append(v.res.ImportContracts, imCon)
	}
}

func (v *visitor) isExist(name string) bool {
	for _, ic := range v.res.ImportContracts {
		if ic.Name == name {
			return true
		}
	}

	return false
}

func (v *visitor) isImport(d *ast.GenDecl) (isImport bool, contractName string) {
	for _, l := range d.Doc.List {
		doc := strings.TrimSpace(l.Text)
		if strings.HasPrefix(strings.ToLower(doc), "//@:import:") {
			splitTemp := strings.Split(doc, ":")
			return true, splitTemp[2]
		}
	}
	return false, ""
}

// parse struct annotation (contract,version,organization,author)
// nolint cyclomatic ... 這復雜度不高啊，拆開反倒不利於閱讀了，我設置了 --cyclo-over=20，其實我可以不用這行注釋了，娃哈哈
func (v *visitor) parseStructs(d *ast.GenDecl, typ *ast.TypeSpec) {
	if v.depth == 1 {
		v.res.UserStruct[typ.Name.Name] = *d
	}

	docs := strings.Split(d.Doc.Text(), "\n")
	for _, comment := range docs {
		comment = strings.TrimSpace(comment)
		if len(comment) > 11 && comment[:11] == "@:contract:" {
			if v.res.ContractName != "" {
				v.reportErr("ContractName is already set", d.Pos())
			}
			v.res.ContractName = strings.TrimSpace(comment[11:])
			if !checkRegex(v.res.ContractName, contractNameExpr) {
				v.reportErr("ContractName -> invalid format:"+v.res.ContractName, d.Pos())
			}
			v.inClass = true
			//if v.res.ContractName != v.res.PackageName {
			//	v.reportErr("PackageName("+v.res.PackageName+")!=ContractName("+v.res.ContractName+")", d.Pos())
			//}
		}
		if len(comment) > 10 && comment[:10] == "@:version:" {
			if v.res.Version != "" {
				v.reportErr("Must only one \"@:version:\" comment", d.Pos())
			}
			v.res.Version = strings.TrimSpace(comment[10:])
			if !checkRegex(v.res.Version, versionExpr) {
				v.reportErr("contract Version -> invalid format:"+v.res.Version, d.Pos())
			}
		}
		if len(comment) > 15 && comment[:15] == "@:organization:" {
			if v.res.OrgID != "" {
				v.reportErr("Must only one \"@:organization:\": comment", d.Pos())
			}
			v.res.OrgID = strings.TrimSpace(comment[15:])
			err := CheckOrgID(v.res.OrgID)
			if err != nil {
				v.reportErr("Organization -> invalid format:"+err.Error(), d.Pos())
			}
		}
		if len(comment) > 9 && comment[:9] == "@:author:" {
			if v.res.Author != "" {
				v.reportErr("Must only one \"@:author:\": comment", d.Pos())
			}
			v.res.Author = strings.TrimSpace(comment[9:])
			if !checkRegex(v.res.Author, authorExpr) {
				v.reportErr("Author -> invalid format:"+v.res.Author, d.Pos())
			}
		}
	}
	if v.inClass && v.depth == 1 {
		if v.res.ContractStructure != "" {
			v.reportErr("You have more ContractStructure:"+v.res.ContractStructure+","+typ.Name.Name, d.Pos())
		}
		v.res.ContractStructure = typ.Name.Name
		if !checkRegex(v.res.ContractStructure, contractClassExpr) {
			v.reportErr("ContractStructure -> invalid format:"+v.res.ContractStructure, d.Pos())
		}
		if !v.checkSDKDeclare(typ) {
			v.reportErr("Contract's first field Must be 'sdk sdk.ISmartContract'", d.Pos())
		}
	}
}

func (v *visitor) reportErr(desc string, pos token.Pos) {
	v.res.ErrFlag = true
	v.res.ErrorDesc = append(v.res.ErrorDesc, desc)
	v.res.ErrorPos = append(v.res.ErrorPos, pos)
}

func (v *visitor) printContractInfo() {
	if v.res.ContractName != "" {
		//fmt.Println("PackageName:", v.res.PackageName)
		//fmt.Println("ContractName:", v.res.ContractName)
		//fmt.Println("ContractStructure:", v.res.ContractStructure)
		if v.res.IsExistInitChain && len(v.res.InitChain.Receiver.Names) != 1 {
			v.reportErr("InitChain has no receiver", v.res.InitChain.pos)
		} else {
			//fmt.Println("InitChain's Receiver:", v.res.InitChain.Receiver.Names[0])
		}
	}
	if v.res.Version != "" {
		//fmt.Println("Version:", v.res.Version)
	}
	if v.res.OrgID != "" {
		//fmt.Println("Organization:", v.res.OrgID)
	}
	if v.res.Author != "" {
		//fmt.Println("Author:", v.res.Author)
	}
}

// all declare must obey our naming specification
func checkRegex(obj string, regex string) bool {
	r, e := regexp.Compile(regex)
	if e != nil {
		return false
	}
	return r.MatchString(obj)
}

// contract structure's first field must be "sdk sdk.ISmartContract"
func (v *visitor) checkSDKDeclare(typ *ast.TypeSpec) bool {
	st, _ := typ.Type.(*ast.StructType)
	l := st.Fields.List
	if len(l) == 0 || len(l[0].Names) == 0 {
		return false
	}
	if l[0].Names[0].Name != "sdk" {
		return false
	}
	if id, ok := l[0].Type.(*ast.SelectorExpr); !ok {
		return false
	} else if id.Sel.Name != "ISmartContract" {
		return false
	} else if x, ok := id.X.(*ast.Ident); !ok {
		return false
	} else if x.Name != "sdk" {
		return false
	}

	return true
}

func importsCollector(res *Result) {
	for imp := range res.Imports {
		res.AllImports[imp] = struct{}{}
		delete(res.Imports, imp)
	}
}

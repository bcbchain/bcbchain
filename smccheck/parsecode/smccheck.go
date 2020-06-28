package parsecode

import (
	"bytes"
	"fmt"
	"github.com/bcbchain/sdk/sdk/types"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Check 分析该目录下的合约代码，进行各种规范检查，提取关键信息
func Check(inPath string) (res *Result, err types.Error) {
	err.ErrorCode = types.CodeOK

	rs := checkFiles(inPath)

	fSet := token.NewFileSet()
	// parseDir 實際並不能遞歸檢查多層級目錄，需要自己去遞歸檢查。實際解析到的也只有一個pkg
	pkgMap, err0 := parser.ParseDir(fSet, inPath, isContractFile, parser.ParseComments)
	if err0 != nil {
		panic(err0)
	}

	if len(pkgMap) != 1 {
		err.ErrorDesc = "parse failed, no pkg or more than 1 pkg\n"
	}

	v := newVisitor()
	v.initTxAndMsgCallee()
	for _, pkg := range pkgMap {
		newKeys := resortFiles(pkg.Files)
		for _, path := range newKeys {
			node := pkg.Files[path]
			// 判断是否符合utf-8要求
			if !isUTF8Encode(path) {
				err.ErrorDesc += "parse failed, contract file encode not utf8\n"
			}

			ast.Walk(v, node)
			importsCollector(v.res)
		}

		// if ITx/IMessage any interface be used in InitChain,UpdateChain or Mine, then report error
		v.parseCallEx(txAndMsgCallee, pkg.Files)

		// if GetTransferToMe interface be used in any method, then mark flag to true;
		// it means this method require transfer token to contract account before it's called;
		v.parseCall(transferCallee, pkg.Files)

		// The basic type of float64/float32 is forbidden in contract
		// The expression of panic is forbidden in contract
		// The expression of for range is forbidden in contract
		// ContractStructure's member variable cannot be called by direct
		v.check(pkg.Files)

		// check ibc function count and prototype
		err.ErrorDesc += v.checkIBC()

		// The cycle call is forbidden in contract, include recursive, eg: A -> B, B -> C, C -> A.
		v.checkCycleCall(pkg.Files)

		// if Emit a standard receipt in contract, then report error
		v.parseEmitCall(pkg.Files)
	}

	// check file about flags use times
	err.ErrorDesc += checkFlag(v, fSet, rs)

	//v.printContractInfo()
	checkImportConflict(v.res)
	checkImportWhitelist(v.res)
	if v.res.ErrFlag {
		for idx, pos := range v.res.ErrorPos {
			err.ErrorDesc += v.res.ErrorDesc[idx] + "\n"
			err.ErrorDesc += fmt.Sprintf("%s %d\n", fSet.Position(pos).Filename, fSet.Position(pos).Line)
		}
	}

	if len(err.ErrorDesc) != 0 {
		err.ErrorCode = 500
		fmt.Println(err.ErrorDesc)
		return
	}
	res = v.res

	return
}

func CheckEX(inPath string) (res *Result, paramsTypes map[string]string, err types.Error) {
	err.ErrorCode = types.CodeOK

	rs := checkFiles(inPath)

	fSet := token.NewFileSet()
	// parseDir 實際並不能遞歸檢查多層級目錄，需要自己去遞歸檢查。實際解析到的也只有一個pkg
	pkgMap, err0 := parser.ParseDir(fSet, inPath, isContractFile, parser.ParseComments)
	if err0 != nil {
		panic(err0)
	}

	if len(pkgMap) != 1 {
		err.ErrorDesc = "parse failed, no pkg or more than 1 pkg\n"
	}

	v := newVisitor()
	v.initTxAndMsgCallee()
	for _, pkg := range pkgMap {
		newKeys := resortFiles(pkg.Files)
		for _, path := range newKeys {
			node := pkg.Files[path]
			// 判断是否符合utf-8要求
			if !isUTF8Encode(path) {
				err.ErrorDesc += "parse failed, contract file encode not utf8\n"
			}

			ast.Walk(v, node)
			importsCollector(v.res)
		}

		// if ITx/IMessage any interface be used in InitChain,UpdateChain or Mine, then report error
		v.parseCallEx(txAndMsgCallee, pkg.Files)

		// if GetTransferToMe interface be used in any method, then mark flag to true;
		// it means this method require transfer token to contract account before it's called;
		v.parseCall(transferCallee, pkg.Files)

		// The basic type of float64/float32 is forbidden in contract
		// The expression of panic is forbidden in contract
		// The expression of for range is forbidden in contract
		// ContractStructure's member variable cannot be called by direct
		v.check(pkg.Files)

		// check ibc function count and prototype
		err.ErrorDesc += v.checkIBC()

		// The cycle call is forbidden in contract, include recursive, eg: A -> B, B -> C, C -> A.
		v.checkCycleCall(pkg.Files)

		// if Emit a standard receipt in contract, then report error
		v.parseEmitCall(pkg.Files)
	}

	// check file about flags use times
	err.ErrorDesc += checkFlag(v, fSet, rs)

	//v.printContractInfo()
	checkImportConflict(v.res)
	checkImportWhitelist(v.res)
	if v.res.ErrFlag {
		for idx, pos := range v.res.ErrorPos {
			err.ErrorDesc += v.res.ErrorDesc[idx] + "\n"
			err.ErrorDesc += fmt.Sprintf("%s %d\n", fSet.Position(pos).Filename, fSet.Position(pos).Line)
		}
	}

	if len(err.ErrorDesc) != 0 {
		err.ErrorCode = 500
		fmt.Println(err.ErrorDesc)
		return
	}
	res = v.res

	paramsTypes = make(map[string]string, 0)
	for _, function := range res.Functions {
		Name := function.Name
		proto := CreatePrototype(function.Method)
		paramsTypes[Name] = proto
	}

	return
}

func resortFiles(files map[string]*ast.File) []string {
	var found bool
	newKeys := make([]string, len(files))
	index := 0
	for path, file := range files {
		for _, comment := range file.Comments {
			for _, item := range comment.List {
				if strings.HasPrefix(item.Text, "//@:contract:") {
					newKeys[0] = path
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if found {
			if newKeys[index] == "" {
				newKeys[index] = path
			}
		} else {
			newKeys[index+1] = path
		}
		index++
	}

	return newKeys
}

func isContractFile(d os.FileInfo) bool {
	return !d.IsDir() &&
		strings.HasSuffix(d.Name(), ".go") &&
		!strings.Contains(d.Name(), "autogen") &&
		!strings.HasSuffix(d.Name(), "_test.go")
}

func checkImportConflict(res *Result) {
	for imp := range res.AllImports {
		for im := range res.AllImports {
			if imp.Name == "." {
				res.ErrFlag = true
				res.ErrorDesc = append(res.ErrorDesc, "dot Import not allowed")
				res.ErrorPos = append(res.ErrorPos, 0)
			}
			if imp != im && imp.Name != "" && imp.Name == im.Name && imp.Path != im.Path {
				res.ErrFlag = true
				res.ErrorDesc = append(res.ErrorDesc, "Import conflict:"+imp.Name+" has more than one path:"+imp.Path+","+im.Path)
				res.ErrorPos = append(res.ErrorPos, 0)
			}
		}
	}
}

func isUTF8Encode(path string) bool {
	resBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return false
	}

	// mod-header
	if resBytes[0] == 0xFE && resBytes[1] == 0xFF {
		// UTF16BE
		return false
	} else if resBytes[0] == 0xFF && resBytes[1] == 0xFE {
		// UTF16LE
		return false
	}

	// contents encode
	if !utf8.Valid(resBytes) {
		return false
	}

	return true
}

// FmtAndWrite - go fmt content and write to filename
func FmtAndWrite(filename, content string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if ee := f.Close(); ee != nil {
			fmt.Println(ee)
		}
	}()

	// Create a FileSet for node. Since the node does not come
	// from a real source file, fSet will be empty.
	fSet := token.NewFileSet()

	// parser.ParseExpr parses the argument and returns the
	// corresponding ast.Node.
	node, err := parser.ParseFile(fSet, "", content, parser.ParseComments)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	err = format.Node(&buf, fSet, node)
	if err != nil {
		return err
	}

	_, err = f.WriteString(buf.String())
	if err != nil {
		return err
	}
	// fmt.Println(n, "byte write to file")
	return nil
}

// Versions - 返回合约版本列表
func CheckVersions(firstContractPath string, res *Result) types.Error {
	retErr := types.Error{ErrorCode: types.CodeOK}

	fInfoS, err := ioutil.ReadDir(firstContractPath)
	if err != nil {
		panic(err)
	}

	var verLen int
	res.Versions = make([]string, 0)
	for _, fInfo := range fInfoS {
		if fInfo.IsDir() && fInfo.Name() != "." && fInfo.Name() != ".." {
			ver := fInfo.Name()
			verSplit := strings.Split(ver, ".")
			if verLen == 0 {
				verLen = len(verSplit)
			} else if verLen != len(verSplit) {
				retErr.ErrorCode = 500
				retErr.ErrorDesc = "version format not match"
			}

			res.Versions = append(res.Versions, fInfo.Name())
		}
	}

	return retErr
}

func checkImportWhitelist(res *Result) {
	var whitelist = map[string]struct{}{
		"bytes": {}, "container/heap": {}, "container/list": {}, "container/ring": {}, "crypto": {},
		"crypto/aes": {}, "crypto/cipher": {}, "crypto/des": {}, "crypto/hmac": {}, "crypto/md5": {},
		"crypto/rc4": {}, "crypto/sha1": {}, "crypto/sha256": {}, "crypto/sha512": {}, "encoding": {},
		"encoding/ascii85": {}, "encoding/asn1": {}, "encoding/base32": {}, "encoding/base64": {},
		"encoding/binary": {}, "encoding/csv": {}, "encoding/gob": {}, "encoding/hex": {}, "encoding/json": {},
		"encoding/pem": {}, "encoding/xml": {}, "errors": {}, "fmt": {}, "hash": {}, "hash/adler32": {},
		"hash/crc32": {}, "hash/crc64": {}, "hash/fnv": {}, "index/suffixarray": {}, "math": {}, "math/big": {},
		"math/bits": {}, "math/cmplx": {}, "reflect": {}, "regexp": {}, "regexp/syntax": {}, "sort": {},
		"strconv": {}, "strings": {}, "unicode": {}, "unicode/utf16": {}, "unicode/utf8": {},
	}

	for imp := range res.AllImports {
		p := strings.TrimSpace(imp.Path)
		p = strings.Replace(p, "\"", "", -1)
		if strings.HasPrefix(p, "github.com/bcbchain/sdk/sdk") || strings.HasPrefix(p, "blockchain/smcsdk/sdk") {
			continue
		}
		if _, ok := whitelist[p]; !ok {
			res.ErrFlag = true
			res.ErrorDesc = append(res.ErrorDesc, "invalid import: "+imp.Path)
			res.ErrorPos = append(res.ErrorPos, 0)
			return
		}
	}
}

type Report struct {
	Count   int64    `json:"count"`
	Flag    string   `json:"flag"`
	ErrDesc string   `json:"errDesc"`
	File    []string `json:"file"`
	Pos     []int64  `json:"pos"`
}

func initReports() []*Report {
	rs := make([]*Report, 0, 12)

	rs = append(rs, &Report{Flag: "//@:contract:", File: make([]string, 0), Pos: make([]int64, 0)})
	rs = append(rs, &Report{Flag: "//@:version:", File: make([]string, 0), Pos: make([]int64, 0)})
	rs = append(rs, &Report{Flag: "//@:organization:", File: make([]string, 0), Pos: make([]int64, 0)})
	rs = append(rs, &Report{Flag: "//@:author:", File: make([]string, 0), Pos: make([]int64, 0)})
	rs = append(rs, &Report{Flag: "//@:public:mine", File: make([]string, 0), Pos: make([]int64, 0)})
	rs = append(rs, &Report{Flag: "//@:constructor", File: make([]string, 0), Pos: make([]int64, 0)})
	rs = append(rs, &Report{Flag: "//@:public:receipt", File: make([]string, 0), Pos: make([]int64, 0)})
	rs = append(rs, &Report{Flag: "//@:public:store:cache", File: make([]string, 0), Pos: make([]int64, 0)})
	rs = append(rs, &Report{Flag: "//@:public:store", File: make([]string, 0), Pos: make([]int64, 0)})
	rs = append(rs, &Report{Flag: "//@:public:method", File: make([]string, 0), Pos: make([]int64, 0)})
	rs = append(rs, &Report{Flag: "//@:public:interface", File: make([]string, 0), Pos: make([]int64, 0)})
	rs = append(rs, &Report{Flag: "//@:import", File: make([]string, 0), Pos: make([]int64, 0)})

	return rs
}

func checkFiles(inPath string) []*Report {

	var err error
	if !filepath.IsAbs(inPath) {
		inPath, err = filepath.Abs(inPath)
		if err != nil {
			panic(err)
		}
	}

	pathSplit := strings.Split(inPath, "/")
	if len(pathSplit) == 1 {
		pathSplit = strings.Split(inPath, "\\")
	}
	i1 := len(pathSplit) - 1
	i2 := i1 - 1
	i3 := i2 - 1

	if pathSplit[i1] == pathSplit[i3] {
		if pathSplit[i1] != pathSplit[i3] {
			panic("wrong path: error direction name")
		}
		if !strings.HasPrefix(pathSplit[i2], "v") {
			panic("wrong path: error version format")
		}
		if !checkRegex(pathSplit[i2][1:], versionExpr) {
			panic("wrong path: error version format")
		}
	}

	rs := initReports()

	err = filepath.Walk(inPath, func(path string, info os.FileInfo, err error) error {
		if isContractFile(info) {
			contents, err := ioutil.ReadFile(path)
			if err != nil {
				panic(err)
			}

			contentStr := string(contents)
			contentSplit := strings.Split(contentStr, "\n")
			if len(contentSplit) == 1 {
				contentSplit = strings.Split(contentStr, "\r\n")
			}

			for i, line := range contentSplit {
				for _, r := range rs {
					if strings.Contains(line, r.Flag) {
						r.Count++
						r.File = append(r.File, path)
						r.Pos = append(r.Pos, int64(i+1))
						break
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		panic(err)
	}

	return rs
}

func checkFlag(v visitor, fSet *token.FileSet, rs []*Report) string {
	errStr := ""

	if v.res.OrgID == "" {
		errStr += "lost organization flag\n"
	}
	if v.res.ContractName == "" {
		errStr += "lost contract flag\n"
	}
	if v.res.Version == "" {
		errStr += "lost version flag\n"
	}
	if v.res.Author == "" {
		errStr += "lost author flag\n"
	}

	for _, r := range rs {
		if r.Flag == "//@:constructor" {
			if r.Count > 2 {
				for i, filePos := range r.File {

					errStr += r.Flag + " flag cannot more than twice\n"
					errStr += fmt.Sprintf("%s %d\n", filePos, r.Pos[i])
				}
			} else {
				errStr += checkConstructor(v, fSet, r)
			}
		} else if r.Flag == "//@:public:store" {
			errStr += checkStore(v, fSet, r)
		} else if r.Flag == "//@:public:store:cache" {
			errStr += checkStoreCache(v, fSet, r)
		} else if r.Flag == "//@:public:method" {
			errStr += checkMethod(v, fSet, r)
		} else if r.Flag == "//@:public:interface" {
			errStr += checkInterface(v, fSet, r)
		} else if r.Flag == "//@:import" {
			errStr += checkImport(v, fSet, r)
		} else if r.Count != 1 {
			for i, filePos := range r.File {
				errStr += r.Flag + " flag should be used only once\n"
				errStr += fmt.Sprintf("%s %d\n", filePos, r.Pos[i])
			}
		} else if r.Flag == "//@:public:receipt" {
			if v.res.Receipts == nil {
				errStr += r.Flag + " flag must followed defined of receipt interface\n"
				errStr += fmt.Sprintf("%s %d\n", r.File[0], r.Pos[0])
			}
		} else if r.Flag == "//@:public:mine" {
			if v.res.IsExistMine == false {
				errStr += r.Flag + " flag must followed Mine function and declare model must be Mine()int64\n"
				errStr += fmt.Sprintf("%s %d\n", r.File[0], r.Pos[0])
			}
		}
	}

	return errStr
}

func checkConstructor(v visitor, fSet *token.FileSet, r *Report) string {
	errStr := ""
	if r.Count == 2 && (!v.res.IsExistUpdateChain || !v.res.IsExistInitChain) {
		initPos := int64(fSet.Position(v.res.InitChain.pos).Line)
		updatePos := int64(fSet.Position(v.res.UpdateChain.pos).Line)

		if initPos != r.Pos[0]+1 && updatePos != r.Pos[0]+1 {
			errStr += "//@:constructor flag must followed InitChain/UpdateChain function\n"
			errStr += fmt.Sprintf("%s %d\n", r.File[0], r.Pos[0])
		}
		if initPos != r.Pos[1]+1 && updatePos != r.Pos[1]+1 {
			errStr += "//@:constructor flag must followed InitChain/UpdateChain function\n"
			errStr += fmt.Sprintf("%s %d\n", r.File[1], r.Pos[1])
		}
	} else if r.Count == 1 && (!v.res.IsExistInitChain && !v.res.IsExistUpdateChain) {
		errStr += "//@:constructor flag must followed InitChain/UpdateChain function\n"
		errStr += fmt.Sprintf("%s %d\n", r.File[0], r.Pos[0])
	}

	return errStr
}

func checkStore(v visitor, fSet *token.FileSet, r *Report) string {
	errStr := ""
	if r.Count != int64(len(v.res.Stores)) {
		for _, store := range v.res.Stores {
			pos := fSet.Position(store.FieldType.Pos()).Line
			for i, comPos := range r.Pos {
				if int64(pos) == comPos+1 {
					r.Pos = append(r.Pos[:i], r.Pos[i+1:]...)
					r.File = append(r.File[:i], r.File[i+1:]...)
					break
				}
			}
		}

		for i := range r.File {
			errStr += "//@:public:store flag must followed member variable defined\n"
			errStr += fmt.Sprintf("%s %d\n", r.File[i], r.Pos[i])
		}
	}

	return errStr
}

func checkStoreCache(v visitor, fSet *token.FileSet, r *Report) string {
	errStr := ""
	if r.Count != int64(len(v.res.StoreCaches)) {
		for _, store := range v.res.StoreCaches {
			pos := fSet.Position(store.FieldType.Pos()).Line
			for i, comPos := range r.Pos {
				if int64(pos) == comPos+1 {
					r.Pos = append(r.Pos[:i], r.Pos[i+1:]...)
					r.File = append(r.File[:i], r.File[i+1:]...)
					break
				}
			}
		}

		for i := range r.File {
			errStr += "//@:public:store:cache flag must followed member variable defined\n"
			errStr += fmt.Sprintf("%s %d\n", r.File[i], r.Pos[i])
		}
	}

	return errStr
}

func checkMethod(v visitor, fSet *token.FileSet, r *Report) string {
	errStr := ""
	if r.Count != int64(len(v.res.MFunctions)) {
		for _, f := range v.res.MFunctions {
			pos := fSet.Position(f.pos).Line
			for i, comPos := range r.Pos {
				if int64(pos) == comPos+1 || int64(pos) == comPos+2 {
					r.Pos = append(r.Pos[:i], r.Pos[i+1:]...)
					r.File = append(r.File[:i], r.File[i+1:]...)
					break
				}
			}
		}

		for i := range r.File {
			errStr += "//@:public:method flag must followed member method defined\n"
			errStr += fmt.Sprintf("%s %d\n", r.File[i], r.Pos[i])
		}
	}

	return errStr
}

func checkInterface(v visitor, fSet *token.FileSet, r *Report) string {
	errStr := ""
	if r.Count != int64(len(v.res.IFunctions)) {
		for _, f := range v.res.IFunctions {
			pos := fSet.Position(f.pos).Line
			for i, comPos := range r.Pos {
				if int64(pos) == comPos+1 || int64(pos) == comPos+2 {
					r.Pos = append(r.Pos[:i], r.Pos[i+1:]...)
					r.File = append(r.File[:i], r.File[i+1:]...)
					break
				}
			}
		}

		for i := range r.File {
			errStr += "//@:public:interface flag must followed member interface defined\n"
			errStr += fmt.Sprintf("%s %d\n", r.File[i], r.Pos[i])
		}
	}

	return errStr
}

func checkImport(v visitor, fSet *token.FileSet, r *Report) string {
	errStr := ""
	if r.Count != int64(len(v.res.ImportContracts)) {
		for _, ic := range v.res.ImportContracts {
			pos := fSet.Position(ic.Pos).Line
			for i, comPos := range r.Pos {
				if int64(pos) == comPos+1 {
					r.Pos = append(r.Pos[:i], r.Pos[i+1:]...)
					r.File = append(r.File[:i], r.File[i+1:]...)
					break
				}
			}
		}

		for i := range r.File {
			errStr += "//@:public:import flag must followed import interface defined\n"
			errStr += fmt.Sprintf("%s %d\n", r.File[i], r.Pos[i])
		}
	}

	return errStr
}

package upgrade1to2

import (
	"blockchain/abciapp_v1.0/bcerrors"
	"blockchain/abciapp_v1.0/smc"
	"blockchain/algorithm"
	"blockchain/smcbuilder"
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/std"
	smcsdk "blockchain/smcsdk/sdk/types"
	"common/dockerlib"
	"encoding/hex"
	"errors"
	"github.com/tendermint/tmlibs/log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const genesisOrgID = "orgJgaGConUyK81zibntUBjQ33PKctpk1K1G"

// Upgrade - upgrade v1 to v2
func (u *Upgrade1to2) Upgrade(conInfoList string) (chainVersion string, smcErr smc.Error) {

	smcErr.ErrorCode = bcerrors.ErrCodeOK

	// check is upgraded or not
	if appState, err := u.getAppState(); err != nil {
		panic(err.Error())
	} else if appState.ChainVersion != 0 {
		smcErr.ErrorCode = bcerrors.ErrCodeInterContractsRuntimeError
		smcErr.ErrorDesc = "v1 to v2 has been upgraded"
		return
	}

	if !(u.Sender.Addr == u.Owner.Addr && u.getGenesisToken().Owner == u.Sender.Addr) {
		smcErr.ErrorCode = bcerrors.ErrCodeNoAuthorization
		smcErr.ErrorDesc = "Only contract owner can do it."
		return
	}

	// update all genesis info, include update genesis contracts and save genesis orgID
	u.updateGenesisInfo()
	allContractAddr := u.getAllContract()
	for _, addr := range *allContractAddr {
		contract := u.getContract(addr)
		if u.isGenesisContract(contract.Name) || u.isTokenContract(contract) {
			u.updateContract(contract)
		} else {
			u.updateV1Contract(contract)
		}
	}

	// update all validators' public key to a byte slice of 32 length
	u.updateValidatorPubKey()

	// forbid v1 mining contract
	u.forbidMining()

	u.GenesisOrg.OrgID = genesisOrgID
	u.GenesisOrg.Name = "genesis"
	u.GenesisOrg.OrgOwner = u.Sender.Addr

	// save v2 contract to v2 cache, because build contract will query current organization's all contracts in v2 cache.
	transID := u.newV2TransactionID()
	defer u.rollbackV2(transID)

	err := u.buildAndSaveNewContract(conInfoList, transID)
	if err != nil {
		smcErr.ErrorCode = bcerrors.ErrCodeInterContractsRuntimeError
		smcErr.ErrorDesc = err.Error()
		return
	}

	// upgrade all token contract that name has prefix: "token-templet-".
	u.upgradeTokenContract()

	if err := u.setOrgAuthDeployContract(&u.Sender.Addr); err != nil {
		smcErr.ErrorCode = bcerrors.ErrCodeInterContractsRuntimeError
		smcErr.ErrorDesc = err.Error()
		return
	}

	chainVersion = "2"
	return
}

func (u *Upgrade1to2) forbidMining() {
	for _, v := range *u.getAllContract() {
		con := u.getContract(v)
		if con.Name == "mining" {
			con.LoseHeight = u.Block.Height + 1
			con.OrgID = genesisOrgID
			u.setContract(con)
			return
		}
	}
}

func (u *Upgrade1to2) updateV1Contract(contract *std.Contract) {
	contract.OrgID = genesisOrgID
	u.setContract(contract)
	versionList := std.ContractVersionList{
		Name:             contract.Name,
		ContractAddrList: []smc.Address{contract.Address},
		EffectHeights:    []int64{contract.EffectHeight},
	}
	u.setContractVersionInfo(&versionList, genesisOrgID)
}

func (u *Upgrade1to2) updateGenesisInfo() {
	u.setGenesisOrgID(genesisOrgID)

	for _, v := range *u.getGenesisContracts() {
		contract := u.getGenesisContract(v)
		contract.OrgID = genesisOrgID
		if contract.Name == "token-basic" {
			contract.Token = u.getGenesisToken().Address
		}
		u.setGenesisContract(contract)
	}
}

func (u *Upgrade1to2) isGenesisContract(name string) bool {
	for _, v := range *u.getGenesisContracts() {
		contract := u.getGenesisContract(v)
		if contract.Name == name && contract.ChainVersion == 0 {
			return true
		}
	}
	return false
}

func (u *Upgrade1to2) isTokenContract(contract *std.Contract) bool {
	if strings.HasPrefix(contract.Name, "token-templet-") && contract.ChainVersion == 0 {
		return true
	}
	return false
}

func (u *Upgrade1to2) updateContract(contract *std.Contract) {
	if contract.Name == "token-basic" {
		contract.OrgID = genesisOrgID
		contract.Token = u.getGenesisToken().Address
		contract.LoseHeight = u.Block.Height + 1
		if contract.LoseHeight == 0 {
			contract.LoseHeight = u.Block.Height + 1
		}
	} else if strings.HasPrefix(contract.Name, "token-templet-") {
		contract.OrgID = genesisOrgID
		contract.Token = contract.Address
		if contract.LoseHeight == 0 {
			contract.LoseHeight = u.Block.Height + 1
		}
	} else if contract.Name == "token-templet" || contract.Name == "token-issue" {
		contract.OrgID = genesisOrgID
		if contract.LoseHeight == 0 {
			contract.LoseHeight = u.Block.Height + 1
		}
	} else {
		contract.OrgID = genesisOrgID
	}
	u.setContract(contract)
}

func (u *Upgrade1to2) buildAndSaveNewContract(conInfoList string, transID int64) error {
	v2Contracts := new([]Contract)
	err := jsoniter.Unmarshal([]byte(conInfoList), v2Contracts)
	if err != nil {
		return err
	}

	if u.checkTokenBasicAndTokenIssue(v2Contracts) == false {
		return errors.New("v2 contracts must contains token-basic and token-issue")
	}

	if u.checkOrgSigner(v2Contracts) == false {
		return errors.New("must only one organization signer")
	}

	builder, err := u.initBuilder()
	if err != nil {
		return err
	}

	currentHeight := u.Block.Height
	genesisToken := u.getGenesisToken()

	var (
		v2ContractAddrList []smc.Address
		orgSigner          string
		genesisOrg         std.Organization
		mineCount          int
	)
	for _, v := range *v2Contracts {
		orgSigner = v.CodeOrgSig.PubKey

		codeHash, err := hex.DecodeString(v.CodeHash)
		if err != nil {
			return err
		}

		addr := algorithm.CalcContractAddress(u.Block.ChainID, genesisOrgID, v.Name, v.Version)
		v2ContractAddrList = append(v2ContractAddrList, addr)
		u.GenesisOrg.ContractAddrList = append(u.GenesisOrg.ContractAddrList, addr)

		codeDevSigBytes, _ := jsoniter.Marshal(v.CodeDevSig)
		codeDevSigBytes, _ = jsoniter.Marshal(string(codeDevSigBytes))
		codeOrgSigBytes, _ := jsoniter.Marshal(v.CodeOrgSig)
		codeOrgSigBytes, _ = jsoniter.Marshal(string(codeOrgSigBytes))
		buildResult := builder.BuildContract(transID, 1, std.ContractMeta{
			Name:         v.Name,
			ContractAddr: addr,
			OrgID:        genesisOrgID,
			Version:      v.Version,
			EffectHeight: currentHeight + 1,
			LoseHeight:   0,
			CodeData:     v.CodeByte,
			CodeHash:     codeHash,
			CodeDevSig:   codeDevSigBytes,
			CodeOrgSig:   codeOrgSigBytes,
		})

		if buildResult.Code != bcerrors.ErrCodeOK {
			return errors.New(buildResult.Error)
		}

		genesisOrg.Name = "genesis"
		genesisOrg.OrgID = genesisOrgID
		genesisOrg.OrgOwner = u.Sender.Addr
		genesisOrg.ContractAddrList = append(genesisOrg.ContractAddrList, addr)
		genesisOrg.OrgCodeHash = buildResult.OrgCodeHash
		u.setOrganizationToV2Cache(transID, 1, &u.GenesisOrg)

		contract := std.Contract{
			Address:      addr,
			Account:      algorithm.CalcContractAddress(u.Block.ChainID, genesisOrgID, v.Name, ""), // 计算account时，version为空
			Owner:        u.Sender.Addr,
			Name:         v.Name,
			Version:      v.Version,
			CodeHash:     codeHash,
			EffectHeight: currentHeight + 1,
			LoseHeight:   0,
			KeyPrefix:    "",
			Methods:      buildResult.Methods,
			Interfaces:   buildResult.Interfaces,
			Token:        "",
			OrgID:        genesisOrgID,
			ChainVersion: 2,
		}
		if contract.Name == "token-basic" {
			contract.Token = genesisToken.Address
		}

		if len(buildResult.Mine) != 0 {
			contract.Account = algorithm.CalcContractAddress(u.Block.ChainID, "", v.Name, "")
			mineCount++
		}

		u.setContract(&contract)
		u.setContractToV2Cache(transID, 1, &contract)

		meta := std.ContractMeta{
			Name:         v.Name,
			ContractAddr: addr,
			OrgID:        genesisOrgID,
			Version:      v.Version,
			EffectHeight: currentHeight + 1,
			LoseHeight:   0,
			CodeData:     v.CodeByte,
			CodeHash:     codeHash,
			CodeDevSig:   codeDevSigBytes,
			CodeOrgSig:   codeOrgSigBytes,
		}
		if err := u.setContractMeta(&meta); err != nil {
			return err
		}

		u.setContractMetaToV2Cache(transID, 1, &meta)

		versionList := std.ContractVersionList{
			Name:             v.Name,
			ContractAddrList: []smc.Address{addr},
			EffectHeights:    []int64{currentHeight},
		}
		u.setContractVersionListToV2Cache(transID, 1, genesisOrgID, &versionList)
		if contract.Name == "token-basic" {
			tb, err := u.getTokenBasicContract()
			if err != nil {
				return err
			}
			versionList.EffectHeights = []int64{tb.EffectHeight, contract.EffectHeight}
			versionList.ContractAddrList = []smc.Address{tb.Address, contract.Address}

			u.setContractVersionInfo(&versionList, genesisOrgID)
		} else if contract.Name == "token-issue" {
			u.V2TokenIssue = contract
			ti, err := u.getTokenIssueContract()
			if err != nil {
				return err
			}
			versionList.EffectHeights = []int64{ti.EffectHeight, contract.EffectHeight}
			versionList.ContractAddrList = []smc.Address{ti.Address, contract.Address}
			u.setContractVersionInfo(&versionList, genesisOrgID)
		} else {
			u.setContractVersionInfo(&versionList, genesisOrgID)
		}

		newConWithHeight := std.ContractWithEffectHeight{
			ContractAddr: addr,
			IsUpgrade:    true,
		}
		height := strconv.FormatInt(contract.EffectHeight, 10)
		conWithHeight := u.getEffectHeightContractAddrs(height)
		conWithHeight = append(conWithHeight, newConWithHeight)
		u.setEffectHeightContractAddrs(height, conWithHeight)
	}

	if mineCount != 1 {
		return errors.New("must only one mining contract")
	}

	// save account all contract address, include v1&v2
	accAccountContractAddr := u.getAccountContractAddrs(&u.Sender.Addr)
	*accAccountContractAddr = append(*accAccountContractAddr, v2ContractAddrList...)
	u.setAccountContractAddrs(&u.Sender.Addr, accAccountContractAddr)

	if signerByte, err := hex.DecodeString(orgSigner); err != nil {
		return err
	} else {
		genesisOrg.Signers = []smcsdk.PubKey{signerByte}
	}
	u.setOrganization(&genesisOrg)

	// save v1 & v2 all contract address
	allAddr := append(*u.getAllContract(), v2ContractAddrList...)
	u.setAllContract(&allAddr)

	return nil
}

func (u *Upgrade1to2) checkTokenBasicAndTokenIssue(v2Contracts *[]Contract) bool {
	isContainsTokenBasic := false
	isContainsTokenIssue := false
	for _, v := range *v2Contracts {
		if v.Name == "token-basic" {
			isContainsTokenBasic = true
		}
		if v.Name == "token-issue" {
			isContainsTokenIssue = true
		}
	}

	if isContainsTokenBasic && isContainsTokenIssue {
		return true
	}
	return false
}

func (u *Upgrade1to2) checkOrgSigner(v2Contracts *[]Contract) bool {
	orgSinger := ""
	for _, v := range *v2Contracts {
		if orgSinger == "" {
			orgSinger = v.CodeOrgSig.PubKey
		} else if orgSinger != v.CodeOrgSig.PubKey {
			return false
		}
	}
	return true
}

func (u *Upgrade1to2) upgradeTokenContract() {
	for _, v := range *u.getAllContract() {
		contract := u.getContract(v)
		if u.isTokenContract(contract) {
			if contract.LoseHeight != u.Block.Height+1 {
				continue
			}
			address := algorithm.CalcContractAddress(u.Block.ChainID, genesisOrgID, contract.Name, u.V2TokenIssue.Version)
			v2Contract := std.Contract{
				Address:      address,
				Account:      algorithm.CalcContractAddress(u.Block.ChainID, genesisOrgID, contract.Name, ""),
				Owner:        contract.Owner,
				Name:         contract.Name,
				Version:      u.V2TokenIssue.Version,
				CodeHash:     u.V2TokenIssue.CodeHash,
				EffectHeight: u.Block.Height + 1,
				LoseHeight:   0,
				KeyPrefix:    "",
				Methods:      u.V2TokenIssue.Methods,
				Interfaces:   u.V2TokenIssue.Interfaces,
				Token:        contract.Address,
				OrgID:        genesisOrgID,
				ChainVersion: u.V2TokenIssue.ChainVersion,
			}
			u.setContract(&v2Contract)

			versionList := std.ContractVersionList{
				Name:             v2Contract.Name,
				ContractAddrList: []smc.Address{contract.Address, v2Contract.Address},
				EffectHeights:    []int64{contract.EffectHeight, v2Contract.EffectHeight},
			}
			u.setContractVersionInfo(&versionList, genesisOrgID)

			accountContractAddrs := u.getAccountContractAddrs(&contract.Owner)
			*accountContractAddrs = append(*accountContractAddrs, address)
			u.setAccountContractAddrs(&contract.Owner, accountContractAddrs)
			allContract := u.getAllContract()
			*allContract = append(*allContract, address)
			u.setAllContract(allContract)
		}
	}
}

func (u *Upgrade1to2) getTokenBasicContract() (*std.Contract, error) {
	for _, v := range *u.getAllContract() {
		con := u.getContract(v)
		if con.Name == "token-basic" {
			return con, nil
		}
	}
	return nil, errors.New("can not get token-basic contract")
}

func (u *Upgrade1to2) getTokenIssueContract() (*std.Contract, error) {
	for _, v := range *u.getAllContract() {
		con := u.getContract(v)
		if con.Name == "token-issue" {
			return con, nil
		}
	}
	return nil, errors.New("can not get token-issue contract")
}

// init smc builder for build v2 contract
func (u *Upgrade1to2) initBuilder() (*smcbuilder.Builder, error) {
	logger := log.NewTMLogger("", "upgrade1to2-build")
	if runtime.GOOS == "windows" {
		ex, err := os.Executable()
		if err != nil {
			return nil, err
		}
		dir := filepath.Dir(ex)
		if dir == "" {
			return nil, errors.New("windows init builder failed")
		}
		smcbuilder.Init(logger, dir+"\\.build")
	} else {
		smcbuilder.Init(logger, os.Getenv("HOME")+"/.build")
	}

	d := dockerlib.GetDockerLib()
	prefix := u.getChainID() + "."
	d.Init(logger)
	d.SetPrefix(prefix)
	return smcbuilder.GetInstance(), nil
}

func (u *Upgrade1to2) updateValidatorPubKey() {
	allValidator := u.getAllValidator()
	for _, v := range allValidator {
		if len(v.NodePubKey) > 32 {
			v.NodePubKey = v.NodePubKey[5:]
			u.setValidator(v)
		}
	}
}

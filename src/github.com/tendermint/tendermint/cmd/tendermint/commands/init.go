package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tendermint/go-crypto"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/types"
	pvm "github.com/tendermint/tendermint/types/priv_validator"
	cmn "github.com/tendermint/tmlibs/common"
)

// InitFilesCmd initialises a fresh Tendermint Core instance.
var InitFilesCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Tendermint",
	Run:   initFiles,
}

func initFilesWithConfig(config *cfg.Config) error {
	// private validator
	privValFile := config.PrivValidatorFile()
	var pv *pvm.FilePV
	if cmn.FileExists(privValFile) {
		pv = pvm.LoadFilePV(privValFile)
		logger.Info("Found private validator", "path", privValFile)
	} else {
		pv = pvm.GenFilePV(privValFile)
		pv.Save()
		logger.Info("Generated private validator", "path", privValFile)
	}

	nodeKeyFile := config.NodeKeyFile()
	if cmn.FileExists(nodeKeyFile) {
		logger.Info("Found node key", "path", nodeKeyFile)
	} else {
		if _, err := p2p.LoadOrGenNodeKey(nodeKeyFile); err != nil {
			return err
		}
		logger.Info("Generated node key", "path", nodeKeyFile)
	}

	// genesis file
	genFile := config.GenesisFile()
	if cmn.FileExists(genFile) {
		logger.Info("Found genesis file", "path", genFile)
	} else {
		genDoc := types.GenesisDoc{
			ChainID: cmn.Fmt("test-chain-%v", cmn.RandStr(6)),
		}
		genDoc.Validators = []types.GenesisValidator{{
			PubKey: pv.GetPubKey(),
			Power:  10,
		}}

		if err := genDoc.SaveAs(genFile); err != nil {
			return err
		}
		logger.Info("Generated genesis file", "path", genFile)
	}

	return nil
}

var (
	genesisPath string
	chainID     string
	byzantium   string
	proxyApp    string
	aAddr       string
	listenPort  string
	listenPortN = 0
)

func AddInitFlags(cmd *cobra.Command) {
	cmd.Flags().String("chain_id", chainID, "Specify the chain ID")
	cmd.Flags().String("genesis_path", genesisPath, "Specify the path of genesis files")
	cmd.Flags().String("follow", byzantium, "Main nodes to follow, split by comma(only for follower)")
	cmd.Flags().String("proxy_app", proxyApp, "Gichain's ip address(only for follower)")
	cmd.Flags().String("listen_port", listenPort, "p2p listen port(only for follower)")
	cmd.Flags().String("a_address", aAddr, "ReAnnounce listen address(only for follower)")
}

func initFiles(cmd *cobra.Command, args []string) {
	var err error

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		for sig := range c {
			fmt.Println("TERM Signal received, exiting.... sig:", sig)
			os.Exit(0)
		}
	}()

	chainID, err = cmd.Flags().GetString("chain_id")
	if err != nil {
		fmt.Printf("init tendermint parse chain_id err: %s\n", err)
		return
	}
	genesisPath, err = cmd.Flags().GetString("genesis_path")
	if err != nil {
		fmt.Printf("init tendermint parse genesis_path err: %s\n", err)
		return
	}
	byzantium, err = cmd.Flags().GetString("follow")
	if err != nil {
		fmt.Printf("init tendermint parse follow err: %s\n", err)
		return
	}
	proxyApp, err = cmd.Flags().GetString("proxy_app")
	if err != nil {
		fmt.Printf("init tendermint parse proxyApp err: %s\n", err)
	}
	aAddr, err = cmd.Flags().GetString("a_address")
	if err != nil {
		fmt.Printf("init tendermint parse announced address err: %s\n", err)
	}
	listenPort, err = cmd.Flags().GetString("listen_port")
	if err != nil {
		fmt.Printf("init tendermint parse listen port err: %s\n", err)
	}
	if listenPort != "" {
		listenPortN, err = strconv.Atoi(listenPort)
		if err != nil {
			fmt.Printf("invalid listenPort: %s, err:%v\n", listenPort, err)
			os.Exit(1)
		}
	}

	if genesisPath != "" {
		if chainID == "" {
			fmt.Printf("init tendermint must use flag \"--genesis_path\" with \"--chain_id\"\n")
			return
		}
		if byzantium != "" {
			fmt.Printf("init tendermint flag \"--genesis_path\" conflict with \"--follow\"\n")
			return
		}
		genesisPath = filepath.Join(genesisPath, chainID)
	} else if byzantium == "" {
		fmt.Printf("init tendermint must use flag \"--genesis_path\" or \"--follow\"\n")
		return
	} else {
		genDoc := getGenDoc(byzantium)
		chainID = genDoc.ChainID
		genesisDir := filepath.Join(config.RootDir, "genesis")
		if _, err := os.Stat(genesisDir); os.IsNotExist(err) {
			if err = os.Mkdir(genesisDir, 0755); err != nil {
				fmt.Println("init tendermint make genesis dir error", err)
				return
			}
		}

		gzFile := filepath.Join(genesisDir, chainID+".tar.gz")
		tarFile := filepath.Join(genesisDir, chainID+".tar")
		pkg := getPkg(byzantium)
		err = ioutil.WriteFile(gzFile, pkg, 0755)
		if err != nil {
			fmt.Println("init tendermint write pkg file error", err)
			return
		}
		err = cmn.UnGzip(gzFile, genesisDir)
		if err != nil {
			fmt.Println("init tendermint unGzip pkg file error", err)
			return
		}
		err = cmn.UnTar(tarFile, genesisDir)
		if err != nil {
			fmt.Println("init tendermint unTar pkg file error", err)
			return
		}
		genesisPath = filepath.Join(genesisDir, chainID)
	}

	// copy files to config dir
	genPath := config.GenesisFile()

	err = types.CopyFile(genesisPath+"/"+chainID+"-genesis.json", genPath)
	if err != nil {
		fmt.Printf("copy file err: %s\n", err)
		return
	}
	err = types.CopyFile(genesisPath+"/"+chainID+"-genesis.json.sig", genPath[:len(genPath)-5]+".json.sig")
	if err != nil {
		fmt.Printf("copy file err: %s\n", err)
		return
	}
	err = types.CopyFile(genesisPath+"/"+chainID+"-validators.json", config.ValidatorsFile())
	if err != nil {
		fmt.Printf("copy file err: %s\n", err)
		return
	}
	// move any .tar.gz file to config directory, they are system contracts. (and maybe garbage :-)
	allFiles, err := ioutil.ReadDir(genesisPath)
	if err != nil {
		fmt.Printf("List Directory err: %s\n", err)
		return
	}
	for _, file := range allFiles {
		if !file.IsDir() &&
			(strings.HasSuffix(file.Name(), ".tar.gz") || file.Name() == "genesis-smart-contract-release-version.txt") {
			err = types.CopyFile(
				filepath.Join(genesisPath, file.Name()),
				filepath.Join(config.RootDir, "config", file.Name()),
			)
			if err != nil {
				fmt.Printf("copy file err: %s\n", err)
				return
			}
		}
	}

	genDoc, err := types.GenesisDocFromFile(config)
	if err != nil {
		fmt.Printf("tendermint can't parse genesis file, %v\n", err)
		return
	}
	if chainID != genDoc.ChainID {
		fmt.Printf("tendermint parsed chainid(%v) is not match path name(%v)\n", genDoc.ChainID, chainID)
		return
	}
	crypto.SetChainId(genDoc.ChainID)

	privValFile := config.PrivValidatorFile()
	var pv *pvm.FilePV
	if cmn.FileExists(privValFile) {
		_ = pvm.LoadFilePV(privValFile)
		logger.Info("Found private validator", "path", privValFile)
	} else {
		pv = pvm.GenFilePV(privValFile)
		pv.Save()
		logger.Info("Generated private validator", "path", privValFile)
	}

	nodeKeyFile := config.NodeKeyFile()
	if cmn.FileExists(nodeKeyFile) {
		logger.Info("Found node key", "path", nodeKeyFile)
	} else {
		if _, err = p2p.LoadOrGenNodeKey(nodeKeyFile); err != nil {
			fmt.Printf("init tendermint parse node_list err: %s\n", err)
			return
		}
		logger.Info("Generated node key", "path", nodeKeyFile)
	}

	if byzantium == "" {
		nodeListFilename := filepath.Join(genesisPath, chainID+"-nodes.json")
		ProcessP2P(*genDoc, nodeListFilename, proxyApp)
	} else {
		ProcessFollower(byzantium, proxyApp, aAddr, listenPortN)
	}

}

type GenesisDoc map[string]json.RawMessage

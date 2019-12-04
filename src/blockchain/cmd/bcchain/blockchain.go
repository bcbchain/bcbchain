package main

import (
	bcchain "blockchain/abciapp/app"
	"blockchain/abciapp/common"
	"blockchain/abciapp/version"
	"blockchain/smcrunctl/invokermgr"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tendermint/abci/server"
	cmn "github.com/tendermint/tmlibs/common"
	tmlog "github.com/tendermint/tmlibs/log"
)

var (
	logger tmlog.Loggerf
)

//RootCmd root cmd
var RootCmd = &cobra.Command{
	Use:   "bcchain",
	Short: "ABCI application",
	Long:  "ABCI application",
}

//Execute starting to execute progress
func Execute() error {
	readConfiguration()
	addCommands()
	addFlags()
	return RootCmd.Execute()
}

//读取配置文件 失败程序将直接终止
func readConfiguration() {
	err := common.GlobalConfig.GetConfig()
	if err != nil {
		panic(err)
	}
}
func addCommands() {
	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(startCmd)
	RootCmd.AddCommand(resetCmd)
}

var (
	debug bool
)

func addFlags() {
	startCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "run mode of debug flag")
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version info",
	Long:  "Show version info",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(version.Version)
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Run the ABCI application",
	Long:  "Run the ABCI application",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdStart(cmd, args)
	},
}

var resetCmd = &cobra.Command{
	Use:   "unsafe_reset_all",
	Short: "(unsafe) Remove all the data",
	Long:  "(unsafe) Remove all the data",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdReset(cmd, args)
	},
}

//cmdReset 清除状态数据库
func cmdReset(cmd *cobra.Command, args []string) error {
	home := os.Getenv("HOME")
	logger = tmlog.NewTMLogger(filepath.Join(home, "log"), "bcchain")
	resetAll(common.GlobalConfig.DBName+".db", logger)
	return nil
}

//cmdStart 程序唯一启动方式
func cmdStart(cmd *cobra.Command, args []string) error {

	home := os.Getenv("HOME")
	logger = tmlog.NewTMLogger(filepath.Join(home, "log"), "bcchain")
	logger.AllowLevel(common.GlobalConfig.LogLevel)
	logger.SetOutputAsync(common.GlobalConfig.LogAsync)
	logger.SetOutputToFile(common.GlobalConfig.LogFile)
	logger.SetOutputToScreen(common.GlobalConfig.LogScreen)
	logger.SetOutputFileSize(common.GlobalConfig.LogSize)

	// Create the application
	app := bcchain.NewBCChainApplication(common.GlobalConfig, logger)

	// upgrade contract binary executable file
	if filePath, exist := common.IsExistUpgradeFile(); exist {
		common.UpgradeBin(filePath, logger)
	}

	// Start the listener
	srv, err := server.NewServer(common.GlobalConfig.Address, common.GlobalConfig.ABCI, app)
	if err != nil {
		return err
	}

	srv.SetLogger(logger)
	if err := srv.Start(); err != nil {
		return err
	}

	invokermgr.Debug = debug
	// Wait forever
	//nolint
	cmn.TrapSignal(func(signal os.Signal) {
		srv.Stop()
	})

	logger.Flush()
	return nil
}

func resetAll(dbDir string, logger tmlog.Loggerf) {
	if err := os.RemoveAll(dbDir); err != nil {
		logger.Error("Error removing directory", "err", err)
		return
	}
	logger.Info("Removed all data", "dir", dbDir)
}

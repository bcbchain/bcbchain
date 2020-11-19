package main

import (
	"fmt"
	bcchain "github.com/bcbchain/bcbchain/abciapp/app"
	"github.com/bcbchain/bcbchain/abciapp/common"
	"github.com/bcbchain/bcbchain/smcdocker"
	"github.com/bcbchain/bcbchain/version"
	"github.com/bcbchain/bclib/tendermint/abci/server"
	cmn "github.com/bcbchain/bclib/tendermint/tmlibs/common"
	tmlog "github.com/bcbchain/bclib/tendermint/tmlibs/log"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
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
	RootCmd.AddCommand(initCmd)
	RootCmd.AddCommand(rollbackCmd)
}

var (
	debug     bool
	followURL string
	rollBack  int
	dbDir     string
)

func addFlags() {
	startCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "run mode of debug flag")
	initCmd.PersistentFlags().StringVarP(&followURL, "follow", "f", "", "Main nodes to follow, split by comma(only for follower)")
	rollbackCmd.PersistentFlags().IntVarP(&rollBack, "rollback", "r", 1, "rollback to dest")
	rollbackCmd.PersistentFlags().StringVarP(&dbDir, "dbDir", "d", "", "levelDB dir")
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

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize bcchain",
	Long:  "Initialize bcchain",
	Args:  cobra.ExactArgs(0),
	Run:   initFiles,
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "rollback appState",
	Long:  "rollback appState to dest",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return rollback(cmd, args)
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
	//logger.AllowLevel(common.GlobalConfig.LogLevel)
	logger.AllowLevel("debug")
	logger.SetOutputAsync(common.GlobalConfig.LogAsync)
	logger.SetOutputToFile(common.GlobalConfig.LogFile)
	logger.SetOutputToScreen(common.GlobalConfig.LogScreen)
	logger.SetOutputFileSize(common.GlobalConfig.LogSize)
	defer logger.Flush()

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

	smcdocker.Debug = debug
	// Wait forever
	//nolint
	cmn.TrapSignal(func(signal os.Signal) {
		srv.Stop()
	})

	return nil
}

func resetAll(dbDir string, logger tmlog.Loggerf) {
	if err := os.RemoveAll(dbDir); err != nil {
		logger.Error("Error removing directory", "err", err)
		return
	}
	logger.Info("Removed all data", "dir", dbDir)
}

package commands

import (
	"errors"
	"github.com/tendermint/tendermint/sidechain"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tmlibs/cli"
	tmflags "github.com/tendermint/tmlibs/cli/flags"
	"github.com/tendermint/tmlibs/log"
)

var (
	config = cfg.DefaultConfig()
	logger = (log.Logger)(nil)
	output = (*os.File)(nil)
)

func init() {
	registerFlagsRootCmd(RootCmd)
	sidechain.ConfigPathFunc = GetConfigFiles
}

func registerFlagsRootCmd(cmd *cobra.Command) {
	cmd.PersistentFlags().String("log_level", config.LogLevel, "Log level")
	// For log customization, to support Log file
	cmd.PersistentFlags().String("log_file", config.LogFile, "Log file")
}

// ParseConfig retrieves the default environment configuration,
// sets up the Tendermint root and ensures that the root exists
func ParseConfig(isInit bool) (*cfg.Config, error) {
	conf := cfg.DefaultConfig()
	confStat, err0 := os.Stat(conf.ConfigFilePath())
	genStat, err1 := os.Stat(conf.GenesisFile())
	if err0 == nil && confStat.Mode().IsRegular() && err1 == nil && genStat.Mode().IsRegular() {
		err := viper.Unmarshal(conf)
		if err != nil {
			return nil, err
		}
		return conf, nil
	}

	tmHome := os.Getenv("TMHOME")
	conf.SetRoot(tmHome)
	confStat, err0 = os.Stat(conf.ConfigFilePath())
	genStat, err1 = os.Stat(conf.GenesisFile())
	if err0 == nil && confStat.Mode().IsRegular() && err1 == nil && genStat.Mode().IsRegular() {
		err := viper.Unmarshal(conf)
		if err != nil {
			return nil, err
		}
		return conf, nil
	}

	pwd, err := os.Getwd()
	if err == nil {
		conf.SetRoot(pwd)
		confStat, err0 = os.Stat(conf.ConfigFilePath())
		genStat, err1 = os.Stat(conf.GenesisFile())
		if err0 == nil && confStat.Mode().IsRegular() && err1 == nil && genStat.Mode().IsRegular() {
			err := viper.Unmarshal(conf)
			if err != nil {
				return nil, err
			}
			return conf, nil
		}
	}

	usr, err := user.Current()
	if err == nil {
		conf.SetRoot(filepath.Join(usr.HomeDir, ".tendermint"))
		confStat, err0 = os.Stat(conf.ConfigFilePath())
		genStat, err1 = os.Stat(conf.GenesisFile())
		if err0 == nil && confStat.Mode().IsRegular() && err1 == nil && genStat.Mode().IsRegular() {
			err := viper.Unmarshal(conf)
			if err != nil {
				return nil, err
			}
			return conf, nil
		}
	}

	if !isInit {
		return nil, errors.New("you must init genesis")
	} else {
		if tmHome != "" {
			conf.SetRoot(tmHome)
		} else {
			conf.SetRoot(filepath.Join(usr.HomeDir, ".tendermint"))
		}
		cfg.EnsureRoot(conf.RootDir)
		return conf, nil
	}

}

// RootCmd is the root command for Tendermint core.
var RootCmd = &cobra.Command{
	Use:   "tendermint",
	Short: "Tendermint Core (BFT Consensus) in Go",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		if cmd.Name() == VersionCmd.Name() {
			return nil
		}

		config, err = ParseConfig(cmd == InitFilesCmd)
		if err != nil {
			return err
		}
		if len(config.LogFile) > 0 {
			output, err = os.OpenFile(config.LogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
			if err != nil {
				return err
			}
		} else {
			output = os.Stdout
		}
		logger1 := log.NewTMLogger(config.LogDir(), "tmcore")
		logger1.SetOutputAsync(true)
		logger1.SetWithThreadID(true)
		logger1.AllowLevel("debug")

		logger = logger1

		logger, err = tmflags.ParseLogLevel(config.LogFile, config.LogLevel, logger, cfg.DefaultLogLevel())
		if err != nil {
			return err
		}
		if viper.GetBool(cli.TraceFlag) {
			logger = log.NewTracingLogger(logger)
		}
		logger = logger.With("module", "main")
		return nil
	},
}

func GetConfig() *cfg.Config {
	return config
}

func GetConfigFiles() (string, string, string, string, string) {
	return config.GenesisFile(), config.ConfigFilePath(), config.DBDir(), config.ValidatorsFile(), config.PrivValidatorFile()
}

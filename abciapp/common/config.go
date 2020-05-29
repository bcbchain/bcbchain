package common

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

//GlobalConfig declares to global
var GlobalConfig Config
var TmCoreURL string

//Config 具体含义请参考 bcchain.yaml
type Config struct {
	Address          string `yaml:"address"`        //default "tcp://127.0.0.1:46658"
	QueryDBAddress   string `yaml:"queryDBAddress"` //default "0.0.0.0:46666"
	ABCI             string `yaml:"abci"`           //default "socket"
	LogLevel         string `yaml:"logLevel"`       //default "debug"
	LogScreen        bool   `yaml:"logScreen"`
	LogFile          bool   `yaml:"logFile"`
	LogAsync         bool   `yaml:"logAsync"`
	LogSize          int    `yaml:"logFileSize"`
	DBName           string `yaml:"dbName"`
	DBIP             string `yaml:"dbIP"`
	DBPort           string `yaml:"dbPort"`
	ChainID          string `yaml:"chainID"`
	ContainerTimeout int64  `yaml:"containerTimeout"`
	Path             string
}

//GetConfig read config to struct
//nolint
func (c *Config) GetConfig() error {
	tmHome := os.Getenv("TMHOME")

	configPaths := []string{
		"/etc/bcchain",
		"$HOME/.config",
		".config",
	}
	if tmHome != "" {
		configPaths = append(configPaths, tmHome)
	}

	viper.SetConfigName("bcchain") // name of config file (without extension)
	viper.SetConfigType("yaml")    // specify ConfigType("YAML")
	// path to look for the config file in
	// call multiple times to add many search paths
	// optionally look for config in the working directory
	for _, configPath := range configPaths {
		viper.AddConfigPath(configPath)

		if _, err := os.Stat(configPath + "/bcchain.yaml"); err == nil {
			c.Path = configPath
		}
	}

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {
		fmt.Println("yamlFile.Get err #%v", err)
		return err
	}

	err = viper.Unmarshal(c)
	if err != nil {
		fmt.Println("Unmarshal error :", err)
		return err
	}
	if len(c.Address) == 0 {
		c.Address = "tcp://127.0.0.1:46658"
	}
	if len(c.QueryDBAddress) == 0 {
		c.Address = "0.0.0.0:46666"
	}
	if len(c.ABCI) == 0 || c.ABCI == "" {
		c.ABCI = "socket"
	}
	if len(c.DBName) == 0 || c.DBName == "" {
		c.DBName = ".appstate"
	}
	if c.ChainID == "" {
		c.ChainID = "local"
	}
	return nil
}

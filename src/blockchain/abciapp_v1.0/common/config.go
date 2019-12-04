package common

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
)

var GlobalConfig Config

//具体含义请参考 bcchain.yaml
type Config struct {
	Address          string `yaml:"address"`          //default "tcp://127.0.0.1:46658"
	Query_DB_Address string `yaml:"query_db_address"` //default "0.0.0.0:46666"
	Abci             string `yaml:"abci"`             //default "socket"
	Log_level        string `yaml:"log_level"`        //default "debug"
	Log_screen       bool   `yaml:"log_screen"`
	Log_file         bool   `yaml:"log_file"`
	Log_async        bool   `yaml:"log_async"`
	Log_size         int    `yaml:"log_file_size"`
	DB_name          string `yaml:"db_name"`
	DB_ip            string `yaml:"db_ip"`
	DB_port          string `yaml:"db_port"`
	Chain_id         string `yaml:"chain_id"`
}

func (c *Config) GetConfig() error {
	tmHome := os.Getenv("TMHOME")

	viper.SetConfigName("bcchain")       // name of config file (without extension)
	viper.SetConfigType("yaml")          // specify ConfigType("YAML")
	viper.AddConfigPath("/etc/bcchain")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.config") // call multiple times to add many search paths
	if tmHome != "" {
		viper.AddConfigPath(tmHome)
	}
	viper.AddConfigPath(".config") // optionally look for config in the working directory

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
	if len(c.Query_DB_Address) == 0 {
		c.Address = "0.0.0.0:46666"
	}
	if len(c.Abci) == 0 || c.Abci == "" {
		c.Abci = "socket"
	}
	if len(c.DB_name) == 0 || c.DB_name == "" {
		c.DB_name = ".appstate"
	}
	if c.Chain_id == "" {
		c.Chain_id = "local"
	}

	return nil
}

package main

import (
	"github.com/bcbchain/bcbchain/abciapp/common"
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"fmt"
	"github.com/spf13/cobra"
	"path"
)

//rollback 状态库回滚，最多回滚100区块
func rollback(cmd *cobra.Command, args []string) error {

	rollbackNum, err := cmd.Flags().GetInt("rollback")
	if err != nil {
		fmt.Printf("rollback bcchain parse rollbackNum err: %s\n", err)
		return err
	}

	dbDir, err := cmd.Flags().GetString("dbDir")
	if err != nil {
		fmt.Printf("rollback bcchain parse dbDir err: %s\n", err)
		return err
	}

	dbPath := path.Join(dbDir, common.GlobalConfig.DBName)

	statedbhelper.Init(dbPath, 100)

	statedbhelper.RollbackStateDB(rollbackNum)

	fmt.Println(statedbhelper.GetWorldAppState(0, 0))

	return nil
}

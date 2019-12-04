package commands

import (
	"fmt"
	"strconv"

	"github.com/tendermint/tendermint/state"

	"github.com/spf13/cobra"

	nm "github.com/tendermint/tendermint/node"
)

var (
	syncStr = ""
)

func AddSyncFlags(cmd *cobra.Command) {
	cmd.Flags().String("to", syncStr, "Sync to block height number")
}

func NewSyncNodeCmd(nodeProvider nm.NodeProvider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync the tendermint node",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			syncStr, err = cmd.Flags().GetString("to")
			if err != nil {
				fmt.Printf("sync tendermint parse flags err: %s\n", err)
				return err
			}
			if syncStr != "" {
				state.SyncTo, err = strconv.ParseInt(syncStr, 10, 64)
				if err != nil {
					fmt.Printf("sync tendermint parse flags err: %s\n", err)
					return err
				}
			}

			n, err := nodeProvider(config, logger)
			if err != nil {
				return fmt.Errorf("failed to create node: %v", err)
			}

			if err := n.Start(); err != nil {
				return fmt.Errorf("failed to sync node: %v", err)
			}
			logger.Info("Syncing node", "nodeInfo", n.Switch().NodeInfo())

			n.RunForever()

			return nil
		},
	}

	AddSyncFlags(cmd)

	return cmd
}

package commands

import (
	"encoding/hex"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/tendermint/types"

	"github.com/tendermint/tendermint/types/priv_validator"
)

// ShowValidatorCmd adds capabilities for showing the validator info.
var ShowValidatorCmd = &cobra.Command{
	Use:   "show_validator",
	Short: "Show this node's validator info",
	Run:   showValidator,
}

func showValidator(cmd *cobra.Command, args []string) {
	genDoc, err := types.GenesisDocFromFile(config)
	if err != nil {
		logger.Error("tendermint can't parse genesis file", "parse", err)
		return
	}
	crypto.SetChainId(genDoc.ChainID)

	privValidator := privval.LoadOrGenFilePV(config.PrivValidatorFile())
	pubKey := privValidator.GetPubKey()
	pubKeyEd := pubKey.(crypto.PubKeyEd25519)
	pubKeyJSONBytes, _ := cdc.MarshalJSON(pubKey)
	fmt.Println(string(pubKeyJSONBytes))
	fmt.Println(hex.EncodeToString(pubKeyEd[:]))
}

package tokenbyb

import (
	"blockchain/abciapp_v1.0/smc"
	"blockchain/abciapp_v1.0/types"
	"common/bignumber_v1.0"
	"encoding/json"
	"math/big"
)

// Receipt for transfer
type bybTransferReceipt struct {
	Token      smc.Address  `json:"token"`      // Token Address
	From       smc.Address  `json:"from"`       // External account address of Sender
	To         smc.Address  `json:"to"`         // External account address of Receiver
	BybBalance []bybBalance `json:"bybBalance"` // Transfer value
}

func (byb *TokenByb) bybReceipt_onTransfer(from, to smc.Address, transByb []bybBalance) {

	// The standard receipt of transfer
	totalBalance := big.NewInt(0)
	for _, b := range transByb {
		*totalBalance = bignumber.Add(*totalBalance, b.Value)
	}
	byb.EventHandler.PackReceiptOfTransfer(byb.State.ContractAddress, from, to, bignumber.NB(totalBalance))

	// The special receipt of byb transfer
	receipt := bybTransferReceipt{
		Token:      byb.State.ContractAddress,
		From:       from,
		To:         to,
		BybBalance: transByb,
	}

	resBytes, _ := json.Marshal(receipt)

	byb.EventHandler.EmitReceipt("bybTransfer", resBytes)
}

func (byb *TokenByb) bybReceipt_onInitToken(token *types.IssueToken, accountAddress smc.Address) {

	byb.EventHandler.PackReceiptOfNewToken(token, accountAddress)
}

func (byb *TokenByb) bybReceipt_onNewBlackHole(addr smc.Address) {

	type bybInitReceipt struct {
		Address string `json:"address"`
	}

	receipt := bybInitReceipt{
		Address: addr,
	}

	resBytes, _ := json.Marshal(receipt)

	byb.EventHandler.EmitReceipt("newBlackHole", resBytes)
}

func (byb *TokenByb) bybReceipt_onNewStockHolder(addr smc.Address, bybt []bybBalance) {

	type bybInitReceipt struct {
		Address    string       `json:"address"`
		BybBalance []bybBalance `json:"bybBalance"`
	}

	receipt := bybInitReceipt{
		Address:    addr,
		BybBalance: bybt,
	}

	resBytes, _ := json.Marshal(receipt)

	byb.EventHandler.EmitReceipt("newStockHolder", resBytes)
}

func (byb *TokenByb) bybReceipt_onDelStockHolder(addr smc.Address) {

	type bybInitReceipt struct {
		Address string `json:"address"`
	}

	receipt := bybInitReceipt{
		Address: addr,
	}

	resBytes, _ := json.Marshal(receipt)

	byb.EventHandler.EmitReceipt("delStockHolder", resBytes)
}

func (byb *TokenByb) bybReceipt_onChangeChromoOwnership(from, to smc.Address, bybt []bybBalance) {
	//transfer receipt
	byb.bybReceipt_onTransfer(from, to, bybt)

	type bybInitReceipt struct {
		FromStockHolder string       `json:"fromStockHolder"`
		ToStockHolder   string       `json:"toStockHolder"`
		BybBalance      []bybBalance `json:"bybBalance"`
	}

	receipt := bybInitReceipt{
		FromStockHolder: from,
		ToStockHolder:   to,
		BybBalance:      bybt,
	}

	resBytes, _ := json.Marshal(receipt)

	byb.EventHandler.EmitReceipt("changeChromoOwnership", resBytes)
}

func (byb *TokenByb) bybReceipt_onSetOwner(addr smc.Address) {
	byb.EventHandler.PackReceiptOfSetOwner(byb.State.ContractAddress, addr)
}

func (byb *TokenByb) bybReceipt_onAddSupply(value, totalSupply bignumber.Number) {
	//The standard receipt
	byb.EventHandler.PackReceiptOfAddSupply(byb.State.ContractAddress, value, totalSupply)
}

func (byb *TokenByb) bybReceipt_onBurn(value, totalSupply bignumber.Number) {
	//The standard receipt
	byb.EventHandler.PackReceiptOfBurn(byb.State.ContractAddress, value, totalSupply)
}

func (byb *TokenByb) bybReceipt_onSetGasPrice(gasPrice uint64) {

	byb.EventHandler.PackReceiptOfSetGasPrice(byb.State.ContractAddress, gasPrice)
}

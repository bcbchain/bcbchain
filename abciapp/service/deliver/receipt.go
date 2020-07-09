package deliver

import (
	"github.com/bcbchain/bclib/tendermint/abci/types"
	abci "github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
)

type ReceiptParser struct {
	receiptChan  chan []common.KVPair // 接收收据。
	endBlockChan chan struct{}        // 所有交易 deliver 结束，等待收据解析结果。

	ResponseChan chan ReceiptResponse // end block 时从此 chan 获取 response。
	SubChans     []chan ReceiptResponse
}

type ReceiptResponse struct {
	HasValidatorUpdate bool
	Validators         []types.Validator

	HasSideChainGenesis bool
	SCGenesisInfo       []*abci.SideChainGenesis
}

func (rp *ReceiptParser) Subscribe(c chan ReceiptResponse) {
	if c == nil {
		panic("can not be nil chan")
	}
	rp.SubChans = append(rp.SubChans, c)
}

// ReceiptsRoutine parse deliver receipts routine.
func (rp *ReceiptParser) ReceiptsRoutine() {

	for {
		select {
		case receipt := <-rp.receiptChan:
			rp.parseReceipt(receipt)

		case <-rp.endBlockChan:
			for {
				// todo 确保 receiptChan 中所有的收据都解析完。
				receipt, ok := <-rp.receiptChan
				if ok {
					rp.parseReceipt(receipt)
				} else {
					break
				}
			}

			rp.pubReceiptResponse()
		}
	}
}

func (rp *ReceiptParser) parseReceipt(receipt []common.KVPair) {

}

func (rp *ReceiptParser) pubReceiptResponse() {
	// todo send to ResponseChan
}

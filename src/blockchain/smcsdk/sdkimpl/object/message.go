package object

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/crypto/sha3"
	"blockchain/smcsdk/sdk/jsoniter"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"blockchain/smcsdk/sdkimpl"
	"bytes"
	"fmt"
	"strings"
)

// Message message detail information
type Message struct {
	smc sdk.ISmartContract //指向智能合约API对象指针

	contract       sdk.IContract    //消息调用的智能合约地址
	methodID       string           //消息调用的智能合约方法ID
	items          []types.HexBytes //消息的数据字段的原始信息（包括方法ID及参数）
	gasPrice       int64            //消息的燃料价格
	sender         sdk.IAccount     //消息发送者的账户信息
	payer          sdk.IAccount     //支付手续费的账户信息
	origins        []types.Address  //消息完整的调用链（用于记录跨合约调用的合约链）
	inputReceipts  []types.KVPair   //级联消息中前一个消息输出的收据作为本次消息的输入
	outputReceipts []types.KVPair   //级联消息中的输出收据
}

var _ sdk.IMessage = (*Message)(nil)
var _ sdkimpl.IAcquireSMC = (*Message)(nil)

// SMC get smart contract object
func (m *Message) SMC() sdk.ISmartContract { return m.smc }

// SetSMC set smart contract object
func (m *Message) SetSMC(smc sdk.ISmartContract) { m.smc = smc }

// To get message's smart contract address
func (m *Message) Contract() sdk.IContract { return m.contract }

// MethodID get message's methodID
func (m *Message) MethodID() string { return m.methodID }

// Data get message's data
func (m *Message) Items() []types.HexBytes { return m.items }

// GasPrice get message's gasPrice
func (m *Message) GasPrice() int64 { return m.gasPrice }

// Sender get message's sender
func (m *Message) Sender() sdk.IAccount { return m.sender }

// Payer get account for pay fee
func (m *Message) Payer() sdk.IAccount { return m.payer }

// Origin get message's origin
func (m *Message) Origins() []types.Address { return m.origins }

// InputReceipts get message's inputReceipts
func (m *Message) InputReceipts() []types.KVPair { return m.inputReceipts }

// OutputReceipts get message's outputReceipts
func (m *Message) OutputReceipts() []types.KVPair { return m.outputReceipts }

// SetContract set value of contract
func (m *Message) SetContract(v sdk.IContract) { m.contract = v }

// FillOutputReceipts fill receipt to output receipt's list
func (m *Message) FillOutputReceipts(receipt types.KVPair) {
	if cap(m.outputReceipts) == 0 {
		m.outputReceipts = make([]types.KVPair, 0)
	}
	m.outputReceipts = append(m.outputReceipts, receipt)
}

// AppendOutput append receipts from new message to origin message
func (m *Message) AppendOutput(receipts []types.KVPair) {
	// AppendOutput Receipts
	for _, r := range receipts {
		keySuffix := string(r.Key[1:])[strings.Index(string(r.Key)[1:], "/")+1:]
		nr := types.KVPair{Key: []byte(fmt.Sprintf("/%d/%s", len(m.OutputReceipts()), keySuffix)), Value: r.Value}
		m.outputReceipts = append(m.outputReceipts, nr)
	}
}

// GetTransferToMe parse receipt that it's transfer receipt
func (m *Message) GetTransferToMe() (transferReceipts []*std.Transfer) {

	transferReceipts = make([]*std.Transfer, 0)
	for _, v := range m.inputReceipts {
		transferReceipt := m.parseToTransfer(v.Value)
		if transferReceipt != nil &&
			transferReceipt.To == m.smc.Message().Contract().Account().Address() {
			transferReceipts = append(transferReceipts, transferReceipt)
		}
	}

	return
}

// parseToTransfer get transfer receipt and parse it
func (m *Message) parseToTransfer(value []byte) *std.Transfer {
	if value == nil {
		return nil
	}

	rpt := std.Receipt{}
	err := jsoniter.Unmarshal(value, &rpt)
	if err != nil {
		sdkimpl.Logger.Errorf("[sdk]Cannot unmarshal receipt data=%v", value)
		return nil
	}

	// check hash
	calcHash := sha3.Sum256([]byte(rpt.Name), []byte(rpt.ContractAddr), rpt.Bytes)
	if bytes.Compare(calcHash, rpt.Hash) != 0 {
		sdkimpl.Logger.Errorf("[sdk]Hash not right calcHash=%v,valueHash=%v", calcHash, rpt.Hash)
		return nil
	}

	if rpt.Name == "std::transfer" {
		var v std.Transfer
		err = jsoniter.Unmarshal(rpt.Bytes, &v)
		if err != nil {
			sdkimpl.Logger.Errorf("[sdk]Cannot unmarshal transfer receipt data=%v", rpt.Bytes)
			return nil
		}
		return &v
	}
	return nil
}

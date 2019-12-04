package evm

import (
	"github.com/hyperledger/burrow/execution/errors"
	"github.com/hyperledger/burrow/execution/exec"
	"github.com/tendermint/tmlibs/log"
)

type EventSink interface {
	Call(call *exec.CallEvent, exception *errors.Exception) error
	CallTransfer(call *exec.TransferData, exception *errors.Exception) error
	Log(log *exec.LogEvent) error
}

type BcEventSink struct {
	Logger log.Logger
	Tags   *[]interface{}
}

func NewBcEventSink(Logger log.Logger, tags *[]interface{}) *BcEventSink {
	return &BcEventSink{Logger: Logger, Tags: tags}
}

func (es *BcEventSink) Call(call *exec.CallEvent, exception *errors.Exception) error {
	es.Logger.Debug("evmCallEvent:", call)
	es.Logger.Debug("evmException:", exception)

	// CallEvent
	*es.Tags = append(*es.Tags, call)
	// Exception
	if exception != nil {
		*es.Tags = append(*es.Tags, exception)
	}

	return nil
}

func (es *BcEventSink) CallTransfer(transfer *exec.TransferData, exception *errors.Exception) error {
	es.Logger.Debug("evmTransferEvent:", transfer)
	es.Logger.Debug("evmException:", exception)

	// CallEvent
	*es.Tags = append(*es.Tags, transfer)
	// Exception
	if exception != nil {
		*es.Tags = append(*es.Tags, exception)
	}

	return nil
}

func (es *BcEventSink) Log(log *exec.LogEvent) error {
	es.Logger.Debug("evmLogEvent:", log)

	// LogEvent
	*es.Tags = append(*es.Tags, log)

	return nil
}

type logFreeEventSink struct {
	EventSink
}

func NewLogFreeEventSink(eventSink EventSink) *logFreeEventSink {
	return &logFreeEventSink{
		EventSink: eventSink,
	}
}

func (esc *logFreeEventSink) Log(log *exec.LogEvent) error {
	return errors.ErrorCodef(errors.ErrorCodeIllegalWrite,
		"Log emitted from contract %v, but current call should be log-free", log.Address)
}

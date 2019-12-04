package types

import (
	"fmt"
	"strings"

	"github.com/tendermint/go-crypto"
	cmn "github.com/tendermint/tmlibs/common"
)

// Volatile state for each Validator
// NOTE: The Accum is not included in Validator.Hash();
// make sure to update that method if changes are made here
type Validator struct {
	Address     crypto.Address `json:"address"`
	PubKey      crypto.PubKey  `json:"pub_key"`
	VotingPower uint64         `json:"voting_power"`
	RewardAddr  crypto.Address `json:"reward_addr"`
	Name        string         `json:"name"`

	Accum int64 `json:"accum"`
}

func NewValidator(pubKey crypto.PubKey, votingPower uint64, reward crypto.Address, name string) *Validator {
	return &Validator{
		Address:     pubKey.Address(crypto.GetChainId()),
		PubKey:      pubKey,
		VotingPower: votingPower,
		RewardAddr:  reward,
		Name:        name,
		Accum:       0,
	}
}

// Creates a new copy of the validator so we can mutate accum.
// Panics if the validator is nil.
func (v *Validator) Copy() *Validator {
	vCopy := *v
	return &vCopy
}

// Returns the one with higher Accum.
func (v *Validator) CompareAccum(other *Validator) *Validator {
	if other == nil {
		return v
	}
	if v.Accum > other.Accum {
		return v
	} else if v.Accum < other.Accum {
		return other
	} else {
		result := strings.Compare(v.Address, other.Address)
		if result < 0 {
			return v
		} else if result > 0 {
			return other
		} else {
			cmn.PanicSanity("Cannot compare identical validators")
			return nil
		}
	}
}

func (v *Validator) String() string {
	if v == nil {
		return "nil-Validator"
	}
	return fmt.Sprintf("Validator{%v %v VP:%v A:%v RW:%v N:%v}",
		v.Address,
		v.PubKey,
		v.VotingPower,
		v.Accum,
		v.RewardAddr,
		v.Name)
}

// Hash computes the unique ID of a validator with a given voting power.
// It excludes the Accum value, which changes with every round.
func (v *Validator) Hash() []byte {
	return aminoHash(struct {
		Address     crypto.Address
		PubKey      crypto.PubKey
		VotingPower uint64
		RewardAddr  crypto.Address
		Name        string
	}{
		v.Address,
		v.PubKey,
		v.VotingPower,
		v.RewardAddr,
		v.Name,
	})
}

//----------------------------------------
// RandValidator

// RandValidator returns a randomized validator, useful for testing.
// UNSTABLE
func RandValidator(randPower bool, minPower int64) (*Validator, PrivValidator) {
	privVal := NewMockPV()
	votePower := minPower
	if randPower {
		votePower += int64(cmn.RandUint32())
	}
	val := NewValidator(privVal.GetPubKey(), uint64(votePower), "", "")
	return val, privVal
}

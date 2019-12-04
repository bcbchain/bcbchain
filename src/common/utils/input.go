package utils

import (
	"bufio"
	"github.com/bgentry/speakeasy"
	"github.com/pkg/errors"
	"os"
)

// MinPassLength is the minimum acceptable password length
const (
	MinPassLength = 8
	MaxPassLength = 20
)

// GetPassword will prompt for a password one-time (to sign a tx)
// It enforces the password length
func getPassword(prompt string, buf *bufio.Reader) (pass string, err error) {
	pass, err = speakeasy.Ask(prompt)
	if err != nil {
		return "", err
	}
	if len(pass) < MinPassLength {
		return "", errors.Errorf("Password must be at least %d characters", MinPassLength)
	}

	if len(pass) > MaxPassLength {
		return "", errors.Errorf("Password must be at most %d characters", MaxPassLength)
	}

	return pass, nil
}

// GetAndCheckPassword will prompt for a password twice to verify they
// match (for creating a new password).
// It enforces the password length. Only parses password once if
// input is piped in.
func GetAndCheckPassword(prompt1, prompt2 string) ([]byte, error) {
	buf := bufio.NewReader(os.Stdin)
	pass1, err := getPassword(prompt1, buf)
	if err != nil {
		return nil, err
	}
	pass2, err := getPassword(prompt2, buf)
	if err != nil {
		return nil, err
	}
	if pass1 != pass2 {
		return nil, errors.New("Password don't match")
	}
	return []byte(pass1), nil
}

func CheckPassword(prompt1 string) ([]byte, error) {
	buf := bufio.NewReader(os.Stdin)
	pass1, err := getPassword(prompt1, buf)
	if err != nil {
		return nil, err
	}
	return []byte(pass1), nil
}

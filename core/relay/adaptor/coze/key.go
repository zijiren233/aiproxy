package coze

import (
	"errors"
	"strings"

	"github.com/labring/aiproxy/core/relay/adaptor"
)

var _ adaptor.KeyValidator = (*Adaptor)(nil)

func (a *Adaptor) ValidateKey(key string) error {
	_, _, err := getTokenAndUserID(key)
	if err != nil {
		return err
	}

	return nil
}

func getTokenAndUserID(key string) (string, string, error) {
	split := strings.Split(key, "|")
	if len(split) != 2 {
		return "", "", errors.New("invalid key format")
	}

	return split[0], split[1], nil
}

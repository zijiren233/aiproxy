package openai

import (
	"encoding/hex"

	"github.com/google/uuid"
	"github.com/labring/aiproxy/common/conv"
)

func shortUUID() string {
	var buf [32]byte
	bytes := uuid.New()
	hex.Encode(buf[:], bytes[:])
	return conv.BytesToString(buf[:])
}

func ChatCompletionID() string {
	return "chatcmpl-" + shortUUID()
}

func CallID() string {
	return "call_" + shortUUID()
}

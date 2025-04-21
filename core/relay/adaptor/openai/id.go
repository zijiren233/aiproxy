package openai

import "github.com/labring/aiproxy/core/common"

func ChatCompletionID() string {
	return "chatcmpl-" + common.ShortUUID()
}

func CallID() string {
	return "call_" + common.ShortUUID()
}

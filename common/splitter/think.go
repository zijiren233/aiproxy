package splitter

import "github.com/labring/aiproxy/common/conv"

const (
	ThinkHead = "<think>\n"
	ThinkTail = "</think>\n"
)

var (
	thinkHeadBytes = conv.StringToBytes(ThinkHead)
	thinkTailBytes = conv.StringToBytes(ThinkTail)
)

func NewThinkSplitter() *Splitter {
	return NewSplitter(thinkHeadBytes, thinkTailBytes)
}

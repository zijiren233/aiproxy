package doubaoaudio

import "github.com/labring/aiproxy/core/relay/adaptor"

var _ adaptor.Features = (*Adaptor)(nil)

func (a *Adaptor) Features() []string {
	return []string{
		"https://www.volcengine.com/docs/6561/1257543",
		"TTS support",
	}
}

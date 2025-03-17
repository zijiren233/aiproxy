package tiktoken

import (
	"strings"
	"sync"

	"github.com/pkoukk/tiktoken-go"
	log "github.com/sirupsen/logrus"
)

// tokenEncoderMap won't grow after initialization
var (
	tokenEncoderMap     = map[string]*tiktoken.Tiktoken{}
	defaultTokenEncoder *tiktoken.Tiktoken
	tokenEncoderLock    sync.RWMutex
)

func init() {
	tiktoken.SetBpeLoader(&embedBpeLoader{})
	gpt35TokenEncoder, err := tiktoken.EncodingForModel("gpt-3.5-turbo")
	if err != nil {
		log.Fatal("failed to get gpt-3.5-turbo token encoder: " + err.Error())
	}
	defaultTokenEncoder = gpt35TokenEncoder
}

func GetTokenEncoder(model string) *tiktoken.Tiktoken {
	tokenEncoderLock.RLock()
	tokenEncoder, ok := tokenEncoderMap[model]
	tokenEncoderLock.RUnlock()
	if ok {
		return tokenEncoder
	}

	tokenEncoderLock.Lock()
	defer tokenEncoderLock.Unlock()
	if tokenEncoder, ok := tokenEncoderMap[model]; ok {
		return tokenEncoder
	}

	tokenEncoder, err := tiktoken.EncodingForModel(model)
	if err != nil {
		if strings.Contains(err.Error(), "no encoding for model") {
			log.Warnf("no encoding for model %s, using encoder for gpt-3.5-turbo", model)
			tokenEncoderMap[model] = defaultTokenEncoder
		} else {
			log.Errorf("failed to get token encoder for model %s: %v", model, err)
		}
		return defaultTokenEncoder
	}

	tokenEncoderMap[model] = tokenEncoder
	return tokenEncoder
}

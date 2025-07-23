package utils

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/labring/aiproxy/core/common"
	model "github.com/labring/aiproxy/core/relay/model"
)

func UnmarshalAnthropicMessageRequest(req *http.Request) (*model.AnthropicMessageRequest, error) {
	var request model.AnthropicMessageRequest

	err := common.UnmarshalRequestReusable(req, &request)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func UnmarshalGeneralOpenAIRequest(req *http.Request) (*model.GeneralOpenAIRequest, error) {
	var request model.GeneralOpenAIRequest

	err := common.UnmarshalRequestReusable(req, &request)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func UnmarshalVideoGenerationJobRequest(
	req *http.Request,
) (*model.VideoGenerationJobRequest, error) {
	var request model.VideoGenerationJobRequest

	err := common.UnmarshalRequestReusable(req, &request)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func UnmarshalImageRequest(req *http.Request) (*model.ImageRequest, error) {
	var request model.ImageRequest

	err := common.UnmarshalRequestReusable(req, &request)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func UnmarshalRerankRequest(req *http.Request) (*model.RerankRequest, error) {
	var request model.RerankRequest

	err := common.UnmarshalRequestReusable(req, &request)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func UnmarshalTTSRequest(req *http.Request) (*model.TextToSpeechRequest, error) {
	var request model.TextToSpeechRequest

	err := common.UnmarshalRequestReusable(req, &request)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func UnmarshalMap(req *http.Request) (map[string]any, error) {
	var request map[string]any

	err := common.UnmarshalRequestReusable(req, &request)
	if err != nil {
		return nil, err
	}

	return request, nil
}

var defaultClient = &http.Client{}

func DoRequest(req *http.Request) (*http.Response, error) {
	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func IsStreamResponse(resp *http.Response) bool {
	return IsStreamResponseWithHeader(resp.Header)
}

func IsStreamResponseWithHeader(header http.Header) bool {
	contentType := header.Get("Content-Type")
	if contentType == "" {
		return false
	}

	return strings.Contains(contentType, "event-stream") ||
		strings.Contains(contentType, "x-ndjson")
}

const scannerBufferSize = 256 * 1024

var scannerBufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, scannerBufferSize)
		return &buf
	},
}

//nolint:forcetypeassert
func GetScannerBuffer() *[]byte {
	v, ok := scannerBufferPool.Get().(*[]byte)
	if !ok {
		panic(fmt.Sprintf("scanner buffer type error: %T, %v", v, v))
	}

	return v
}

func PutScannerBuffer(buf *[]byte) {
	if cap(*buf) != scannerBufferSize {
		return
	}

	scannerBufferPool.Put(buf)
}

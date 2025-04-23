package jina

import (
	"bytes"
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/relay/meta"
)

//nolint:gocritic
func ConvertEmbeddingsRequest(meta *meta.Meta, req *http.Request) (string, http.Header, io.Reader, error) {
	reqMap := make(map[string]any)
	err := common.UnmarshalBodyReusable(req, &reqMap)
	if err != nil {
		return "", nil, nil, err
	}

	reqMap["model"] = meta.ActualModel

	switch v := reqMap["input"].(type) {
	case string:
		reqMap["input"] = []string{v}
	}

	delete(reqMap, "encoding_format")

	jsonData, err := sonic.Marshal(reqMap)
	if err != nil {
		return "", nil, nil, err
	}
	return http.MethodPost, nil, bytes.NewReader(jsonData), nil
}

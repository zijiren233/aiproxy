package openai

import (
	"bytes"
	"errors"
	"net/http"

	"github.com/bytedance/sonic/ast"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
)

func ConvertEmbeddingsRequest(
	meta *meta.Meta,
	req *http.Request,
	inputToSlices bool,
) (*adaptor.ConvertRequestResult, error) {
	node, err := common.UnmarshalBody2Node(req)
	if err != nil {
		return nil, err
	}

	_, err = node.Set("model", ast.NewString(meta.ActualModel))
	if err != nil {
		return nil, err
	}

	if inputToSlices {
		inputNode := node.Get("input")
		if inputNode.Exists() {
			inputString, err := inputNode.String()
			if err != nil {
				if !errors.Is(err, ast.ErrUnsupportType) {
					return nil, err
				}
			} else {
				_, err = node.SetAny("input", []string{inputString})
				if err != nil {
					return nil, err
				}
			}
		}
	}

	jsonData, err := node.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return &adaptor.ConvertRequestResult{
		Header: nil,
		Body:   bytes.NewReader(jsonData),
	}, nil
}

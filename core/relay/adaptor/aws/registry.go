package aws

import (
	"github.com/labring/aiproxy/core/model"
	claude "github.com/labring/aiproxy/core/relay/adaptor/aws/claude"
	llama3 "github.com/labring/aiproxy/core/relay/adaptor/aws/llama3"
	"github.com/labring/aiproxy/core/relay/adaptor/aws/utils"
)

type ModelType int

const (
	AwsClaude ModelType = iota + 1
	AwsLlama3
)

type Model struct {
	config *model.ModelConfig
	_type  ModelType
}

var adaptors = map[string]Model{}

func init() {
	for _, model := range claude.AwsModelIDMap {
		adaptors[model.Model] = Model{config: &model.ModelConfig, _type: AwsClaude}
	}
	for _, model := range llama3.AwsModelIDMap {
		adaptors[model.Model] = Model{config: &model.ModelConfig, _type: AwsLlama3}
	}
}

func GetAdaptor(model string) utils.AwsAdapter {
	adaptorType := adaptors[model]
	switch adaptorType._type {
	case AwsClaude:
		return &claude.Adaptor{}
	case AwsLlama3:
		return &llama3.Adaptor{}
	default:
		return nil
	}
}

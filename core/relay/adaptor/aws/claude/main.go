// Package aws provides the AWS adaptor for the relay service.
package aws

import (
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"github.com/labring/aiproxy/core/common/render"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/anthropic"
	"github.com/labring/aiproxy/core/relay/adaptor/aws/utils"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/pkg/errors"
)

type awsModelItem struct {
	ID string
	model.ModelConfig
}

// AwsModelIDMap maps internal model identifiers to AWS model identifiers.
// For more details, see: https://docs.aws.amazon.com/bedrock/latest/userguide/model-ids.html

var AwsModelIDMap = map[string]awsModelItem{
	"claude-instant-1.2": {
		ModelConfig: model.ModelConfig{
			Model: "claude-instant-1.2",
			Type:  mode.ChatCompletions,
			Owner: model.ModelOwnerAnthropic,
		},
		ID: "anthropic.claude-instant-v1",
	},
	"claude-2.0": {
		ModelConfig: model.ModelConfig{
			Model: "claude-2.0",
			Type:  mode.ChatCompletions,
			Owner: model.ModelOwnerAnthropic,
		},
		ID: "anthropic.claude-v2",
	},
	"claude-2.1": {
		ModelConfig: model.ModelConfig{
			Model: "claude-2.1",
			Type:  mode.ChatCompletions,
			Owner: model.ModelOwnerAnthropic,
		},
		ID: "anthropic.claude-v2:1",
	},
	"claude-3-haiku-20240307": {
		ModelConfig: model.ModelConfig{
			Model: "claude-3-haiku-20240307",
			Type:  mode.ChatCompletions,
			Owner: model.ModelOwnerAnthropic,
		},
		ID: "anthropic.claude-3-haiku-20240307-v1:0",
	},
	"claude-3-5-sonnet-latest": {
		ModelConfig: model.ModelConfig{
			Model: "claude-3-5-sonnet-latest",
			Type:  mode.ChatCompletions,
			Owner: model.ModelOwnerAnthropic,
		},
		ID: "anthropic.claude-3-5-sonnet-20241022-v2:0",
	},
	"claude-3-5-haiku-20241022": {
		ModelConfig: model.ModelConfig{
			Model: "claude-3-5-haiku-20241022",
			Type:  mode.ChatCompletions,
			Owner: model.ModelOwnerAnthropic,
		},
		ID: "anthropic.claude-3-5-haiku-20241022-v1:0",
	},
}

func awsModelID(requestModel string) (string, error) {
	if awsModelID, ok := AwsModelIDMap[requestModel]; ok {
		return awsModelID.ID, nil
	}

	return "", errors.Errorf("model %s not found", requestModel)
}

func Handler(meta *meta.Meta, c *gin.Context) (*model.Usage, *relaymodel.ErrorWithStatusCode) {
	awsModelID, err := awsModelID(meta.ActualModel)
	if err != nil {
		return nil, utils.WrapErr(errors.Wrap(err, "awsModelID"))
	}

	awsReq := &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(awsModelID),
		Accept:      aws.String("application/json"),
		ContentType: aws.String("application/json"),
	}

	convReq, ok := meta.Get(ConvertedRequest)
	if !ok {
		return nil, utils.WrapErr(errors.New("request not found"))
	}
	claudeReq := convReq.(*anthropic.Request)
	awsClaudeReq := &Request{
		AnthropicVersion: "bedrock-2023-05-31",
	}
	if err = copier.Copy(awsClaudeReq, claudeReq); err != nil {
		return nil, utils.WrapErr(errors.Wrap(err, "copy request"))
	}

	awsReq.Body, err = sonic.Marshal(awsClaudeReq)
	if err != nil {
		return nil, utils.WrapErr(errors.Wrap(err, "marshal request"))
	}

	awsClient, err := utils.AwsClientFromMeta(meta)
	if err != nil {
		return nil, utils.WrapErr(errors.Wrap(err, "get aws client"))
	}

	awsResp, err := awsClient.InvokeModel(c.Request.Context(), awsReq)
	if err != nil {
		return nil, utils.WrapErr(errors.Wrap(err, "InvokeModel"))
	}

	claudeResponse := new(anthropic.Response)
	err = sonic.Unmarshal(awsResp.Body, claudeResponse)
	if err != nil {
		return nil, utils.WrapErr(errors.Wrap(err, "unmarshal response"))
	}

	openaiResp := anthropic.Response2OpenAI(meta, claudeResponse)
	c.JSON(http.StatusOK, openaiResp)
	return openaiResp.Usage.ToModelUsage(), nil
}

func StreamHandler(m *meta.Meta, c *gin.Context) (*model.Usage, *relaymodel.ErrorWithStatusCode) {
	log := middleware.GetLogger(c)
	awsModelID, err := awsModelID(m.ActualModel)
	if err != nil {
		return nil, utils.WrapErr(errors.Wrap(err, "awsModelID"))
	}

	awsReq := &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(awsModelID),
		Accept:      aws.String("application/json"),
		ContentType: aws.String("application/json"),
	}

	convReq, ok := m.Get(ConvertedRequest)
	if !ok {
		return nil, utils.WrapErr(errors.New("request not found"))
	}
	claudeReq, ok := convReq.(*anthropic.Request)
	if !ok {
		return nil, utils.WrapErr(errors.New("request not found"))
	}

	awsClaudeReq := &Request{
		AnthropicVersion: "bedrock-2023-05-31",
	}
	if err = copier.Copy(awsClaudeReq, claudeReq); err != nil {
		return nil, utils.WrapErr(errors.Wrap(err, "copy request"))
	}
	awsReq.Body, err = sonic.Marshal(awsClaudeReq)
	if err != nil {
		return nil, utils.WrapErr(errors.Wrap(err, "marshal request"))
	}

	awsClient, err := utils.AwsClientFromMeta(m)
	if err != nil {
		return nil, utils.WrapErr(errors.Wrap(err, "get aws client"))
	}

	awsResp, err := awsClient.InvokeModelWithResponseStream(c.Request.Context(), awsReq)
	if err != nil {
		return nil, utils.WrapErr(errors.Wrap(err, "InvokeModelWithResponseStream"))
	}
	stream := awsResp.GetStream()
	defer stream.Close()

	var usage *relaymodel.Usage
	responseText := strings.Builder{}
	var writed bool

	// c.Stream(func(_ io.Writer) bool {

	// })
	for event := range stream.Events() {
		switch v := event.(type) {
		case *types.ResponseStreamMemberChunk:
			response, err := anthropic.StreamResponse2OpenAI(m, v.Value.Bytes)
			if err != nil {
				if writed {
					log.Errorf("response error: %+v", err)
					continue
				}
				return usage.ToModelUsage(), err
			}
			if response == nil {
				continue
			}
			if response != nil {
				switch {
				case response.Usage != nil:
					if usage == nil {
						usage = &relaymodel.Usage{}
					}
					usage.Add(response.Usage)
					if usage.PromptTokens == 0 {
						usage.PromptTokens = m.InputTokens
						usage.TotalTokens += m.InputTokens
					}
					response.Usage = usage
					responseText.Reset()
				case usage == nil:
					for _, choice := range response.Choices {
						responseText.WriteString(choice.Delta.StringContent())
					}
				default:
					response.Usage = usage
				}
			}

			_ = render.ObjectData(c, response)
			writed = true
		case *types.UnknownUnionMember:
			log.Error("unknown tag: " + v.Tag)
			continue
		default:
			log.Errorf("union is nil or unknown type: %v", v)
			continue
		}
	}

	if usage == nil {
		usage = &relaymodel.Usage{
			PromptTokens:     m.InputTokens,
			CompletionTokens: openai.CountTokenText(responseText.String(), m.OriginModel),
			TotalTokens:      m.InputTokens + openai.CountTokenText(responseText.String(), m.OriginModel),
		}
		_ = render.ObjectData(c, &relaymodel.ChatCompletionsStreamResponse{
			ID:      openai.ChatCompletionID(),
			Model:   m.OriginModel,
			Object:  relaymodel.ChatCompletionChunk,
			Created: time.Now().Unix(),
			Choices: []*relaymodel.ChatCompletionsStreamResponseChoice{},
			Usage:   usage,
		})
	}

	render.Done(c)

	return usage.ToModelUsage(), nil
}

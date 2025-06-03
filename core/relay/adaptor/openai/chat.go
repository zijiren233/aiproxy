package openai

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/conv"
	"github.com/labring/aiproxy/core/common/render"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

const (
	DataPrefix       = "data:"
	Done             = "[DONE]"
	DataPrefixLength = len(DataPrefix)
)

var (
	DataPrefixBytes = conv.StringToBytes(DataPrefix)
	DoneBytes       = conv.StringToBytes(Done)
)

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

func ConvertTextRequest(
	meta *meta.Meta,
	req *http.Request,
	doNotPatchStreamOptionsIncludeUsage bool,
) (*adaptor.ConvertRequestResult, error) {
	reqMap := make(map[string]any)
	err := common.UnmarshalBodyReusable(req, &reqMap)
	if err != nil {
		return nil, err
	}

	if !doNotPatchStreamOptionsIncludeUsage {
		if err := patchStreamOptions(reqMap); err != nil {
			return nil, err
		}
	}

	reqMap["model"] = meta.ActualModel
	jsonData, err := sonic.Marshal(reqMap)
	if err != nil {
		return nil, err
	}
	return &adaptor.ConvertRequestResult{
		Method: http.MethodPost,
		Header: nil,
		Body:   bytes.NewReader(jsonData),
	}, nil
}

func patchStreamOptions(reqMap map[string]any) error {
	stream, ok := reqMap["stream"]
	if !ok {
		return nil
	}

	streamBool, ok := stream.(bool)
	if !ok {
		return errors.New("stream is not a boolean")
	}

	if !streamBool {
		return nil
	}

	streamOptions, ok := reqMap["stream_options"].(map[string]any)
	if !ok {
		if reqMap["stream_options"] != nil {
			return errors.New("stream_options is not a map")
		}
		reqMap["stream_options"] = map[string]any{
			"include_usage": true,
		}
		return nil
	}

	streamOptions["include_usage"] = true
	return nil
}

func GetUsageOrChatChoicesResponseFromNode(
	node *ast.Node,
) (*relaymodel.Usage, []*relaymodel.ChatCompletionsStreamResponseChoice, error) {
	var usage *relaymodel.Usage
	usageNode, err := node.Get("usage").Raw()
	if err != nil {
		if !errors.Is(err, ast.ErrNotExist) {
			return nil, nil, err
		}
	} else {
		err = sonic.UnmarshalString(usageNode, &usage)
		if err != nil {
			return nil, nil, err
		}
	}

	if usage != nil {
		return usage, nil, nil
	}

	var choices []*relaymodel.ChatCompletionsStreamResponseChoice
	choicesNode, err := node.Get("choices").Raw()
	if err != nil {
		if !errors.Is(err, ast.ErrNotExist) {
			return nil, nil, err
		}
	} else {
		err = sonic.UnmarshalString(choicesNode, &choices)
		if err != nil {
			return nil, nil, err
		}
	}
	return nil, choices, nil
}

type PreHandler func(meta *meta.Meta, node *ast.Node) error

func StreamHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
	preHandler PreHandler,
) (*model.Usage, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return nil, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	log := middleware.GetLogger(c)

	responseText := strings.Builder{}

	scanner := bufio.NewScanner(resp.Body)
	buf := GetScannerBuffer()
	defer PutScannerBuffer(buf)
	scanner.Buffer(*buf, cap(*buf))

	var usage *relaymodel.Usage

	for scanner.Scan() {
		data := scanner.Bytes()
		if len(data) < DataPrefixLength { // ignore blank line or wrong format
			continue
		}
		if !slices.Equal(data[:DataPrefixLength], DataPrefixBytes) {
			continue
		}
		data = bytes.TrimSpace(data[DataPrefixLength:])
		if slices.Equal(data, DoneBytes) {
			break
		}

		node, err := sonic.Get(data)
		if err != nil {
			log.Error("error unmarshalling stream response: " + err.Error())
			continue
		}
		if preHandler != nil {
			err := preHandler(meta, &node)
			if err != nil {
				log.Error("error pre handler: " + err.Error())
				continue
			}
		}
		u, ch, err := GetUsageOrChatChoicesResponseFromNode(&node)
		if err != nil {
			log.Error("error unmarshalling stream response: " + err.Error())
			continue
		}
		if u != nil {
			usage = u
			responseText.Reset()
		}
		for _, choice := range ch {
			if usage == nil {
				if choice.Text != "" {
					responseText.WriteString(choice.Text)
				} else {
					responseText.WriteString(choice.Delta.StringContent())
				}
			}
		}

		_, err = node.Set("model", ast.NewString(meta.OriginModel))
		if err != nil {
			log.Error("error set model: " + err.Error())
		}

		_ = render.ObjectData(c, &node)
	}

	if err := scanner.Err(); err != nil {
		log.Error("error reading stream: " + err.Error())
	}

	if usage == nil || (usage.TotalTokens == 0 && responseText.Len() > 0) {
		usage = ResponseText2Usage(
			responseText.String(),
			meta.ActualModel,
			int64(meta.RequestUsage.InputTokens),
		)
		_ = render.ObjectData(c, &relaymodel.ChatCompletionsStreamResponse{
			ID:      ChatCompletionID(),
			Model:   meta.OriginModel,
			Object:  relaymodel.ChatCompletionChunkObject,
			Created: time.Now().Unix(),
			Choices: []*relaymodel.ChatCompletionsStreamResponseChoice{},
			Usage:   usage,
		})
	} else if usage.TotalTokens != 0 && usage.PromptTokens == 0 { // some channels don't return prompt tokens & completion tokens
		usage.PromptTokens = int64(meta.RequestUsage.InputTokens)
		usage.CompletionTokens = usage.TotalTokens - int64(meta.RequestUsage.InputTokens)
	}

	render.Done(c)

	return usage.ToModelUsage(), nil
}

func GetUsageOrChoicesResponseFromNode(
	node *ast.Node,
) (*relaymodel.Usage, []*relaymodel.TextResponseChoice, error) {
	var usage *relaymodel.Usage
	usageNode, err := node.Get("usage").Raw()
	if err != nil {
		if !errors.Is(err, ast.ErrNotExist) {
			return nil, nil, err
		}
	} else {
		err = sonic.UnmarshalString(usageNode, &usage)
		if err != nil {
			return nil, nil, err
		}
	}

	if usage != nil {
		return usage, nil, nil
	}

	var choices []*relaymodel.TextResponseChoice
	choicesNode, err := node.Get("choices").Raw()
	if err != nil {
		if !errors.Is(err, ast.ErrNotExist) {
			return nil, nil, err
		}
	} else {
		err = sonic.UnmarshalString(choicesNode, &choices)
		if err != nil {
			return nil, nil, err
		}
	}
	return nil, choices, nil
}

func Handler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
	preHandler PreHandler,
) (*model.Usage, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return nil, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	log := middleware.GetLogger(c)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, relaymodel.WrapperOpenAIError(
			err,
			"read_response_body_failed",
			http.StatusInternalServerError,
		)
	}

	node, err := sonic.Get(responseBody)
	if err != nil {
		return nil, relaymodel.WrapperOpenAIError(
			err,
			"unmarshal_response_body_failed",
			http.StatusInternalServerError,
		)
	}
	if preHandler != nil {
		err := preHandler(meta, &node)
		if err != nil {
			return nil, relaymodel.WrapperOpenAIError(
				err,
				"pre_handler_failed",
				http.StatusInternalServerError,
			)
		}
	}
	usage, choices, err := GetUsageOrChoicesResponseFromNode(&node)
	if err != nil {
		return nil, relaymodel.WrapperOpenAIError(
			err,
			"unmarshal_response_body_failed",
			http.StatusInternalServerError,
		)
	}

	if usage == nil || usage.TotalTokens == 0 ||
		(usage.PromptTokens == 0 && usage.CompletionTokens == 0) {
		var completionTokens int64
		for _, choice := range choices {
			if choice.Text != "" {
				completionTokens += CountTokenText(choice.Text, meta.ActualModel)
				continue
			}
			completionTokens += CountTokenText(choice.Message.StringContent(), meta.ActualModel)
		}
		usage = &relaymodel.Usage{
			PromptTokens:     int64(meta.RequestUsage.InputTokens),
			CompletionTokens: completionTokens,
			TotalTokens:      int64(meta.RequestUsage.InputTokens) + completionTokens,
		}
		_, err = node.Set("usage", ast.NewAny(usage))
		if err != nil {
			return usage.ToModelUsage(), relaymodel.WrapperOpenAIError(
				err,
				"set_usage_failed",
				http.StatusInternalServerError,
			)
		}
	} else if usage.TotalTokens != 0 && usage.PromptTokens == 0 { // some channels don't return prompt tokens & completion tokens
		usage.PromptTokens = int64(meta.RequestUsage.InputTokens)
		usage.CompletionTokens = usage.TotalTokens - int64(meta.RequestUsage.InputTokens)
		_, err = node.Set("usage", ast.NewAny(usage))
		if err != nil {
			return usage.ToModelUsage(), relaymodel.WrapperOpenAIError(err, "set_usage_failed", http.StatusInternalServerError)
		}
	}

	_, err = node.Set("model", ast.NewString(meta.OriginModel))
	if err != nil {
		return usage.ToModelUsage(), relaymodel.WrapperOpenAIError(
			err,
			"set_model_failed",
			http.StatusInternalServerError,
		)
	}

	newData, err := sonic.Marshal(&node)
	if err != nil {
		return usage.ToModelUsage(), relaymodel.WrapperOpenAIError(
			err,
			"marshal_response_body_failed",
			http.StatusInternalServerError,
		)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(newData)))
	_, err = c.Writer.Write(newData)
	if err != nil {
		log.Warnf("write response body failed: %v", err)
	}
	return usage.ToModelUsage(), nil
}

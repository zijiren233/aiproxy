package ali

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/image"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
	log "github.com/sirupsen/logrus"
)

const MetaResponseFormat = "response_format"

type qwenImageOpenAIRequest struct {
	relaymodel.ImageRequest
	NegativePrompt string `json:"negative_prompt,omitempty"`
	PromptExtend   *bool  `json:"prompt_extend,omitempty"`
	Watermark      *bool  `json:"watermark,omitempty"`
	Seed           *int64 `json:"seed,omitempty"`
}

func isQwenImageModel(meta *meta.Meta) bool {
	if meta == nil {
		return false
	}

	return isQwenImageModelName(meta.OriginModel) || isQwenImageModelName(meta.ActualModel)
}

func isQwenImageModelName(modelName string) bool {
	modelName = strings.ToLower(modelName)
	return strings.HasPrefix(modelName, "qwen-image")
}

func ConvertImageRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	if isQwenImageModel(meta) {
		return ConvertQwenImageGenerationRequest(meta, req)
	}

	request, err := utils.UnmarshalImageRequest(req)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	request.Model = meta.ActualModel

	var imageRequest ImageRequest

	imageRequest.Input.Prompt = request.Prompt
	imageRequest.Model = request.Model
	imageRequest.Parameters.Size = strings.ReplaceAll(request.Size, "x", "*")
	imageRequest.Parameters.N = request.N
	imageRequest.ResponseFormat = request.ResponseFormat

	meta.Set(MetaResponseFormat, request.ResponseFormat)

	data, err := sonic.Marshal(&imageRequest)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	return adaptor.ConvertResult{
		Header: http.Header{
			"X-Dashscope-Async": {"enable"},
			"Content-Type":      {"application/json"},
			"Content-Length":    {strconv.Itoa(len(data))},
		},
		Body: bytes.NewReader(data),
	}, nil
}

func ConvertQwenImageGenerationRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	var request qwenImageOpenAIRequest

	err := common.UnmarshalRequestReusable(req, &request)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	meta.Set(MetaResponseFormat, request.ResponseFormat)

	imageRequest, err := buildQwenImageRequest(meta, request.Prompt, &request)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	data, err := sonic.Marshal(&imageRequest)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	return adaptor.ConvertResult{
		Header: http.Header{
			"Content-Type":   {"application/json"},
			"Content-Length": {strconv.Itoa(len(data))},
		},
		Body: bytes.NewReader(data),
	}, nil
}

func ConvertQwenImageEditRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	err := common.ParseMultipartFormWithLimit(req)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	imageRequest := qwenImageOpenAIRequest{
		ImageRequest: relaymodel.ImageRequest{
			Model:          req.PostFormValue("model"),
			Prompt:         req.PostFormValue("prompt"),
			Size:           req.PostFormValue("size"),
			ResponseFormat: req.PostFormValue("response_format"),
		},
		NegativePrompt: req.PostFormValue("negative_prompt"),
	}

	if n := req.PostFormValue("n"); n != "" {
		imageRequest.N, err = strconv.Atoi(n)
		if err != nil {
			return adaptor.ConvertResult{}, fmt.Errorf("invalid n: %w", err)
		}
	}

	if promptExtend := req.PostFormValue("prompt_extend"); promptExtend != "" {
		value, err := strconv.ParseBool(promptExtend)
		if err != nil {
			return adaptor.ConvertResult{}, fmt.Errorf("invalid prompt_extend: %w", err)
		}

		imageRequest.PromptExtend = &value
	}

	if watermark := req.PostFormValue("watermark"); watermark != "" {
		value, err := strconv.ParseBool(watermark)
		if err != nil {
			return adaptor.ConvertResult{}, fmt.Errorf("invalid watermark: %w", err)
		}

		imageRequest.Watermark = &value
	}

	if seed := req.PostFormValue("seed"); seed != "" {
		value, err := strconv.ParseInt(seed, 10, 64)
		if err != nil {
			return adaptor.ConvertResult{}, fmt.Errorf("invalid seed: %w", err)
		}

		imageRequest.Seed = &value
	}

	meta.Set(MetaResponseFormat, imageRequest.ResponseFormat)

	qwenRequest, err := buildQwenImageRequest(meta, imageRequest.Prompt, &imageRequest)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	userMessage := &qwenRequest.Input.Messages[0]

	fileHeaders := qwenImageEditFileHeaders(req.MultipartForm.File)
	if len(fileHeaders) == 0 {
		return adaptor.ConvertResult{}, errors.New("image is required")
	}

	if len(fileHeaders) > 3 {
		return adaptor.ConvertResult{}, errors.New("image supports at most 3 files")
	}

	imageContents := make([]map[string]any, 0, len(fileHeaders))
	for _, fileHeader := range fileHeaders {
		imageData, err := multipartImageFileToDataURL(fileHeader)
		if err != nil {
			return adaptor.ConvertResult{}, err
		}

		imageContents = append(imageContents, map[string]any{"image": imageData})
	}

	userMessage.Content = append(imageContents, userMessage.Content...)

	data, err := sonic.Marshal(&qwenRequest)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	return adaptor.ConvertResult{
		Header: http.Header{
			"Content-Type":   {"application/json"},
			"Content-Length": {strconv.Itoa(len(data))},
		},
		Body: bytes.NewReader(data),
	}, nil
}

func buildQwenImageRequest(
	meta *meta.Meta,
	prompt string,
	request *qwenImageOpenAIRequest,
) (MultimodalImageRequest, error) {
	var imageRequest MultimodalImageRequest

	if strings.TrimSpace(prompt) == "" {
		return imageRequest, errors.New("prompt is required")
	}

	imageRequest.Model = meta.ActualModel
	imageRequest.Input.Messages = []MultimodalImageMessage{
		{
			Role:    "user",
			Content: []map[string]any{{"text": prompt}},
		},
	}

	parameters := make(map[string]any)
	if request != nil {
		if size := qwenImageSize(request.Size); size != "" && qwenImageSupportsSize(meta) {
			parameters["size"] = size
		}

		if request.N > 0 && qwenImageSupportsN(meta) {
			if request.N > 6 {
				return imageRequest, errors.New("n must be between 1 and 6")
			}

			parameters["n"] = request.N
		} else if request.N > 1 {
			return imageRequest, errors.New("n must be 1 for this model")
		}

		if request.NegativePrompt != "" {
			parameters["negative_prompt"] = request.NegativePrompt
		}

		if request.PromptExtend != nil && qwenImageSupportsPromptExtend(meta) {
			parameters["prompt_extend"] = *request.PromptExtend
		}

		if request.Watermark != nil {
			parameters["watermark"] = *request.Watermark
		}

		if request.Seed != nil {
			parameters["seed"] = *request.Seed
		}
	}

	if len(parameters) > 0 {
		imageRequest.Parameters = parameters
	}

	return imageRequest, nil
}

func qwenImageSize(size string) string {
	if size == "" || size == "auto" {
		return ""
	}

	return strings.ReplaceAll(size, "x", "*")
}

func qwenImageRuleModel(meta *meta.Meta) string {
	if meta == nil {
		return ""
	}

	if isQwenImageModelName(meta.ActualModel) {
		return strings.ToLower(meta.ActualModel)
	}

	return strings.ToLower(meta.OriginModel)
}

func qwenImageSupportsN(meta *meta.Meta) bool {
	modelName := qwenImageRuleModel(meta)
	if strings.HasPrefix(modelName, "qwen-image-2.0") {
		return true
	}

	if meta != nil && meta.Mode == mode.ImagesEdits {
		return strings.HasPrefix(modelName, "qwen-image-edit-max") ||
			strings.HasPrefix(modelName, "qwen-image-edit-plus")
	}

	return false
}

func qwenImageSupportsSize(meta *meta.Meta) bool {
	return meta == nil ||
		meta.Mode != mode.ImagesEdits ||
		qwenImageRuleModel(meta) != "qwen-image-edit"
}

func qwenImageSupportsPromptExtend(meta *meta.Meta) bool {
	return qwenImageSupportsSize(meta)
}

func qwenImageEditFileHeaders(files map[string][]*multipart.FileHeader) []*multipart.FileHeader {
	fileHeaders := make([]*multipart.FileHeader, 0, len(files["image"])+len(files["image[]"]))
	fileHeaders = append(fileHeaders, files["image"]...)
	fileHeaders = append(fileHeaders, files["image[]"]...)

	return fileHeaders
}

func multipartImageFileToDataURL(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := io.ReadAll(common.LimitReader(file, image.MaxImageSize+1))
	if err != nil {
		return "", err
	}

	if len(data) > image.MaxImageSize {
		return "", fmt.Errorf("image too large: max: %d", image.MaxImageSize)
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	if !image.IsImageURL(contentType) {
		if ext := strings.ToLower(filepath.Ext(fileHeader.Filename)); ext != "" {
			if detected := mime.TypeByExtension(ext); detected != "" {
				contentType = detected
			}
		}
	}

	if !image.IsImageURL(contentType) {
		return "", errors.New("image file is not an image")
	}

	contentType = image.TrimImageContentType(contentType)

	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func ImageHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if isQwenImageModel(meta) {
		return QwenImageHandler(meta, c, resp)
	}

	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	log := common.GetLogger(c)

	responseFormat, _ := meta.MustGet(MetaResponseFormat).(string)

	var aliTaskResponse TaskResponse

	err := common.UnmarshalResponse(resp, &aliTaskResponse)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIError(
			err,
			"unmarshal_response_body_failed",
			http.StatusInternalServerError,
		)
	}

	if aliTaskResponse.Message != "" {
		log.Error("aliAsyncTask err: " + aliTaskResponse.Message)

		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIError(
			errors.New(aliTaskResponse.Message),
			"ali_async_task_failed",
			http.StatusInternalServerError,
		)
	}

	aliResponse, err := asyncTaskWait(c, aliTaskResponse.Output.TaskID, meta.Channel.Key)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIError(
			err,
			"ali_async_task_wait_failed",
			http.StatusInternalServerError,
		)
	}

	if aliResponse.Output.TaskStatus != "SUCCEEDED" {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIErrorWithMessage(
			aliResponse.Output.Message,
			"ali_error",
			resp.StatusCode,
		)
	}

	fullTextResponse := responseAli2OpenAIImage(c.Request.Context(), aliResponse, responseFormat)

	jsonResponse, err := sonic.Marshal(fullTextResponse)
	if err != nil {
		return adaptor.DoResponseResult{
				Usage: fullTextResponse.Usage.ToModelUsage(),
			}, relaymodel.WrapperOpenAIError(
				err,
				"marshal_response_body_failed",
				http.StatusInternalServerError,
			)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(jsonResponse)))
	_, _ = c.Writer.Write(jsonResponse)

	return adaptor.DoResponseResult{Usage: fullTextResponse.Usage.ToModelUsage()}, nil
}

func QwenImageHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	var aliResponse MultimodalImageResponse
	if err := common.UnmarshalResponse(resp, &aliResponse); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIError(
			err,
			"unmarshal_response_body_failed",
			http.StatusInternalServerError,
		)
	}

	if aliResponse.Code != "" || aliResponse.Message != "" {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIErrorWithMessage(
			aliResponse.Message,
			aliResponse.Code,
			http.StatusInternalServerError,
		)
	}

	imageResponse := responseQwenImage2OpenAI(
		c.Request.Context(),
		&aliResponse,
		meta.GetString(MetaResponseFormat),
	)

	jsonResponse, err := sonic.Marshal(imageResponse)
	if err != nil {
		return adaptor.DoResponseResult{
				Usage: imageResponse.Usage.ToModelUsage(),
			}, relaymodel.WrapperOpenAIError(
				err,
				"marshal_response_body_failed",
				http.StatusInternalServerError,
			)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(jsonResponse)))
	_, _ = c.Writer.Write(jsonResponse)

	return adaptor.DoResponseResult{
		Usage:      imageResponse.Usage.ToModelUsage(),
		UpstreamID: aliResponse.RequestID,
	}, nil
}

func asyncTask(ctx context.Context, taskID, key string) (*TaskResponse, error) {
	url := "https://dashscope.aliyuncs.com/api/v1/tasks/" + taskID

	var aliResponse TaskResponse

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &aliResponse, err
	}

	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return &aliResponse, err
	}
	defer resp.Body.Close()

	var response TaskResponse

	err = sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return &aliResponse, err
	}

	return &response, nil
}

func asyncTaskWait(ctx context.Context, taskID, key string) (*TaskResponse, error) {
	waitSeconds := 2
	step := 0
	maxStep := 20

	for {
		step++

		rsp, err := asyncTask(ctx, taskID, key)
		if err != nil {
			return nil, err
		}

		if rsp.Output.TaskStatus == "" {
			return rsp, nil
		}

		switch rsp.Output.TaskStatus {
		case "FAILED":
			fallthrough
		case "CANCELED":
			fallthrough
		case "SUCCEEDED":
			fallthrough
		case "UNKNOWN":
			return rsp, nil
		}

		if step >= maxStep {
			break
		}

		time.Sleep(time.Duration(waitSeconds) * time.Second)
	}

	return nil, errors.New("aliAsyncTaskWait timeout")
}

func responseAli2OpenAIImage(
	ctx context.Context,
	response *TaskResponse,
	responseFormat string,
) *relaymodel.ImageResponse {
	imageResponse := relaymodel.ImageResponse{
		Created: time.Now().Unix(),
	}

	for _, data := range response.Output.Results {
		var b64Json string
		if responseFormat == "b64_json" {
			// 读取 data.Url 的图片数据并转存到 b64Json
			_, imageData, err := image.GetImageFromURL(ctx, data.URL)
			if err != nil {
				// 处理获取图片数据失败的情况
				log.Error("getImageData Error getting image data: " + err.Error())
				continue
			}

			// 将图片数据转为 Base64 编码的字符串
			b64Json = imageData
		} else {
			// 如果 responseFormat 不是 "b64_json"，则直接使用 data.B64Image
			b64Json = data.B64Image
		}

		imageResponse.Data = append(imageResponse.Data, &relaymodel.ImageData{
			URL:           data.URL,
			B64Json:       b64Json,
			RevisedPrompt: "",
		})
	}

	imageResponse.Usage = &relaymodel.ImageUsage{
		OutputTokens: int64(len(imageResponse.Data)),
		TotalTokens:  int64(len(imageResponse.Data)),
	}

	return &imageResponse
}

func responseQwenImage2OpenAI(
	ctx context.Context,
	response *MultimodalImageResponse,
	responseFormat string,
) *relaymodel.ImageResponse {
	imageResponse := relaymodel.ImageResponse{
		Created: time.Now().Unix(),
	}

	appendData := func(url, b64Image, revisedPrompt string) {
		if url == "" && b64Image == "" {
			return
		}

		b64Json := b64Image
		if responseFormat == "b64_json" && b64Json == "" && url != "" {
			_, imageData, err := image.GetImageFromURL(ctx, url)
			if err != nil {
				log.Error("getImageData Error getting image data: " + err.Error())
			} else {
				b64Json = imageData
			}
		}

		imageResponse.Data = append(imageResponse.Data, &relaymodel.ImageData{
			URL:           url,
			B64Json:       b64Json,
			RevisedPrompt: revisedPrompt,
		})
	}

	for _, choice := range response.Output.Choices {
		for _, content := range choice.Message.Content {
			appendData(
				firstNonEmpty(content.Image, content.URL),
				content.B64Image,
				content.ActualPrompt,
			)
		}
	}

	for _, result := range response.Output.Results {
		appendData(
			firstNonEmpty(result.Image, result.URL),
			result.B64Image,
			result.ActualPrompt,
		)
	}

	outputTokens := response.Usage.ImageCount
	if outputTokens == 0 {
		outputTokens = int64(len(imageResponse.Data))
	}

	imageResponse.Usage = &relaymodel.ImageUsage{
		OutputTokens: outputTokens,
		TotalTokens:  outputTokens,
		OutputTokensDetails: &relaymodel.ImageOutputTokensDetails{
			ImageTokens: outputTokens,
		},
	}

	return &imageResponse
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

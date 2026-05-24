package gemini

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	relayutils "github.com/labring/aiproxy/core/relay/utils"
)

const (
	geminiVideoTTL           = 7 * 24 * time.Hour
	geminiVideoLocalIDPrefix = "gemini_op_"

	defaultGeminiVideoDurationSeconds = 8
	defaultGeminiVideoResolution      = "720p"

	metaGeminiVideoSeconds    = "gemini_video_seconds"
	metaGeminiVideoVariants   = "gemini_video_variants"
	metaGeminiVideoResolution = "gemini_video_resolution"
)

type geminiVideoRequest struct {
	Instances  []geminiVideoInstance `json:"instances,omitempty"`
	Parameters geminiVideoParameters `json:"parameters,omitempty"`
}

type geminiVideoInstance struct {
	Prompt string            `json:"prompt,omitempty"`
	Image  *geminiVideoMedia `json:"image,omitempty"`
	Video  *geminiVideoMedia `json:"video,omitempty"`
}

type geminiVideoMedia struct {
	InlineData         *relaymodel.GeminiInlineData `json:"inlineData,omitempty"`
	FileData           *relaymodel.GeminiFileData   `json:"fileData,omitempty"`
	BytesBase64Encoded string                       `json:"bytesBase64Encoded,omitempty"`
	MimeType           string                       `json:"mimeType,omitempty"`
	GCSURI             string                       `json:"gcsUri,omitempty"`
	FileURI            string                       `json:"fileUri,omitempty"`
}

type geminiVideoParameters struct {
	AspectRatio      string `json:"aspectRatio,omitempty"`
	Resolution       string `json:"resolution,omitempty"`
	DurationSeconds  int    `json:"durationSeconds,omitempty"`
	NumberOfVideos   int    `json:"numberOfVideos,omitempty"`
	NegativePrompt   string `json:"negativePrompt,omitempty"`
	PersonGeneration string `json:"personGeneration,omitempty"`
}

type geminiVideoStoreMetadata struct {
	OperationName string `json:"operation_name,omitempty"`
	Prompt        string `json:"prompt,omitempty"`
	Resolution    string `json:"resolution,omitempty"`
	Seconds       int    `json:"seconds,omitempty"`
	Variants      int    `json:"variants,omitempty"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
}

type geminiOperation = relaymodel.GeminiVideoOperation

func NativeVideoConvertRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	cfg, err := loadConfig(meta)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	body, err := common.GetRequestBodyReusable(req)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	if cfg.EnablePersonGenerationAllowAll {
		body, err = ensureNativeVideoPersonGenerationAllowAll(body)
		if err != nil {
			return adaptor.ConvertResult{}, err
		}
	}

	if meta != nil {
		if node, err := common.GetJSONNodeNoCopy(body); err == nil {
			parameters := node.Get("parameters")

			seconds := firstPositiveIntNode(
				&node,
				parameters,
				defaultGeminiVideoDurationSeconds,
				"durationSeconds",
			)

			variants := intNode(&node, "numberOfVideos")
			if variants == 0 {
				variants = intNode(parameters, "numberOfVideos")
			}

			if variants <= 0 {
				variants = 1
			}

			resolution := firstStringNode(&node, parameters, "resolution")
			if resolution == "" {
				resolution = defaultGeminiVideoResolution
			}

			width, height := requestVideoDimensionsFromAspectRatio(
				firstStringNode(&node, parameters, "aspectRatio"),
				resolution,
			)
			setGeminiVideoRequestMetadata(meta, seconds, variants, resolution, width, height)
		}
	}

	return adaptor.ConvertResult{
		Header: http.Header{
			"Content-Type":   {"application/json"},
			"Content-Length": {strconv.Itoa(len(body))},
		},
		Body: bytes.NewReader(body),
	}, nil
}

func ConvertVideoNoBodyRequest(
	_ *meta.Meta,
	_ *http.Request,
) (adaptor.ConvertResult, error) {
	return adaptor.ConvertResult{}, nil
}

func ConvertVideoGenerationsGetJobsRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	return ConvertVideoNoBodyRequest(meta, req)
}

func ConvertVideoGenerationsContentRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	return ConvertVideoNoBodyRequest(meta, req)
}

func ConvertVideosGetRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	return ConvertVideoNoBodyRequest(meta, req)
}

func ConvertVideosContentRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	return ConvertVideoNoBodyRequest(meta, req)
}

func ConvertVideoGenerationJobRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	return convertOpenAIVideoGenerationJobRequest(meta, req)
}

func ConvertVideosRequest(meta *meta.Meta, req *http.Request) (adaptor.ConvertResult, error) {
	return convertOpenAIVideosRequest(meta, req)
}

func convertOpenAIVideoGenerationJobRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	request, err := parseOpenAIVideoGenerationJobRequest(req)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	return convertGeminiVideoRequest(meta, request)
}

func convertOpenAIVideosRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	request, err := parseOpenAIVideosRequest(req)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	return convertGeminiVideoRequest(meta, request)
}

func convertGeminiVideoRequest(
	meta *meta.Meta,
	request geminiVideoRequest,
) (adaptor.ConvertResult, error) {
	cfg, err := loadConfig(meta)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	if len(request.Instances) == 0 {
		return adaptor.ConvertResult{}, errors.New("prompt or input_reference is required")
	}

	applyVideoPersonGenerationConfig(&request, cfg)

	if meta != nil {
		meta.Set("gemini_video_prompt", request.Instances[0].Prompt)
		width, height := requestVideoDimensionsFromAspectRatio(
			request.Parameters.AspectRatio,
			geminiVideoMetadataResolution(request.Parameters.Resolution),
		)
		setGeminiVideoRequestMetadata(
			meta,
			geminiVideoMetadataDurationSeconds(request.Parameters.DurationSeconds),
			request.Parameters.NumberOfVideos,
			geminiVideoMetadataResolution(request.Parameters.Resolution),
			width,
			height,
		)
	}

	body, err := sonic.Marshal(request)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	return adaptor.ConvertResult{
		Header: http.Header{
			"Content-Type":   {"application/json"},
			"Content-Length": {strconv.Itoa(len(body))},
		},
		Body: bytes.NewReader(body),
	}, nil
}

func ConvertVideoRequestParametersToVertex(body []byte) ([]byte, error) {
	node, err := common.GetJSONNodeNoCopy(body)
	if err != nil {
		return nil, err
	}

	convertGeminiVideoNumberOfVideosToVertexObject(&node)
	parameters := node.Get("parameters")
	convertGeminiVideoNumberOfVideosToVertexObject(parameters)
	convertGeminiVideoMediaToVertex(&node)

	return node.MarshalJSON()
}

func convertGeminiVideoNumberOfVideosToVertexObject(node *ast.Node) {
	if node == nil || !node.Exists() || node.TypeSafe() == ast.V_NULL {
		return
	}

	numberOfVideos := node.Get("numberOfVideos")
	if numberOfVideos == nil ||
		!numberOfVideos.Exists() ||
		numberOfVideos.TypeSafe() == ast.V_NULL {
		return
	}

	sampleCount := node.Get("sampleCount")
	if sampleCount == nil || !sampleCount.Exists() || sampleCount.TypeSafe() == ast.V_NULL {
		if count, ok := intFromNode(numberOfVideos); ok {
			_, _ = node.Set("sampleCount", ast.NewNumber(strconv.Itoa(count)))
		}
	}

	_, _ = node.Unset("numberOfVideos")
}

func convertGeminiVideoMediaToVertex(node *ast.Node) {
	if node == nil || !node.Exists() || node.TypeSafe() == ast.V_NULL {
		return
	}

	instances := node.Get("instances")
	if instances == nil || !instances.Exists() || instances.TypeSafe() != ast.V_ARRAY {
		return
	}

	length, err := instances.Len()
	if err != nil {
		return
	}

	for i := range length {
		instance := instances.Index(i)
		if instance == nil || !instance.Exists() || instance.TypeSafe() != ast.V_OBJECT {
			continue
		}

		convertGeminiVideoMediaObjectToVertex(instance.Get("image"))
		convertGeminiVideoMediaObjectToVertex(instance.Get("video"))
	}
}

func convertGeminiVideoMediaObjectToVertex(media *ast.Node) {
	if media == nil || !media.Exists() || media.TypeSafe() != ast.V_OBJECT {
		return
	}

	inlineData := media.Get("inlineData")
	if inlineData != nil && inlineData.Exists() && inlineData.TypeSafe() == ast.V_OBJECT {
		if mimeType := inlineData.Get("mimeType"); mimeType != nil && mimeType.Exists() {
			if value, err := mimeType.String(); err == nil {
				_, _ = media.Set("mimeType", ast.NewString(value))
			}
		}

		if data := inlineData.Get("data"); data != nil && data.Exists() {
			if value, err := data.String(); err == nil {
				_, _ = media.Set("bytesBase64Encoded", ast.NewString(value))
			}
		}

		_, _ = media.Unset("inlineData")
	}

	fileData := media.Get("fileData")
	if fileData != nil && fileData.Exists() && fileData.TypeSafe() == ast.V_OBJECT {
		if fileURI := fileData.Get("fileUri"); fileURI != nil && fileURI.Exists() {
			uri, err := fileURI.String()
			if err == nil && strings.HasPrefix(strings.TrimSpace(uri), "gs://") {
				_, _ = media.Set("gcsUri", ast.NewString(uri))
			} else if err == nil {
				_, _ = media.Set("fileUri", ast.NewString(uri))
			}
		}

		_, _ = media.Unset("fileData")
	}
}

func applyVideoPersonGenerationConfig(request *geminiVideoRequest, cfg Config) {
	if request == nil || !cfg.EnablePersonGenerationAllowAll {
		return
	}

	if request.Parameters.PersonGeneration == "" {
		request.Parameters.PersonGeneration = "allow_all"
	}
}

func ensureNativeVideoPersonGenerationAllowAll(body []byte) ([]byte, error) {
	node, err := common.GetJSONNodeNoCopy(body)
	if err != nil {
		return nil, err
	}

	parameters := node.Get("parameters")
	if parameters == nil || !parameters.Exists() || parameters.TypeSafe() == ast.V_NULL {
		if _, err := node.Set("parameters", ast.NewObject(nil)); err != nil {
			return nil, err
		}

		parameters = node.Get("parameters")
	}

	personGeneration := parameters.Get("personGeneration")
	if personGeneration == nil ||
		!personGeneration.Exists() ||
		personGeneration.TypeSafe() == ast.V_NULL {
		if _, err := parameters.Set("personGeneration", ast.NewString("allow_all")); err != nil {
			return nil, err
		}
	}

	return node.MarshalJSON()
}

func parseOpenAIVideoGenerationJobRequest(req *http.Request) (geminiVideoRequest, error) {
	if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
		return parseMultipartOpenAIVideoGenerationJobRequest(req)
	}

	node, err := common.UnmarshalRequest2NodeReusable(req)
	if err != nil {
		return geminiVideoRequest{}, err
	}

	return parseJSONOpenAIVideoGenerationJobRequest(&node), nil
}

func parseOpenAIVideosRequest(req *http.Request) (geminiVideoRequest, error) {
	if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
		return parseMultipartOpenAIVideosRequest(req)
	}

	node, err := common.UnmarshalRequest2NodeReusable(req)
	if err != nil {
		return geminiVideoRequest{}, err
	}

	return parseJSONOpenAIVideosRequest(&node), nil
}

func parseJSONOpenAIVideoGenerationJobRequest(node *ast.Node) geminiVideoRequest {
	request := parseJSONOpenAIVideoCommonRequest(node)
	request.Parameters.DurationSeconds = intNode(node, "n_seconds")
	request.Parameters.NumberOfVideos = intNode(node, "n_variants")

	if request.Parameters.NumberOfVideos <= 0 {
		request.Parameters.NumberOfVideos = 1
	}

	return request
}

func parseJSONOpenAIVideosRequest(node *ast.Node) geminiVideoRequest {
	request := parseJSONOpenAIVideoCommonRequest(node)
	request.Parameters.DurationSeconds = intNode(node, "seconds")
	request.Parameters.NumberOfVideos = 1

	return request
}

func parseJSONOpenAIVideoCommonRequest(node *ast.Node) geminiVideoRequest {
	size := stringNode(node, "size")
	request := geminiVideoRequest{
		Parameters: geminiVideoParameters{
			AspectRatio:      geminiVideoAspectRatioFromSize(size),
			Resolution:       geminiVideoResolutionFromSize(size),
			NegativePrompt:   stringNode(node, "negative_prompt"),
			PersonGeneration: stringNode(node, "person_generation"),
		},
	}

	instance := geminiVideoInstance{
		Prompt: stringNode(node, "prompt"),
	}

	if media := mediaFromString(firstNonEmpty(
		stringNode(node, "input_reference"),
		stringNode(node, "image"),
		stringNode(node, "image_url"),
	)); media != nil {
		instance.Image = media
	}

	if media := mediaFromString(stringNode(node, "video_url")); media != nil {
		instance.Video = media
	}

	if instance.Prompt != "" || instance.Image != nil || instance.Video != nil {
		request.Instances = append(request.Instances, instance)
	}

	return request
}

func parseMultipartOpenAIVideoGenerationJobRequest(req *http.Request) (geminiVideoRequest, error) {
	request, err := parseMultipartOpenAIVideoCommonRequest(req)
	if err != nil {
		return geminiVideoRequest{}, err
	}

	setOptionalPositiveInt(&request.Parameters.DurationSeconds, req.PostFormValue("n_seconds"))
	setOptionalPositiveInt(&request.Parameters.NumberOfVideos, req.PostFormValue("n_variants"))

	if request.Parameters.NumberOfVideos <= 0 {
		request.Parameters.NumberOfVideos = 1
	}

	return request, nil
}

func parseMultipartOpenAIVideosRequest(req *http.Request) (geminiVideoRequest, error) {
	request, err := parseMultipartOpenAIVideoCommonRequest(req)
	if err != nil {
		return geminiVideoRequest{}, err
	}

	setOptionalPositiveInt(&request.Parameters.DurationSeconds, req.PostFormValue("seconds"))
	request.Parameters.NumberOfVideos = 1

	return request, nil
}

func parseMultipartOpenAIVideoCommonRequest(req *http.Request) (geminiVideoRequest, error) {
	if err := common.ParseMultipartFormWithLimit(req); err != nil {
		return geminiVideoRequest{}, fmt.Errorf("parse multipart form: %w", err)
	}

	size := req.PostFormValue("size")
	request := geminiVideoRequest{
		Parameters: geminiVideoParameters{
			AspectRatio:      geminiVideoAspectRatioFromSize(size),
			Resolution:       geminiVideoResolutionFromSize(size),
			NegativePrompt:   strings.TrimSpace(req.PostFormValue("negative_prompt")),
			PersonGeneration: strings.TrimSpace(req.PostFormValue("person_generation")),
		},
	}

	instance := geminiVideoInstance{Prompt: req.PostFormValue("prompt")}
	if media := mediaFromString(firstFormValue(
		req.MultipartForm.Value,
		"input_reference",
		"image",
		"image_url",
	)); media != nil {
		instance.Image = media
	}

	if media := mediaFromString(firstFormValue(
		req.MultipartForm.Value,
		"video_url",
	)); media != nil {
		instance.Video = media
	}

	if media, err := multipartGeminiVideoMedia(
		req.MultipartForm.File,
		"image",
		"input_reference",
	); err != nil {
		return geminiVideoRequest{}, err
	} else if media != nil {
		instance.Image = media
	}

	if media, err := multipartGeminiVideoMedia(req.MultipartForm.File, "video"); err != nil {
		return geminiVideoRequest{}, err
	} else if media != nil {
		instance.Video = media
	}

	if instance.Prompt != "" || instance.Image != nil || instance.Video != nil {
		request.Instances = append(request.Instances, instance)
	}

	return request, nil
}

func stringNode(node *ast.Node, names ...string) string {
	for _, name := range names {
		valueNode := node.Get(name)
		if valueNode == nil || !valueNode.Exists() || valueNode.TypeSafe() == ast.V_NULL {
			continue
		}

		value, err := valueNode.String()
		if err == nil && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}

		if valueNode.TypeSafe() == ast.V_OBJECT {
			urlNode := valueNode.Get("url")
			if urlNode != nil && urlNode.Exists() {
				value, err := urlNode.String()
				if err == nil && strings.TrimSpace(value) != "" {
					return strings.TrimSpace(value)
				}
			}
		}
	}

	return ""
}

func intNode(node *ast.Node, names ...string) int {
	for _, name := range names {
		value, ok := intFromNode(node.Get(name))
		if ok {
			return value
		}
	}

	return 0
}

func firstPositiveIntNode(
	node *ast.Node,
	parameters *ast.Node,
	defaultValue int,
	name string,
) int {
	value := intNode(node, name)
	if value > 0 {
		return value
	}

	value = intNode(parameters, name)
	if value > 0 {
		return value
	}

	return defaultValue
}

func firstStringNode(node, parameters *ast.Node, name string) string {
	value := stringNode(node, name)
	if value != "" {
		return value
	}

	return stringNode(parameters, name)
}

func intFromNode(node *ast.Node) (int, bool) {
	if node == nil || !node.Exists() || node.TypeSafe() == ast.V_NULL {
		return 0, false
	}

	if node.TypeSafe() == ast.V_STRING {
		value, err := node.String()
		if err != nil {
			return 0, false
		}

		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0, false
		}

		return parsed, true
	}

	value, err := node.Int64()
	if err != nil {
		return 0, false
	}

	return int(value), true
}

func mediaFromString(value string) *geminiVideoMedia {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	if mimeType, data, ok := parseMediaDataURL(value, "image"); ok {
		return &geminiVideoMedia{
			InlineData: &relaymodel.GeminiInlineData{
				MimeType: mimeType,
				Data:     data,
			},
		}
	}

	if mimeType, data, ok := parseMediaDataURL(value, "video"); ok {
		return &geminiVideoMedia{
			InlineData: &relaymodel.GeminiInlineData{
				MimeType: mimeType,
				Data:     data,
			},
		}
	}

	return &geminiVideoMedia{FileData: &relaymodel.GeminiFileData{FileURI: value}}
}

func firstFormValue(values map[string][]string, names ...string) string {
	for _, name := range names {
		for _, value := range values[name] {
			value = strings.TrimSpace(value)
			if value != "" {
				return value
			}
		}
	}

	return ""
}

func setOptionalPositiveInt(target *int, values ...string) {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		parsed, err := strconv.Atoi(value)
		if err == nil && parsed > 0 {
			*target = parsed
			return
		}
	}
}

func multipartGeminiVideoMedia(
	files map[string][]*multipart.FileHeader,
	names ...string,
) (*geminiVideoMedia, error) {
	var selected []*multipart.FileHeader
	for _, name := range names {
		selected = append(selected, files[name]...)
	}

	if len(selected) == 0 {
		return nil, nil
	}

	if len(selected) > 1 {
		return nil, errors.New("gemini video supports at most one media file per field")
	}

	fileHeader := selected[0]

	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(common.LimitReader(file, maxGeminiMediaSize+1))
	if err != nil {
		return nil, err
	}

	if len(data) > maxGeminiMediaSize {
		return nil, fmt.Errorf("media too large: max: %d", maxGeminiMediaSize)
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}

	return &geminiVideoMedia{
		InlineData: &relaymodel.GeminiInlineData{
			Data:     base64.StdEncoding.EncodeToString(data),
			MimeType: mimeType,
		},
	}, nil
}

func geminiVideoAspectRatioFromSize(size string) string {
	return relaymodel.VideoAspectRatioFromSize(size)
}

func geminiVideoResolutionFromSize(size string) string {
	size = strings.TrimSpace(size)
	if size == "" {
		return ""
	}

	if width, height, ok := relaymodel.ParseVideoDimensions(size); ok {
		return relaymodel.VideoResolutionFromDimensions(width, height)
	}

	switch strings.ToLower(size) {
	case "480p", "720p", "1080p", "4k":
		return strings.ToLower(size)
	default:
		return ""
	}
}

func NativeVideoHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return adaptor.DoResponseResult{}, ErrorHandler(resp)
	}

	defer resp.Body.Close()

	body, err := common.GetResponseBody(resp)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperGeminiError(
			err,
			http.StatusInternalServerError,
		)
	}

	var operation geminiOperation
	if err := sonic.Unmarshal(body, &operation); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperGeminiError(
			err,
			http.StatusInternalServerError,
		)
	}

	upstreamName := operation.Name
	if operation.Name != "" && meta != nil {
		operationID := nativeGeminiVideoStoreID(operation.Name)
		// Native Gemini clients poll the operation name directly, so retain enough
		// channel/model ownership data to route that GET back to the original channel.
		if err := saveGeminiVideoJobStore(
			meta,
			store,
			operationID,
			operation.Name,
			time.Now().Add(geminiVideoTTL),
		); err != nil {
			common.GetLogger(c).Errorf("save gemini native video operation store failed: %v", err)
		}
	}

	if publicName := publicNativeGeminiVideoOperationName(meta, operation.Name); publicName != "" {
		patchedBody, marshalErr := rewriteGeminiOperationName(body, publicName)
		if marshalErr == nil {
			body = patchedBody
		}
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	_, _ = c.Writer.Write(body)

	return adaptor.DoResponseResult{
		UpstreamID:   upstreamName,
		AsyncUsage:   upstreamName != "",
		UsageContext: geminiVideoUsageContext(meta, &operation),
	}, nil
}

func NativeVideoOperationHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return adaptor.DoResponseResult{}, ErrorHandler(resp)
	}

	defer resp.Body.Close()

	body, err := common.GetResponseBody(resp)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperGeminiError(
			err,
			http.StatusInternalServerError,
		)
	}

	var operation geminiOperation
	if err := sonic.Unmarshal(body, &operation); err == nil {
		publicName := publicNativeGeminiVideoOperationName(meta, operation.Name)
		if publicName != "" {
			patchedBody, marshalErr := rewriteGeminiOperationName(body, publicName)
			if marshalErr == nil {
				body = patchedBody
			}
		}
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	_, _ = c.Writer.Write(body)

	return adaptor.DoResponseResult{}, nil
}

func VideoGenerationJobSubmitHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	operation, relayErr := readGeminiVideoOperation(resp)
	if relayErr != nil {
		return adaptor.DoResponseResult{}, relayErr
	}

	expiresAt := time.Now().Add(geminiVideoTTL)
	jobID := geminiVideoLocalID(operation.Name)

	if err := saveGeminiVideoJobStore(
		meta,
		store,
		jobID,
		operation.Name,
		expiresAt,
	); err != nil {
		common.GetLogger(c).Errorf("save gemini video job store failed: %v", err)
	}

	job := buildGeminiVideoJob(meta, jobID, &operation)
	if job.Status == relaymodel.VideoGenerationJobStatusSucceeded {
		for _, generation := range job.Generations {
			if err := saveGeminiVideoStore(
				meta,
				store,
				generation.ID,
				operation.Name,
				expiresAt,
			); err != nil {
				common.GetLogger(c).Errorf("save gemini video generation store failed: %v", err)
			}
		}
	}

	return writeGeminiVideoObject(c, job, adaptor.DoResponseResult{
		UpstreamID:   operation.Name,
		AsyncUsage:   true,
		UsageContext: geminiVideoUsageContext(meta, &operation),
	})
}

func VideosSubmitHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	operation, relayErr := readGeminiVideoOperation(resp)
	if relayErr != nil {
		return adaptor.DoResponseResult{}, relayErr
	}

	expiresAt := time.Now().Add(geminiVideoTTL)
	videoID := geminiVideoLocalID(operation.Name)

	if err := saveGeminiVideoStore(
		meta,
		store,
		videoID,
		operation.Name,
		expiresAt,
	); err != nil {
		common.GetLogger(c).Errorf("save gemini video store failed: %v", err)
	}

	return writeGeminiVideoObject(
		c,
		buildGeminiVideo(meta, videoID, &operation),
		adaptor.DoResponseResult{
			UpstreamID:   operation.Name,
			AsyncUsage:   true,
			UsageContext: geminiVideoUsageContext(meta, &operation),
		},
	)
}

func readGeminiVideoOperation(resp *http.Response) (geminiOperation, adaptor.Error) {
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return geminiOperation{}, openai.VideoErrorHanlder(resp)
	}

	defer resp.Body.Close()

	var operation geminiOperation
	if err := common.UnmarshalResponse(resp, &operation); err != nil {
		return geminiOperation{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	if operation.Name == "" {
		return geminiOperation{}, relaymodel.WrapperOpenAIVideoErrorWithMessage(
			"missing operation name in gemini video response",
			http.StatusInternalServerError,
		)
	}

	return operation, nil
}

func VideoGenerationJobStatusHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, openai.VideoErrorHanlder(resp)
	}

	defer resp.Body.Close()

	var operation geminiOperation
	if err := common.UnmarshalResponse(resp, &operation); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	applyStoredGeminiVideoRequestMetadata(meta, store, model.VideoJobStoreID(meta.JobID))

	if operation.Name == "" {
		operationName, err := resolveVideoStatusOperationName(meta, store)
		if err != nil {
			return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
				err,
				http.StatusInternalServerError,
			)
		}

		operation.Name = operationName
	}

	expiresAt := time.Now().Add(geminiVideoTTL)
	localID := geminiVideoLocalID(operation.Name)
	job := buildGeminiVideoJob(meta, localID, &operation)

	if job.Status == relaymodel.VideoGenerationJobStatusSucceeded {
		for _, generation := range job.Generations {
			if err := saveGeminiVideoStore(
				meta,
				store,
				generation.ID,
				operation.Name,
				expiresAt,
			); err != nil {
				common.GetLogger(c).Errorf("save gemini video generation store failed: %v", err)
			}
		}
	}

	return writeGeminiVideoObject(
		c,
		job,
		adaptor.DoResponseResult{UsageContext: geminiVideoUsageContext(meta, &operation)},
	)
}

func VideosStatusHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, openai.VideoErrorHanlder(resp)
	}

	defer resp.Body.Close()

	var operation geminiOperation
	if err := common.UnmarshalResponse(resp, &operation); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	applyStoredGeminiVideoRequestMetadata(meta, store, model.VideoGenerationStoreID(meta.VideoID))

	if operation.Name == "" {
		operationName, err := resolveVideosStatusOperationName(meta, store)
		if err != nil {
			return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
				err,
				http.StatusInternalServerError,
			)
		}

		operation.Name = operationName
	}

	localID := geminiVideoLocalID(operation.Name)
	if len(geminiVideoURLs(&operation)) > 0 {
		if err := saveGeminiVideoStore(
			meta,
			store,
			localID,
			operation.Name,
			time.Now().Add(geminiVideoTTL),
		); err != nil {
			common.GetLogger(c).Errorf("save gemini video store failed: %v", err)
		}
	}

	return writeGeminiVideoObject(
		c,
		buildGeminiVideo(meta, localID, &operation),
		adaptor.DoResponseResult{UsageContext: geminiVideoUsageContext(meta, &operation)},
	)
}

func VideoGenerationJobContentHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	return fetchGeminiVideoContentHandler(meta, c, resp, meta.GenerationID)
}

func VideosContentHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	return fetchGeminiVideoContentHandler(meta, c, resp, meta.VideoID)
}

func fetchGeminiVideoContentHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
	id string,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, openai.VideoErrorHanlder(resp)
	}

	defer resp.Body.Close()

	var operation geminiOperation
	if err := common.UnmarshalResponse(resp, &operation); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	videoURL := geminiVideoURLByID(&operation, id)
	if videoURL == "" {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoErrorWithMessage(
			"video url is empty",
			http.StatusInternalServerError,
		)
	}

	videoResp, err := fetchGeminiVideoContent(c.Request.Context(), meta, videoURL)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}
	defer videoResp.Body.Close()

	if videoResp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoErrorWithMessage(
			fmt.Sprintf("unexpected video status code: %d", videoResp.StatusCode),
			http.StatusInternalServerError,
		)
	}

	c.Writer.Header().
		Set("Content-Type", firstNonEmpty(videoResp.Header.Get("Content-Type"), "video/mp4"))
	c.Writer.Header().Set("Content-Length", videoResp.Header.Get("Content-Length"))
	_, _ = io.Copy(c.Writer, videoResp.Body)

	return adaptor.DoResponseResult{UpstreamID: id}, nil
}

func resolveVideoStatusOperationName(meta *meta.Meta, store adaptor.Store) (string, error) {
	if meta == nil || meta.JobID == "" {
		return "", nil
	}

	return ResolveVideoJobOperationID(meta, store, meta.JobID)
}

func resolveVideosStatusOperationName(meta *meta.Meta, store adaptor.Store) (string, error) {
	if meta == nil || meta.VideoID == "" {
		return "", nil
	}

	return ResolveVideoGenerationOperationID(meta, store, meta.VideoID)
}

func buildGeminiVideoJob(
	meta *meta.Meta,
	id string,
	operation *geminiOperation,
) relaymodel.VideoGenerationJob {
	now := time.Now().Unix()
	status := geminiOperationVideoJobStatus(operation)

	var finishedAt *int64
	if status == relaymodel.VideoGenerationJobStatusSucceeded || geminiOperationFailed(operation) {
		finishedAt = &now
	}

	urls := geminiVideoURLs(operation)
	generations := make([]relaymodel.VideoGenerations, 0, len(urls))

	for index := range urls {
		generationID := geminiVideoGenerationID(id, index)
		generations = append(generations, relaymodel.VideoGenerations{
			Object:    "video.generation",
			ID:        generationID,
			JobID:     id,
			CreatedAt: now,
			Width:     requestVideoWidth(meta),
			Height:    requestVideoHeight(meta),
			Prompt:    metaPrompt(meta),
			NSeconds:  requestVideoSeconds(meta),
		})
	}

	return relaymodel.VideoGenerationJob{
		Object:      "video.generation.job",
		ID:          id,
		Status:      status,
		CreatedAt:   now,
		FinishedAt:  finishedAt,
		Prompt:      metaPrompt(meta),
		Model:       meta.OriginModel,
		NVariants:   maxInt(requestVideoVariants(meta), 1),
		NSeconds:    requestVideoSeconds(meta),
		Width:       requestVideoWidth(meta),
		Height:      requestVideoHeight(meta),
		Generations: generations,
	}
}

func buildGeminiVideo(
	meta *meta.Meta,
	id string,
	operation *geminiOperation,
) relaymodel.Video {
	status := geminiOperationVideoStatus(operation)
	video := relaymodel.Video{
		ID:        id,
		Object:    "video",
		CreatedAt: time.Now().Unix(),
		Status:    status,
		Model:     meta.OriginModel,
		Prompt:    metaPrompt(meta),
		Seconds:   requestVideoSeconds(meta),
		Size:      requestVideoResolution(meta),
		Progress:  geminiVideoProgress(status),
	}

	if operation != nil && operation.Error != nil {
		video.Error = map[string]any{
			"code":    operation.Error.Code,
			"message": operation.Error.Message,
		}
	}

	return video
}

func geminiOperationVideoJobStatus(operation *geminiOperation) relaymodel.VideoGenerationJobStatus {
	if geminiOperationSucceeded(operation) {
		return relaymodel.VideoGenerationJobStatusSucceeded
	}

	if geminiOperationFailed(operation) {
		return "failed"
	}

	return relaymodel.VideoGenerationJobStatusQueued
}

func geminiOperationVideoStatus(operation *geminiOperation) relaymodel.VideoStatus {
	if geminiOperationSucceeded(operation) {
		return relaymodel.VideoStatusCompleted
	}

	if geminiOperationFailed(operation) {
		return relaymodel.VideoStatusFailed
	}

	return relaymodel.VideoStatusQueued
}

func geminiOperationSucceeded(operation *geminiOperation) bool {
	return operation != nil &&
		operation.Done &&
		len(geminiVideoURLs(operation)) > 0 &&
		operation.Error == nil
}

func geminiOperationFailed(operation *geminiOperation) bool {
	return operation != nil && operation.Done && operation.Error != nil
}

func geminiVideoProgress(status relaymodel.VideoStatus) int {
	if status == relaymodel.VideoStatusCompleted {
		return 100
	}

	return 0
}

func geminiVideoURLs(operation *geminiOperation) []string {
	if operation == nil {
		return nil
	}

	samples := operation.Response.GenerateVideoResponse.GeneratedSamples
	urls := make([]string, 0, len(samples))

	for _, sample := range samples {
		if sample.Video.URI != "" {
			urls = append(urls, sample.Video.URI)
		}
	}

	return urls
}

func geminiVideoGenerationID(operationName string, index int) string {
	if index == 0 {
		return operationName
	}

	return fmt.Sprintf("%s:%d", operationName, index)
}

func geminiVideoURLByID(operation *geminiOperation, id string) string {
	urls := geminiVideoURLs(operation)
	if len(urls) == 0 {
		return ""
	}

	index := geminiVideoGenerationIndex(id)
	if index < 0 || index >= len(urls) {
		return ""
	}

	return urls[index]
}

func geminiVideoGenerationIndex(id string) int {
	_, suffix, ok := strings.Cut(id, ":")
	if !ok {
		return 0
	}

	index, err := strconv.Atoi(suffix)
	if err != nil || index < 0 {
		return -1
	}

	return index
}

func VideoOperationID(id string) string {
	operationID, _, _ := strings.Cut(id, ":")

	return geminiVideoOperationID(operationID)
}

func ResolveVideoOperationID(
	meta *meta.Meta,
	store adaptor.Store,
	id string,
) (string, error) {
	return ResolveVideoJobOperationID(meta, store, id)
}

func ResolveVideoJobOperationID(
	meta *meta.Meta,
	store adaptor.Store,
	id string,
) (string, error) {
	operationID, _, _ := strings.Cut(id, ":")

	return resolveGeminiVideoOperationID(
		meta,
		store,
		operationID,
		geminiVideoStoredJobOperationName,
	)
}

func ResolveVideoGenerationOperationID(
	meta *meta.Meta,
	store adaptor.Store,
	id string,
) (string, error) {
	operationID, _, _ := strings.Cut(id, ":")

	return resolveGeminiVideoOperationID(
		meta,
		store,
		operationID,
		geminiVideoStoredGenerationOperationName,
	)
}

func geminiVideoLocalID(operationName string) string {
	if operationName == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(operationName))
	encoded := hex.EncodeToString(sum[:])

	return geminiVideoLocalIDPrefix + encoded
}

func geminiVideoOperationID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}

	return id
}

func resolveGeminiVideoOperationID(
	meta *meta.Meta,
	store adaptor.Store,
	id string,
	storedOperationName func(*meta.Meta, adaptor.Store, string) (string, error),
) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", nil
	}

	if !strings.HasPrefix(id, geminiVideoLocalIDPrefix) {
		return geminiVideoOperationID(id), nil
	}

	if store != nil && meta != nil {
		if operationName, err := storedOperationName(meta, store, id); err != nil {
			return "", err
		} else if operationName != "" {
			return operationName, nil
		}
	}

	return geminiVideoOperationID(id), nil
}

func getNativeVideoOperationRequestURL(
	meta *meta.Meta,
	store adaptor.Store,
) (adaptor.RequestURL, error) {
	operationName := nativeGeminiVideoUpstreamOperationName(meta, store)
	if operationName == "" {
		return adaptor.RequestURL{}, errors.New("operation id is empty")
	}

	return getOperationRequestURL(meta, operationName)
}

func NativeGeminiVideoUpstreamOperationName(meta *meta.Meta, store adaptor.Store) string {
	return nativeGeminiVideoUpstreamOperationName(meta, store)
}

func nativeGeminiVideoUpstreamOperationName(meta *meta.Meta, store adaptor.Store) string {
	if meta == nil || meta.OperationID == "" {
		return ""
	}

	if operationName := storedNativeGeminiVideoOperationName(meta, store); operationName != "" {
		return operationName
	}

	return nativeGeminiVideoOperationNameForModel(meta.ActualModel, meta.OperationID)
}

func publicNativeGeminiVideoOperationName(meta *meta.Meta, upstreamName string) string {
	if meta == nil {
		return ""
	}

	operationID := operationIDFromGeminiOperationName(upstreamName)
	if operationID == "" {
		operationID = meta.OperationID
	}

	if operationID == "" {
		return ""
	}

	return nativeGeminiVideoOperationNameForModel(meta.OriginModel, operationID)
}

func storedNativeGeminiVideoOperationName(meta *meta.Meta, store adaptor.Store) string {
	if meta == nil || meta.OperationID == "" || meta.Token.ID == 0 {
		return ""
	}

	if store != nil {
		cache, err := store.GetStore(
			meta.Group.ID,
			meta.Token.ID,
			model.VideoJobStoreID(meta.OperationID),
		)
		if err != nil {
			return ""
		}

		return strings.TrimSpace(cache.Metadata)
	} else {
		cache, err := model.CacheGetStore(
			meta.Group.ID,
			meta.Token.ID,
			model.VideoJobStoreID(meta.OperationID),
		)
		if err != nil {
			return ""
		}

		return strings.TrimSpace(cache.Metadata)
	}
}

func nativeGeminiVideoOperationNameForModel(modelName, operationID string) string {
	operationID = strings.Trim(operationID, "/")
	if operationID == "" {
		return ""
	}

	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return "operations/" + operationID
	}

	return "models/" + modelName + "/operations/" + operationID
}

func rewriteGeminiOperationName(body []byte, name string) ([]byte, error) {
	node, err := common.GetJSONNodeNoCopy(body)
	if err != nil {
		return nil, err
	}

	if _, err := node.Set("name", ast.NewString(name)); err != nil {
		return nil, err
	}

	return node.MarshalJSON()
}

func nativeGeminiVideoStoreID(operationName string) string {
	if operationID := operationIDFromGeminiOperationName(operationName); operationID != "" {
		return operationID
	}

	return geminiVideoLocalID(operationName)
}

func operationIDFromGeminiOperationName(operationName string) string {
	operationName = strings.Trim(strings.TrimSpace(operationName), "/")
	if operationName == "" {
		return ""
	}

	if operationID, ok := strings.CutPrefix(operationName, "operations/"); ok {
		return operationID
	}

	_, operationID, ok := strings.Cut(operationName, "/operations/")
	if !ok {
		return ""
	}

	return strings.Trim(operationID, "/")
}

func geminiVideoStoredJobOperationName(
	meta *meta.Meta,
	store adaptor.Store,
	id string,
) (string, error) {
	metadata, err := geminiVideoStoredMetadata(meta, store, model.VideoJobStoreID(id))
	if err != nil {
		return "", err
	}

	return metadata.OperationName, nil
}

func geminiVideoStoredGenerationOperationName(
	meta *meta.Meta,
	store adaptor.Store,
	id string,
) (string, error) {
	metadata, err := geminiVideoStoredMetadata(meta, store, model.VideoGenerationStoreID(id))
	if err != nil {
		return "", err
	}

	return metadata.OperationName, nil
}

func geminiVideoStoredMetadata(
	meta *meta.Meta,
	store adaptor.Store,
	storeID string,
) (geminiVideoStoreMetadata, error) {
	if store == nil || meta == nil || storeID == "" {
		return geminiVideoStoreMetadata{}, nil
	}

	cache, err := store.GetStore(meta.Group.ID, meta.Token.ID, storeID)
	if err != nil {
		return geminiVideoStoreMetadata{}, err
	}

	return parseGeminiVideoStoreMetadata(cache.Metadata), nil
}

func applyStoredGeminiVideoRequestMetadata(
	meta *meta.Meta,
	store adaptor.Store,
	storeID string,
) {
	metadata, err := geminiVideoStoredMetadata(meta, store, storeID)
	if err != nil {
		return
	}

	if metadata.Prompt != "" {
		meta.Set("gemini_video_prompt", metadata.Prompt)
	}

	if metadata.Seconds > 0 ||
		metadata.Variants > 0 ||
		metadata.Resolution != "" ||
		metadata.Width > 0 ||
		metadata.Height > 0 {
		setGeminiVideoRequestMetadata(
			meta,
			metadata.Seconds,
			metadata.Variants,
			metadata.Resolution,
			metadata.Width,
			metadata.Height,
		)
	}
}

func parseGeminiVideoStoreMetadata(value string) geminiVideoStoreMetadata {
	value = strings.TrimSpace(value)
	if value == "" {
		return geminiVideoStoreMetadata{}
	}

	var metadata geminiVideoStoreMetadata
	if err := sonic.UnmarshalString(value, &metadata); err == nil && metadata.OperationName != "" {
		return metadata
	}

	return geminiVideoStoreMetadata{OperationName: value}
}

func requestVideoSeconds(meta *meta.Meta) int {
	if meta == nil {
		return 0
	}

	return meta.GetInt("gemini_video_seconds")
}

func requestVideoVariants(meta *meta.Meta) int {
	if meta == nil {
		return 0
	}

	return meta.GetInt("gemini_video_variants")
}

func requestVideoResolution(meta *meta.Meta) string {
	if meta == nil {
		return ""
	}

	return meta.GetString(metaGeminiVideoResolution)
}

func requestVideoWidth(meta *meta.Meta) int {
	if meta == nil {
		return 0
	}

	return meta.GetInt("gemini_video_width")
}

func requestVideoHeight(meta *meta.Meta) int {
	if meta == nil {
		return 0
	}

	return meta.GetInt("gemini_video_height")
}

func metaPrompt(meta *meta.Meta) string {
	if meta == nil {
		return ""
	}

	return meta.GetString("gemini_video_prompt")
}

func saveGeminiVideoJobStore(
	meta *meta.Meta,
	store adaptor.Store,
	id string,
	operationName string,
	expiresAt time.Time,
) error {
	if store == nil || id == "" {
		return nil
	}

	return store.SaveStore(adaptor.StoreCache{
		ID:        model.VideoJobStoreID(id),
		GroupID:   meta.Group.ID,
		TokenID:   meta.Token.ID,
		ChannelID: meta.Channel.ID,
		Model:     meta.OriginModel,
		Metadata:  geminiVideoStoreMetadataString(meta, operationName),
		ExpiresAt: expiresAt,
	})
}

func saveGeminiVideoStore(
	meta *meta.Meta,
	store adaptor.Store,
	id string,
	operationName string,
	expiresAt time.Time,
) error {
	if store == nil || id == "" {
		return nil
	}

	return store.SaveStore(adaptor.StoreCache{
		ID:        model.VideoGenerationStoreID(id),
		GroupID:   meta.Group.ID,
		TokenID:   meta.Token.ID,
		ChannelID: meta.Channel.ID,
		Model:     meta.OriginModel,
		Metadata:  geminiVideoStoreMetadataString(meta, operationName),
		ExpiresAt: expiresAt,
	})
}

func geminiVideoStoreMetadataString(meta *meta.Meta, operationName string) string {
	metadata := geminiVideoStoreMetadata{
		OperationName: operationName,
		Prompt:        metaPrompt(meta),
		Seconds:       requestVideoSeconds(meta),
		Variants:      requestVideoVariants(meta),
		Resolution:    requestVideoResolution(meta),
		Width:         requestVideoWidth(meta),
		Height:        requestVideoHeight(meta),
	}

	body, err := sonic.MarshalString(metadata)
	if err != nil {
		return operationName
	}

	return body
}

func writeGeminiVideoObject(
	c *gin.Context,
	value any,
	result adaptor.DoResponseResult,
) (adaptor.DoResponseResult, adaptor.Error) {
	body, err := sonic.Marshal(value)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	_, _ = c.Writer.Write(body)

	return result, nil
}

func fetchGeminiVideoContent(
	ctx context.Context,
	meta *meta.Meta,
	videoURL string,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, videoURL, nil)
	if err != nil {
		return nil, err
	}

	if meta != nil && meta.Channel.Key != "" {
		req.Header.Set("X-Goog-Api-Key", meta.Channel.Key)
	}

	var (
		proxyURL      string
		skipTLSVerify bool
	)
	if meta != nil {
		proxyURL = meta.Channel.ProxyURL
		skipTLSVerify = meta.Channel.SkipTLSVerify
	}

	client, err := relayutils.LoadHTTPClientWithTLSConfigE(0, proxyURL, skipTLSVerify)
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}

func geminiVideoUsageContext(meta *meta.Meta, operation *geminiOperation) model.UsageContext {
	context := model.UsageContext{}
	if meta != nil {
		context = meta.RequestUsageContext
		if resolution := meta.GetString(metaGeminiVideoResolution); resolution != "" {
			context.Resolution = resolution
		}
	}

	if operation == nil {
		return context
	}

	return context
}

func setGeminiVideoRequestMetadata(
	meta *meta.Meta,
	seconds int,
	variants int,
	resolution string,
	width int,
	height int,
) {
	if meta == nil {
		return
	}

	if variants <= 0 {
		variants = 1
	}

	meta.Set(metaGeminiVideoSeconds, seconds)
	meta.Set(metaGeminiVideoVariants, variants)

	if resolution != "" {
		meta.Set(metaGeminiVideoResolution, resolution)
	}

	if width > 0 && height > 0 {
		meta.Set("gemini_video_width", width)
		meta.Set("gemini_video_height", height)
	}
}

func requestVideoDimensionsFromAspectRatio(aspectRatio, resolution string) (int, int) {
	shortSide := videoResolutionShortSide(resolution)
	if shortSide <= 0 {
		return 0, 0
	}

	widthRatio, heightRatio := parseVideoAspectRatio(aspectRatio)
	if widthRatio <= 0 || heightRatio <= 0 {
		widthRatio, heightRatio = 16, 9
	}

	if widthRatio >= heightRatio {
		return shortSide * widthRatio / heightRatio, shortSide
	}

	return shortSide, shortSide * heightRatio / widthRatio
}

func videoResolutionShortSide(resolution string) int {
	resolution = strings.ToLower(strings.TrimSpace(resolution))
	switch resolution {
	case "480p":
		return 480
	case "720p":
		return 720
	case "1080p":
		return 1080
	case "4k":
		return 2160
	default:
		return 0
	}
}

func parseVideoAspectRatio(aspectRatio string) (int, int) {
	left, right, ok := strings.Cut(strings.TrimSpace(aspectRatio), ":")
	if !ok {
		return 0, 0
	}

	widthRatio, err := strconv.Atoi(left)
	if err != nil {
		return 0, 0
	}

	heightRatio, err := strconv.Atoi(right)
	if err != nil {
		return 0, 0
	}

	return widthRatio, heightRatio
}

func geminiVideoMetadataDurationSeconds(seconds int) int {
	if seconds > 0 {
		return seconds
	}

	return defaultGeminiVideoDurationSeconds
}

func geminiVideoMetadataResolution(resolution string) string {
	resolution = strings.TrimSpace(resolution)
	if resolution != "" {
		return resolution
	}

	return defaultGeminiVideoResolution
}

func geminiVideoRequestUsage(meta *meta.Meta) model.Usage {
	if meta == nil {
		return model.Usage{}
	}

	seconds := meta.GetInt(metaGeminiVideoSeconds)
	if seconds <= 0 {
		return model.Usage{}
	}

	variants := meta.GetInt(metaGeminiVideoVariants)
	if variants <= 0 {
		variants = 1
	}

	tokens := model.ZeroNullInt64(int64(seconds * variants))

	return model.Usage{
		OutputTokens: tokens,
		TotalTokens:  tokens,
	}
}

func maxInt(values ...int) int {
	maxValue := 0
	for _, value := range values {
		if value > maxValue {
			maxValue = value
		}
	}

	return maxValue
}

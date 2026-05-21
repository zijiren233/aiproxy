package ali

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/image"
	coremodel "github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	relayutils "github.com/labring/aiproxy/core/relay/utils"
)

var _ adaptor.AsyncUsageFetcher = (*Adaptor)(nil)

const aliVideoTaskTTL = 24 * time.Hour

const (
	metaAliVideoPrompt  = "ali_video_prompt"
	metaAliVideoSeconds = "ali_video_seconds"
	metaAliVideoWidth   = "ali_video_width"
	metaAliVideoHeight  = "ali_video_height"
	metaAliVideoSize    = "ali_video_size"
)

type aliVideoOpenAIRequest struct {
	relaymodel.VideoGenerationJobRequest
	Seconds            any             `json:"seconds,omitempty"`
	InputReference     string          `json:"input_reference,omitempty"`
	ImageURL           string          `json:"image_url,omitempty"`
	ImgURL             string          `json:"img_url,omitempty"`
	FirstFrameURL      string          `json:"first_frame_url,omitempty"`
	LastFrameURL       string          `json:"last_frame_url,omitempty"`
	FirstClipURL       string          `json:"first_clip_url,omitempty"`
	AudioURL           string          `json:"audio_url,omitempty"`
	VideoURL           string          `json:"video_url,omitempty"`
	NegativePrompt     string          `json:"negative_prompt,omitempty"`
	Size               string          `json:"size,omitempty"`
	Ratio              string          `json:"ratio,omitempty"`
	PromptExtend       *bool           `json:"prompt_extend,omitempty"`
	Watermark          *bool           `json:"watermark,omitempty"`
	Seed               *int64          `json:"seed,omitempty"`
	ShotType           string          `json:"shot_type,omitempty"`
	Audio              *bool           `json:"audio,omitempty"`
	Template           string          `json:"template,omitempty"`
	Media              []aliVideoMedia `json:"media,omitempty"`
	ReferenceURLs      []string        `json:"reference_urls,omitempty"`
	ReferenceImageURLs []string        `json:"reference_image_urls,omitempty"`
	Input              map[string]any  `json:"input,omitempty"`
	Parameters         map[string]any  `json:"parameters,omitempty"`
	Metadata           map[string]any  `json:"metadata,omitempty"`
	Ext                map[string]any  `json:"ext,omitempty"`
}

type aliVideoMedia struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type aliVideoRequest struct {
	Model      string         `json:"model"`
	Input      map[string]any `json:"input"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

type AliVideoTaskResponse struct {
	RequestID  string        `json:"request_id,omitempty"`
	Code       string        `json:"code,omitempty"`
	Message    string        `json:"message,omitempty"`
	Output     AliVideoTask  `json:"output,omitempty"`
	Usage      AliVideoUsage `json:"usage,omitempty"`
	StatusCode int           `json:"status_code,omitempty"`
}

type AliVideoTask struct {
	TaskID         string `json:"task_id,omitempty"`
	TaskStatus     string `json:"task_status,omitempty"`
	SubmitTime     string `json:"submit_time,omitempty"`
	ScheduledTime  string `json:"scheduled_time,omitempty"`
	EndTime        string `json:"end_time,omitempty"`
	VideoURL       string `json:"video_url,omitempty"`
	OutputVideoURL string `json:"output_video_url,omitempty"`
	OrigPrompt     string `json:"orig_prompt,omitempty"`
	Code           string `json:"code,omitempty"`
	Message        string `json:"message,omitempty"`
}

type AliVideoUsage struct {
	Duration            int64  `json:"duration,omitempty"`
	InputVideoDuration  int64  `json:"input_video_duration,omitempty"`
	OutputVideoDuration int64  `json:"output_video_duration,omitempty"`
	VideoDuration       int64  `json:"video_duration,omitempty"`
	VideoCount          int64  `json:"video_count,omitempty"`
	SR                  any    `json:"SR,omitempty"`
	Ratio               string `json:"ratio,omitempty"`
	Audio               *bool  `json:"audio,omitempty"`
}

func getAliVideoRequestURL(baseURL string, meta *meta.Meta) (adaptor.RequestURL, error) {
	path := "/api/v1/services/aigc/video-generation/video-synthesis"
	method := http.MethodPost

	taskID := aliVideoTaskID(meta)
	if taskID != "" {
		path = "/api/v1/tasks/" + taskID
		method = http.MethodGet
	}

	targetURL, err := url.JoinPath(baseURL, path)
	if err != nil {
		return adaptor.RequestURL{}, err
	}

	return adaptor.RequestURL{
		Method: method,
		URL:    targetURL,
	}, nil
}

func aliVideoTaskID(meta *meta.Meta) string {
	if meta == nil {
		return ""
	}

	switch meta.Mode {
	case mode.VideoGenerationsGetJobs:
		return meta.JobID
	case mode.VideoGenerationsContent:
		return meta.GenerationID
	case mode.VideosGet, mode.VideosContent:
		return meta.VideoID
	default:
		return ""
	}
}

func ConvertAliVideoRequest(meta *meta.Meta, req *http.Request) (adaptor.ConvertResult, error) {
	request, err := unmarshalAliVideoRequest(req)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	if request.NVariants > 1 {
		return adaptor.ConvertResult{}, errors.New("n_variants must be 1 for Ali video models")
	}

	if err := hydrateAliVideoRemixReference(meta, req, request); err != nil {
		return adaptor.ConvertResult{}, err
	}

	setAliVideoRequestMetadata(meta, request)

	body := aliVideoRequest{
		Model: meta.ActualModel,
		Input: buildAliVideoInput(meta, request),
	}

	parameters := buildAliVideoParameters(meta, request)
	if len(parameters) > 0 {
		body.Parameters = parameters
	}

	data, err := sonic.Marshal(&body)
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

func setAliVideoRequestMetadata(meta *meta.Meta, request *aliVideoOpenAIRequest) {
	if request.Prompt != "" {
		meta.Set(metaAliVideoPrompt, request.Prompt)
	}

	if seconds := aliVideoRequestSeconds(request); seconds > 0 {
		meta.Set(metaAliVideoSeconds, seconds)
	}

	width, height := aliVideoRequestDimensions(request)
	if width > 0 && height > 0 {
		meta.Set(metaAliVideoWidth, width)
		meta.Set(metaAliVideoHeight, height)
		meta.Set(metaAliVideoSize, fmt.Sprintf("%dx%d", width, height))
	} else if request.Size != "" {
		meta.Set(metaAliVideoSize, request.Size)
	}
}

func hydrateAliVideoRemixReference(
	meta *meta.Meta,
	req *http.Request,
	request *aliVideoOpenAIRequest,
) error {
	if meta.Mode != mode.VideosRemix || requestHasAliVideoReference(request) {
		return nil
	}

	if meta.VideoID == "" {
		return errors.New("video_id is required")
	}

	channel := &coremodel.Channel{
		BaseURL:       meta.Channel.BaseURL,
		Key:           meta.Channel.Key,
		ProxyURL:      meta.Channel.ProxyURL,
		SkipTLSVerify: meta.Channel.SkipTLSVerify,
	}

	task, err := fetchAliVideoTask(req.Context(), channel, meta.Channel.BaseURL, meta.VideoID)
	if err != nil {
		return fmt.Errorf("fetch remix source video: %w", err)
	}

	videoURL := firstNonEmpty(task.Output.VideoURL, task.Output.OutputVideoURL)
	if videoURL == "" {
		return errors.New("remix source video url is empty")
	}

	request.VideoURL = videoURL

	return nil
}

func requestHasAliVideoReference(request *aliVideoOpenAIRequest) bool {
	return request.InputReference != "" ||
		request.ImageURL != "" ||
		request.ImgURL != "" ||
		request.FirstFrameURL != "" ||
		request.FirstClipURL != "" ||
		request.VideoURL != "" ||
		len(request.Media) > 0 ||
		len(request.ReferenceURLs) > 0 ||
		len(request.ReferenceImageURLs) > 0
}

func unmarshalAliVideoRequest(req *http.Request) (*aliVideoOpenAIRequest, error) {
	var request aliVideoOpenAIRequest

	contentType := req.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		if err := common.UnmarshalRequestReusable(req, &request); err != nil {
			return nil, err
		}

		if err := rejectAliVideoDangerousAliases(req, &request); err != nil {
			return nil, err
		}

		return &request, nil
	}

	if err := common.ParseMultipartFormWithLimit(req); err != nil {
		return nil, err
	}

	request.Model = req.PostFormValue("model")
	request.Prompt = req.PostFormValue("prompt")
	request.InputReference = req.PostFormValue("input_reference")
	request.Size = req.PostFormValue("size")
	request.Ratio = req.PostFormValue("ratio")
	request.NegativePrompt = req.PostFormValue("negative_prompt")
	request.ImageURL = req.PostFormValue("image_url")
	request.ImgURL = req.PostFormValue("img_url")
	request.FirstFrameURL = req.PostFormValue("first_frame_url")
	request.LastFrameURL = req.PostFormValue("last_frame_url")
	request.FirstClipURL = req.PostFormValue("first_clip_url")
	request.AudioURL = req.PostFormValue("audio_url")
	request.VideoURL = req.PostFormValue("video_url")
	request.ShotType = req.PostFormValue("shot_type")
	request.Template = req.PostFormValue("template")

	if err := parseAliVideoFormFields(req, &request); err != nil {
		return nil, err
	}

	if err := rejectAliVideoDangerousAliases(req, &request); err != nil {
		return nil, err
	}

	if request.InputReference == "" {
		value, err := multipartVideoReferenceToDataURL(req.MultipartForm.File)
		if err != nil {
			return nil, err
		}

		request.InputReference = value
	}

	return &request, nil
}

func rejectAliVideoDangerousAliases(req *http.Request, request *aliVideoOpenAIRequest) error {
	if request.Parameters != nil {
		if err := rejectAliVideoDangerousParameterAliases(request.Parameters); err != nil {
			return err
		}
	}

	if err := rejectAliVideoDangerousMetadataAliases(request.Metadata); err != nil {
		return err
	}

	if err := rejectAliVideoDangerousMetadataAliases(request.Ext); err != nil {
		return err
	}

	if request.Input != nil {
		if _, ok := request.Input["resolution"]; ok {
			return errors.New("resolution is not supported, use size")
		}
	}

	contentType := req.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if req.PostFormValue("duration") != "" {
			return errors.New("duration is not supported, use seconds")
		}

		if req.PostFormValue("resolution") != "" {
			return errors.New("resolution is not supported, use size")
		}

		return nil
	}

	node, err := common.UnmarshalRequest2NodeReusable(req)
	if err != nil {
		return err
	}

	if value := node.Get("duration"); value != nil &&
		value.Exists() &&
		value.TypeSafe() != ast.V_NULL {
		return errors.New("duration is not supported, use seconds")
	}

	if value := node.Get("resolution"); value != nil &&
		value.Exists() &&
		value.TypeSafe() != ast.V_NULL {
		return errors.New("resolution is not supported, use size")
	}

	inputNode := node.Get("input")
	if inputNode != nil && inputNode.Exists() && inputNode.TypeSafe() != ast.V_NULL {
		if value := inputNode.Get("resolution"); value != nil &&
			value.Exists() &&
			value.TypeSafe() != ast.V_NULL {
			return errors.New("resolution is not supported, use size")
		}
	}

	return nil
}

func rejectAliVideoDangerousMetadataAliases(metadata map[string]any) error {
	if len(metadata) == 0 {
		return nil
	}

	if rawParameters, ok := metadata["parameters"].(map[string]any); ok {
		if err := rejectAliVideoDangerousParameterAliases(rawParameters); err != nil {
			return err
		}
	}

	return nil
}

func rejectAliVideoDangerousParameterAliases(parameters map[string]any) error {
	if _, ok := parameters["duration"]; ok {
		return errors.New("duration is not supported in parameters, use seconds")
	}

	if _, ok := parameters["resolution"]; ok {
		return errors.New("resolution is not supported in parameters, use size")
	}

	return nil
}

func parseAliVideoFormFields(req *http.Request, request *aliVideoOpenAIRequest) error {
	var err error

	if seconds := req.PostFormValue("seconds"); seconds != "" {
		request.Seconds = seconds
	}

	if nSeconds := req.PostFormValue("n_seconds"); nSeconds != "" {
		request.NSeconds, err = strconv.Atoi(nSeconds)
		if err != nil {
			return fmt.Errorf("invalid n_seconds: %w", err)
		}
	}

	if nVariants := req.PostFormValue("n_variants"); nVariants != "" {
		request.NVariants, err = strconv.Atoi(nVariants)
		if err != nil {
			return fmt.Errorf("invalid n_variants: %w", err)
		}
	}

	if width := req.PostFormValue("width"); width != "" {
		request.Width, err = strconv.Atoi(width)
		if err != nil {
			return fmt.Errorf("invalid width: %w", err)
		}
	}

	if height := req.PostFormValue("height"); height != "" {
		request.Height, err = strconv.Atoi(height)
		if err != nil {
			return fmt.Errorf("invalid height: %w", err)
		}
	}

	if promptExtend := req.PostFormValue("prompt_extend"); promptExtend != "" {
		value, err := strconv.ParseBool(promptExtend)
		if err != nil {
			return fmt.Errorf("invalid prompt_extend: %w", err)
		}

		request.PromptExtend = &value
	}

	if watermark := req.PostFormValue("watermark"); watermark != "" {
		value, err := strconv.ParseBool(watermark)
		if err != nil {
			return fmt.Errorf("invalid watermark: %w", err)
		}

		request.Watermark = &value
	}

	if audio := req.PostFormValue("audio"); audio != "" {
		value, err := strconv.ParseBool(audio)
		if err != nil {
			return fmt.Errorf("invalid audio: %w", err)
		}

		request.Audio = &value
	}

	if seed := req.PostFormValue("seed"); seed != "" {
		value, err := strconv.ParseInt(seed, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid seed: %w", err)
		}

		request.Seed = &value
	}

	for _, field := range []struct {
		name   string
		target *map[string]any
	}{
		{name: "input", target: &request.Input},
		{name: "parameters", target: &request.Parameters},
		{name: "metadata", target: &request.Metadata},
		{name: "ext", target: &request.Ext},
	} {
		if err := parseAliVideoJSONFormField(req, field.name, field.target); err != nil {
			return err
		}
	}

	return nil
}

func parseAliVideoJSONFormField(req *http.Request, name string, target *map[string]any) error {
	value := req.PostFormValue(name)
	if value == "" {
		return nil
	}

	var parsed map[string]any
	if err := sonic.UnmarshalString(value, &parsed); err != nil {
		return fmt.Errorf("invalid %s: %w", name, err)
	}

	*target = parsed

	return nil
}

func multipartVideoReferenceToDataURL(
	files map[string][]*multipart.FileHeader,
) (string, error) {
	fileHeaders := files["input_reference"]

	fileHeaders = append(fileHeaders, files["input_reference[]"]...)
	if len(fileHeaders) == 0 {
		return "", nil
	}

	if len(fileHeaders) > 1 {
		return "", errors.New("input_reference supports at most 1 file")
	}

	return multipartVideoReferenceFileToDataURL(fileHeaders[0])
}

func multipartVideoReferenceFileToDataURL(fileHeader *multipart.FileHeader) (string, error) {
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
		return "", fmt.Errorf("input_reference too large: max: %d", image.MaxImageSize)
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	if !image.IsImageURL(contentType) {
		return "", errors.New("input_reference file is not an image")
	}

	contentType = image.TrimImageContentType(contentType)

	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func buildAliVideoInput(meta *meta.Meta, request *aliVideoOpenAIRequest) map[string]any {
	input := cloneMap(request.Input)
	mergeAliVideoMetadataInput(input, request.Metadata)
	mergeAliVideoMetadataInput(input, request.Ext)

	if request.Prompt != "" {
		input["prompt"] = request.Prompt
	}

	if request.NegativePrompt != "" {
		input["negative_prompt"] = request.NegativePrompt
	}

	if request.AudioURL != "" {
		input["audio_url"] = request.AudioURL
	}

	inputReference := strings.TrimSpace(request.InputReference)

	referenceURLs := append([]string(nil), request.ReferenceURLs...)
	if len(referenceURLs) == 0 {
		referenceURLs = stringSliceFromAny(
			firstMapValue("reference_urls", request.Metadata, request.Ext),
		)
	}

	media := append([]aliVideoMedia(nil), request.Media...)
	if len(media) == 0 {
		media = mediaFromAny(firstMapValue("media", request.Metadata, request.Ext))
	}

	if len(media) == 0 {
		media = buildAliVideoMedia(meta, request, inputReference, referenceURLs)
	}

	if len(media) > 0 && videoModelUsesMedia(meta) {
		input["media"] = media
	} else {
		if url := firstNonEmpty(
			request.ImgURL,
			request.ImageURL,
			inputReference,
			request.FirstFrameURL,
		); url != "" {
			input["img_url"] = url
		}

		if request.FirstFrameURL != "" && request.FirstFrameURL != input["img_url"] {
			input["first_frame_url"] = request.FirstFrameURL
		}

		if request.LastFrameURL != "" {
			input["last_frame_url"] = request.LastFrameURL
		}

		if len(referenceURLs) > 0 {
			input["reference_urls"] = referenceURLs
		}
	}

	return input
}

func buildAliVideoMedia(
	meta *meta.Meta,
	request *aliVideoOpenAIRequest,
	inputReference string,
	referenceURLs []string,
) []aliVideoMedia {
	media := make([]aliVideoMedia, 0, 4+len(referenceURLs)+len(request.ReferenceImageURLs))

	if url := firstNonEmpty(
		inputReference,
		request.ImgURL,
		request.ImageURL,
		request.FirstFrameURL,
	); url != "" {
		media = append(media, aliVideoMedia{Type: "first_frame", URL: url})
	}

	if request.LastFrameURL != "" {
		media = append(media, aliVideoMedia{Type: "last_frame", URL: request.LastFrameURL})
	}

	if request.AudioURL != "" && videoModelUsesMedia(meta) {
		media = append(media, aliVideoMedia{Type: "driving_audio", URL: request.AudioURL})
	}

	if request.FirstClipURL != "" {
		media = append(media, aliVideoMedia{Type: "first_clip", URL: request.FirstClipURL})
	} else if request.VideoURL != "" {
		mediaType := "first_clip"
		if isAliVideoEditModel(meta) || isHappyHorseVideoEditModel(meta) {
			mediaType = "video"
		}

		media = append(media, aliVideoMedia{Type: mediaType, URL: request.VideoURL})
	}

	for _, url := range request.ReferenceImageURLs {
		if url != "" {
			media = append(media, aliVideoMedia{Type: "reference_image", URL: url})
		}
	}

	if isHappyHorseReferenceVideoModel(meta) {
		for _, url := range referenceURLs {
			if url != "" {
				media = append(media, aliVideoMedia{Type: "reference_image", URL: url})
			}
		}
	}

	return media
}

func buildAliVideoParameters(
	meta *meta.Meta,
	request *aliVideoOpenAIRequest,
) map[string]any {
	parameters := cloneMap(request.Parameters)
	mergeAliVideoMetadataParameters(parameters, request.Metadata)
	mergeAliVideoMetadataParameters(parameters, request.Ext)

	if seconds := aliVideoRequestSeconds(request); seconds > 0 {
		parameters["duration"] = seconds
	} else if request.NSeconds > 0 {
		parameters["duration"] = request.NSeconds
	}

	if request.Size != "" {
		setAliVideoSize(meta, parameters, request.Size)
	} else if size := sizeFromOpenAIRequest(request); size != "" {
		setAliVideoSize(meta, parameters, size)
	} else if resolution := resolutionFromOpenAIRequest(request); resolution != "" {
		parameters["resolution"] = resolution
	}

	if request.Ratio != "" {
		parameters["ratio"] = request.Ratio
	}

	if request.PromptExtend != nil {
		parameters["prompt_extend"] = *request.PromptExtend
	}

	if request.Watermark != nil {
		parameters["watermark"] = *request.Watermark
	}

	if request.Seed != nil {
		parameters["seed"] = *request.Seed
	}

	if request.ShotType != "" {
		parameters["shot_type"] = request.ShotType
	}

	if request.Audio != nil {
		parameters["audio"] = *request.Audio
	}

	if request.Template != "" {
		parameters["template"] = request.Template
	}

	return parameters
}

func aliVideoRequestSeconds(request *aliVideoOpenAIRequest) int {
	switch value := request.Seconds.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		seconds, _ := strconv.Atoi(value.String())
		return seconds
	case string:
		seconds, _ := strconv.Atoi(value)
		return seconds
	default:
		return 0
	}
}

func aliVideoRequestDimensions(request *aliVideoOpenAIRequest) (int, int) {
	if request.Width > 0 && request.Height > 0 {
		return request.Width, request.Height
	}

	return dimensionsFromAliVideoSize(request.Size)
}

func dimensionsFromAliVideoSize(size string) (int, int) {
	size = strings.TrimSpace(strings.ToLower(size))
	size = strings.ReplaceAll(size, "*", "x")

	if !strings.Contains(size, "x") {
		return 0, 0
	}

	widthText, heightText, ok := strings.Cut(size, "x")
	if !ok {
		return 0, 0
	}

	width, widthErr := strconv.Atoi(widthText)
	height, heightErr := strconv.Atoi(heightText)

	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return 0, 0
	}

	return width, height
}

func setAliVideoSize(meta *meta.Meta, parameters map[string]any, size string) {
	size = strings.ReplaceAll(size, "x", "*")
	if videoModelUsesResolution(meta) && strings.HasSuffix(strings.ToUpper(size), "P") {
		parameters["resolution"] = strings.ToUpper(size)
		return
	}

	parameters["size"] = size
}

func resolutionFromOpenAIRequest(request *aliVideoOpenAIRequest) string {
	switch {
	case request.Height >= 1000:
		return "1080P"
	case request.Height >= 700:
		return "720P"
	case request.Height >= 400:
		return "480P"
	default:
		return ""
	}
}

func sizeFromOpenAIRequest(request *aliVideoOpenAIRequest) string {
	if request.Width <= 0 || request.Height <= 0 {
		return ""
	}

	return fmt.Sprintf("%d*%d", request.Width, request.Height)
}

func mergeAliVideoMetadataInput(input, metadata map[string]any) {
	if len(metadata) == 0 {
		return
	}

	if rawInput, ok := metadata["input"].(map[string]any); ok {
		maps.Copy(input, rawInput)
	}
}

func mergeAliVideoMetadataParameters(parameters, metadata map[string]any) {
	if len(metadata) == 0 {
		return
	}

	if rawParameters, ok := metadata["parameters"].(map[string]any); ok {
		maps.Copy(parameters, rawParameters)
	}

	for key, value := range metadata {
		if key == "input" || key == "parameters" || isAliVideoMetadataInputKey(key) {
			continue
		}

		parameters[key] = value
	}
}

func isAliVideoMetadataInputKey(key string) bool {
	switch key {
	case "media",
		"reference_urls",
		"img_url",
		"image_url",
		"first_frame_url",
		"last_frame_url",
		"first_clip_url",
		"audio_url",
		"video_url",
		"negative_prompt":
		return true
	default:
		return false
	}
}

func cloneMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	maps.Copy(output, input)

	return output
}

func firstMapValue(key string, maps ...map[string]any) any {
	for _, values := range maps {
		if values == nil {
			continue
		}

		if value, ok := values[key]; ok {
			return value
		}
	}

	return nil
}

func stringSliceFromAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if itemString, ok := item.(string); ok && itemString != "" {
				values = append(values, itemString)
			}
		}

		return values
	default:
		return nil
	}
}

func mediaFromAny(value any) []aliVideoMedia {
	items, ok := value.([]any)
	if !ok {
		return nil
	}

	media := make([]aliVideoMedia, 0, len(items))
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		mediaType, _ := itemMap["type"].(string)

		mediaURL, _ := itemMap["url"].(string)
		if mediaType != "" && mediaURL != "" {
			media = append(media, aliVideoMedia{Type: mediaType, URL: mediaURL})
		}
	}

	return media
}

func videoModelUsesMedia(meta *meta.Meta) bool {
	return isHappyHorseVideoModel(meta) || isWan27VideoModel(meta)
}

func videoModelUsesResolution(meta *meta.Meta) bool {
	return isHappyHorseVideoModel(meta) || isWan27VideoModel(meta)
}

func isHappyHorseVideoModel(meta *meta.Meta) bool {
	return isHappyHorseVideoModelName(meta.OriginModel) ||
		isHappyHorseVideoModelName(meta.ActualModel)
}

func isHappyHorseVideoModelName(modelName string) bool {
	return strings.HasPrefix(strings.ToLower(modelName), "happyhorse-")
}

func isHappyHorseReferenceVideoModel(meta *meta.Meta) bool {
	return strings.EqualFold(meta.OriginModel, "happyhorse-1.0-r2v") ||
		strings.EqualFold(meta.ActualModel, "happyhorse-1.0-r2v")
}

func isHappyHorseVideoEditModel(meta *meta.Meta) bool {
	return strings.EqualFold(meta.OriginModel, "happyhorse-1.0-video-edit") ||
		strings.EqualFold(meta.ActualModel, "happyhorse-1.0-video-edit")
}

func isWan27VideoModel(meta *meta.Meta) bool {
	return strings.HasPrefix(strings.ToLower(meta.OriginModel), "wan2.7-") ||
		strings.HasPrefix(strings.ToLower(meta.ActualModel), "wan2.7-")
}

func isAliVideoEditModel(meta *meta.Meta) bool {
	return strings.Contains(strings.ToLower(meta.OriginModel), "videoedit") ||
		strings.Contains(strings.ToLower(meta.ActualModel), "videoedit")
}

func AliVideoHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return adaptor.DoResponseResult{}, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	var aliResponse AliVideoTaskResponse
	if err := common.UnmarshalResponse(resp, &aliResponse); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	if aliResponse.Code != "" || aliResponse.Message != "" {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoErrorWithMessage(
			firstNonEmpty(aliResponse.Message, aliResponse.Code),
			http.StatusInternalServerError,
		)
	}

	job := aliVideoTaskToOpenAIJob(meta, &aliResponse)
	if err := saveAliVideoJobStore(meta, store, job.ID); err != nil {
		common.GetLogger(c).Errorf("save video job store failed: %v", err)
	}

	jsonResponse, err := sonic.Marshal(job)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(jsonResponse)))
	_, _ = c.Writer.Write(jsonResponse)

	return adaptor.DoResponseResult{
		UpstreamID: job.ID,
		AsyncUsage: true,
	}, nil
}

func AliVideoGetJobsHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	var aliResponse AliVideoTaskResponse
	if err := common.UnmarshalResponse(resp, &aliResponse); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	job := aliVideoTaskToOpenAIJob(meta, &aliResponse)
	if job.Status == relaymodel.VideoGenerationJobStatusSucceeded {
		for _, generation := range job.Generations {
			if err := saveAliVideoGenerationStore(
				meta,
				store,
				generation.ID,
				job.ExpiresAt,
			); err != nil {
				common.GetLogger(c).Errorf("save video generation store failed: %v", err)
			}
		}
	}

	jsonResponse, err := sonic.Marshal(job)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(jsonResponse)))
	_, _ = c.Writer.Write(jsonResponse)

	return adaptor.DoResponseResult{UpstreamID: job.ID}, nil
}

func AliVideoContentHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	var aliResponse AliVideoTaskResponse
	if err := common.UnmarshalResponse(resp, &aliResponse); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	videoURL := firstNonEmpty(aliResponse.Output.VideoURL, aliResponse.Output.OutputVideoURL)
	if videoURL == "" {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoErrorWithMessage(
			"video url is empty",
			http.StatusInternalServerError,
		)
	}

	videoResp, err := fetchAliVideoContent(c.Request.Context(), meta, videoURL)
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

	return adaptor.DoResponseResult{UpstreamID: aliResponse.Output.TaskID}, nil
}

func fetchAliVideoContent(
	ctx context.Context,
	meta *meta.Meta,
	videoURL string,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, videoURL, nil)
	if err != nil {
		return nil, err
	}

	var (
		proxyURL      string
		skipTLSVerify bool
	)
	if meta != nil {
		proxyURL = meta.Channel.ProxyURL
		skipTLSVerify = meta.Channel.SkipTLSVerify
	}

	client, err := relayutils.LoadHTTPClientWithTLSConfigE(
		0,
		proxyURL,
		skipTLSVerify,
	)
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}

func aliVideoTaskToOpenAIJob(
	meta *meta.Meta,
	response *AliVideoTaskResponse,
) *relaymodel.VideoGenerationJob {
	now := time.Now()
	createdAt := parseAliVideoTime(response.Output.SubmitTime, now).Unix()
	finishedAt := parseAliVideoOptionalTime(response.Output.EndTime)

	expiresAtTime := now.Add(aliVideoTaskTTL)
	if finishedAt != nil {
		expiresAtTime = time.Unix(*finishedAt, 0).Add(aliVideoTaskTTL)
	}

	expiresAt := expiresAtTime.Unix()
	taskID := response.Output.TaskID
	status := aliVideoStatusToOpenAI(response.Output.TaskStatus)
	nSeconds := int(aliVideoOutputSeconds(response.Usage))

	nVariants := int(response.Usage.VideoCount)
	if nVariants == 0 {
		nVariants = 1
	}

	width, height := aliVideoDimensions(response.Usage)

	prompt := firstNonEmpty(response.Output.OrigPrompt, meta.GetString(metaAliVideoPrompt))
	if nSeconds == 0 {
		nSeconds = meta.GetInt(metaAliVideoSeconds)
	}

	if width == 0 || height == 0 {
		width = meta.GetInt(metaAliVideoWidth)
		height = meta.GetInt(metaAliVideoHeight)
	}

	job := &relaymodel.VideoGenerationJob{
		Object:      relaymodel.VideoGenerationJobObject,
		ID:          taskID,
		Status:      status,
		CreatedAt:   createdAt,
		FinishedAt:  finishedAt,
		ExpiresAt:   &expiresAt,
		Generations: []relaymodel.VideoGenerations{},
		Prompt:      prompt,
		Model:       meta.OriginModel,
		NVariants:   nVariants,
		NSeconds:    nSeconds,
		Width:       width,
		Height:      height,
	}

	if status == relaymodel.VideoGenerationJobStatusSucceeded &&
		firstNonEmpty(response.Output.VideoURL, response.Output.OutputVideoURL) != "" {
		job.Generations = append(job.Generations, relaymodel.VideoGenerations{
			Object:    relaymodel.VideoGenerationObject,
			ID:        taskID,
			JobID:     taskID,
			CreatedAt: createdAt,
			Width:     width,
			Height:    height,
			Prompt:    prompt,
			NSeconds:  nSeconds,
		})
	}

	if response.Output.Message != "" {
		finishReason := response.Output.Message
		job.FinishReason = &finishReason
	}

	return job
}

func AliVideosHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return adaptor.DoResponseResult{}, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	var aliResponse AliVideoTaskResponse
	if err := common.UnmarshalResponse(resp, &aliResponse); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	if aliResponse.Code != "" || aliResponse.Message != "" {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoErrorWithMessage(
			firstNonEmpty(aliResponse.Message, aliResponse.Code),
			http.StatusInternalServerError,
		)
	}

	video := aliVideoTaskToOpenAIVideo(meta, &aliResponse)
	if err := saveAliVideoGenerationStore(meta, store, video.ID, nil); err != nil {
		common.GetLogger(c).Errorf("save video store failed: %v", err)
	}

	jsonResponse, err := sonic.Marshal(video)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(jsonResponse)))
	_, _ = c.Writer.Write(jsonResponse)

	return adaptor.DoResponseResult{
		UpstreamID: video.ID,
		AsyncUsage: true,
	}, nil
}

func AliVideoGetHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	var aliResponse AliVideoTaskResponse
	if err := common.UnmarshalResponse(resp, &aliResponse); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	video := aliVideoTaskToOpenAIVideo(meta, &aliResponse)
	if video.Status == relaymodel.VideoStatusCompleted {
		if err := saveAliVideoGenerationStore(meta, store, video.ID, nil); err != nil {
			common.GetLogger(c).Errorf("save video store failed: %v", err)
		}
	}

	jsonResponse, err := sonic.Marshal(video)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(jsonResponse)))
	_, _ = c.Writer.Write(jsonResponse)

	return adaptor.DoResponseResult{UpstreamID: video.ID}, nil
}

func aliVideoStatusToOpenAI(status string) relaymodel.VideoGenerationJobStatus {
	switch strings.ToUpper(status) {
	case "PENDING":
		return relaymodel.VideoGenerationJobStatusQueued
	case "RUNNING":
		return relaymodel.VideoGenerationJobStatusRunning
	case "SUCCEEDED":
		return relaymodel.VideoGenerationJobStatusSucceeded
	case "FAILED":
		return "failed"
	case "CANCELED", "CANCELLED":
		return "cancelled"
	case "UNKNOWN":
		return "unknown"
	default:
		if status == "" {
			return relaymodel.VideoGenerationJobStatusQueued
		}

		return strings.ToLower(status)
	}
}

func aliVideoStatusToOpenAIVideo(status string) relaymodel.VideoStatus {
	switch strings.ToUpper(status) {
	case "PENDING":
		return relaymodel.VideoStatusQueued
	case "RUNNING":
		return relaymodel.VideoStatusInProgress
	case "SUCCEEDED":
		return relaymodel.VideoStatusCompleted
	case "FAILED":
		return relaymodel.VideoStatusFailed
	case "CANCELED", "CANCELLED":
		return relaymodel.VideoStatusCancelled
	default:
		if status == "" {
			return relaymodel.VideoStatusQueued
		}

		return strings.ToLower(status)
	}
}

func aliVideoTaskToOpenAIVideo(
	meta *meta.Meta,
	response *AliVideoTaskResponse,
) *relaymodel.Video {
	now := time.Now()
	createdAt := parseAliVideoTime(response.Output.SubmitTime, now).Unix()
	width, height := aliVideoDimensions(response.Usage)

	seconds := int(aliVideoOutputSeconds(response.Usage))
	if seconds == 0 {
		seconds = meta.GetInt(metaAliVideoSeconds)
	}

	size := aliVideoSize(width, height)
	if size == "" {
		size = meta.GetString(metaAliVideoSize)
	}

	video := &relaymodel.Video{
		ID:        response.Output.TaskID,
		Object:    relaymodel.VideoObject,
		CreatedAt: createdAt,
		Status:    aliVideoStatusToOpenAIVideo(response.Output.TaskStatus),
		Model:     meta.OriginModel,
		Prompt:    firstNonEmpty(response.Output.OrigPrompt, meta.GetString(metaAliVideoPrompt)),
		Seconds:   seconds,
		Size:      size,
	}

	switch video.Status {
	case relaymodel.VideoStatusCompleted:
		video.Progress = 100
	case relaymodel.VideoStatusQueued:
		video.Progress = 0
	case relaymodel.VideoStatusInProgress:
		video.Progress = 50
	}

	message := firstNonEmpty(response.Output.Message, response.Message)
	if message != "" && video.Status == relaymodel.VideoStatusFailed {
		video.Error = map[string]any{"message": message}
	}

	return video
}

func aliVideoSize(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	return fmt.Sprintf("%dx%d", width, height)
}

func aliVideoOutputSeconds(usage AliVideoUsage) int64 {
	if usage.OutputVideoDuration > 0 {
		return usage.OutputVideoDuration
	}

	if usage.VideoDuration > 0 {
		return usage.VideoDuration
	}

	if usage.Duration > usage.InputVideoDuration {
		return usage.Duration - usage.InputVideoDuration
	}

	return usage.Duration
}

func aliVideoUsageToModelUsage(usage AliVideoUsage) coremodel.Usage {
	input := usage.InputVideoDuration
	output := aliVideoOutputSeconds(usage)

	total := usage.Duration
	if total == 0 {
		total = input + output
	}

	return coremodel.Usage{
		VideoInputTokens: coremodel.ZeroNullInt64(input),
		OutputTokens:     coremodel.ZeroNullInt64(output),
		TotalTokens:      coremodel.ZeroNullInt64(total),
	}
}

func aliVideoDimensions(usage AliVideoUsage) (int, int) {
	resolution := intFromAny(usage.SR)
	if resolution == 0 {
		return 0, 0
	}

	switch usage.Ratio {
	case "9:16":
		return resolution * 9 / 16, resolution
	case "1:1":
		return resolution, resolution
	case "4:3":
		return resolution * 4 / 3, resolution
	case "3:4":
		return resolution * 3 / 4, resolution
	default:
		return resolution * 16 / 9, resolution
	}
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		value, _ := strconv.Atoi(typed)
		return value
	default:
		return 0
	}
}

func parseAliVideoTime(value string, fallback time.Time) time.Time {
	if parsed := parseAliVideoOptionalTime(value); parsed != nil {
		return time.Unix(*parsed, 0)
	}

	return fallback
}

func parseAliVideoOptionalTime(value string) *int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	for _, layout := range []string{
		"2006-01-02 15:04:05.000",
		time.DateTime,
		time.RFC3339Nano,
		time.RFC3339,
	} {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			unix := parsed.Unix()
			return &unix
		}
	}

	return nil
}

func saveAliVideoJobStore(meta *meta.Meta, store adaptor.Store, jobID string) error {
	if store == nil || jobID == "" {
		return nil
	}

	return store.SaveStore(adaptor.StoreCache{
		ID:        coremodel.VideoJobStoreID(jobID),
		GroupID:   meta.Group.ID,
		TokenID:   meta.Token.ID,
		ChannelID: meta.Channel.ID,
		Model:     meta.OriginModel,
		ExpiresAt: time.Now().Add(aliVideoTaskTTL),
	})
}

func saveAliVideoGenerationStore(
	meta *meta.Meta,
	store adaptor.Store,
	generationID string,
	expiresAt *int64,
) error {
	if store == nil || generationID == "" {
		return nil
	}

	expiresAtTime := time.Now().Add(aliVideoTaskTTL)
	if expiresAt != nil {
		expiresAtTime = time.Unix(*expiresAt, 0)
	}

	return store.SaveStore(adaptor.StoreCache{
		ID:        coremodel.VideoGenerationStoreID(generationID),
		GroupID:   meta.Group.ID,
		TokenID:   meta.Token.ID,
		ChannelID: meta.Channel.ID,
		Model:     meta.OriginModel,
		ExpiresAt: expiresAtTime,
	})
}

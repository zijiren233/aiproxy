package doubao

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
	coremodel "github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	relayutils "github.com/labring/aiproxy/core/relay/utils"
)

const (
	metaDoubaoVideoRequest = "doubao_video_request"
	doubaoVideoTTL         = 7 * 24 * time.Hour
)

type doubaoVideoRequest struct {
	Model                 string               `json:"model,omitempty"`
	Content               []doubaoVideoContent `json:"content,omitempty"`
	CallbackURL           string               `json:"callback_url,omitempty"`
	ReturnLastFrame       *bool                `json:"return_last_frame,omitempty"`
	ServiceTier           string               `json:"service_tier,omitempty"`
	ExecutionExpiresAfter *int                 `json:"execution_expires_after,omitempty"`
	GenerateAudio         *bool                `json:"generate_audio,omitempty"`
	Draft                 *bool                `json:"draft,omitempty"`
	Tools                 []map[string]any     `json:"tools,omitempty"`
	SafetyIdentifier      string               `json:"safety_identifier,omitempty"`
	Priority              *int                 `json:"priority,omitempty"`
	Resolution            string               `json:"resolution,omitempty"`
	Ratio                 string               `json:"ratio,omitempty"`
	Duration              *int                 `json:"duration,omitempty"`
	Frames                *int                 `json:"frames,omitempty"`
	FramesPerSecond       *int                 `json:"framespersecond,omitempty"`
	Seed                  any                  `json:"seed,omitempty"`
	CameraFixed           *bool                `json:"camera_fixed,omitempty"`
	Watermark             *bool                `json:"watermark,omitempty"`
}

type doubaoVideoContent struct {
	Type      string                 `json:"type,omitempty"`
	Text      string                 `json:"text,omitempty"`
	ImageURL  *doubaoVideoURLContent `json:"image_url,omitempty"`
	VideoURL  *doubaoVideoURLContent `json:"video_url,omitempty"`
	AudioURL  *doubaoVideoURLContent `json:"audio_url,omitempty"`
	DraftTask *doubaoDraftTask       `json:"draft_task,omitempty"`
	Role      string                 `json:"role,omitempty"`
}

type doubaoVideoURLContent struct {
	URL string `json:"url,omitempty"`
}

type doubaoDraftTask struct {
	ID string `json:"id,omitempty"`
}

type doubaoVideoTaskResponse struct {
	ID                    string                  `json:"id,omitempty"`
	Model                 string                  `json:"model,omitempty"`
	Status                string                  `json:"status,omitempty"`
	Error                 *relaymodel.OpenAIError `json:"error,omitempty"`
	Content               doubaoVideoOutput       `json:"content,omitempty"`
	Usage                 doubaoVideoUsage        `json:"usage,omitempty"`
	Seed                  int64                   `json:"seed,omitempty"`
	Resolution            string                  `json:"resolution,omitempty"`
	Ratio                 string                  `json:"ratio,omitempty"`
	Duration              int                     `json:"duration,omitempty"`
	Frames                int                     `json:"frames,omitempty"`
	FramesPerSecond       int                     `json:"framespersecond,omitempty"`
	CreatedAt             int64                   `json:"created_at,omitempty"`
	UpdatedAt             int64                   `json:"updated_at,omitempty"`
	ServiceTier           string                  `json:"service_tier,omitempty"`
	ExecutionExpiresAfter int64                   `json:"execution_expires_after,omitempty"`
	GenerateAudio         *bool                   `json:"generate_audio,omitempty"`
	Draft                 *bool                   `json:"draft,omitempty"`
	DraftTaskID           string                  `json:"draft_task_id,omitempty"`
}

type doubaoVideoOutput struct {
	VideoURL     string `json:"video_url,omitempty"`
	LastFrameURL string `json:"last_frame_url,omitempty"`
	FileURL      string `json:"file_url,omitempty"`
}

type doubaoVideoUsage struct {
	CompletionTokens int64                `json:"completion_tokens,omitempty"`
	TotalTokens      int64                `json:"total_tokens,omitempty"`
	ToolUsage        doubaoVideoToolUsage `json:"tool_usage,omitempty"`
}

type doubaoVideoToolUsage struct {
	WebSearch int64 `json:"web_search,omitempty"`
}

func ConvertVideoRequest(meta *meta.Meta, req *http.Request) (adaptor.ConvertResult, error) {
	request, err := parseDoubaoVideoRequest(req)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	if len(request.Content) == 0 {
		return adaptor.ConvertResult{}, errors.New("content is required")
	}

	request.Model = meta.ActualModel
	meta.Set(metaDoubaoVideoRequest, request)

	data, err := sonic.Marshal(&request)
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

func parseDoubaoVideoRequest(req *http.Request) (doubaoVideoRequest, error) {
	if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
		return parseDoubaoMultipartVideoRequest(req)
	}

	var raw map[string]any
	if err := common.UnmarshalRequestReusable(req, &raw); err != nil {
		return doubaoVideoRequest{}, err
	}

	return parseDoubaoJSONVideoRequest(raw), nil
}

func parseDoubaoJSONVideoRequest(raw map[string]any) doubaoVideoRequest {
	request := doubaoVideoRequest{
		Content:          doubaoVideoContentFromAny(raw["content"]),
		CallbackURL:      stringFromAny(raw["callback_url"]),
		ServiceTier:      stringFromAny(raw["service_tier"]),
		SafetyIdentifier: stringFromAny(raw["safety_identifier"]),
		Resolution: firstNonEmptyString(
			stringFromAny(raw["resolution"]),
			doubaoVideoResolutionFromSize(stringFromAny(raw["size"])),
		),
		Ratio: firstNonEmptyString(
			stringFromAny(raw["ratio"]),
			ratioFromSize(stringFromAny(raw["size"])),
		),
		Seed:                  raw["seed"],
		ExecutionExpiresAfter: intPtrFromAny(raw["execution_expires_after"]),
		GenerateAudio:         boolPtrFromAny(raw["generate_audio"]),
		Draft:                 boolPtrFromAny(raw["draft"]),
		Priority:              intPtrFromAny(raw["priority"]),
		Duration:              intPtrFromAny(firstPresent(raw, "seconds", "n_seconds")),
		Frames:                intPtrFromAny(raw["frames"]),
		FramesPerSecond:       intPtrFromAny(firstPresent(raw, "framespersecond", "fps")),
		CameraFixed:           boolPtrFromAny(raw["camera_fixed"]),
		Watermark:             boolPtrFromAny(raw["watermark"]),
	}

	if request.Content == nil {
		request.Content = doubaoVideoContentFromOpenAI(raw)
	}

	if tools, ok := raw["tools"].([]any); ok {
		request.Tools = make([]map[string]any, 0, len(tools))
		for _, item := range tools {
			if tool, ok := item.(map[string]any); ok {
				request.Tools = append(request.Tools, tool)
			}
		}
	}

	return request
}

func parseDoubaoMultipartVideoRequest(req *http.Request) (doubaoVideoRequest, error) {
	if err := common.ParseMultipartFormWithLimit(req); err != nil {
		return doubaoVideoRequest{}, fmt.Errorf("parse multipart form: %w", err)
	}

	request := doubaoVideoRequest{
		CallbackURL:      req.PostFormValue("callback_url"),
		ServiceTier:      req.PostFormValue("service_tier"),
		SafetyIdentifier: req.PostFormValue("safety_identifier"),
		Resolution: firstNonEmptyString(
			req.PostFormValue("resolution"),
			doubaoVideoResolutionFromSize(req.PostFormValue("size")),
		),
		Ratio: firstNonEmptyString(
			req.PostFormValue("ratio"),
			ratioFromSize(req.PostFormValue("size")),
		),
		Content: []doubaoVideoContent{},
	}

	if prompt := req.PostFormValue("prompt"); prompt != "" {
		request.Content = append(request.Content, doubaoVideoContent{Type: "text", Text: prompt})
	}

	setOptionalInt(&request.Duration, req.PostFormValue("seconds"), req.PostFormValue("n_seconds"))
	setOptionalInt(&request.Frames, req.PostFormValue("frames"))
	setOptionalInt(
		&request.FramesPerSecond,
		req.PostFormValue("framespersecond"),
		req.PostFormValue("fps"),
	)
	setOptionalInt(&request.ExecutionExpiresAfter, req.PostFormValue("execution_expires_after"))
	setOptionalInt(&request.Priority, req.PostFormValue("priority"))

	request.GenerateAudio = boolPtrFromString(req.PostFormValue("generate_audio"))
	request.Draft = boolPtrFromString(req.PostFormValue("draft"))
	request.CameraFixed = boolPtrFromString(req.PostFormValue("camera_fixed"))
	request.Watermark = boolPtrFromString(req.PostFormValue("watermark"))

	if seed := strings.TrimSpace(req.PostFormValue("seed")); seed != "" {
		request.Seed = seed
	}

	addFormURLContents(&request.Content, req.MultipartForm.Value)

	if err := addMultipartFileContents(&request.Content, req.MultipartForm.File); err != nil {
		return doubaoVideoRequest{}, err
	}

	return request, nil
}

func doubaoVideoContentFromAny(value any) []doubaoVideoContent {
	items, ok := value.([]any)
	if !ok {
		return nil
	}

	content := make([]doubaoVideoContent, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}

		content = append(content, doubaoVideoContentFromMap(m))
	}

	return content
}

func doubaoVideoContentFromMap(m map[string]any) doubaoVideoContent {
	item := doubaoVideoContent{
		Type: strings.TrimSpace(stringFromAny(m["type"])),
		Text: stringFromAny(m["text"]),
		Role: stringFromAny(m["role"]),
	}

	if item.Type == "" && item.Text != "" {
		item.Type = "text"
	}

	switch item.Type {
	case "image_url":
		item.ImageURL = &doubaoVideoURLContent{URL: nestedURL(m["image_url"])}
	case "video_url":
		item.VideoURL = &doubaoVideoURLContent{URL: nestedURL(m["video_url"])}
	case "audio_url":
		item.AudioURL = &doubaoVideoURLContent{URL: nestedURL(m["audio_url"])}
	case "input_audio":
		item.Type = "audio_url"

		item.AudioURL = openAIAudioToDoubaoURL(m["input_audio"])
		if item.Role == "" {
			item.Role = "reference_audio"
		}
	case "draft_task":
		item.DraftTask = &doubaoDraftTask{ID: nestedID(m["draft_task"])}
	}

	return item
}

func doubaoVideoContentFromOpenAI(raw map[string]any) []doubaoVideoContent {
	content := []doubaoVideoContent{}

	if prompt := stringFromAny(raw["prompt"]); prompt != "" {
		content = append(content, doubaoVideoContent{Type: "text", Text: prompt})
	}

	addStringContent := func(contentType string, value any, role string) {
		urlValue := stringFromAny(value)
		if urlValue == "" {
			return
		}

		item := doubaoVideoContent{Type: contentType, Role: role}
		switch contentType {
		case "image_url":
			item.ImageURL = &doubaoVideoURLContent{URL: urlValue}
		case "video_url":
			item.VideoURL = &doubaoVideoURLContent{URL: urlValue}
		case "audio_url":
			item.AudioURL = &doubaoVideoURLContent{URL: urlValue}
		}

		content = append(content, item)
	}

	addStringContent("image_url", firstPresent(raw, "input_reference", "image", "image_url"), "")
	addStringContent("image_url", raw["first_frame_url"], "first_frame")
	addStringContent("image_url", raw["last_frame_url"], "last_frame")
	addStringContent("video_url", raw["video_url"], "reference_video")
	addStringContent("audio_url", raw["audio_url"], "reference_audio")

	if inputAudio, ok := raw["input_audio"].(map[string]any); ok {
		content = append(content, doubaoVideoContent{
			Type:     "audio_url",
			AudioURL: openAIAudioToDoubaoURL(inputAudio),
			Role:     "reference_audio",
		})
	}

	if draftTaskID := stringFromAny(
		firstPresent(raw, "draft_task_id", "video_id"),
	); draftTaskID != "" {
		content = append(content, doubaoVideoContent{
			Type:      "draft_task",
			DraftTask: &doubaoDraftTask{ID: draftTaskID},
		})
	}

	return content
}

func addFormURLContents(content *[]doubaoVideoContent, values map[string][]string) {
	add := func(contentType, key, role string) {
		for _, value := range values[key] {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}

			item := doubaoVideoContent{Type: contentType, Role: role}
			switch contentType {
			case "image_url":
				item.ImageURL = &doubaoVideoURLContent{URL: value}
			case "video_url":
				item.VideoURL = &doubaoVideoURLContent{URL: value}
			case "audio_url":
				item.AudioURL = &doubaoVideoURLContent{URL: value}
			}

			*content = append(*content, item)
		}
	}

	add("image_url", "input_reference", "")
	add("image_url", "image", "")
	add("image_url", "first_frame_url", "first_frame")
	add("image_url", "last_frame_url", "last_frame")
	add("video_url", "video_url", "reference_video")
	add("audio_url", "audio_url", "reference_audio")
}

func addMultipartFileContents(
	content *[]doubaoVideoContent,
	files map[string][]*multipart.FileHeader,
) error {
	for _, key := range []string{"input_reference", "image"} {
		for _, fileHeader := range files[key] {
			dataURL, err := multipartMediaDataURL(fileHeader, "image")
			if err != nil {
				return err
			}

			*content = append(*content, doubaoVideoContent{
				Type:     "image_url",
				ImageURL: &doubaoVideoURLContent{URL: dataURL},
			})
		}
	}

	for _, fileHeader := range files["video"] {
		dataURL, err := multipartMediaDataURL(fileHeader, "video")
		if err != nil {
			return err
		}

		*content = append(*content, doubaoVideoContent{
			Type:     "video_url",
			VideoURL: &doubaoVideoURLContent{URL: dataURL},
			Role:     "reference_video",
		})
	}

	for _, fileHeader := range files["audio"] {
		dataURL, err := multipartMediaDataURL(fileHeader, "audio")
		if err != nil {
			return err
		}

		*content = append(*content, doubaoVideoContent{
			Type:     "audio_url",
			AudioURL: &doubaoVideoURLContent{URL: dataURL},
			Role:     "reference_audio",
		})
	}

	return nil
}

func multipartMediaDataURL(fileHeader *multipart.FileHeader, mediaType string) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := io.ReadAll(common.LimitReader(file, common.MaxRequestBodySize+1))
	if err != nil {
		return "", err
	}

	if len(data) > common.MaxRequestBodySize {
		return "", fmt.Errorf("%s too large: max: %d", mediaType, common.MaxRequestBodySize)
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	if ext := strings.ToLower(filepath.Ext(fileHeader.Filename)); ext != "" {
		if detected := mime.TypeByExtension(ext); detected != "" &&
			!strings.HasPrefix(contentType, mediaType+"/") {
			contentType = detected
		}
	}

	if !strings.HasPrefix(contentType, mediaType+"/") {
		return "", fmt.Errorf("%s file is not %s", mediaType, mediaType)
	}

	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func VideoSubmitHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return adaptor.DoResponseResult{}, openai.VideoErrorHanlder(resp)
	}

	defer resp.Body.Close()

	var response doubaoVideoTaskResponse
	if err := common.UnmarshalResponse(resp, &response); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	if response.ID == "" {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoErrorWithMessage(
			"missing id in doubao video response",
			http.StatusInternalServerError,
		)
	}

	if response.Status == "" {
		response.Status = "queued"
	}

	expiresAt := doubaoVideoExpiresAt(response)
	if meta.Mode == mode.Videos {
		if err := saveDoubaoVideoStore(meta, store, response.ID, expiresAt); err != nil {
			common.GetLogger(c).Errorf("save doubao video store failed: %v", err)
		}

		video := buildDoubaoVideo(meta, response.ID, &response)

		return writeDoubaoVideoObject(c, video, adaptor.DoResponseResult{
			UpstreamID: response.ID,
			AsyncUsage: true,
		})
	}

	if err := saveDoubaoVideoJobStore(meta, store, response.ID, expiresAt); err != nil {
		common.GetLogger(c).Errorf("save doubao video job store failed: %v", err)
	}

	job := buildDoubaoVideoJob(meta, response.ID, &response)
	if job.Status == relaymodel.VideoGenerationJobStatusSucceeded {
		for _, generation := range job.Generations {
			if err := saveDoubaoVideoStore(meta, store, generation.ID, expiresAt); err != nil {
				common.GetLogger(c).
					Errorf("save doubao video generation store failed: %v", err)
			}
		}
	}

	return writeDoubaoVideoObject(c, job, adaptor.DoResponseResult{
		UpstreamID: response.ID,
		AsyncUsage: true,
	})
}

func VideoStatusHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, openai.VideoErrorHanlder(resp)
	}

	defer resp.Body.Close()

	var response doubaoVideoTaskResponse
	if err := common.UnmarshalResponse(resp, &response); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	if response.ID == "" {
		response.ID = firstNonEmptyString(meta.JobID, meta.VideoID)
	}

	expiresAt := doubaoVideoExpiresAt(response)
	if meta.Mode == mode.VideosGet {
		if response.Content.VideoURL != "" || response.Content.FileURL != "" {
			if err := saveDoubaoVideoStore(meta, store, response.ID, expiresAt); err != nil {
				common.GetLogger(c).Errorf("save doubao video store failed: %v", err)
			}
		}

		return writeDoubaoVideoObject(
			c,
			buildDoubaoVideo(meta, response.ID, &response),
			adaptor.DoResponseResult{UpstreamID: response.ID},
		)
	}

	job := buildDoubaoVideoJob(meta, response.ID, &response)
	if job.Status == relaymodel.VideoGenerationJobStatusSucceeded {
		for _, generation := range job.Generations {
			if err := saveDoubaoVideoStore(meta, store, generation.ID, expiresAt); err != nil {
				common.GetLogger(c).Errorf("save doubao video generation store failed: %v", err)
			}
		}
	}

	return writeDoubaoVideoObject(c, job, adaptor.DoResponseResult{})
}

func VideoContentHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, openai.VideoErrorHanlder(resp)
	}

	defer resp.Body.Close()

	var response doubaoVideoTaskResponse
	if err := common.UnmarshalResponse(resp, &response); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	videoURL := firstNonEmptyString(response.Content.VideoURL, response.Content.FileURL)
	if videoURL == "" {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoErrorWithMessage(
			"video url is empty",
			http.StatusInternalServerError,
		)
	}

	videoResp, err := fetchDoubaoVideoContent(c.Request.Context(), meta, videoURL)
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
		Set("Content-Type", firstNonEmptyString(videoResp.Header.Get("Content-Type"), "video/mp4"))
	c.Writer.Header().Set("Content-Length", videoResp.Header.Get("Content-Length"))
	_, _ = io.Copy(c.Writer, videoResp.Body)

	return adaptor.DoResponseResult{UpstreamID: doubaoContentUpstreamID(meta)}, nil
}

func buildDoubaoVideoJob(
	meta *meta.Meta,
	id string,
	response *doubaoVideoTaskResponse,
) relaymodel.VideoGenerationJob {
	now := time.Now().Unix()
	createdAt := firstPositiveInt64(response.CreatedAt, now)
	expiresAt := doubaoVideoExpiresAt(*response).Unix()
	request := doubaoVideoRequestFromMeta(meta)
	status := doubaoVideoJobStatus(response.Status)

	job := relaymodel.VideoGenerationJob{
		Object:      relaymodel.VideoGenerationJobObject,
		ID:          id,
		Status:      status,
		CreatedAt:   createdAt,
		ExpiresAt:   &expiresAt,
		Generations: []relaymodel.VideoGenerations{},
		Prompt:      doubaoVideoPrompt(request),
		Model:       meta.OriginModel,
		NVariants:   1,
		NSeconds:    firstPositiveInt(response.Duration, intFromPtr(request.Duration)),
	}

	job.Width, job.Height = doubaoVideoDimensions(response.Resolution, response.Ratio)

	if status == relaymodel.VideoGenerationJobStatusSucceeded ||
		status == relaymodel.VideoGenerationJobStatus("failed") {
		finishedAt := firstPositiveInt64(response.UpdatedAt, now)
		job.FinishedAt = &finishedAt
	}

	if response.Error != nil && response.Error.Message != "" {
		reason := response.Error.Message
		job.FinishReason = &reason
	}

	if status == relaymodel.VideoGenerationJobStatusSucceeded &&
		firstNonEmptyString(response.Content.VideoURL, response.Content.FileURL) != "" {
		job.Generations = append(job.Generations, relaymodel.VideoGenerations{
			Object:    relaymodel.VideoGenerationObject,
			ID:        id,
			JobID:     id,
			CreatedAt: firstPositiveInt64(response.UpdatedAt, now),
			Width:     job.Width,
			Height:    job.Height,
			Prompt:    job.Prompt,
			NSeconds:  job.NSeconds,
		})
	}

	return job
}

func buildDoubaoVideo(
	meta *meta.Meta,
	id string,
	response *doubaoVideoTaskResponse,
) relaymodel.Video {
	now := time.Now().Unix()
	request := doubaoVideoRequestFromMeta(meta)
	video := relaymodel.Video{
		ID:        id,
		Object:    relaymodel.VideoObject,
		CreatedAt: firstPositiveInt64(response.CreatedAt, now),
		Status:    doubaoVideoStatus(response.Status),
		Model:     meta.OriginModel,
		Prompt:    doubaoVideoPrompt(request),
		Seconds:   firstPositiveInt(response.Duration, intFromPtr(request.Duration)),
		Size:      response.Resolution,
	}

	switch video.Status {
	case relaymodel.VideoStatusCompleted:
		video.Progress = 100
	case relaymodel.VideoStatusInProgress:
		video.Progress = 50
	case relaymodel.VideoStatusQueued:
		video.Progress = 0
	}

	if response.Error != nil && response.Error.Message != "" {
		video.Error = map[string]any{"message": response.Error.Message}
	}

	return video
}

func doubaoVideoUsageToModelUsage(usage doubaoVideoUsage) coremodel.Usage {
	output := usage.CompletionTokens
	if output == 0 {
		output = usage.TotalTokens
	}

	total := usage.TotalTokens
	if total == 0 {
		total = output
	}

	return coremodel.Usage{
		OutputTokens:   coremodel.ZeroNullInt64(output),
		TotalTokens:    coremodel.ZeroNullInt64(total),
		WebSearchCount: coremodel.ZeroNullInt64(usage.ToolUsage.WebSearch),
	}
}

func doubaoVideoUsageContext(response *doubaoVideoTaskResponse) coremodel.UsageContext {
	if response == nil {
		return coremodel.UsageContext{}
	}

	return coremodel.UsageContext{
		PriceCondition: coremodel.UsagePriceCondition{
			Size: response.Resolution,
		},
		ServiceTier: response.ServiceTier,
	}
}

func writeDoubaoVideoObject(
	c *gin.Context,
	value any,
	result adaptor.DoResponseResult,
) (adaptor.DoResponseResult, adaptor.Error) {
	data, err := sonic.Marshal(value)
	if err != nil {
		return result, relaymodel.WrapperOpenAIVideoError(err, http.StatusInternalServerError)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = c.Writer.Write(data)

	return result, nil
}

func fetchDoubaoVideoContent(
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

	client, err := relayutils.LoadHTTPClientWithTLSConfigE(0, proxyURL, skipTLSVerify)
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}

func saveDoubaoVideoJobStore(
	meta *meta.Meta,
	store adaptor.Store,
	jobID string,
	expiresAt time.Time,
) error {
	if store == nil || jobID == "" {
		return nil
	}

	return store.SaveStore(adaptor.StoreCache{
		ID:        coremodel.VideoJobStoreID(jobID),
		GroupID:   meta.Group.ID,
		TokenID:   meta.Token.ID,
		ChannelID: meta.Channel.ID,
		Model:     meta.OriginModel,
		ExpiresAt: expiresAt,
	})
}

func saveDoubaoVideoStore(
	meta *meta.Meta,
	store adaptor.Store,
	videoID string,
	expiresAt time.Time,
) error {
	if store == nil || videoID == "" {
		return nil
	}

	return store.SaveStore(adaptor.StoreCache{
		ID:        coremodel.VideoGenerationStoreID(videoID),
		GroupID:   meta.Group.ID,
		TokenID:   meta.Token.ID,
		ChannelID: meta.Channel.ID,
		Model:     meta.OriginModel,
		ExpiresAt: expiresAt,
	})
}

func doubaoVideoRequestFromMeta(meta *meta.Meta) doubaoVideoRequest {
	if meta == nil {
		return doubaoVideoRequest{}
	}

	if value, ok := meta.Get(metaDoubaoVideoRequest); ok {
		request, _ := value.(doubaoVideoRequest)
		return request
	}

	return doubaoVideoRequest{}
}

func doubaoVideoPrompt(request doubaoVideoRequest) string {
	for _, item := range request.Content {
		if item.Type == "text" && item.Text != "" {
			return item.Text
		}
	}

	return ""
}

func doubaoVideoExpiresAt(response doubaoVideoTaskResponse) time.Time {
	if response.CreatedAt > 0 && response.ExecutionExpiresAfter > 0 {
		return time.Unix(response.CreatedAt+response.ExecutionExpiresAfter, 0)
	}

	return time.Now().Add(doubaoVideoTTL)
}

func doubaoVideoDimensions(resolution, ratio string) (int, int) {
	var height int
	switch strings.ToLower(strings.TrimSpace(resolution)) {
	case "1080p":
		height = 1080
	case "720p":
		height = 720
	case "480p":
		height = 480
	default:
		return 0, 0
	}

	switch strings.TrimSpace(ratio) {
	case "9:16":
		return height * 9 / 16, height
	case "1:1":
		return height, height
	case "4:3":
		return height * 4 / 3, height
	case "3:4":
		return height * 3 / 4, height
	case "21:9":
		return height * 21 / 9, height
	default:
		return height * 16 / 9, height
	}
}

func doubaoVideoJobStatus(status string) relaymodel.VideoGenerationJobStatus {
	switch strings.ToLower(status) {
	case "succeeded":
		return relaymodel.VideoGenerationJobStatusSucceeded
	case "running":
		return relaymodel.VideoGenerationJobStatusRunning
	case "failed":
		return relaymodel.VideoGenerationJobStatus("failed")
	case "expired":
		return relaymodel.VideoGenerationJobStatus("expired")
	case "cancelled", "canceled":
		return relaymodel.VideoGenerationJobStatus("cancelled")
	default:
		return relaymodel.VideoGenerationJobStatusQueued
	}
}

func doubaoVideoStatus(status string) relaymodel.VideoStatus {
	switch strings.ToLower(status) {
	case "succeeded":
		return relaymodel.VideoStatusCompleted
	case "running":
		return relaymodel.VideoStatusInProgress
	case "failed":
		return relaymodel.VideoStatusFailed
	case "cancelled", "canceled":
		return relaymodel.VideoStatusCancelled
	default:
		return relaymodel.VideoStatusQueued
	}
}

func doubaoContentUpstreamID(meta *meta.Meta) string {
	if meta == nil {
		return ""
	}

	if meta.Mode == mode.VideosContent {
		return meta.VideoID
	}

	return meta.GenerationID
}

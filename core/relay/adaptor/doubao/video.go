package doubao

import (
	"bytes"
	"context"
	"encoding/base64"
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
	"github.com/labring/aiproxy/core/relay/meta"
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

type doubaoOpenAIVideoMode string

const (
	doubaoOpenAIVideoModeCreate doubaoOpenAIVideoMode = "create"
	doubaoOpenAIVideoModeEdit   doubaoOpenAIVideoMode = "edit"
	doubaoOpenAIVideoModeExtend doubaoOpenAIVideoMode = "extend"
)

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

type doubaoVideoStoreMetadata struct {
	Prompt     string `json:"prompt,omitempty"`
	Resolution string `json:"resolution,omitempty"`
	Ratio      string `json:"ratio,omitempty"`
	Duration   int    `json:"duration,omitempty"`
}

func ConvertVideoGenerationJobRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	return convertDoubaoVideoGenerationJobRequest(meta, req)
}

func ConvertVideosRequest(meta *meta.Meta, req *http.Request) (adaptor.ConvertResult, error) {
	return convertDoubaoVideosRequest(meta, req)
}

func ConvertVideosEditRequest(meta *meta.Meta, req *http.Request) (adaptor.ConvertResult, error) {
	return convertDoubaoVideosEditRequest(meta, req)
}

func ConvertVideosExtensionRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	return convertDoubaoVideosExtensionRequest(meta, req)
}

func ConvertVideoGenerationsGetJobsRequest(
	_ *meta.Meta,
	_ *http.Request,
) (adaptor.ConvertResult, error) {
	return adaptor.ConvertResult{}, nil
}

func ConvertVideoGenerationsContentRequest(
	_ *meta.Meta,
	_ *http.Request,
) (adaptor.ConvertResult, error) {
	return adaptor.ConvertResult{}, nil
}

func ConvertVideosGetRequest(_ *meta.Meta, _ *http.Request) (adaptor.ConvertResult, error) {
	return adaptor.ConvertResult{}, nil
}

func ConvertVideosContentRequest(_ *meta.Meta, _ *http.Request) (adaptor.ConvertResult, error) {
	return adaptor.ConvertResult{}, nil
}

func convertDoubaoVideoGenerationJobRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	request, err := parseDoubaoVideoGenerationJobRequest(req)
	if err != nil {
		return adaptor.ConvertResult{}, convertRequestError(meta, err.Error())
	}

	return convertDoubaoVideoRequest(meta, request)
}

func convertDoubaoVideosRequest(meta *meta.Meta, req *http.Request) (adaptor.ConvertResult, error) {
	request, err := parseDoubaoVideosRequest(req)
	if err != nil {
		return adaptor.ConvertResult{}, convertRequestError(meta, err.Error())
	}

	return convertDoubaoVideoRequest(meta, request)
}

func convertDoubaoVideosEditRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	request, err := parseDoubaoVideosEditRequest(req)
	if err != nil {
		return adaptor.ConvertResult{}, convertRequestError(meta, err.Error())
	}

	return convertDoubaoVideoRequest(meta, request)
}

func convertDoubaoVideosExtensionRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	request, err := parseDoubaoVideosExtensionRequest(req)
	if err != nil {
		return adaptor.ConvertResult{}, convertRequestError(meta, err.Error())
	}

	return convertDoubaoVideoRequest(meta, request)
}

func convertDoubaoVideoRequest(
	meta *meta.Meta,
	request doubaoVideoRequest,
) (adaptor.ConvertResult, error) {
	if len(request.Content) == 0 {
		return adaptor.ConvertResult{}, convertRequestError(meta, "content is required")
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

func parseDoubaoVideoGenerationJobRequest(req *http.Request) (doubaoVideoRequest, error) {
	if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
		return parseDoubaoMultipartVideoGenerationJobRequest(req)
	}

	var raw map[string]any
	if err := common.UnmarshalRequestReusable(req, &raw); err != nil {
		return doubaoVideoRequest{}, err
	}

	return parseDoubaoJSONVideoGenerationJobRequest(raw), nil
}

func parseDoubaoVideosRequest(req *http.Request) (doubaoVideoRequest, error) {
	if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
		return parseDoubaoMultipartVideosRequest(req, doubaoOpenAIVideoModeCreate)
	}

	var raw map[string]any
	if err := common.UnmarshalRequestReusable(req, &raw); err != nil {
		return doubaoVideoRequest{}, err
	}

	return parseDoubaoJSONVideosRequest(raw), nil
}

func parseDoubaoVideosEditRequest(req *http.Request) (doubaoVideoRequest, error) {
	return parseDoubaoVideosModeRequest(req, doubaoOpenAIVideoModeEdit)
}

func parseDoubaoVideosExtensionRequest(req *http.Request) (doubaoVideoRequest, error) {
	return parseDoubaoVideosModeRequest(req, doubaoOpenAIVideoModeExtend)
}

func parseDoubaoVideosModeRequest(
	req *http.Request,
	openAIMode doubaoOpenAIVideoMode,
) (doubaoVideoRequest, error) {
	if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
		return parseDoubaoMultipartVideosRequest(req, openAIMode)
	}

	var raw map[string]any
	if err := common.UnmarshalRequestReusable(req, &raw); err != nil {
		return doubaoVideoRequest{}, err
	}

	request := parseDoubaoJSONVideosRequest(raw)
	addDoubaoOpenAIVideoField(&request.Content, raw["video"], openAIMode)

	return request, nil
}

func parseDoubaoJSONVideoGenerationJobRequest(raw map[string]any) doubaoVideoRequest {
	request := parseDoubaoJSONOpenAIVideoCommonRequest(raw, doubaoVideoJobSizeFromJSON(raw))
	request.Duration = intPtrFromAny(raw["n_seconds"])

	return request
}

func parseDoubaoJSONVideosRequest(raw map[string]any) doubaoVideoRequest {
	request := parseDoubaoJSONOpenAIVideoCommonRequest(raw, stringFromAny(raw["size"]))
	request.Duration = intPtrFromAny(raw["seconds"])

	return request
}

func parseDoubaoJSONOpenAIVideoCommonRequest(raw map[string]any, size string) doubaoVideoRequest {
	request := doubaoVideoRequest{
		Content:          doubaoVideoContentFromAny(raw["content"]),
		CallbackURL:      stringFromAny(raw["callback_url"]),
		ServiceTier:      stringFromAny(raw["service_tier"]),
		SafetyIdentifier: stringFromAny(raw["safety_identifier"]),
		Resolution: firstNonEmptyString(
			stringFromAny(raw["resolution"]),
			doubaoVideoResolutionFromSize(size),
		),
		Ratio: firstNonEmptyString(
			stringFromAny(raw["ratio"]),
			ratioFromSize(size),
		),
		Seed:                  raw["seed"],
		ExecutionExpiresAfter: intPtrFromAny(raw["execution_expires_after"]),
		GenerateAudio:         boolPtrFromAny(raw["generate_audio"]),
		Draft:                 boolPtrFromAny(raw["draft"]),
		Priority:              intPtrFromAny(raw["priority"]),
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

func parseDoubaoMultipartVideoGenerationJobRequest(req *http.Request) (doubaoVideoRequest, error) {
	request, err := parseDoubaoMultipartOpenAIVideoCommonRequest(
		req,
		doubaoVideoJobSizeFromForm,
	)
	if err != nil {
		return doubaoVideoRequest{}, err
	}

	setOptionalInt(&request.Duration, req.PostFormValue("n_seconds"))

	return request, nil
}

func parseDoubaoMultipartVideosRequest(
	req *http.Request,
	openAIMode doubaoOpenAIVideoMode,
) (doubaoVideoRequest, error) {
	request, err := parseDoubaoMultipartOpenAIVideoCommonRequest(req, doubaoVideoSizeFromForm)
	if err != nil {
		return doubaoVideoRequest{}, err
	}

	setOptionalInt(&request.Duration, req.PostFormValue("seconds"))

	addDoubaoOpenAIVideoField(
		&request.Content,
		req.PostFormValue("video"),
		openAIMode,
	)

	return request, nil
}

func parseDoubaoMultipartOpenAIVideoCommonRequest(
	req *http.Request,
	sizeFromForm func(*http.Request) string,
) (doubaoVideoRequest, error) {
	if err := common.ParseMultipartFormWithLimit(req); err != nil {
		return doubaoVideoRequest{}, fmt.Errorf("parse multipart form: %w", err)
	}

	size := ""
	if sizeFromForm != nil {
		size = sizeFromForm(req)
	}

	request := doubaoVideoRequest{
		CallbackURL:      req.PostFormValue("callback_url"),
		ServiceTier:      req.PostFormValue("service_tier"),
		SafetyIdentifier: req.PostFormValue("safety_identifier"),
		Resolution: firstNonEmptyString(
			req.PostFormValue("resolution"),
			doubaoVideoResolutionFromSize(size),
		),
		Ratio: firstNonEmptyString(
			req.PostFormValue("ratio"),
			ratioFromSize(size),
		),
		Content: []doubaoVideoContent{},
	}

	if prompt := req.PostFormValue("prompt"); prompt != "" {
		request.Content = append(request.Content, doubaoVideoContent{Type: "text", Text: prompt})
	}

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

func doubaoVideoSizeFromForm(req *http.Request) string {
	return req.PostFormValue("size")
}

func doubaoVideoJobSizeFromJSON(raw map[string]any) string {
	width := intFromPtr(intPtrFromAny(raw["width"]))

	height := intFromPtr(intPtrFromAny(raw["height"]))
	if width <= 0 || height <= 0 {
		return ""
	}

	return fmt.Sprintf("%dx%d", width, height)
}

func doubaoVideoJobSizeFromForm(req *http.Request) string {
	width := intFromPtr(intPtrFromAny(req.PostFormValue("width")))

	height := intFromPtr(intPtrFromAny(req.PostFormValue("height")))
	if width <= 0 || height <= 0 {
		return ""
	}

	return fmt.Sprintf("%dx%d", width, height)
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

	if draftTaskID := doubaoVideoDraftTaskIDFromRaw(raw); draftTaskID != "" {
		addDoubaoDraftTaskContent(&content, draftTaskID)
	}

	return content
}

func addDoubaoOpenAIVideoField(
	content *[]doubaoVideoContent,
	value any,
	openAIMode doubaoOpenAIVideoMode,
) {
	if openAIMode == doubaoOpenAIVideoModeCreate {
		return
	}

	videoURL := strings.TrimSpace(stringFromAny(value))
	if videoURL == "" {
		return
	}

	if strings.HasPrefix(videoURL, "video_") || strings.HasPrefix(videoURL, "doubao_") {
		addDoubaoDraftTaskContent(content, videoURL)
		return
	}

	roleForMode := func() string {
		if openAIMode == doubaoOpenAIVideoModeExtend {
			return "first_video"
		}

		return "reference_video"
	}

	for i := range *content {
		item := &(*content)[i]
		if item.Type != "video_url" || item.Role != "reference_video" {
			continue
		}

		urlValue := ""
		if item.VideoURL != nil {
			urlValue = item.VideoURL.URL
		}

		if urlValue == videoURL {
			item.Role = roleForMode()
			return
		}
	}

	*content = append(*content, doubaoVideoContent{
		Type:     "video_url",
		VideoURL: &doubaoVideoURLContent{URL: videoURL},
		Role:     roleForMode(),
	})
}

func doubaoVideoDraftTaskIDFromRaw(raw map[string]any) string {
	return stringFromAny(firstPresent(raw, "draft_task_id", "video_id"))
}

func addDoubaoDraftTaskContent(content *[]doubaoVideoContent, draftTaskID string) {
	draftTaskID = strings.TrimSpace(draftTaskID)
	if draftTaskID == "" {
		return
	}

	for _, item := range *content {
		if item.Type == "draft_task" && item.DraftTask != nil && item.DraftTask.ID == draftTaskID {
			return
		}
	}

	*content = append(*content, doubaoVideoContent{
		Type:      "draft_task",
		DraftTask: &doubaoDraftTask{ID: draftTaskID},
	})
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
	add("video_url", "video", "reference_video")
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

func VideoGenerationJobSubmitHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	response, relayErr := readDoubaoVideoTaskResponse(resp)
	if relayErr != nil {
		return adaptor.DoResponseResult{}, relayErr
	}

	expiresAt := doubaoVideoExpiresAt(response)
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
		UsageContext: doubaoVideoUsageContext(
			&response,
		).WithFallback(doubaoVideoRequestUsageContext(meta)),
	})
}

func VideosSubmitHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	response, relayErr := readDoubaoVideoTaskResponse(resp)
	if relayErr != nil {
		return adaptor.DoResponseResult{}, relayErr
	}

	expiresAt := doubaoVideoExpiresAt(response)
	if err := saveDoubaoVideoStore(meta, store, response.ID, expiresAt); err != nil {
		common.GetLogger(c).Errorf("save doubao video store failed: %v", err)
	}

	video := buildDoubaoVideo(meta, response.ID, &response)

	return writeDoubaoVideoObject(c, video, adaptor.DoResponseResult{
		UpstreamID: response.ID,
		AsyncUsage: true,
		UsageContext: doubaoVideoUsageContext(
			&response,
		).WithFallback(doubaoVideoRequestUsageContext(meta)),
	})
}

func readDoubaoVideoTaskResponse(
	resp *http.Response,
) (relaymodel.DoubaoVideoTaskResponse, adaptor.Error) {
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return relaymodel.DoubaoVideoTaskResponse{}, OpenAIVideoErrorHandler(resp)
	}

	defer resp.Body.Close()

	var response relaymodel.DoubaoVideoTaskResponse
	if err := common.UnmarshalResponse(resp, &response); err != nil {
		return relaymodel.DoubaoVideoTaskResponse{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	if response.ID == "" {
		return relaymodel.DoubaoVideoTaskResponse{}, relaymodel.WrapperOpenAIVideoErrorWithMessage(
			"missing id in doubao video response",
			http.StatusInternalServerError,
		)
	}

	if response.Status == "" {
		response.Status = "queued"
	}

	return response, nil
}

func VideoGenerationJobStatusHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, OpenAIVideoErrorHandler(resp)
	}

	defer resp.Body.Close()

	var response relaymodel.DoubaoVideoTaskResponse
	if err := common.UnmarshalResponse(resp, &response); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	if response.ID == "" {
		response.ID = meta.JobID
	}

	applyStoredDoubaoVideoRequestMetadata(
		meta,
		store,
		coremodel.VideoJobStoreID(response.ID),
		&response,
	)

	expiresAt := doubaoVideoExpiresAt(response)
	job := buildDoubaoVideoJob(meta, response.ID, &response)

	if job.Status == relaymodel.VideoGenerationJobStatusSucceeded {
		for _, generation := range job.Generations {
			if err := saveDoubaoVideoStore(meta, store, generation.ID, expiresAt); err != nil {
				common.GetLogger(c).Errorf("save doubao video generation store failed: %v", err)
			}
		}
	}

	return writeDoubaoVideoObject(c, job, adaptor.DoResponseResult{
		UsageContext: doubaoVideoUsageContext(
			&response,
		).WithFallback(doubaoVideoRequestUsageContext(meta)),
	})
}

func VideosStatusHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, OpenAIVideoErrorHandler(resp)
	}

	defer resp.Body.Close()

	var response relaymodel.DoubaoVideoTaskResponse
	if err := common.UnmarshalResponse(resp, &response); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	if response.ID == "" {
		response.ID = meta.VideoID
	}

	applyStoredDoubaoVideoRequestMetadata(
		meta,
		store,
		coremodel.VideoGenerationStoreID(response.ID),
		&response,
	)

	expiresAt := doubaoVideoExpiresAt(response)
	if response.Content.VideoURL != "" || response.Content.FileURL != "" {
		if err := saveDoubaoVideoStore(meta, store, response.ID, expiresAt); err != nil {
			common.GetLogger(c).Errorf("save doubao video store failed: %v", err)
		}
	}

	return writeDoubaoVideoObject(
		c,
		buildDoubaoVideo(meta, response.ID, &response),
		adaptor.DoResponseResult{
			UpstreamID: response.ID,
			UsageContext: doubaoVideoUsageContext(
				&response,
			).WithFallback(doubaoVideoRequestUsageContext(meta)),
		},
	)
}

func VideoGenerationJobContentHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	return fetchDoubaoVideoContentHandler(meta, c, resp, meta.GenerationID)
}

func VideosContentHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	return fetchDoubaoVideoContentHandler(meta, c, resp, meta.VideoID)
}

func fetchDoubaoVideoContentHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
	id string,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, OpenAIVideoErrorHandler(resp)
	}

	defer resp.Body.Close()

	var response relaymodel.DoubaoVideoTaskResponse
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

	return adaptor.DoResponseResult{UpstreamID: id}, nil
}

func buildDoubaoVideoJob(
	meta *meta.Meta,
	id string,
	response *relaymodel.DoubaoVideoTaskResponse,
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

	resolution, ratio := doubaoVideoResolutionAndRatio(response, request)
	job.Width, job.Height = doubaoVideoDimensions(resolution, ratio)

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
	response *relaymodel.DoubaoVideoTaskResponse,
) relaymodel.Video {
	now := time.Now().Unix()
	request := doubaoVideoRequestFromMeta(meta)
	resolution, ratio := doubaoVideoResolutionAndRatio(response, request)
	video := relaymodel.Video{
		ID:        id,
		Object:    relaymodel.VideoObject,
		CreatedAt: firstPositiveInt64(response.CreatedAt, now),
		Status:    doubaoVideoStatus(response.Status),
		Model:     meta.OriginModel,
		Prompt:    doubaoVideoPrompt(request),
		Seconds:   firstPositiveInt(response.Duration, intFromPtr(request.Duration)),
		Size:      doubaoVideoSize(resolution, ratio),
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

func doubaoVideoUsageToModelUsage(usage relaymodel.DoubaoVideoUsage) coremodel.Usage {
	// Seedance video usage is returned as completion/total tokens by Ark, and
	// model pricing uses those tokens directly rather than generated seconds.
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

func doubaoVideoUsageContext(response *relaymodel.DoubaoVideoTaskResponse) coremodel.UsageContext {
	if response == nil {
		return coremodel.UsageContext{}
	}

	resolution := strings.TrimSpace(response.Resolution)
	ratio := strings.TrimSpace(response.Ratio)

	return coremodel.UsageContext{
		Resolution:       doubaoVideoSize(resolution, ratio),
		NativeResolution: resolution,
		ServiceTier:      response.ServiceTier,
	}
}

func doubaoVideoRequestUsageContext(meta *meta.Meta) coremodel.UsageContext {
	request := doubaoVideoRequestFromMeta(meta)

	return coremodel.UsageContext{
		Resolution:       doubaoVideoSize(request.Resolution, request.Ratio),
		NativeResolution: request.Resolution,
		ServiceTier:      request.ServiceTier,
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
		Metadata:  doubaoVideoStoreMetadataString(meta),
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
		Metadata:  doubaoVideoStoreMetadataString(meta),
		ExpiresAt: expiresAt,
	})
}

func doubaoVideoStoreMetadataString(meta *meta.Meta) string {
	request := doubaoVideoRequestFromMeta(meta)
	metadata := doubaoVideoStoreMetadata{
		Prompt:     doubaoVideoPrompt(request),
		Resolution: request.Resolution,
		Ratio:      request.Ratio,
		Duration:   intFromPtr(request.Duration),
	}

	data, err := sonic.MarshalString(metadata)
	if err != nil {
		return ""
	}

	return data
}

func applyStoredDoubaoVideoRequestMetadata(
	meta *meta.Meta,
	store adaptor.Store,
	storeID string,
	response *relaymodel.DoubaoVideoTaskResponse,
) {
	if meta == nil || store == nil || storeID == "" || response == nil {
		return
	}

	cache, err := store.GetStore(meta.Group.ID, meta.Token.ID, storeID)
	if err != nil || cache.Metadata == "" {
		return
	}

	var metadata doubaoVideoStoreMetadata
	if err := sonic.UnmarshalString(cache.Metadata, &metadata); err != nil {
		return
	}

	var request doubaoVideoRequest
	if value, ok := meta.Get(metaDoubaoVideoRequest); ok {
		request, _ = value.(doubaoVideoRequest)
	}

	if doubaoVideoPrompt(request) == "" && metadata.Prompt != "" {
		request.Content = append(
			request.Content,
			doubaoVideoContent{Type: "text", Text: metadata.Prompt},
		)
	}

	if request.Resolution == "" {
		request.Resolution = metadata.Resolution
	}

	if request.Ratio == "" {
		request.Ratio = metadata.Ratio
	}

	if request.Duration == nil && metadata.Duration > 0 {
		duration := metadata.Duration
		request.Duration = &duration
	}

	if response.Resolution == "" {
		response.Resolution = metadata.Resolution
	}

	if response.Ratio == "" {
		response.Ratio = metadata.Ratio
	}

	if response.Duration == 0 {
		response.Duration = metadata.Duration
	}

	if len(request.Content) > 0 ||
		request.Resolution != "" ||
		request.Ratio != "" ||
		request.Duration != nil {
		meta.Set(metaDoubaoVideoRequest, request)
	}
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

func doubaoVideoExpiresAt(response relaymodel.DoubaoVideoTaskResponse) time.Time {
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
		return height, height * 16 / 9
	case "1:1":
		return height, height
	case "4:3":
		return height * 4 / 3, height
	case "3:4":
		return height, height * 4 / 3
	case "21:9":
		return height * 21 / 9, height
	default:
		return height * 16 / 9, height
	}
}

func doubaoVideoResolutionAndRatio(
	response *relaymodel.DoubaoVideoTaskResponse,
	request doubaoVideoRequest,
) (string, string) {
	if response == nil {
		return request.Resolution, request.Ratio
	}

	return firstNonEmptyString(response.Resolution, request.Resolution),
		firstNonEmptyString(response.Ratio, request.Ratio)
}

func doubaoVideoSize(resolution, ratio string) string {
	width, height := doubaoVideoDimensions(resolution, ratio)
	if width <= 0 || height <= 0 {
		return ""
	}

	return fmt.Sprintf("%dx%d", width, height)
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

package siliconflow

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
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	relayutils "github.com/labring/aiproxy/core/relay/utils"
)

const (
	metaVideoRequest    = "siliconflow_video_request"
	siliconFlowVideoTTL = 24 * time.Hour
)

type videoSubmitRequest struct {
	Model          string `json:"model,omitempty"`
	Prompt         string `json:"prompt,omitempty"`
	ImageSize      string `json:"image_size,omitempty"`
	Image          string `json:"image,omitempty"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	Seed           any    `json:"seed,omitempty"`
}

type videoSubmitResponse struct {
	RequestID string `json:"requestId"`
}

type videoStatusRequest struct {
	RequestID string `json:"requestId"`
}

type videoStatusResponse struct {
	Status  string             `json:"status"`
	Reason  string             `json:"reason,omitempty"`
	Results videoStatusResults `json:"results,omitempty"`
}

type videoStatusResults struct {
	Videos  []videoStatusVideo `json:"videos,omitempty"`
	Timings map[string]any     `json:"timings,omitempty"`
	Seed    int64              `json:"seed,omitempty"`
}

type videoStatusVideo struct {
	URL string `json:"url"`
}

func ConvertVideoRequest(meta *meta.Meta, req *http.Request) (adaptor.ConvertResult, error) {
	if meta.Mode == mode.VideosRemix {
		return adaptor.ConvertResult{}, errors.New("siliconflow does not support videos remix")
	}

	var request videoSubmitRequest

	if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
		parsed, err := multipartVideoSubmitRequest(req)
		if err != nil {
			return adaptor.ConvertResult{}, err
		}

		request = parsed
	} else {
		var reqMap map[string]any
		if err := common.UnmarshalRequestReusable(req, &reqMap); err != nil {
			return adaptor.ConvertResult{}, err
		}

		request = jsonVideoSubmitRequest(reqMap)
	}

	request.Model = meta.ActualModel
	meta.Set(metaVideoRequest, request)

	data, err := sonic.Marshal(request)
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

func ConvertVideoStatusRequest(meta *meta.Meta, _ *http.Request) (adaptor.ConvertResult, error) {
	data, err := sonic.Marshal(videoStatusRequest{RequestID: meta.JobID})
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

func ConvertVideoContentStatusRequest(
	meta *meta.Meta,
	_ *http.Request,
) (adaptor.ConvertResult, error) {
	data, err := sonic.Marshal(videoStatusRequest{RequestID: meta.GenerationID})
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

func ConvertVideosStatusRequest(meta *meta.Meta, _ *http.Request) (adaptor.ConvertResult, error) {
	data, err := sonic.Marshal(videoStatusRequest{RequestID: meta.VideoID})
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

func jsonVideoSubmitRequest(reqMap map[string]any) videoSubmitRequest {
	request := videoSubmitRequest{
		Prompt:         stringFromMap(reqMap, "prompt"),
		ImageSize:      videoImageSize(reqMap),
		Image:          videoImage(reqMap),
		NegativePrompt: stringFromMap(reqMap, "negative_prompt"),
		Seed:           reqMap["seed"],
	}

	return request
}

func multipartVideoSubmitRequest(req *http.Request) (videoSubmitRequest, error) {
	if err := common.ParseMultipartFormWithLimit(req); err != nil {
		return videoSubmitRequest{}, fmt.Errorf("parse multipart form: %w", err)
	}

	request := videoSubmitRequest{
		Prompt:         req.PostFormValue("prompt"),
		ImageSize:      strings.TrimSpace(req.PostFormValue("size")),
		NegativePrompt: req.PostFormValue("negative_prompt"),
	}

	if request.ImageSize == "" {
		width := req.PostFormValue("width")

		height := req.PostFormValue("height")
		if width != "" && height != "" {
			request.ImageSize = width + "x" + height
		}
	}

	if seed := strings.TrimSpace(req.PostFormValue("seed")); seed != "" {
		request.Seed = seed
	}

	imageValue := req.PostFormValue("input_reference")
	if imageValue == "" {
		imageValue = req.PostFormValue("image")
	}

	if imageValue != "" {
		request.Image = imageValue
		return request, nil
	}

	imageData, err := multipartVideoImageDataURL(req.MultipartForm.File)
	if err != nil {
		return videoSubmitRequest{}, err
	}

	request.Image = imageData

	return request, nil
}

func videoImageSize(reqMap map[string]any) string {
	if size := stringFromMap(reqMap, "size"); size != "" {
		return size
	}

	width, widthOK := intFromAny(reqMap["width"])

	height, heightOK := intFromAny(reqMap["height"])
	if widthOK && heightOK && width > 0 && height > 0 {
		return fmt.Sprintf("%dx%d", width, height)
	}

	if imageSize := stringFromMap(reqMap, "image_size"); imageSize != "" {
		return imageSize
	}

	return ""
}

func videoImage(reqMap map[string]any) string {
	if inputReference := stringFromMap(reqMap, "input_reference"); inputReference != "" {
		return inputReference
	}

	if image := stringFromMap(reqMap, "image"); image != "" {
		return image
	}

	return ""
}

func stringFromMap(reqMap map[string]any, key string) string {
	value, ok := reqMap[key]
	if !ok {
		return ""
	}

	str, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(str)
}

func intFromAny(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}

		return parsed, true
	default:
		return 0, false
	}
}

func multipartVideoImageDataURL(files map[string][]*multipart.FileHeader) (string, error) {
	fileHeaders := make(
		[]*multipart.FileHeader,
		0,
		len(files["input_reference"])+len(files["image"]),
	)
	fileHeaders = append(fileHeaders, files["input_reference"]...)
	fileHeaders = append(fileHeaders, files["image"]...)

	if len(fileHeaders) == 0 {
		return "", nil
	}

	if len(fileHeaders) > 1 {
		return "", errors.New("video image supports at most 1 file")
	}

	return multipartImageDataURL(fileHeaders[0])
}

func multipartImageDataURL(fileHeader *multipart.FileHeader) (string, error) {
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

func VideoSubmitHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, ErrorHandler(resp)
	}

	defer resp.Body.Close()

	var response videoSubmitResponse
	if err := common.UnmarshalResponse(resp, &response); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	if response.RequestID == "" {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoErrorWithMessage(
			"missing requestId in siliconflow video submit response",
			http.StatusInternalServerError,
		)
	}

	meta.RequestUsage = siliconFlowVideoSubmitUsage()

	if meta.Mode == mode.Videos {
		video := buildVideo(meta, response.RequestID, relaymodel.VideoStatusQueued, nil)
		if err := saveVideoGenerationStore(
			meta,
			store,
			response.RequestID,
			time.Now().Add(siliconFlowVideoTTL),
		); err != nil {
			common.GetLogger(c).Errorf("save siliconflow video store failed: %v", err)
		}

		data, err := sonic.Marshal(video)
		if err != nil {
			return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
				err,
				http.StatusInternalServerError,
			)
		}

		c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.Header().Set("Content-Length", strconv.Itoa(len(data)))
		_, _ = c.Writer.Write(data)

		return adaptor.DoResponseResult{
			UpstreamID: response.RequestID,
			AsyncUsage: true,
		}, nil
	}

	if err := saveVideoJobStore(
		meta,
		store,
		response.RequestID,
		time.Now().Add(siliconFlowVideoTTL),
	); err != nil {
		common.GetLogger(c).Errorf("save siliconflow video job store failed: %v", err)
	}

	job := buildVideoJob(meta, response.RequestID, relaymodel.VideoGenerationJobStatusQueued, nil)

	data, err := sonic.Marshal(job)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = c.Writer.Write(data)

	return adaptor.DoResponseResult{
		UpstreamID: response.RequestID,
		AsyncUsage: true,
	}, nil
}

func siliconFlowVideoSubmitUsage() model.Usage {
	return model.Usage{
		OutputTokens: model.ZeroNullInt64(1),
		TotalTokens:  model.ZeroNullInt64(1),
	}
}

func VideoStatusHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, ErrorHandler(resp)
	}

	defer resp.Body.Close()

	var response videoStatusResponse
	if err := common.UnmarshalResponse(resp, &response); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	if meta.Mode == mode.VideosGet {
		video := buildVideo(
			meta,
			meta.VideoID,
			siliconFlowVideoStatusToOpenAI(response.Status),
			&response,
		)
		if video.Status == relaymodel.VideoStatusCompleted {
			if err := saveVideoGenerationStore(
				meta,
				store,
				video.ID,
				time.Now().Add(siliconFlowVideoTTL),
			); err != nil {
				common.GetLogger(c).Errorf("save siliconflow video store failed: %v", err)
			}
		}

		data, err := sonic.Marshal(video)
		if err != nil {
			return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
				err,
				http.StatusInternalServerError,
			)
		}

		c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.Header().Set("Content-Length", strconv.Itoa(len(data)))
		_, _ = c.Writer.Write(data)

		return adaptor.DoResponseResult{UpstreamID: video.ID}, nil
	}

	job := buildVideoJob(meta, meta.JobID, siliconFlowVideoStatus(response.Status), &response)

	if response.Status == "Succeed" {
		expiresAt := time.Now().Add(siliconFlowVideoTTL)
		for _, generation := range job.Generations {
			if err := saveVideoGenerationStore(meta, store, generation.ID, expiresAt); err != nil {
				common.GetLogger(c).
					Errorf("save siliconflow video generation store failed: %v", err)
			}
		}
	}

	data, err := sonic.Marshal(job)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = c.Writer.Write(data)

	return adaptor.DoResponseResult{}, nil
}

func VideoContentHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, ErrorHandler(resp)
	}

	defer resp.Body.Close()

	var response videoStatusResponse
	if err := common.UnmarshalResponse(resp, &response); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	videoURL := firstSiliconFlowVideoURL(response.Results.Videos)
	if videoURL == "" {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIVideoErrorWithMessage(
			"video url is empty",
			http.StatusInternalServerError,
		)
	}

	videoResp, err := fetchSiliconFlowVideoContent(c.Request.Context(), meta, videoURL)
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

	return adaptor.DoResponseResult{UpstreamID: siliconFlowContentUpstreamID(meta)}, nil
}

func fetchSiliconFlowVideoContent(
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

func firstSiliconFlowVideoURL(videos []videoStatusVideo) string {
	for _, video := range videos {
		if video.URL != "" {
			return video.URL
		}
	}

	return ""
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

func siliconFlowContentUpstreamID(meta *meta.Meta) string {
	if meta == nil {
		return ""
	}

	if meta.Mode == mode.VideosContent {
		return meta.VideoID
	}

	return meta.GenerationID
}

func buildVideo(
	meta *meta.Meta,
	id string,
	status relaymodel.VideoStatus,
	response *videoStatusResponse,
) relaymodel.Video {
	now := time.Now().Unix()

	var request videoSubmitRequest
	if value, ok := meta.Get(metaVideoRequest); ok {
		request, _ = value.(videoSubmitRequest)
	}

	video := relaymodel.Video{
		ID:        id,
		Object:    relaymodel.VideoObject,
		CreatedAt: now,
		Status:    status,
		Model:     meta.OriginModel,
		Prompt:    request.Prompt,
		Size:      request.ImageSize,
	}

	switch video.Status {
	case relaymodel.VideoStatusCompleted:
		video.Progress = 100
	case relaymodel.VideoStatusInProgress:
		video.Progress = 50
	case relaymodel.VideoStatusQueued:
		video.Progress = 0
	}

	if response != nil && response.Status == "Failed" {
		reason := response.Reason
		if reason == "" {
			reason = "failed"
		}

		video.Error = map[string]any{"message": reason}
	}

	return video
}

func buildVideoJob(
	meta *meta.Meta,
	id string,
	status relaymodel.VideoGenerationJobStatus,
	response *videoStatusResponse,
) relaymodel.VideoGenerationJob {
	now := time.Now().Unix()
	expiresAt := now + int64((24 * time.Hour).Seconds())

	var request videoSubmitRequest
	if value, ok := meta.Get(metaVideoRequest); ok {
		request, _ = value.(videoSubmitRequest)
	}

	job := relaymodel.VideoGenerationJob{
		Object:      relaymodel.VideoGenerationJobObject,
		ID:          id,
		Status:      status,
		CreatedAt:   now,
		ExpiresAt:   &expiresAt,
		Generations: []relaymodel.VideoGenerations{},
		Prompt:      request.Prompt,
		Model:       meta.OriginModel,
		NVariants:   1,
	}

	if request.ImageSize != "" {
		job.Width, job.Height = parseSize(request.ImageSize)
	}

	if response == nil {
		return job
	}

	if response.Status == "Succeed" || response.Status == "Failed" {
		job.FinishedAt = &now
	}

	if response.Status == "Failed" {
		reason := response.Reason
		if reason == "" {
			reason = "failed"
		}

		job.FinishReason = &reason
	}

	for _, video := range response.Results.Videos {
		if video.URL == "" {
			continue
		}

		job.Generations = append(job.Generations, relaymodel.VideoGenerations{
			Object:    relaymodel.VideoGenerationObject,
			ID:        id,
			JobID:     id,
			CreatedAt: now,
			Width:     job.Width,
			Height:    job.Height,
			Prompt:    job.Prompt,
		})

		break
	}

	return job
}

func siliconFlowVideoStatus(status string) relaymodel.VideoGenerationJobStatus {
	switch status {
	case "Succeed":
		return relaymodel.VideoGenerationJobStatusSucceeded
	case "InProgress":
		return relaymodel.VideoGenerationJobStatusRunning
	case "Failed":
		return relaymodel.VideoGenerationJobStatus("failed")
	default:
		return relaymodel.VideoGenerationJobStatusQueued
	}
}

func siliconFlowVideoStatusToOpenAI(status string) relaymodel.VideoStatus {
	switch status {
	case "Succeed":
		return relaymodel.VideoStatusCompleted
	case "InProgress":
		return relaymodel.VideoStatusInProgress
	case "Failed":
		return relaymodel.VideoStatusFailed
	default:
		return relaymodel.VideoStatusQueued
	}
}

func parseSize(size string) (int, int) {
	width, height, ok := strings.Cut(size, "x")
	if !ok {
		return 0, 0
	}

	parsedWidth, err := strconv.Atoi(strings.TrimSpace(width))
	if err != nil {
		return 0, 0
	}

	parsedHeight, err := strconv.Atoi(strings.TrimSpace(height))
	if err != nil {
		return 0, 0
	}

	return parsedWidth, parsedHeight
}

func saveVideoJobStore(
	meta *meta.Meta,
	store adaptor.Store,
	jobID string,
	expiresAt time.Time,
) error {
	if store == nil {
		return nil
	}

	return store.SaveStore(adaptor.StoreCache{
		ID:        model.VideoJobStoreID(jobID),
		GroupID:   meta.Group.ID,
		TokenID:   meta.Token.ID,
		ChannelID: meta.Channel.ID,
		Model:     meta.OriginModel,
		ExpiresAt: expiresAt,
	})
}

func saveVideoGenerationStore(
	meta *meta.Meta,
	store adaptor.Store,
	generationID string,
	expiresAt time.Time,
) error {
	if store == nil || generationID == "" {
		return nil
	}

	return store.SaveStore(adaptor.StoreCache{
		ID:        model.VideoGenerationStoreID(generationID),
		GroupID:   meta.Group.ID,
		TokenID:   meta.Token.ID,
		ChannelID: meta.Channel.ID,
		Model:     meta.OriginModel,
		ExpiresAt: expiresAt,
	})
}

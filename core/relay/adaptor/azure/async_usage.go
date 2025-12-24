package azure

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

// Ensure Adaptor implements AsyncUsageFetcher
var _ model.AsyncUsageFetcher = (*Adaptor)(nil)

// asyncJobData represents the job_id stored in AsyncUsageInfo.Data
type asyncJobData struct {
	JobID string `json:"job_id"`
}

// FetchAsyncUsage implements model.AsyncUsageFetcher for Azure adaptor
// It fetches the status of async tasks (like video generation) and returns usage when completed
func (a *Adaptor) FetchAsyncUsage(
	ctx context.Context,
	channel *model.Channel,
	info *model.AsyncUsageInfo,
) (model.Usage, bool, error) {
	switch mode.Mode(info.Mode) {
	case mode.VideoGenerationsJobs:
		return a.fetchVideoJobUsage(ctx, channel, info)
	default:
		return model.Usage{}, false, fmt.Errorf("unsupported async mode: %d", info.Mode)
	}
}

// fetchVideoJobUsage fetches the status of a video generation job from Azure
func (a *Adaptor) fetchVideoJobUsage(
	ctx context.Context,
	channel *model.Channel,
	info *model.AsyncUsageInfo,
) (model.Usage, bool, error) {
	// Parse job_id from Data
	var jobData asyncJobData
	if err := sonic.UnmarshalString(info.Data, &jobData); err != nil {
		return model.Usage{}, false, fmt.Errorf("failed to parse job data: %w", err)
	}

	if jobData.JobID == "" {
		return model.Usage{}, false, errors.New("job_id is empty")
	}

	// Get token and API version from channel key
	token, apiVersion, err := GetTokenAndAPIVersion(channel.Key)
	if err != nil {
		return model.Usage{}, false, fmt.Errorf("failed to parse channel key: %w", err)
	}

	// Construct the Azure-specific request URL
	// Format: https://{resource}.openai.azure.com/openai/v1/video/generations/jobs/{job_id}?api-version={version}
	baseURL := channel.BaseURL
	if baseURL == "" {
		baseURL = a.DefaultBaseURL()
	}

	requestURL, err := url.JoinPath(baseURL, "/openai/v1/video/generations/jobs", jobData.JobID)
	if err != nil {
		return model.Usage{}, false, fmt.Errorf("failed to construct URL: %w", err)
	}

	requestURL = fmt.Sprintf("%s?api-version=%s", requestURL, apiVersion)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return model.Usage{}, false, fmt.Errorf("failed to create request: %w", err)
	}

	// Azure uses Api-Key header instead of Authorization
	req.Header.Set("Api-Key", token)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return model.Usage{}, false, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return model.Usage{}, false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var job relaymodel.VideoGenerationJob
	if err := sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&job); err != nil {
		return model.Usage{}, false, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check job status
	switch job.Status {
	case relaymodel.VideoGenerationJobStatusSucceeded:
		// Job completed, calculate usage
		usage := calculateVideoUsage(&job)
		return usage, true, nil

	case relaymodel.VideoGenerationJobStatusQueued,
		relaymodel.VideoGenerationJobStatusProcessing,
		relaymodel.VideoGenerationJobStatusRunning:
		// Job still in progress
		return model.Usage{}, false, nil

	default:
		// Job failed or unknown status
		return model.Usage{}, false, fmt.Errorf("job failed with status: %s", job.Status)
	}
}

// calculateVideoUsage calculates the usage for a completed video generation job
// Usage is based on video duration (n_seconds) and number of variants
func calculateVideoUsage(job *relaymodel.VideoGenerationJob) model.Usage {
	// Calculate total seconds of video generated
	// Each variant is a separate video of n_seconds duration
	totalSeconds := job.NSeconds * job.NVariants

	// For video generation, we use OutputTokens to store the total seconds
	// This allows the pricing system to charge per second of video generated
	return model.Usage{
		OutputTokens: model.ZeroNullInt64(totalSeconds),
		TotalTokens:  model.ZeroNullInt64(totalSeconds),
	}
}

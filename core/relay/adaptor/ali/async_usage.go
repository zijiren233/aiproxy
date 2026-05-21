package ali

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/bytedance/sonic"
	coremodel "github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/mode"
	relayutils "github.com/labring/aiproxy/core/relay/utils"
)

var _ adaptor.AsyncUsageFetcher = (*Adaptor)(nil)

func (a *Adaptor) FetchAsyncUsage(
	ctx context.Context,
	channel *coremodel.Channel,
	info *coremodel.AsyncUsageInfo,
) (coremodel.Usage, bool, error) {
	switch mode.Mode(info.Mode) {
	case mode.VideoGenerationsJobs, mode.Videos, mode.VideosRemix:
		return a.fetchAliVideoJobUsage(ctx, channel, info)
	default:
		return coremodel.Usage{}, false, fmt.Errorf("unsupported async usage mode: %d", info.Mode)
	}
}

func (a *Adaptor) fetchAliVideoJobUsage(
	ctx context.Context,
	channel *coremodel.Channel,
	info *coremodel.AsyncUsageInfo,
) (coremodel.Usage, bool, error) {
	if info.UpstreamID == "" {
		return coremodel.Usage{}, false, errors.New("upstream id is empty")
	}

	response, err := fetchAliVideoTask(ctx, channel, info.BaseURL, info.UpstreamID)
	if err != nil {
		return coremodel.Usage{}, false, err
	}

	switch strings.ToUpper(response.Output.TaskStatus) {
	case "SUCCEEDED":
		return aliVideoUsageToModelUsage(response.Usage), true, nil
	case "PENDING", "RUNNING", "":
		return coremodel.Usage{}, false, nil
	case "FAILED", "CANCELED", "CANCELLED", "UNKNOWN":
		return coremodel.Usage{}, true, fmt.Errorf(
			"ali video task ended with status %q: %s",
			response.Output.TaskStatus,
			firstNonEmpty(response.Output.Message, response.Message),
		)
	default:
		return coremodel.Usage{}, false, nil
	}
}

func fetchAliVideoTask(
	ctx context.Context,
	channel *coremodel.Channel,
	baseURL string,
	taskID string,
) (*AliVideoTaskResponse, error) {
	if baseURL == "" && channel != nil {
		baseURL = channel.BaseURL
	}

	if baseURL == "" {
		baseURL = (&Adaptor{}).DefaultBaseURL()
	}

	taskURL, err := url.JoinPath(baseURL, "/api/v1/tasks", taskID)
	if err != nil {
		return nil, fmt.Errorf("build ali video task url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, taskURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new ali video task request: %w", err)
	}

	if channel != nil {
		req.Header.Set("Authorization", "Bearer "+channel.Key)
	}

	var (
		proxyURL      string
		skipTLSVerify bool
	)
	if channel != nil {
		proxyURL = channel.ProxyURL
		skipTLSVerify = channel.SkipTLSVerify
	}

	client, err := relayutils.LoadHTTPClientWithTLSConfigE(
		0,
		proxyURL,
		skipTLSVerify,
	)
	if err != nil {
		return nil, fmt.Errorf("load http client: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do ali video task request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response AliVideoTaskResponse
	if err := sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode ali video task: %w", err)
	}

	return &response, nil
}

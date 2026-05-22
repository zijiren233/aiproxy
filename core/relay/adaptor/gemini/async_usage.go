package gemini

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relayutils "github.com/labring/aiproxy/core/relay/utils"
)

var _ adaptor.AsyncUsageFetcher = (*Adaptor)(nil)

func (a *Adaptor) FetchAsyncUsage(
	ctx context.Context,
	request adaptor.AsyncUsageRequest,
) (model.Usage, model.UsageContext, bool, error) {
	info := request.Info
	if info == nil {
		return model.Usage{}, model.UsageContext{}, false, errors.New("async usage info is nil")
	}

	switch mode.Mode(info.Mode) {
	case mode.GeminiVideo, mode.VideoGenerationsJobs, mode.Videos:
	default:
		return model.Usage{}, model.UsageContext{}, false, fmt.Errorf(
			"unsupported async usage mode: %d",
			info.Mode,
		)
	}

	operation, err := a.fetchVideoOperation(ctx, request.Channel, info)
	if err != nil {
		return model.Usage{}, model.UsageContext{}, false, err
	}

	if !operation.Done {
		return model.Usage{}, model.UsageContext{}, false, nil
	}

	if operation.Error != nil {
		return model.Usage{}, model.UsageContext{}, true, fmt.Errorf(
			"gemini video operation failed: %s",
			operation.Error.Message,
		)
	}

	return info.Usage, info.UsageContext, true, nil
}

func (a *Adaptor) fetchVideoOperation(
	ctx context.Context,
	channel *model.Channel,
	info *model.AsyncUsageInfo,
) (*geminiOperation, error) {
	if info.UpstreamID == "" {
		return nil, errors.New("upstream id is empty")
	}

	requestMeta := meta.NewMeta(
		channel,
		mode.Mode(info.Mode),
		info.Model,
		model.ModelConfig{Model: info.Model},
	)
	if info.BaseURL != "" {
		requestMeta.Channel.BaseURL = info.BaseURL
	}

	requestURL, err := getOperationRequestURL(requestMeta, info.UpstreamID)
	if err != nil {
		return nil, fmt.Errorf("build gemini video operation url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.URL, nil)
	if err != nil {
		return nil, err
	}

	if channel != nil {
		req.Header.Set("X-Goog-Api-Key", channel.Key)
	}

	var (
		proxyURL      string
		skipTLSVerify bool
	)
	if channel != nil {
		proxyURL = channel.ProxyURL
		skipTLSVerify = channel.SkipTLSVerify
	}

	client, err := relayutils.LoadHTTPClientWithTLSConfigE(0, proxyURL, skipTLSVerify)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var operation geminiOperation
	if err := sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&operation); err != nil {
		return nil, fmt.Errorf("decode gemini video operation: %w", err)
	}

	return &operation, nil
}

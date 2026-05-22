package utils_test

import (
	"io"
	"testing"

	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/labring/aiproxy/core/relay/utils"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestGeminiVideo(t *testing.T) {
	body, relayMode, err := utils.BuildRequest(model.ModelConfig{
		Model: "veo-3.1-generate-preview",
		Type:  mode.GeminiVideo,
	})
	require.NoError(t, err)
	require.Equal(t, mode.GeminiVideo, relayMode)

	data, err := io.ReadAll(body)
	require.NoError(t, err)
	require.JSONEq(
		t,
		`{
			"instances":[{"prompt":"A calm cinematic shot of clouds moving over a mountain."}],
			"parameters":{"aspectRatio":"16:9","durationSeconds":5,"numberOfVideos":1}
		}`,
		string(data),
	)
}

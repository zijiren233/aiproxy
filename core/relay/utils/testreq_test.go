package utils_test

import (
	"io"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
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
			"instances":[{"prompt":"A calm cinematic shot of clouds moving over a mountain."}]
		}`,
		string(data),
	)
}

func TestBuildImagesGenerationsRequestUsesImagePrompt(t *testing.T) {
	body, relayMode, err := utils.BuildRequest(model.ModelConfig{
		Model: "gemini-3.1-flash-image-preview",
		Type:  mode.GeminiImage,
	})
	require.NoError(t, err)
	require.Equal(t, mode.ImagesGenerations, relayMode)

	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var request relaymodel.GeneralOpenAIRequest
	require.NoError(t, sonic.Unmarshal(data, &request))
	require.Equal(t, "gemini-3.1-flash-image-preview", request.Model)
	require.NotEqual(t, "hi", request.Prompt)
	require.Contains(t, request.Prompt, "red square")
	require.Equal(t, "1024x1024", request.Size)
}

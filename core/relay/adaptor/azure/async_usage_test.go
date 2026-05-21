package azure_test

import (
	"testing"

	"github.com/labring/aiproxy/core/relay/model"
	"github.com/stretchr/testify/require"
)

func TestVideoPriceSizeFromDimensionsNormalizesSize(t *testing.T) {
	t.Parallel()

	size := model.VideoPriceSizeFromDimensions(1280, 720)

	require.Equal(t, "720p", size)
}

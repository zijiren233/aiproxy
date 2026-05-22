//nolint:testpackage
package openai

import (
	"testing"

	"github.com/labring/aiproxy/core/relay/model"
	"github.com/stretchr/testify/require"
)

func TestCalculateVideoUsageNormalizesResolution(t *testing.T) {
	t.Parallel()

	_, usageContext := calculateVideoUsage(&model.VideoGenerationJob{
		NVariants: 1,
		NSeconds:  5,
		Width:     1280,
		Height:    720,
	})

	require.Equal(t, "720p", usageContext.PriceCondition.Resolution)
}

func TestCalculateVideoUsageNormalizesPortraitResolution(t *testing.T) {
	t.Parallel()

	_, usageContext := calculateVideoUsage(&model.VideoGenerationJob{
		NVariants: 1,
		NSeconds:  5,
		Width:     720,
		Height:    1280,
	})

	require.Equal(t, "720p", usageContext.PriceCondition.Resolution)
}

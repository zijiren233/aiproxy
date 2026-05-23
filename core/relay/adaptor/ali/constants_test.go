//nolint:testpackage
package ali

import (
	"testing"

	"github.com/labring/aiproxy/core/common/consume"
	"github.com/labring/aiproxy/core/model"
)

func aliModelPriceForTest(t *testing.T, modelName string) model.Price {
	t.Helper()

	for _, mc := range ModelList {
		if mc.Model == modelName {
			return mc.Price
		}
	}

	t.Fatalf("model %s not found", modelName)

	return model.Price{}
}

func TestAliImagePriceUsesSuccessfulImageCount(t *testing.T) {
	t.Parallel()

	amount := consume.CalculateAmountDetail(
		200,
		model.Usage{
			OutputTokens:      3,
			ImageOutputTokens: 3,
			TotalTokens:       3,
		},
		model.UsageContext{},
		aliModelPriceForTest(t, "qwen-image-plus"),
	)

	if amount.UsedAmount != 0.6 {
		t.Fatalf("expected 0.6 image amount, got %#v", amount)
	}
}

func TestAliVideoPriceUsesOutputSecondsAndResolution(t *testing.T) {
	t.Parallel()

	amount := consume.CalculateAmountDetail(
		200,
		model.Usage{
			OutputTokens: 5,
			TotalTokens:  5,
		},
		model.UsageContext{
			Resolution: "720p",
		},
		aliModelPriceForTest(t, "wan2.5-t2v-preview"),
	)

	if amount.UsedAmount != 3 {
		t.Fatalf("expected 3 video amount, got %#v", amount)
	}
}

func TestAliVideoPriceFallsBackToMostExpensiveResolution(t *testing.T) {
	t.Parallel()

	amount := consume.CalculateAmountDetail(
		200,
		model.Usage{
			OutputTokens: 5,
			TotalTokens:  5,
		},
		model.UsageContext{},
		aliModelPriceForTest(t, "wan2.5-t2v-preview"),
	)

	if amount.UsedAmount != 5 {
		t.Fatalf("expected 5 video amount, got %#v", amount)
	}
}

func TestAliVideoEditPriceUsesInputAndOutputSeconds(t *testing.T) {
	t.Parallel()

	amount := consume.CalculateAmountDetail(
		200,
		model.Usage{
			VideoInputTokens: 2,
			OutputTokens:     5,
			TotalTokens:      7,
		},
		model.UsageContext{
			Resolution: "1080p",
		},
		aliModelPriceForTest(t, "happyhorse-1.0-video-edit"),
	)

	if amount.UsedAmount != 11.2 {
		t.Fatalf("expected 11.2 video edit amount, got %#v", amount)
	}
}

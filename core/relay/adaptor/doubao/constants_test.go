//nolint:testpackage
package doubao

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common/consume"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/controller"
)

func doubaoModelPriceForTest(t *testing.T, modelName string) model.Price {
	t.Helper()

	for _, mc := range ModelList {
		if mc.Model == modelName {
			return mc.Price
		}
	}

	t.Fatalf("model %s not found", modelName)

	return model.Price{}
}

func TestDoubaoSeedancePriceUsesReturnedCompletionTokens(t *testing.T) {
	t.Parallel()

	amount := consume.CalculateAmountDetail(
		200,
		model.Usage{
			OutputTokens: 1000,
			TotalTokens:  1000,
		},
		model.UsageContext{
			Resolution: "1080p",
		},
		doubaoModelPriceForTest(t, "doubao-seedance-2-0"),
	)

	if amount.UsedAmount != 51 {
		t.Fatalf("expected 51 token amount, got %#v", amount)
	}
}

func TestDoubaoSeedancePriceFallsBackToMostExpensiveResolution(t *testing.T) {
	t.Parallel()

	amount := consume.CalculateAmountDetail(
		200,
		model.Usage{
			OutputTokens: 1000,
			TotalTokens:  1000,
		},
		model.UsageContext{},
		doubaoModelPriceForTest(t, "doubao-seedance-2-0"),
	)

	if amount.UsedAmount != 51 {
		t.Fatalf("expected 51 token amount, got %#v", amount)
	}
}

func TestDoubaoSeedanceConditionalPriceKeepsTokenUnitAfterVideoController(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/video/generations/jobs",
		bytes.NewBufferString(`{
			"model":"doubao-seedance-2-0",
			"prompt":"A city street",
			"n_seconds":5,
			"size":"1080p"
		}`),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	price, err := controller.GetVideoGenerationJobRequestPrice(ctx, model.ModelConfig{
		Price: doubaoModelPriceForTest(t, "doubao-seedance-2-0"),
	})
	if err != nil {
		t.Fatalf("GetVideoGenerationJobRequestPrice returned error: %v", err)
	}

	amount := consume.CalculateAmountDetail(
		http.StatusOK,
		model.Usage{
			OutputTokens: 1000,
			TotalTokens:  1000,
		},
		model.UsageContext{
			Resolution: "1080p",
		},
		price,
	)

	if amount.UsedAmount != 51 {
		t.Fatalf("expected 51 token amount after controller price projection, got %#v", amount)
	}
}

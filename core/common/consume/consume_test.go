package consume_test

import (
	"net/http"
	"testing"

	"github.com/labring/aiproxy/core/common/consume"
	"github.com/labring/aiproxy/core/model"
)

func TestCalculateAmount(t *testing.T) {
	tests := []struct {
		name  string
		code  int
		usage model.Usage
		price model.Price
		want  float64
	}{
		{
			name: "Per-Request Pricing (OK)",
			code: http.StatusOK,
			usage: model.Usage{
				InputTokens:  1000,
				OutputTokens: 500,
			},
			price: model.Price{
				PerRequestPrice: 2.5,
			},
			want: 2.5,
		},
		{
			name: "Per-Request Pricing (Non-OK)",
			code: http.StatusBadRequest,
			usage: model.Usage{
				InputTokens:  1000,
				OutputTokens: 500,
			},
			price: model.Price{
				PerRequestPrice: 2.5,
			},
			want: 0,
		},
		{
			name: "Simple Pricing",
			code: http.StatusOK,
			usage: model.Usage{
				InputTokens:  1000,
				OutputTokens: 2000,
			},
			price: model.Price{
				InputPrice:  0.001,
				OutputPrice: 0.002,
			},
			want: 0.005, // 0.001 * 1000/1000 + 0.002 * 2000/1000
		},
		{
			name: "Simple Pricing With Unit 1",
			code: http.StatusOK,
			usage: model.Usage{
				InputTokens:  1000,
				OutputTokens: 2000,
			},
			price: model.Price{
				InputPrice:      0.001,
				InputPriceUnit:  1,
				OutputPrice:     0.002,
				OutputPriceUnit: 2,
			},
			want: 3, // 0.001 * 1000/1 + 0.002 * 2000/2
		},
		{
			name: "Images Pricing",
			code: http.StatusOK,
			usage: model.Usage{
				InputTokens:      2000,
				ImageInputTokens: 1000,
				OutputTokens:     3000,
			},
			price: model.Price{
				InputPrice:      0.001,
				ImageInputPrice: 0.003,
				OutputPrice:     0.004,
			},
			want: 0.016, // 0.001 * (2000-1000)/1000 + 0.003 * 1000/1000 + 0.004 * 4000/1000
		},
		{
			name: "Cached Token Pricing",
			code: http.StatusOK,
			usage: model.Usage{
				InputTokens:         4000,
				CacheCreationTokens: 1000,
				CachedTokens:        2000,
			},
			price: model.Price{
				InputPrice:         0.01,
				CacheCreationPrice: 0.1,
				CachedPrice:        0.001,
			},
			want: 0.112, // 0.01 * (4000-1000-2000)/1000 + 0.1 * 1000/1000 + 0.001 * 2000/1000
		},
		{
			name: "Web Search Pricing",
			code: http.StatusOK,
			usage: model.Usage{
				WebSearchCount: 2,
			},
			price: model.Price{
				WebSearchPrice:     0.5,
				WebSearchPriceUnit: 1,
			},
			want: 1, // 0.5 * 2/1
		},
		{
			name: "Thinking Mode Output Pricing (ON)",
			code: http.StatusOK,
			usage: model.Usage{
				OutputTokens:    2000,
				ReasoningTokens: 1000,
			},
			price: model.Price{
				OutputPrice:             0.01,
				ThinkingModeOutputPrice: 0.03,
			},
			want: 0.06, // 0.03 * 2000/1000
		},
		{
			name: "Thinking Mode Output Pricing (OFF)",
			code: http.StatusOK,
			usage: model.Usage{
				OutputTokens: 2000,
			},
			price: model.Price{
				OutputPrice:             0.01,
				ThinkingModeOutputPrice: 0.03,
			},
			want: 0.02, // 0.01 * 2000/1000
		},
	}

	for _, tt := range tests {
		got := consume.CalculateAmount(tt.code, tt.usage, tt.price)
		if got != tt.want {
			t.Errorf("CalculateAmount()\n%s\n\tgot: %v\n\twant: %v\n\t", tt.name, got, tt.want)
		}
	}
}

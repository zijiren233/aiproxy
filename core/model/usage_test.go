package model_test

import (
	"testing"

	"github.com/labring/aiproxy/core/model"
)

func TestPrice_ValidateConditionalPrices(t *testing.T) {
	tests := []struct {
		name    string
		price   model.Price
		wantErr bool
	}{
		{
			name: "Empty conditional prices",
			price: model.Price{
				ConditionalPrices: []model.ConditionalPrice{},
			},
			wantErr: false,
		},
		{
			name: "Nil conditional prices",
			price: model.Price{
				ConditionalPrices: nil,
			},
			wantErr: false,
		},
		{
			name: "Valid single condition",
			price: model.Price{
				ConditionalPrices: []model.ConditionalPrice{
					{
						Condition: model.PriceCondition{
							InputTokenMin:  0,
							InputTokenMax:  32000,
							OutputTokenMin: 0,
							OutputTokenMax: 200,
						},
						Price: model.Price{
							InputPrice:  0.0008,
							OutputPrice: 0.002,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid multiple conditions - doubao-seed-1.6 example",
			price: model.Price{
				ConditionalPrices: []model.ConditionalPrice{
					{
						Condition: model.PriceCondition{
							InputTokenMin:  0,
							InputTokenMax:  32000,
							OutputTokenMin: 0,
							OutputTokenMax: 200,
						},
						Price: model.Price{
							InputPrice:  0.0008,
							OutputPrice: 0.002,
						},
					},
					{
						Condition: model.PriceCondition{
							InputTokenMin:  0,
							InputTokenMax:  32000,
							OutputTokenMin: 201,
							OutputTokenMax: 16000,
						},
						Price: model.Price{
							InputPrice:  0.0008,
							OutputPrice: 0.008,
						},
					},
					{
						Condition: model.PriceCondition{
							InputTokenMin: 32001,
							InputTokenMax: 128000,
						},
						Price: model.Price{
							InputPrice:  0.0012,
							OutputPrice: 0.016,
						},
					},
					{
						Condition: model.PriceCondition{
							InputTokenMin: 128001,
							InputTokenMax: 256000,
						},
						Price: model.Price{
							InputPrice:  0.0024,
							OutputPrice: 0.024,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid input token range - min > max",
			price: model.Price{
				ConditionalPrices: []model.ConditionalPrice{
					{
						Condition: model.PriceCondition{
							InputTokenMin: 32000,
							InputTokenMax: 1000, // min > max
						},
						Price: model.Price{
							InputPrice:  0.0008,
							OutputPrice: 0.002,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid output token range - min > max",
			price: model.Price{
				ConditionalPrices: []model.ConditionalPrice{
					{
						Condition: model.PriceCondition{
							InputTokenMin:  0,
							InputTokenMax:  32000,
							OutputTokenMin: 1000,
							OutputTokenMax: 500, // min > max
						},
						Price: model.Price{
							InputPrice:  0.0008,
							OutputPrice: 0.002,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Overlapping input ranges with overlapping output ranges",
			price: model.Price{
				ConditionalPrices: []model.ConditionalPrice{
					{
						Condition: model.PriceCondition{
							InputTokenMin:  0,
							InputTokenMax:  32000,
							OutputTokenMin: 0,
							OutputTokenMax: 500,
						},
						Price: model.Price{
							InputPrice:  0.0008,
							OutputPrice: 0.002,
						},
					},
					{
						Condition: model.PriceCondition{
							InputTokenMin:  20000, // overlaps with previous
							InputTokenMax:  50000,
							OutputTokenMin: 200, // overlaps with previous
							OutputTokenMax: 1000,
						},
						Price: model.Price{
							InputPrice:  0.0012,
							OutputPrice: 0.008,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Overlapping input ranges but non-overlapping output ranges (valid)",
			price: model.Price{
				ConditionalPrices: []model.ConditionalPrice{
					{
						Condition: model.PriceCondition{
							InputTokenMin:  0,
							InputTokenMax:  32000,
							OutputTokenMin: 0,
							OutputTokenMax: 200,
						},
						Price: model.Price{
							InputPrice:  0.0008,
							OutputPrice: 0.002,
						},
					},
					{
						Condition: model.PriceCondition{
							InputTokenMin:  0,     // same input range
							InputTokenMax:  32000, // same input range
							OutputTokenMin: 201,   // non-overlapping output range
							OutputTokenMax: 16000,
						},
						Price: model.Price{
							InputPrice:  0.0008,
							OutputPrice: 0.008,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Improperly ordered conditions",
			price: model.Price{
				ConditionalPrices: []model.ConditionalPrice{
					{
						Condition: model.PriceCondition{
							InputTokenMin: 32001,
							InputTokenMax: 128000,
						},
						Price: model.Price{
							InputPrice:  0.0012,
							OutputPrice: 0.016,
						},
					},
					{
						Condition: model.PriceCondition{
							InputTokenMin: 0,
							InputTokenMax: 32000, // should come before the previous one
						},
						Price: model.Price{
							InputPrice:  0.0008,
							OutputPrice: 0.002,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Valid consecutive ranges",
			price: model.Price{
				ConditionalPrices: []model.ConditionalPrice{
					{
						Condition: model.PriceCondition{
							InputTokenMin: 0,
							InputTokenMax: 32000,
						},
						Price: model.Price{
							InputPrice:  0.0008,
							OutputPrice: 0.002,
						},
					},
					{
						Condition: model.PriceCondition{
							InputTokenMin: 32001, // consecutive with previous
							InputTokenMax: 128000,
						},
						Price: model.Price{
							InputPrice:  0.0012,
							OutputPrice: 0.016,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Gap between ranges (valid)",
			price: model.Price{
				ConditionalPrices: []model.ConditionalPrice{
					{
						Condition: model.PriceCondition{
							InputTokenMin: 0,
							InputTokenMax: 32000,
						},
						Price: model.Price{
							InputPrice:  0.0008,
							OutputPrice: 0.002,
						},
					},
					{
						Condition: model.PriceCondition{
							InputTokenMin: 50000, // gap between 32000 and 50000
							InputTokenMax: 128000,
						},
						Price: model.Price{
							InputPrice:  0.0012,
							OutputPrice: 0.016,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Unbounded ranges (zero values)",
			price: model.Price{
				ConditionalPrices: []model.ConditionalPrice{
					{
						Condition: model.PriceCondition{
							InputTokenMin: 0, // unbounded min
							InputTokenMax: 0, // unbounded max
						},
						Price: model.Price{
							InputPrice:  0.001,
							OutputPrice: 0.002,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Mixed bounded and unbounded ranges",
			price: model.Price{
				ConditionalPrices: []model.ConditionalPrice{
					{
						Condition: model.PriceCondition{
							InputTokenMin: 0,
							InputTokenMax: 32000,
						},
						Price: model.Price{
							InputPrice:  0.0008,
							OutputPrice: 0.002,
						},
					},
					{
						Condition: model.PriceCondition{
							InputTokenMin: 32001,
							InputTokenMax: 0, // unbounded max
						},
						Price: model.Price{
							InputPrice:  0.0012,
							OutputPrice: 0.016,
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.price.ValidateConditionalPrices()

			if tt.wantErr {
				if err == nil {
					t.Errorf("%s: ValidateConditionalPrices() expected error but got nil", tt.name)
				}
				return
			}
			if err != nil {
				t.Errorf("%s: ValidateConditionalPrices() unexpected error = %v", tt.name, err)
			}
		})
	}
}

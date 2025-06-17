package model

import (
	"fmt"
	"math"
)

type PriceCondition struct {
	InputTokenMin  int64 `json:"input_token_min,omitempty"`
	InputTokenMax  int64 `json:"input_token_max,omitempty"`
	OutputTokenMin int64 `json:"output_token_min,omitempty"`
	OutputTokenMax int64 `json:"output_token_max,omitempty"`
}

type ConditionalPrice struct {
	Condition PriceCondition `json:"condition"`
	Price     Price          `json:"price"`
}

type Price struct {
	PerRequestPrice ZeroNullFloat64 `json:"per_request_price,omitempty"`

	InputPrice     ZeroNullFloat64 `json:"input_price,omitempty"`
	InputPriceUnit ZeroNullInt64   `json:"input_price_unit,omitempty"`

	ImageInputPrice     ZeroNullFloat64 `json:"image_input_price,omitempty"`
	ImageInputPriceUnit ZeroNullInt64   `json:"image_input_price_unit,omitempty"`

	OutputPrice     ZeroNullFloat64 `json:"output_price,omitempty"`
	OutputPriceUnit ZeroNullInt64   `json:"output_price_unit,omitempty"`

	// when ThinkingModeOutputPrice and ReasoningTokens are not 0, OutputPrice and OutputPriceUnit
	// will be overwritten
	ThinkingModeOutputPrice     ZeroNullFloat64 `json:"thinking_mode_output_price,omitempty"`
	ThinkingModeOutputPriceUnit ZeroNullInt64   `json:"thinking_mode_output_price_unit,omitempty"`

	CachedPrice     ZeroNullFloat64 `json:"cached_price,omitempty"`
	CachedPriceUnit ZeroNullInt64   `json:"cached_price_unit,omitempty"`

	CacheCreationPrice     ZeroNullFloat64 `json:"cache_creation_price,omitempty"`
	CacheCreationPriceUnit ZeroNullInt64   `json:"cache_creation_price_unit,omitempty"`

	WebSearchPrice     ZeroNullFloat64 `json:"web_search_price,omitempty"`
	WebSearchPriceUnit ZeroNullInt64   `json:"web_search_price_unit,omitempty"`

	ConditionalPrices []ConditionalPrice `gorm:"serializer:fastjson;type:text" json:"conditional_prices,omitempty"`
}

func (p *Price) ValidateConditionalPrices() error {
	if len(p.ConditionalPrices) == 0 {
		return nil
	}

	for i, conditionalPrice := range p.ConditionalPrices {
		condition := conditionalPrice.Condition

		// Validate individual condition ranges
		if condition.InputTokenMin > 0 && condition.InputTokenMax > 0 {
			if condition.InputTokenMin > condition.InputTokenMax {
				return fmt.Errorf(
					"conditional price %d: input token min (%d) cannot be greater than max (%d)",
					i,
					condition.InputTokenMin,
					condition.InputTokenMax,
				)
			}
		}

		if condition.OutputTokenMin > 0 && condition.OutputTokenMax > 0 {
			if condition.OutputTokenMin > condition.OutputTokenMax {
				return fmt.Errorf(
					"conditional price %d: output token min (%d) cannot be greater than max (%d)",
					i,
					condition.OutputTokenMin,
					condition.OutputTokenMax,
				)
			}
		}

		// Check for overlaps with other conditions
		for j := i + 1; j < len(p.ConditionalPrices); j++ {
			otherCondition := p.ConditionalPrices[j].Condition

			// Check input token range overlap
			if hasRangeOverlap(
				condition.InputTokenMin, condition.InputTokenMax,
				otherCondition.InputTokenMin, otherCondition.InputTokenMax,
			) {
				// If input ranges overlap, check if output ranges also overlap
				if hasRangeOverlap(
					condition.OutputTokenMin, condition.OutputTokenMax,
					otherCondition.OutputTokenMin, otherCondition.OutputTokenMax,
				) {
					return fmt.Errorf(
						"conditional prices %d and %d have overlapping conditions",
						i,
						j,
					)
				}
			}
		}
	}

	// Check if conditions are sorted by input token ranges (optional ordering check)
	if err := p.validateConditionalPriceOrdering(); err != nil {
		return err
	}

	return nil
}

// hasRangeOverlap checks if two ranges overlap
// Range is defined by [min, max], where 0 means unbounded
func hasRangeOverlap(min1, max1, min2, max2 int64) bool {
	// Convert 0 to appropriate bounds for comparison
	actualMin1 := min1
	actualMax1 := max1
	actualMin2 := min2
	actualMax2 := max2

	if actualMin1 == 0 {
		actualMin1 = 0
	}
	if actualMax1 == 0 {
		actualMax1 = math.MaxInt64
	}
	if actualMin2 == 0 {
		actualMin2 = 0
	}
	if actualMax2 == 0 {
		actualMax2 = math.MaxInt64
	}

	// Check if ranges overlap: range1.max >= range2.min && range1.min <= range2.max
	return actualMax1 >= actualMin2 && actualMin1 <= actualMax2
}

// validateConditionalPriceOrdering checks if conditional prices are properly ordered
func (p *Price) validateConditionalPriceOrdering() error {
	if len(p.ConditionalPrices) <= 1 {
		return nil
	}

	for i := 0; i < len(p.ConditionalPrices)-1; i++ {
		current := p.ConditionalPrices[i].Condition
		next := p.ConditionalPrices[i+1].Condition

		// Check if input token ranges are in ascending order
		// Compare the starting points of ranges
		currentInputMin := current.InputTokenMin
		nextInputMin := next.InputTokenMin

		// If current range starts after next range, it's improperly ordered
		if currentInputMin > nextInputMin {
			return fmt.Errorf("conditional prices %d and %d are not properly ordered: "+
				"current min (%d) should not be greater than next min (%d)",
				i, i+1, currentInputMin, nextInputMin)
		}

		// If they have the same starting point, check the ending points
		if currentInputMin == nextInputMin {
			currentInputMax := current.InputTokenMax
			nextInputMax := next.InputTokenMax

			// Handle unbounded ranges (0 means unbounded)
			if currentInputMax == 0 {
				currentInputMax = math.MaxInt64
			}
			if nextInputMax == 0 {
				nextInputMax = math.MaxInt64
			}

			if currentInputMax > nextInputMax {
				return fmt.Errorf("conditional prices %d and %d are not properly ordered: "+
					"ranges with same min should be ordered by max",
					i, i+1)
			}
		}
	}

	return nil
}

func (p *Price) SelectConditionalPrice(usage Usage) Price {
	if len(p.ConditionalPrices) == 0 {
		return *p
	}

	inputTokens := int64(usage.InputTokens)
	outputTokens := int64(usage.OutputTokens)

	for _, conditionalPrice := range p.ConditionalPrices {
		condition := conditionalPrice.Condition

		if condition.InputTokenMin > 0 && inputTokens < condition.InputTokenMin {
			continue
		}
		if condition.InputTokenMax > 0 && inputTokens > condition.InputTokenMax {
			continue
		}

		if condition.OutputTokenMin > 0 && outputTokens < condition.OutputTokenMin {
			continue
		}
		if condition.OutputTokenMax > 0 && outputTokens > condition.OutputTokenMax {
			continue
		}

		return conditionalPrice.Price
	}

	return *p
}

func (p *Price) GetInputPriceUnit() int64 {
	if p.InputPriceUnit > 0 {
		return int64(p.InputPriceUnit)
	}
	return PriceUnit
}

func (p *Price) GetImageInputPriceUnit() int64 {
	if p.ImageInputPriceUnit > 0 {
		return int64(p.ImageInputPriceUnit)
	}
	return PriceUnit
}

func (p *Price) GetOutputPriceUnit() int64 {
	if p.OutputPriceUnit > 0 {
		return int64(p.OutputPriceUnit)
	}
	return PriceUnit
}

func (p *Price) GetCachedPriceUnit() int64 {
	if p.CachedPriceUnit > 0 {
		return int64(p.CachedPriceUnit)
	}
	return PriceUnit
}

func (p *Price) GetCacheCreationPriceUnit() int64 {
	if p.CacheCreationPriceUnit > 0 {
		return int64(p.CacheCreationPriceUnit)
	}
	return PriceUnit
}

func (p *Price) GetWebSearchPriceUnit() int64 {
	if p.WebSearchPriceUnit > 0 {
		return int64(p.WebSearchPriceUnit)
	}
	return PriceUnit
}

type Usage struct {
	InputTokens         ZeroNullInt64 `json:"input_tokens,omitempty"`
	ImageInputTokens    ZeroNullInt64 `json:"image_input_tokens,omitempty"`
	OutputTokens        ZeroNullInt64 `json:"output_tokens,omitempty"`
	CachedTokens        ZeroNullInt64 `json:"cached_tokens,omitempty"`
	CacheCreationTokens ZeroNullInt64 `json:"cache_creation_tokens,omitempty"`
	ReasoningTokens     ZeroNullInt64 `json:"reasoning_tokens,omitempty"`
	TotalTokens         ZeroNullInt64 `json:"total_tokens,omitempty"`
	WebSearchCount      ZeroNullInt64 `json:"web_search_count,omitempty"`
}

func (u *Usage) Add(other Usage) {
	u.InputTokens += other.InputTokens
	u.ImageInputTokens += other.ImageInputTokens
	u.OutputTokens += other.OutputTokens
	u.CachedTokens += other.CachedTokens
	u.CacheCreationTokens += other.CacheCreationTokens
	u.TotalTokens += other.TotalTokens
	u.WebSearchCount += other.WebSearchCount
}

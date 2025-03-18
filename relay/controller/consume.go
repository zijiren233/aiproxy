package controller

import (
	"github.com/labring/aiproxy/model"
	"github.com/shopspring/decimal"
)

type PreCheckGroupBalanceReq struct {
	InputTokens        int
	MaxTokens          int
	InputPrice         float64
	OutputPrice        float64
	CachedPrice        float64
	CacheCreationPrice float64
}

func getPreConsumedAmount(req *PreCheckGroupBalanceReq) float64 {
	if req == nil || req.InputPrice == 0 || (req.InputTokens == 0 && req.MaxTokens == 0) {
		return 0
	}
	preConsumedTokens := int64(req.InputTokens)
	if req.MaxTokens != 0 {
		preConsumedTokens += int64(req.MaxTokens)
	}
	return decimal.
		NewFromInt(preConsumedTokens).
		Mul(decimal.NewFromFloat(req.InputPrice)).
		Div(decimal.NewFromInt(model.PriceUnit)).
		InexactFloat64()
}

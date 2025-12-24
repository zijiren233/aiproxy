package adaptor

import (
	"testing"

	"github.com/labring/aiproxy/core/model"
	"github.com/stretchr/testify/assert"
)

func TestSyncUsage(t *testing.T) {
	t.Run("NewSyncUsage creates correct instance", func(t *testing.T) {
		usage := model.Usage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		}

		syncUsage := NewSyncUsage(usage)

		assert.Equal(t, usage, syncUsage.Usage())
		assert.False(t, syncUsage.IsAsync())
		assert.Nil(t, syncUsage.AsyncInfo())
	})

	t.Run("SyncUsage with empty usage", func(t *testing.T) {
		syncUsage := NewSyncUsage(model.Usage{})

		assert.Equal(t, model.Usage{}, syncUsage.Usage())
		assert.False(t, syncUsage.IsAsync())
		assert.Nil(t, syncUsage.AsyncInfo())
	})

	t.Run("SyncUsage with complex usage", func(t *testing.T) {
		usage := model.Usage{
			InputTokens:         1000,
			OutputTokens:        500,
			TotalTokens:         1500,
			CachedTokens:        200,
			CacheCreationTokens: 50,
			ReasoningTokens:     300,
			ImageInputTokens:    10,
			AudioInputTokens:    5,
			ImageOutputTokens:   3,
			WebSearchCount:      2,
		}

		syncUsage := NewSyncUsage(usage)

		result := syncUsage.Usage()
		assert.Equal(t, model.ZeroNullInt64(1000), result.InputTokens)
		assert.Equal(t, model.ZeroNullInt64(500), result.OutputTokens)
		assert.Equal(t, model.ZeroNullInt64(1500), result.TotalTokens)
		assert.Equal(t, model.ZeroNullInt64(200), result.CachedTokens)
		assert.Equal(t, model.ZeroNullInt64(50), result.CacheCreationTokens)
		assert.Equal(t, model.ZeroNullInt64(300), result.ReasoningTokens)
		assert.Equal(t, model.ZeroNullInt64(10), result.ImageInputTokens)
		assert.Equal(t, model.ZeroNullInt64(5), result.AudioInputTokens)
		assert.Equal(t, model.ZeroNullInt64(3), result.ImageOutputTokens)
		assert.Equal(t, model.ZeroNullInt64(2), result.WebSearchCount)
	})
}

func TestAsyncUsage(t *testing.T) {
	t.Run("NewAsyncUsage creates correct instance", func(t *testing.T) {
		info := &model.AsyncUsageInfo{
			Mode:      1,
			Model:     "gpt-4",
			ChannelID: 10,
			GroupID:   "group-1",
			TokenID:   5,
			Data:      `{"job_id":"job-123"}`,
		}

		asyncUsage := NewAsyncUsage(info)

		assert.True(t, asyncUsage.IsAsync())
		assert.Equal(t, model.Usage{}, asyncUsage.Usage()) // Should return empty usage
		assert.NotNil(t, asyncUsage.AsyncInfo())
		assert.Equal(t, info, asyncUsage.AsyncInfo())
	})

	t.Run("AsyncUsage returns empty usage", func(t *testing.T) {
		info := &model.AsyncUsageInfo{
			Mode:      2,
			Model:     "video-model",
			ChannelID: 1,
			GroupID:   "g1",
			TokenID:   1,
		}

		asyncUsage := NewAsyncUsage(info)
		usage := asyncUsage.Usage()

		// Async usage should always return empty/zero usage
		assert.Equal(t, model.ZeroNullInt64(0), usage.InputTokens)
		assert.Equal(t, model.ZeroNullInt64(0), usage.OutputTokens)
		assert.Equal(t, model.ZeroNullInt64(0), usage.TotalTokens)
	})

	t.Run("AsyncUsage with nil info", func(t *testing.T) {
		asyncUsage := NewAsyncUsage(nil)

		assert.True(t, asyncUsage.IsAsync())
		assert.Equal(t, model.Usage{}, asyncUsage.Usage())
		assert.Nil(t, asyncUsage.AsyncInfo())
	})

	t.Run("AsyncInfo contains all fields", func(t *testing.T) {
		info := &model.AsyncUsageInfo{
			Mode:      3,
			Model:     "sora-turbo",
			ChannelID: 100,
			GroupID:   "enterprise-group",
			TokenID:   42,
			TokenName: "api-token",
			Data:      `{"job_id":"video-job-456","format":"mp4"}`,
			Price: model.Price{
				InputPrice:  0.01,
				OutputPrice: 0.02,
			},
		}

		asyncUsage := NewAsyncUsage(info)
		result := asyncUsage.AsyncInfo()

		assert.Equal(t, 3, result.Mode)
		assert.Equal(t, "sora-turbo", result.Model)
		assert.Equal(t, 100, result.ChannelID)
		assert.Equal(t, "enterprise-group", result.GroupID)
		assert.Equal(t, 42, result.TokenID)
		assert.Equal(t, "api-token", result.TokenName)
		assert.Contains(t, result.Data, "video-job-456")
		assert.Equal(t, model.ZeroNullFloat64(0.01), result.Price.InputPrice)
		assert.Equal(t, model.ZeroNullFloat64(0.02), result.Price.OutputPrice)
	})
}

func TestUsageResultInterface(t *testing.T) {
	t.Run("SyncUsage implements UsageResult", func(t *testing.T) {
		var _ UsageResult = NewSyncUsage(model.Usage{})
	})

	t.Run("AsyncUsage implements UsageResult", func(t *testing.T) {
		var _ UsageResult = NewAsyncUsage(nil)
	})

	t.Run("polymorphic usage handling", func(t *testing.T) {
		// Test that both types can be used through the interface
		usageResults := []UsageResult{
			NewSyncUsage(model.Usage{InputTokens: 100}),
			NewAsyncUsage(&model.AsyncUsageInfo{Model: "async-model"}),
			NewSyncUsage(model.Usage{OutputTokens: 50}),
		}

		syncCount := 0
		asyncCount := 0

		for _, result := range usageResults {
			if result.IsAsync() {
				asyncCount++

				assert.NotNil(t, result.AsyncInfo())
			} else {
				syncCount++

				assert.Nil(t, result.AsyncInfo())
			}
		}

		assert.Equal(t, 2, syncCount)
		assert.Equal(t, 1, asyncCount)
	})

	t.Run("processing sync and async usage", func(t *testing.T) {
		// Simulate the pattern used in relay controller
		processUsageResult := func(result UsageResult) (model.Usage, *model.AsyncUsageInfo) {
			if result.IsAsync() {
				return model.Usage{}, result.AsyncInfo()
			}
			return result.Usage(), nil
		}

		// Test with sync usage
		syncResult := NewSyncUsage(model.Usage{InputTokens: 500})
		usage, asyncInfo := processUsageResult(syncResult)
		assert.Equal(t, model.ZeroNullInt64(500), usage.InputTokens)
		assert.Nil(t, asyncInfo)

		// Test with async usage
		asyncResult := NewAsyncUsage(&model.AsyncUsageInfo{
			Model: "video-gen",
			Data:  `{"job_id":"123"}`,
		})
		usage, asyncInfo = processUsageResult(asyncResult)
		assert.Equal(t, model.ZeroNullInt64(0), usage.InputTokens)
		assert.NotNil(t, asyncInfo)
		assert.Equal(t, "video-gen", asyncInfo.Model)
	})
}

func TestUsageResultNilSafety(t *testing.T) {
	t.Run("nil UsageResult handling pattern", func(t *testing.T) {
		// Test the typical nil check pattern
		var result UsageResult

		// When result is nil, we should handle it gracefully
		assert.Nil(t, result)

		// This is how code should check for nil
		if result != nil && result.IsAsync() {
			t.Error("should not reach here")
		}
	})

	t.Run("AsyncUsage with nil AsyncInfo is still async", func(t *testing.T) {
		asyncUsage := NewAsyncUsage(nil)

		// Even with nil info, IsAsync should return true
		// This maintains the semantic meaning of the type
		assert.True(t, asyncUsage.IsAsync())
		assert.Nil(t, asyncUsage.AsyncInfo())
	})
}

func TestSyncUsageValueSemantics(t *testing.T) {
	t.Run("SyncUsage is a value type", func(t *testing.T) {
		usage := model.Usage{InputTokens: 100}
		syncUsage1 := NewSyncUsage(usage)

		// Modify the original usage
		usage.InputTokens = 200

		// SyncUsage should retain the original value (copy semantics)
		assert.Equal(t, model.ZeroNullInt64(100), syncUsage1.Usage().InputTokens)
	})

	t.Run("Multiple SyncUsage from same source are independent", func(t *testing.T) {
		usage := model.Usage{InputTokens: 100}
		syncUsage1 := NewSyncUsage(usage)
		syncUsage2 := NewSyncUsage(usage)

		// Both should have the same value
		assert.Equal(t, syncUsage1.Usage(), syncUsage2.Usage())
	})
}

func TestAsyncUsageReferenceSemantics(t *testing.T) {
	t.Run("AsyncUsage holds reference to AsyncUsageInfo", func(t *testing.T) {
		info := &model.AsyncUsageInfo{
			Model:     "original-model",
			ChannelID: 1,
		}
		asyncUsage := NewAsyncUsage(info)

		// Modify the original info
		info.Model = "modified-model"
		info.ChannelID = 2

		// AsyncUsage should reflect the changes (reference semantics)
		assert.Equal(t, "modified-model", asyncUsage.AsyncInfo().Model)
		assert.Equal(t, 2, asyncUsage.AsyncInfo().ChannelID)
	})

	t.Run("Multiple AsyncUsage share same AsyncUsageInfo", func(t *testing.T) {
		info := &model.AsyncUsageInfo{Model: "shared-model"}
		asyncUsage1 := NewAsyncUsage(info)
		asyncUsage2 := NewAsyncUsage(info)

		// Both should point to the same info
		assert.Same(t, asyncUsage1.AsyncInfo(), asyncUsage2.AsyncInfo())
	})
}

package model

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) func() {
	t.Helper()

	// Use shared cache mode for in-memory SQLite so all goroutines share the same database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// Limit to 1 connection to ensure all operations use the same in-memory database
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	// Run migrations
	err = db.AutoMigrate(
		&AsyncUsageInfo{},
		&Log{},
		&Summary{},
		&GroupSummary{},
	)
	require.NoError(t, err)

	// Save original LogDB and restore it after test
	originalLogDB := LogDB
	LogDB = db

	// Clear any pending batch data from previous tests
	ResetBatchData()

	return func() {
		LogDB = originalLogDB

		ResetBatchData() // Clear batch data on cleanup too

		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
}

func TestAsyncUsageInfo_Constants(t *testing.T) {
	assert.Equal(t, "pending", AsyncUsageStatusPending)
	assert.Equal(t, "completed", AsyncUsageStatusCompleted)
	assert.Equal(t, "failed", AsyncUsageStatusFailed)
}

func TestCreateAsyncUsageInfo(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name    string
		info    *AsyncUsageInfo
		wantErr bool
	}{
		{
			name: "create basic async usage info",
			info: &AsyncUsageInfo{
				RequestID: "req-123",
				RequestAt: time.Now(),
				Mode:      1,
				Model:     "gpt-4",
				ChannelID: 1,
				GroupID:   "group-1",
				TokenID:   1,
				TokenName: "token-1",
				Price: Price{
					InputPrice:  0.001,
					OutputPrice: 0.002,
				},
				Data: `{"job_id":"job-123"}`,
			},
			wantErr: false,
		},
		{
			name: "create with empty data",
			info: &AsyncUsageInfo{
				RequestID: "req-456",
				RequestAt: time.Now(),
				Mode:      2,
				Model:     "gpt-3.5-turbo",
				ChannelID: 2,
				GroupID:   "group-2",
				TokenID:   2,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateAsyncUsageInfo(tt.info)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotZero(t, tt.info.ID)
			assert.Equal(t, AsyncUsageStatusPending, tt.info.Status)
			assert.False(t, tt.info.CreatedAt.IsZero())
			assert.False(t, tt.info.UpdatedAt.IsZero())
		})
	}
}

func TestGetPendingAsyncUsages(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create test data
	now := time.Now()
	infos := []*AsyncUsageInfo{
		{
			RequestID: "req-1",
			RequestAt: now,
			Mode:      1,
			Model:     "gpt-4",
			ChannelID: 1,
			GroupID:   "group-1",
			TokenID:   1,
		},
		{
			RequestID: "req-2",
			RequestAt: now.Add(time.Second),
			Mode:      1,
			Model:     "gpt-4",
			ChannelID: 1,
			GroupID:   "group-1",
			TokenID:   1,
		},
		{
			RequestID: "req-3",
			RequestAt: now.Add(2 * time.Second),
			Mode:      2,
			Model:     "gpt-3.5",
			ChannelID: 2,
			GroupID:   "group-2",
			TokenID:   2,
		},
	}

	for _, info := range infos {
		err := CreateAsyncUsageInfo(info)
		require.NoError(t, err)
	}

	// Mark one as completed
	infos[1].Status = AsyncUsageStatusCompleted
	err := UpdateAsyncUsageInfo(infos[1])
	require.NoError(t, err)

	t.Run("get all pending", func(t *testing.T) {
		results, err := GetPendingAsyncUsages(10)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("get with limit", func(t *testing.T) {
		results, err := GetPendingAsyncUsages(1)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "req-1", results[0].RequestID) // Oldest first (ASC order)
	})

	t.Run("empty result when no pending", func(t *testing.T) {
		// Mark all as completed
		for _, info := range infos {
			info.Status = AsyncUsageStatusCompleted
			_ = UpdateAsyncUsageInfo(info)
		}

		results, err := GetPendingAsyncUsages(10)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestGetPendingAsyncUsagesByMode(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create test data with different modes
	now := time.Now()
	infos := []*AsyncUsageInfo{
		{
			RequestID: "req-mode1-1",
			RequestAt: now,
			Mode:      1,
			Model:     "m1",
			ChannelID: 1,
			GroupID:   "g1",
			TokenID:   1,
		},
		{
			RequestID: "req-mode1-2",
			RequestAt: now,
			Mode:      1,
			Model:     "m1",
			ChannelID: 1,
			GroupID:   "g1",
			TokenID:   1,
		},
		{
			RequestID: "req-mode2-1",
			RequestAt: now,
			Mode:      2,
			Model:     "m2",
			ChannelID: 2,
			GroupID:   "g2",
			TokenID:   2,
		},
		{
			RequestID: "req-mode3-1",
			RequestAt: now,
			Mode:      3,
			Model:     "m3",
			ChannelID: 3,
			GroupID:   "g3",
			TokenID:   3,
		},
	}

	for _, info := range infos {
		err := CreateAsyncUsageInfo(info)
		require.NoError(t, err)
	}

	t.Run("get mode 1", func(t *testing.T) {
		results, err := GetPendingAsyncUsagesByMode(1, 10)
		require.NoError(t, err)
		assert.Len(t, results, 2)

		for _, r := range results {
			assert.Equal(t, 1, r.Mode)
		}
	})

	t.Run("get mode 2", func(t *testing.T) {
		results, err := GetPendingAsyncUsagesByMode(2, 10)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "req-mode2-1", results[0].RequestID)
	})

	t.Run("get non-existent mode", func(t *testing.T) {
		results, err := GetPendingAsyncUsagesByMode(999, 10)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestUpdateAsyncUsageInfo(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create initial record
	info := &AsyncUsageInfo{
		RequestID: "req-update",
		RequestAt: time.Now(),
		Mode:      1,
		Model:     "gpt-4",
		ChannelID: 1,
		GroupID:   "group-1",
		TokenID:   1,
	}

	err := CreateAsyncUsageInfo(info)
	require.NoError(t, err)

	originalUpdatedAt := info.UpdatedAt

	// Wait a bit to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	// Update the record
	info.Status = AsyncUsageStatusCompleted
	info.Usage = Usage{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}
	info.UsedAmount = 0.5
	info.RetryCount = 2

	err = UpdateAsyncUsageInfo(info)
	require.NoError(t, err)

	// Verify update
	assert.True(t, info.UpdatedAt.After(originalUpdatedAt))

	// Fetch and verify
	var fetched AsyncUsageInfo

	err = LogDB.First(&fetched, info.ID).Error
	require.NoError(t, err)

	assert.Equal(t, AsyncUsageStatusCompleted, fetched.Status)
	assert.Equal(t, ZeroNullInt64(100), fetched.Usage.InputTokens)
	assert.Equal(t, ZeroNullInt64(50), fetched.Usage.OutputTokens)
	assert.Equal(t, 0.5, fetched.UsedAmount)
	assert.Equal(t, 2, fetched.RetryCount)
}

func TestDeleteAsyncUsageInfo(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create record
	info := &AsyncUsageInfo{
		RequestID: "req-delete",
		RequestAt: time.Now(),
		Mode:      1,
		Model:     "gpt-4",
		ChannelID: 1,
		GroupID:   "group-1",
		TokenID:   1,
	}

	err := CreateAsyncUsageInfo(info)
	require.NoError(t, err)

	id := info.ID

	// Verify exists
	var count int64
	LogDB.Model(&AsyncUsageInfo{}).Where("id = ?", id).Count(&count)
	assert.Equal(t, int64(1), count)

	// Delete
	err = DeleteAsyncUsageInfo(id)
	require.NoError(t, err)

	// Verify deleted
	LogDB.Model(&AsyncUsageInfo{}).Where("id = ?", id).Count(&count)
	assert.Equal(t, int64(0), count)

	// Delete non-existent should not error
	err = DeleteAsyncUsageInfo(99999)
	assert.NoError(t, err)
}

func TestCleanupCompletedAsyncUsages(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now()

	// Create test data with different statuses and times
	infos := []*AsyncUsageInfo{
		// Old completed - should be deleted
		{
			RequestID: "req-old-completed",
			RequestAt: now.Add(-48 * time.Hour),
			Mode:      1,
			Model:     "m1",
			ChannelID: 1,
			GroupID:   "g1",
			TokenID:   1,
			Status:    AsyncUsageStatusCompleted,
		},
		// Recent completed - should NOT be deleted
		{
			RequestID: "req-new-completed",
			RequestAt: now,
			Mode:      1,
			Model:     "m1",
			ChannelID: 1,
			GroupID:   "g1",
			TokenID:   1,
			Status:    AsyncUsageStatusCompleted,
		},
		// Old pending - should NOT be deleted (different status)
		{
			RequestID: "req-old-pending",
			RequestAt: now.Add(-48 * time.Hour),
			Mode:      1,
			Model:     "m1",
			ChannelID: 1,
			GroupID:   "g1",
			TokenID:   1,
			Status:    AsyncUsageStatusPending,
		},
		// Old failed - should NOT be deleted (different status)
		{
			RequestID: "req-old-failed",
			RequestAt: now.Add(-48 * time.Hour),
			Mode:      1,
			Model:     "m1",
			ChannelID: 1,
			GroupID:   "g1",
			TokenID:   1,
			Status:    AsyncUsageStatusFailed,
		},
	}

	for _, info := range infos {
		// Directly insert to control status and timestamps
		info.CreatedAt = info.RequestAt
		info.UpdatedAt = info.RequestAt
		err := LogDB.Create(info).Error
		require.NoError(t, err)
	}

	// Cleanup completed records older than 24 hours
	err := CleanupCompletedAsyncUsages(24 * time.Hour)
	require.NoError(t, err)

	// Verify results
	var remaining []*AsyncUsageInfo

	err = LogDB.Find(&remaining).Error
	require.NoError(t, err)

	assert.Len(t, remaining, 3) // All except old completed

	// Verify the deleted one is gone
	for _, r := range remaining {
		assert.NotEqual(t, "req-old-completed", r.RequestID)
	}
}

func TestAsyncUsageInfo_PriceAndUsageSerialization(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create record with Price and Usage objects
	info := &AsyncUsageInfo{
		RequestID: "req-serialization",
		RequestAt: time.Now(),
		Mode:      1,
		Model:     "gpt-4",
		ChannelID: 1,
		GroupID:   "group-1",
		TokenID:   1,
		TokenName: "test-token",
		Price: Price{
			InputPrice:         0.001,
			OutputPrice:        0.002,
			CachedPrice:        0.0005,
			CacheCreationPrice: 0.003,
		},
		Usage: Usage{
			InputTokens:         100,
			OutputTokens:        50,
			TotalTokens:         150,
			CachedTokens:        20,
			CacheCreationTokens: 10,
			ReasoningTokens:     30,
		},
		UsedAmount: 0.25,
	}

	err := CreateAsyncUsageInfo(info)
	require.NoError(t, err)

	// Fetch and verify serialization/deserialization
	var fetched AsyncUsageInfo

	err = LogDB.First(&fetched, info.ID).Error
	require.NoError(t, err)

	// Verify Price
	assert.Equal(t, info.Price.InputPrice, fetched.Price.InputPrice)
	assert.Equal(t, info.Price.OutputPrice, fetched.Price.OutputPrice)
	assert.Equal(t, info.Price.CachedPrice, fetched.Price.CachedPrice)
	assert.Equal(t, info.Price.CacheCreationPrice, fetched.Price.CacheCreationPrice)

	// Verify Usage
	assert.Equal(t, info.Usage.InputTokens, fetched.Usage.InputTokens)
	assert.Equal(t, info.Usage.OutputTokens, fetched.Usage.OutputTokens)
	assert.Equal(t, info.Usage.TotalTokens, fetched.Usage.TotalTokens)
	assert.Equal(t, info.Usage.CachedTokens, fetched.Usage.CachedTokens)
	assert.Equal(t, info.Usage.CacheCreationTokens, fetched.Usage.CacheCreationTokens)
	assert.Equal(t, info.Usage.ReasoningTokens, fetched.Usage.ReasoningTokens)
}

func TestAsyncUsageInfo_DataField(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name string
		data string
	}{
		{
			name: "simple job id",
			data: `{"job_id":"job-123"}`,
		},
		{
			name: "complex data",
			data: `{"job_id":"job-456","type":"video","format":"mp4","duration":60}`,
		},
		{
			name: "empty data",
			data: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &AsyncUsageInfo{
				RequestID: "req-" + tt.name,
				RequestAt: time.Now(),
				Mode:      1,
				Model:     "model",
				ChannelID: 1,
				GroupID:   "g1",
				TokenID:   1,
				Data:      tt.data,
			}

			err := CreateAsyncUsageInfo(info)
			require.NoError(t, err)

			var fetched AsyncUsageInfo

			err = LogDB.First(&fetched, info.ID).Error
			require.NoError(t, err)

			assert.Equal(t, tt.data, fetched.Data)
		})
	}
}

func TestUpdateLogUsageByRequestID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create a log entry first
	log := &Log{
		RequestID: "req-log-123",
		ChannelID: 5,
		Model:     "gpt-4-turbo",
		GroupID:   "test-group",
		TokenID:   10,
		TokenName: "test-token",
	}
	err := LogDB.Create(log).Error
	require.NoError(t, err)

	// Update usage
	usage := Usage{
		InputTokens:         1000,
		OutputTokens:        500,
		TotalTokens:         1500,
		CachedTokens:        100,
		CacheCreationTokens: 50,
		ReasoningTokens:     200,
		ImageInputTokens:    10,
		AudioInputTokens:    5,
		ImageOutputTokens:   3,
		WebSearchCount:      2,
	}
	amount := 1.5

	channelID, modelName, err := UpdateLogUsageByRequestID("req-log-123", usage, amount)
	require.NoError(t, err)
	assert.Equal(t, 5, channelID)
	assert.Equal(t, "gpt-4-turbo", modelName)

	// Verify the log was updated
	var updated Log

	err = LogDB.Where("request_id = ?", "req-log-123").First(&updated).Error
	require.NoError(t, err)

	assert.Equal(t, ZeroNullInt64(1000), updated.Usage.InputTokens)
	assert.Equal(t, ZeroNullInt64(500), updated.Usage.OutputTokens)
	assert.Equal(t, ZeroNullInt64(1500), updated.Usage.TotalTokens)
	assert.Equal(t, ZeroNullInt64(100), updated.Usage.CachedTokens)
	assert.Equal(t, ZeroNullInt64(50), updated.Usage.CacheCreationTokens)
	assert.Equal(t, ZeroNullInt64(200), updated.Usage.ReasoningTokens)
	assert.Equal(t, ZeroNullInt64(10), updated.Usage.ImageInputTokens)
	assert.Equal(t, ZeroNullInt64(5), updated.Usage.AudioInputTokens)
	assert.Equal(t, ZeroNullInt64(3), updated.Usage.ImageOutputTokens)
	assert.Equal(t, ZeroNullInt64(2), updated.Usage.WebSearchCount)
	assert.Equal(t, 1.5, updated.UsedAmount)
}

func TestUpdateLogUsageByRequestID_NotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	usage := Usage{InputTokens: 100}
	_, _, err := UpdateLogUsageByRequestID("non-existent-req", usage, 0.1)
	assert.Error(t, err)
}

func TestUpdateSummaryForAsyncUsage(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create async usage info
	requestAt := time.Now().Truncate(time.Hour) // Ensure it's on an hour boundary
	info := &AsyncUsageInfo{
		RequestID: "req-summary-test",
		RequestAt: requestAt,
		Mode:      1,
		Model:     "gpt-4",
		ChannelID: 10,
		GroupID:   "test-group",
		TokenID:   1,
		TokenName: "test-token",
	}

	usage := Usage{
		InputTokens:  500,
		OutputTokens: 250,
		TotalTokens:  750,
	}
	amount := 0.75

	// Add to batch
	UpdateSummaryForAsyncUsage(info, usage, amount)

	// Process batch to persist data
	ProcessBatchUpdatesSummary()

	// Verify Summary was created/updated
	var summary Summary

	hourTimestamp := requestAt.Truncate(time.Hour).Unix()
	err := LogDB.Where("channel_id = ? AND model = ? AND hour_timestamp = ?",
		info.ChannelID, info.Model, hourTimestamp).First(&summary).Error
	require.NoError(t, err)

	assert.Equal(t, ZeroNullInt64(500), summary.Data.InputTokens)
	assert.Equal(t, ZeroNullInt64(250), summary.Data.OutputTokens)
	assert.Equal(t, 0.75, summary.Data.UsedAmount)

	// Verify GroupSummary was created/updated
	var groupSummary GroupSummary

	err = LogDB.Where("group_id = ? AND token_name = ? AND model = ? AND hour_timestamp = ?",
		info.GroupID, info.TokenName, info.Model, hourTimestamp).First(&groupSummary).Error
	require.NoError(t, err)

	assert.Equal(t, ZeroNullInt64(500), groupSummary.Data.InputTokens)
	assert.Equal(t, ZeroNullInt64(250), groupSummary.Data.OutputTokens)
	assert.Equal(t, 0.75, groupSummary.Data.UsedAmount)
}

func TestUpdateSummaryForAsyncUsage_AddToExisting(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	requestAt := time.Now().Truncate(time.Hour)
	hourTimestamp := requestAt.Unix()

	// Create existing summary
	existingSummary := &Summary{
		Unique: SummaryUnique{
			ChannelID:     10,
			Model:         "gpt-4",
			HourTimestamp: hourTimestamp,
		},
		Data: SummaryData{
			Usage: Usage{
				InputTokens:  100,
				OutputTokens: 50,
				TotalTokens:  150,
			},
			UsedAmount: 0.15,
			Count: Count{
				RequestCount: 1,
			},
		},
	}
	err := LogDB.Create(existingSummary).Error
	require.NoError(t, err)

	// Update with async usage
	info := &AsyncUsageInfo{
		RequestID: "req-add-test",
		RequestAt: requestAt,
		Mode:      1,
		Model:     "gpt-4",
		ChannelID: 10,
		GroupID:   "g1",
		TokenID:   1,
		TokenName: "t1",
	}

	usage := Usage{
		InputTokens:  200,
		OutputTokens: 100,
		TotalTokens:  300,
	}
	amount := 0.30

	// Add to batch
	UpdateSummaryForAsyncUsage(info, usage, amount)

	// Process batch to persist data
	ProcessBatchUpdatesSummary()

	// Verify Summary was updated (values should be added)
	var summary Summary

	err = LogDB.Where("channel_id = ? AND model = ? AND hour_timestamp = ?",
		10, "gpt-4", hourTimestamp).First(&summary).Error
	require.NoError(t, err)

	// Tokens should be added
	assert.Equal(t, ZeroNullInt64(300), summary.Data.InputTokens)  // 100 + 200
	assert.Equal(t, ZeroNullInt64(150), summary.Data.OutputTokens) // 50 + 100
	assert.InDelta(t, 0.45, summary.Data.UsedAmount, 0.001)        // 0.15 + 0.30

	// Request count should NOT be incremented (async usage doesn't count as new request)
	assert.Equal(t, int64(1), summary.Data.RequestCount)
}

func TestAsyncUsageInfo_StatusTransitions(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	info := &AsyncUsageInfo{
		RequestID: "req-status",
		RequestAt: time.Now(),
		Mode:      1,
		Model:     "gpt-4",
		ChannelID: 1,
		GroupID:   "g1",
		TokenID:   1,
	}

	// Create - should be pending
	err := CreateAsyncUsageInfo(info)
	require.NoError(t, err)
	assert.Equal(t, AsyncUsageStatusPending, info.Status)

	// Transition to completed
	info.Status = AsyncUsageStatusCompleted
	info.Usage = Usage{InputTokens: 100}
	err = UpdateAsyncUsageInfo(info)
	require.NoError(t, err)

	var fetched AsyncUsageInfo

	err = LogDB.First(&fetched, info.ID).Error
	require.NoError(t, err)
	assert.Equal(t, AsyncUsageStatusCompleted, fetched.Status)

	// Create another and transition to failed
	info2 := &AsyncUsageInfo{
		RequestID: "req-status-2",
		RequestAt: time.Now(),
		Mode:      1,
		Model:     "gpt-4",
		ChannelID: 1,
		GroupID:   "g1",
		TokenID:   1,
	}
	err = CreateAsyncUsageInfo(info2)
	require.NoError(t, err)

	info2.Status = AsyncUsageStatusFailed
	info2.Error = "API error: rate limit exceeded"
	info2.RetryCount = 3
	err = UpdateAsyncUsageInfo(info2)
	require.NoError(t, err)

	var fetched2 AsyncUsageInfo

	err = LogDB.First(&fetched2, info2.ID).Error
	require.NoError(t, err)
	assert.Equal(t, AsyncUsageStatusFailed, fetched2.Status)
	assert.Equal(t, "API error: rate limit exceeded", fetched2.Error)
	assert.Equal(t, 3, fetched2.RetryCount)
}

func TestAsyncUsageInfo_Indexes(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create multiple records to test index usage
	now := time.Now()
	for i := range 10 {
		info := &AsyncUsageInfo{
			RequestID: "req-idx-" + string(rune('0'+i)),
			RequestAt: now,
			Mode:      i % 3,
			Model:     "model-" + string(rune('a'+i%2)),
			ChannelID: i % 5,
			GroupID:   "group-" + string(rune('0'+i%3)),
			TokenID:   i % 4,
			Status:    []string{AsyncUsageStatusPending, AsyncUsageStatusCompleted, AsyncUsageStatusFailed}[i%3],
		}
		info.CreatedAt = now
		info.UpdatedAt = now
		err := LogDB.Create(info).Error
		require.NoError(t, err)
	}

	// Query by status (indexed)
	var results []*AsyncUsageInfo

	err := LogDB.Where("status = ?", AsyncUsageStatusPending).Find(&results).Error
	require.NoError(t, err)

	for _, r := range results {
		assert.Equal(t, AsyncUsageStatusPending, r.Status)
	}

	// Query by mode (indexed)
	err = LogDB.Where("mode = ?", 1).Find(&results).Error
	require.NoError(t, err)

	for _, r := range results {
		assert.Equal(t, 1, r.Mode)
	}

	// Query by channel_id (indexed)
	err = LogDB.Where("channel_id = ?", 2).Find(&results).Error
	require.NoError(t, err)

	for _, r := range results {
		assert.Equal(t, 2, r.ChannelID)
	}

	// Query by group_id (indexed)
	err = LogDB.Where("group_id = ?", "group-1").Find(&results).Error
	require.NoError(t, err)

	for _, r := range results {
		assert.Equal(t, "group-1", r.GroupID)
	}
}

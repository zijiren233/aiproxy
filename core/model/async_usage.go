package model

import (
	"context"
	"time"
)

// AsyncUsageInfo contains metadata for async usage tracking
// This is designed to be stored in database with generic Data field for JSON
type AsyncUsageInfo struct {
	ID         int       `gorm:"primaryKey"                    json:"id"`                    // Auto-increment ID
	RequestID  string    `gorm:"size:64;index"                 json:"request_id"`            // Request ID to find/update the log entry
	RequestAt  time.Time `                                     json:"request_at"`            // Request time for calculating hour timestamp
	Mode       int       `gorm:"index"                         json:"mode"`                  // Relay mode (mode.Mode)
	Model      string    `gorm:"size:128"                      json:"model"`                 // Model name
	ChannelID  int       `gorm:"index"                         json:"channel_id"`            // Channel ID
	GroupID    string    `gorm:"size:64;index"                 json:"group_id"`              // Group ID
	TokenID    int       `gorm:"index"                         json:"token_id"`              // Token ID
	TokenName  string    `gorm:"size:128"                      json:"token_name,omitempty"`  // Token name for consumption
	Price      Price     `gorm:"serializer:fastjson;type:text" json:"price"`                 // Price for calculating amount
	Data       string    `gorm:"type:text"                     json:"data,omitempty"`        // Generic JSON data for adaptor-specific info (e.g., job_id)
	Status     string    `gorm:"size:32;index;default:pending" json:"status"`                // pending, completed, failed
	Usage      Usage     `gorm:"serializer:fastjson;type:text" json:"usage"`                 // Usage when completed
	UsedAmount float64   `                                     json:"used_amount,omitempty"` // Calculated amount when completed
	Error      string    `gorm:"type:text"                     json:"error,omitempty"`       // Error message if failed

	RetryCount int       `json:"retry_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

const (
	AsyncUsageStatusPending   = "pending"
	AsyncUsageStatusCompleted = "completed"
	AsyncUsageStatusFailed    = "failed"
)

// AsyncUsageFetcher is an interface that adaptors can implement to fetch async usage
type AsyncUsageFetcher interface {
	// FetchAsyncUsage fetches the usage for an async task
	// ctx: context for cancellation and timeout
	// channel: the channel containing API key and base URL
	// info: the async usage metadata
	// Returns the usage, whether the task is completed, and any error
	FetchAsyncUsage(
		ctx context.Context,
		channel *Channel,
		info *AsyncUsageInfo,
	) (Usage, bool, error)
}

// CreateAsyncUsageInfo creates a new async usage info record
func CreateAsyncUsageInfo(info *AsyncUsageInfo) error {
	info.Status = AsyncUsageStatusPending
	info.CreatedAt = time.Now()
	info.UpdatedAt = time.Now()
	return LogDB.Create(info).Error
}

// GetPendingAsyncUsages returns pending async usage records for processing
func GetPendingAsyncUsages(limit int) ([]*AsyncUsageInfo, error) {
	var results []*AsyncUsageInfo

	err := LogDB.Where("status = ?", AsyncUsageStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&results).Error

	return results, err
}

// GetPendingAsyncUsagesByMode returns pending async usage records of a specific mode
func GetPendingAsyncUsagesByMode(mode, limit int) ([]*AsyncUsageInfo, error) {
	var results []*AsyncUsageInfo

	err := LogDB.Where("status = ? AND mode = ?", AsyncUsageStatusPending, mode).
		Order("created_at ASC").
		Limit(limit).
		Find(&results).Error

	return results, err
}

// UpdateAsyncUsageInfo updates an async usage info record
func UpdateAsyncUsageInfo(info *AsyncUsageInfo) error {
	info.UpdatedAt = time.Now()
	return LogDB.Save(info).Error
}

// DeleteAsyncUsageInfo deletes an async usage info record
func DeleteAsyncUsageInfo(id int) error {
	return LogDB.Delete(&AsyncUsageInfo{}, "id = ?", id).Error
}

// CleanupCompletedAsyncUsages removes completed async usage records older than duration
func CleanupCompletedAsyncUsages(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	return LogDB.Where("status = ? AND updated_at < ?", AsyncUsageStatusCompleted, cutoff).
		Delete(&AsyncUsageInfo{}).Error
}

// UpdateLogUsageByRequestID updates the usage and amount for a log entry by RequestID
// It also returns the log entry's ChannelID and Model for updating summaries
func UpdateLogUsageByRequestID(
	requestID string,
	usage Usage,
	amount float64,
) (channelID int, modelName string, err error) {
	var logEntry Log

	err = LogDB.Where("request_id = ?", requestID).First(&logEntry).Error
	if err != nil {
		return 0, "", err
	}

	// Update the log entry with usage and amount, then save
	logEntry.Usage = usage
	logEntry.UsedAmount = amount
	err = LogDB.Save(&logEntry).Error

	return logEntry.ChannelID, logEntry.Model, err
}

// UpdateSummaryForAsyncUsage updates the Summary and GroupSummary tables with async usage data
// Note: This only updates usage/amount fields, not request counts since the request was already counted
func UpdateSummaryForAsyncUsage(info *AsyncUsageInfo, usage Usage, amount float64) {
	// Calculate hour timestamp from RequestAt
	hourTimestamp := info.RequestAt.Truncate(time.Hour).Unix()

	// Use batch update to avoid direct database calls
	BatchUpdateSummaryOnlyUsage(
		info.ChannelID,
		info.GroupID,
		info.TokenName,
		info.Model,
		hourTimestamp,
		usage,
		amount,
	)
}

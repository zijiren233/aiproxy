package model

import (
	"time"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/config"
	"gorm.io/gorm"
)

type RetryLog struct {
	RequestBody           string          `gorm:"type:text"                           json:"request_body,omitempty"`
	ResponseBody          string          `gorm:"type:text"                           json:"response_body,omitempty"`
	RequestBodyTruncated  bool            `json:"request_body_truncated,omitempty"`
	ResponseBodyTruncated bool            `json:"response_body_truncated,omitempty"`
	RequestAt             time.Time       `json:"request_at"`
	RetryAt               time.Time       `json:"retry_at,omitempty"`
	TTFBMilliseconds      ZeroNullInt64   `json:"ttfb_milliseconds,omitempty"`
	CreatedAt             time.Time       `gorm:"autoCreateTime;index"                json:"created_at"`
	Model                 string          `json:"model"`
	RequestID             EmptyNullString `gorm:"index:,where:request_id is not null" json:"request_id"`
	ID                    int             `gorm:"primaryKey"                          json:"id"`
	ChannelID             int             `json:"channel,omitempty"`
	Code                  int             `gorm:"index"                               json:"code,omitempty"`
	Mode                  int             `json:"mode,omitempty"`
	RetryTimes            ZeroNullInt64   `json:"retry_times,omitempty"`
}

func (d *RetryLog) BeforeSave(_ *gorm.DB) (err error) {
	if reqMax := config.GetLogDetailRequestBodyMaxSize(); reqMax > 0 && int64(len(d.RequestBody)) > reqMax {
		d.RequestBody = common.TruncateByRune(d.RequestBody, int(reqMax)) + "..."
		d.RequestBodyTruncated = true
	}
	if respMax := config.GetLogDetailResponseBodyMaxSize(); respMax > 0 && int64(len(d.ResponseBody)) > respMax {
		d.ResponseBody = common.TruncateByRune(d.ResponseBody, int(respMax)) + "..."
		d.ResponseBodyTruncated = true
	}
	return
}

func (l *RetryLog) MarshalJSON() ([]byte, error) {
	type Alias RetryLog
	a := &struct {
		*Alias
		CreatedAt int64 `json:"created_at"`
		RequestAt int64 `json:"request_at"`
		RetryAt   int64 `json:"retry_at,omitempty"`
	}{
		Alias:     (*Alias)(l),
		CreatedAt: l.CreatedAt.UnixMilli(),
		RequestAt: l.RequestAt.UnixMilli(),
	}
	if !l.RetryAt.IsZero() {
		a.RetryAt = l.RetryAt.UnixMilli()
	}
	return sonic.Marshal(a)
}

func RecordRetryLog(
	requestID string,
	createAt time.Time,
	requestAt time.Time,
	retryAt time.Time,
	firstByteAt time.Time,
	code int,
	channelID int,
	modelName string,
	mode int,
	retryTimes int,
	requestDetail *RequestDetail,
) error {
	if createAt.IsZero() {
		createAt = time.Now()
	}
	if requestAt.IsZero() {
		requestAt = createAt
	}
	if firstByteAt.IsZero() || firstByteAt.Before(requestAt) {
		firstByteAt = requestAt
	}
	log := &RetryLog{
		RequestID:        EmptyNullString(requestID),
		RequestAt:        requestAt,
		CreatedAt:        createAt,
		RetryAt:          retryAt,
		TTFBMilliseconds: ZeroNullInt64(firstByteAt.Sub(requestAt).Milliseconds()),
		Code:             code,
		Model:            modelName,
		Mode:             mode,
		ChannelID:        channelID,
		RetryTimes:       ZeroNullInt64(retryTimes),
		RequestBody:      requestDetail.RequestBody,
		ResponseBody:     requestDetail.ResponseBody,
	}
	return LogDB.Create(log).Error
}

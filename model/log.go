package model

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/common"
	"github.com/labring/aiproxy/common/config"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type RequestDetail struct {
	CreatedAt             time.Time `gorm:"autoCreateTime;index"              json:"-"`
	RequestBody           string    `gorm:"type:text"                         json:"request_body,omitempty"`
	ResponseBody          string    `gorm:"type:text"                         json:"response_body,omitempty"`
	RequestBodyTruncated  bool      `json:"request_body_truncated,omitempty"`
	ResponseBodyTruncated bool      `json:"response_body_truncated,omitempty"`
	ID                    int       `gorm:"primaryKey"                        json:"id"`
	LogID                 int       `gorm:"index"                             json:"log_id"`
}

func (d *RequestDetail) BeforeSave(_ *gorm.DB) (err error) {
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

type Price struct {
	InputPrice         float64 `json:"input_price,omitempty"`
	OutputPrice        float64 `json:"output_price,omitempty"`
	CachedPrice        float64 `json:"cached_price,omitempty"`
	CacheCreationPrice float64 `json:"cache_creation_price,omitempty"`
}

type Usage struct {
	InputTokens         int `json:"input_tokens,omitempty"`
	OutputTokens        int `json:"output_tokens,omitempty"`
	CachedTokens        int `json:"cached_tokens,omitempty"`
	CacheCreationTokens int `json:"cache_creation_tokens,omitempty"`
	TotalTokens         int `json:"total_tokens,omitempty"`
}

type Log struct {
	RequestDetail        *RequestDetail `gorm:"foreignKey:LogID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"request_detail,omitempty"`
	RequestAt            time.Time      `gorm:"index"                                                          json:"request_at"`
	TimestampTruncByDay  int64          `json:"timestamp_trunc_by_day"`
	TimestampTruncByHour int64          `json:"timestamp_trunc_by_hour"`
	CreatedAt            time.Time      `gorm:"autoCreateTime;index"                                           json:"created_at"`
	TokenName            string         `json:"token_name,omitempty"`
	Endpoint             string         `json:"endpoint"`
	Content              string         `gorm:"type:text"                                                      json:"content,omitempty"`
	GroupID              string         `gorm:"index"                                                          json:"group,omitempty"`
	Model                string         `gorm:"index"                                                          json:"model"`
	RequestID            string         `gorm:"index"                                                          json:"request_id"`
	ID                   int            `gorm:"primaryKey"                                                     json:"id"`
	TokenID              int            `gorm:"index"                                                          json:"token_id,omitempty"`
	ChannelID            int            `gorm:"index"                                                          json:"channel,omitempty"`
	Code                 int            `gorm:"index"                                                          json:"code,omitempty"`
	Mode                 int            `json:"mode,omitempty"`
	IP                   string         `gorm:"index"                                                          json:"ip,omitempty"`
	RetryTimes           int            `json:"retry_times,omitempty"`
	DownstreamResult     bool           `json:"downstream_result,omitempty"`
	Price                Price          `gorm:"embedded"                                                       json:"price,omitempty"`
	Usage                Usage          `gorm:"embedded"                                                       json:"usage,omitempty"`
	UsedAmount           float64        `json:"used_amount,omitempty"`
}

func CreateLogIndexes(db *gorm.DB) error {
	var indexes []string
	if common.UsingSQLite {
		// not support INCLUDE
		indexes = []string{
			// used by global search logs
			"CREATE INDEX IF NOT EXISTS idx_model_reqat ON logs (model, request_at)",
			// used by global search logs
			"CREATE INDEX IF NOT EXISTS idx_channel_reqat ON logs (channel_id, request_at)",
			// used by global search logs
			"CREATE INDEX IF NOT EXISTS idx_channel_model_reqat ON logs (channel_id, model, request_at)",

			// global day indexes, used by global dashboard
			"CREATE INDEX IF NOT EXISTS idx_model_reqat_truncday ON logs (model, request_at, timestamp_trunc_by_day)",
			// global hour indexes, used by global dashboard
			"CREATE INDEX IF NOT EXISTS idx_model_reqat_trunchour ON logs (model, request_at, timestamp_trunc_by_hour)",

			// used by search group logs
			"CREATE INDEX IF NOT EXISTS idx_group_token_reqat ON logs (group_id, token_name, request_at)",
			// used by search group logs
			"CREATE INDEX IF NOT EXISTS idx_group_model_reqat ON logs (group_id, model, request_at)",
			// used by search group logs
			"CREATE INDEX IF NOT EXISTS idx_group_token_model_reqat ON logs (group_id, token_name, model, request_at)",

			// day indexes, used by dashboard
			"CREATE INDEX IF NOT EXISTS idx_group_reqat_truncday ON logs (group_id, request_at, timestamp_trunc_by_day)",
			"CREATE INDEX IF NOT EXISTS idx_group_model_reqat_truncday ON logs (group_id, model, request_at, timestamp_trunc_by_day)",
			"CREATE INDEX IF NOT EXISTS idx_group_token_reqat_truncday ON logs (group_id, token_name, request_at, timestamp_trunc_by_day)",
			"CREATE INDEX IF NOT EXISTS idx_group_model_token_reqat_truncday ON logs (group_id, model, token_name, request_at, timestamp_trunc_by_day)",
			// hour indexes, used by dashboard
			"CREATE INDEX IF NOT EXISTS idx_group_reqat_trunchour ON logs (group_id, request_at, timestamp_trunc_by_hour)",
			"CREATE INDEX IF NOT EXISTS idx_group_model_reqat_trunchour ON logs (group_id, model, request_at, timestamp_trunc_by_hour)",
			"CREATE INDEX IF NOT EXISTS idx_group_token_reqat_trunchour ON logs (group_id, token_name, request_at, timestamp_trunc_by_hour)",
			"CREATE INDEX IF NOT EXISTS idx_group_model_token_reqat_trunchour ON logs (group_id, model, token_name, request_at, timestamp_trunc_by_hour)",
		}
	} else {
		indexes = []string{
			// used by global search logs
			"CREATE INDEX IF NOT EXISTS idx_model_reqat ON logs (model, request_at) INCLUDE (code, request_id, downstream_result)",
			// used by global search logs
			"CREATE INDEX IF NOT EXISTS idx_channel_reqat ON logs (channel_id, request_at) INCLUDE (code, request_id, downstream_result)",
			// used by global search logs
			"CREATE INDEX IF NOT EXISTS idx_channel_model_reqat ON logs (channel_id, model, request_at) INCLUDE (code, request_id, downstream_result)",

			// global day indexes, used by global dashboard
			"CREATE INDEX IF NOT EXISTS idx_model_reqat_truncday ON logs (model, request_at, timestamp_trunc_by_day) INCLUDE (code, downstream_result)",
			// global hour indexes, used by global dashboard
			"CREATE INDEX IF NOT EXISTS idx_model_reqat_trunchour ON logs (model, request_at, timestamp_trunc_by_hour) INCLUDE (code, downstream_result)",

			// used by search group logs
			"CREATE INDEX IF NOT EXISTS idx_group_token_reqat ON logs (group_id, token_name, request_at) INCLUDE (code, request_id, downstream_result)",
			// used by search group logs
			"CREATE INDEX IF NOT EXISTS idx_group_token_reqat ON logs (group_id, token_name, request_at) INCLUDE (code, request_id, downstream_result)",
			// used by search group logs
			"CREATE INDEX IF NOT EXISTS idx_group_model_reqat ON logs (group_id, model, request_at) INCLUDE (code, request_id, downstream_result)",
			// used by search group logs
			"CREATE INDEX IF NOT EXISTS idx_group_token_model_reqat ON logs (group_id, token_name, model, request_at) INCLUDE (code, request_id, downstream_result)",

			// day indexes, used by dashboard
			"CREATE INDEX IF NOT EXISTS idx_group_reqat_truncday ON logs (group_id, request_at, timestamp_trunc_by_day) INCLUDE (downstream_result)",
			"CREATE INDEX IF NOT EXISTS idx_group_model_reqat_truncday ON logs (group_id, model, request_at, timestamp_trunc_by_day) INCLUDE (downstream_result)",
			"CREATE INDEX IF NOT EXISTS idx_group_token_reqat_truncday ON logs (group_id, token_name, request_at, timestamp_trunc_by_day) INCLUDE (downstream_result)",
			"CREATE INDEX IF NOT EXISTS idx_group_model_token_reqat_truncday ON logs (group_id, model, token_name, request_at, timestamp_trunc_by_day) INCLUDE (downstream_result)",
			// hour indexes, used by dashboard
			"CREATE INDEX IF NOT EXISTS idx_group_reqat_trunchour ON logs (group_id, request_at, timestamp_trunc_by_hour) INCLUDE (downstream_result)",
			"CREATE INDEX IF NOT EXISTS idx_group_model_reqat_trunchour ON logs (group_id, model, request_at, timestamp_trunc_by_hour) INCLUDE (downstream_result)",
			"CREATE INDEX IF NOT EXISTS idx_group_token_reqat_trunchour ON logs (group_id, token_name, request_at, timestamp_trunc_by_hour) INCLUDE (downstream_result)",
			"CREATE INDEX IF NOT EXISTS idx_group_model_token_reqat_trunchour ON logs (group_id, model, token_name, request_at, timestamp_trunc_by_hour) INCLUDE (downstream_result)",
		}
	}

	for _, index := range indexes {
		if err := db.Exec(index).Error; err != nil {
			return err
		}
	}

	return nil
}

const (
	contentMaxSize = 2 * 1024 // 2KB
)

func (l *Log) BeforeSave(_ *gorm.DB) (err error) {
	if len(l.Content) > contentMaxSize {
		l.Content = common.TruncateByRune(l.Content, contentMaxSize) + "..."
	}
	if l.TimestampTruncByDay == 0 {
		l.TimestampTruncByDay = l.RequestAt.Truncate(24 * time.Hour).Unix()
	}
	if l.TimestampTruncByHour == 0 {
		l.TimestampTruncByHour = l.RequestAt.Truncate(time.Hour).Unix()
	}
	return
}

func (l *Log) MarshalJSON() ([]byte, error) {
	type Alias Log
	return sonic.Marshal(&struct {
		*Alias
		CreatedAt int64 `json:"created_at"`
		RequestAt int64 `json:"request_at"`
	}{
		Alias:     (*Alias)(l),
		CreatedAt: l.CreatedAt.UnixMilli(),
		RequestAt: l.RequestAt.UnixMilli(),
	})
}

func GetLogDetail(logID int) (*RequestDetail, error) {
	var detail RequestDetail
	err := LogDB.
		Model(&RequestDetail{}).
		Where("log_id = ?", logID).
		First(&detail).Error
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

func GetGroupLogDetail(logID int, group string) (*RequestDetail, error) {
	if group == "" || group == "*" {
		return nil, errors.New("invalid group parameter")
	}
	var detail RequestDetail
	err := LogDB.
		Model(&RequestDetail{}).
		Joins("JOIN logs ON logs.id = request_details.log_id").
		Where("logs.group_id = ?", group).
		Where("log_id = ?", logID).
		First(&detail).Error
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

const defaultCleanLogBatchSize = 1000

func CleanLog(batchSize int) error {
	err := cleanLog(batchSize)
	if err != nil {
		return err
	}
	return cleanLogDetail(batchSize)
}

func cleanLog(batchSize int) error {
	logStorageHours := config.GetLogStorageHours()
	if logStorageHours <= 0 {
		return nil
	}
	if batchSize <= 0 {
		batchSize = defaultCleanLogBatchSize
	}
	return LogDB.
		Session(&gorm.Session{SkipDefaultTransaction: true}).
		Where(
			"created_at < ?",
			time.Now().Add(-time.Duration(logStorageHours)*time.Hour),
		).
		Limit(batchSize).
		Delete(&Log{}).Error
}

func cleanLogDetail(batchSize int) error {
	detailStorageHours := config.GetLogDetailStorageHours()
	if detailStorageHours <= 0 {
		return nil
	}
	if batchSize <= 0 {
		batchSize = defaultCleanLogBatchSize
	}
	return LogDB.
		Session(&gorm.Session{SkipDefaultTransaction: true}).
		Where(
			"created_at < ?",
			time.Now().Add(-time.Duration(detailStorageHours)*time.Hour),
		).
		Limit(batchSize).
		Delete(&RequestDetail{}).Error
}

func RecordConsumeLog(
	requestID string,
	requestAt time.Time,
	group string,
	code int,
	channelID int,
	modelName string,
	tokenID int,
	tokenName string,
	endpoint string,
	content string,
	mode int,
	ip string,
	retryTimes int,
	requestDetail *RequestDetail,
	downstreamResult bool,
	usage Usage,
	modelPrice Price,
	amount float64,
) error {
	log := &Log{
		RequestID:        requestID,
		RequestAt:        requestAt,
		GroupID:          group,
		CreatedAt:        time.Now(),
		Code:             code,
		TokenID:          tokenID,
		TokenName:        tokenName,
		Model:            modelName,
		Mode:             mode,
		IP:               ip,
		ChannelID:        channelID,
		Endpoint:         endpoint,
		Content:          content,
		RetryTimes:       retryTimes,
		RequestDetail:    requestDetail,
		DownstreamResult: downstreamResult,
		Price:            modelPrice,
		Usage:            usage,
		UsedAmount:       amount,
	}
	return LogDB.Create(log).Error
}

func getLogOrder(order string) string {
	prefix, suffix, _ := strings.Cut(order, "-")
	switch prefix {
	case "request_at", "id", "created_at":
		switch suffix {
		case "asc":
			return prefix + " asc"
		default:
			return prefix + " desc"
		}
	default:
		return "request_at desc"
	}
}

type CodeType string

const (
	CodeTypeAll     CodeType = "all"
	CodeTypeSuccess CodeType = "success"
	CodeTypeError   CodeType = "error"
)

type GetLogsResult struct {
	Logs  []*Log `json:"logs"`
	Total int64  `json:"total"`
}

type GetGroupLogsResult struct {
	GetLogsResult
	Models     []string `json:"models"`
	TokenNames []string `json:"token_names"`
}

func buildGetLogsQuery(
	group string,
	startTimestamp time.Time,
	endTimestamp time.Time,
	modelName string,
	requestID string,
	tokenID int,
	tokenName string,
	channelID int,
	endpoint string,
	mode int,
	codeType CodeType,
	ip string,
	resultOnly bool,
) *gorm.DB {
	tx := LogDB.Model(&Log{})

	if group == "" {
		tx = tx.Where("group_id IS NULL OR group_id = ''")
	} else if group != "*" {
		tx = tx.Where("group_id = ?", group)
	}

	switch {
	case !startTimestamp.IsZero() && !endTimestamp.IsZero():
		tx = tx.Where("request_at BETWEEN ? AND ?", startTimestamp, endTimestamp)
	case !startTimestamp.IsZero():
		tx = tx.Where("request_at >= ?", startTimestamp)
	case !endTimestamp.IsZero():
		tx = tx.Where("request_at <= ?", endTimestamp)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if modelName != "" {
		tx = tx.Where("model = ?", modelName)
	}
	if mode != 0 {
		tx = tx.Where("mode = ?", mode)
	}
	if requestID != "" {
		tx = tx.Where("request_id = ?", requestID)
	}
	if tokenID != 0 {
		tx = tx.Where("token_id = ?", tokenID)
	}
	if channelID != 0 {
		tx = tx.Where("channel_id = ?", channelID)
	}
	if endpoint != "" {
		tx = tx.Where("endpoint = ?", endpoint)
	}
	if ip != "" {
		tx = tx.Where("ip = ?", ip)
	}
	switch codeType {
	case CodeTypeSuccess:
		tx = tx.Where("code = 200")
	case CodeTypeError:
		tx = tx.Where("code != 200")
	}
	if resultOnly {
		tx = tx.Where("downstream_result = true")
	}
	return tx
}

func getLogs(
	group string,
	startTimestamp time.Time,
	endTimestamp time.Time,
	modelName string,
	requestID string,
	tokenID int,
	tokenName string,
	channelID int,
	endpoint string,
	order string,
	mode int,
	codeType CodeType,
	withBody bool,
	ip string,
	page int,
	perPage int,
	resultOnly bool,
) (int64, []*Log, error) {
	var total int64
	var logs []*Log

	g := new(errgroup.Group)

	g.Go(func() error {
		return buildGetLogsQuery(
			group,
			startTimestamp,
			endTimestamp,
			modelName,
			requestID,
			tokenID,
			tokenName,
			channelID,
			endpoint,
			mode,
			codeType,
			ip,
			resultOnly,
		).Count(&total).Error
	})

	g.Go(func() error {
		query := buildGetLogsQuery(
			group,
			startTimestamp,
			endTimestamp,
			modelName,
			requestID,
			tokenID,
			tokenName,
			channelID,
			endpoint,
			mode,
			codeType,
			ip,
			resultOnly,
		)
		if withBody {
			query = query.Preload("RequestDetail")
		} else {
			query = query.Preload("RequestDetail", func(db *gorm.DB) *gorm.DB {
				return db.Select("id", "log_id")
			})
		}

		limit, offset := toLimitOffset(page, perPage)
		return query.
			Order(getLogOrder(order)).
			Limit(limit).
			Offset(offset).
			Find(&logs).Error
	})

	if err := g.Wait(); err != nil {
		return 0, nil, err
	}

	return total, logs, nil
}

func GetLogs(
	group string,
	startTimestamp time.Time,
	endTimestamp time.Time,
	modelName string,
	requestID string,
	tokenID int,
	tokenName string,
	channelID int,
	endpoint string,
	order string,
	mode int,
	codeType CodeType,
	withBody bool,
	ip string,
	page int,
	perPage int,
	resultOnly bool,
) (*GetLogsResult, error) {
	total, logs, err := getLogs(group, startTimestamp, endTimestamp, modelName, requestID, tokenID, tokenName, channelID, endpoint, order, mode, codeType, withBody, ip, page, perPage, resultOnly)
	if err != nil {
		return nil, err
	}

	result := &GetLogsResult{
		Logs:  logs,
		Total: total,
	}

	return result, nil
}

func GetGroupLogs(
	group string,
	startTimestamp time.Time,
	endTimestamp time.Time,
	modelName string,
	requestID string,
	tokenID int,
	tokenName string,
	channelID int,
	endpoint string,
	order string,
	mode int,
	codeType CodeType,
	withBody bool,
	ip string,
	page int,
	perPage int,
	resultOnly bool,
) (*GetGroupLogsResult, error) {
	if group == "" {
		return nil, errors.New("group is required")
	}

	var (
		total      int64
		logs       []*Log
		tokenNames []string
		models     []string
	)

	g := new(errgroup.Group)

	g.Go(func() error {
		var err error
		total, logs, err = getLogs(group, startTimestamp, endTimestamp, modelName, requestID, tokenID, tokenName, channelID, endpoint, order, mode, codeType, withBody, ip, page, perPage, resultOnly)
		return err
	})

	g.Go(func() error {
		var err error
		tokenNames, err = GetUsedTokenNames(group, startTimestamp, endTimestamp)
		return err
	})

	g.Go(func() error {
		var err error
		models, err = GetUsedModels(group, startTimestamp, endTimestamp)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &GetGroupLogsResult{
		GetLogsResult: GetLogsResult{
			Logs:  logs,
			Total: total,
		},
		Models:     models,
		TokenNames: tokenNames,
	}, nil
}

func buildSearchLogsQuery(
	group string,
	keyword string,
	endpoint string,
	requestID string,
	tokenID int,
	tokenName string,
	modelName string,
	startTimestamp time.Time,
	endTimestamp time.Time,
	channelID int,
	mode int,
	codeType CodeType,
	ip string,
	resultOnly bool,
) *gorm.DB {
	tx := LogDB.Model(&Log{})

	if group == "" {
		tx = tx.Where("group_id IS NULL OR group_id = ''")
	} else if group != "*" {
		tx = tx.Where("group_id = ?", group)
	}

	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}

	if modelName != "" {
		tx = tx.Where("model = ?", modelName)
	}

	switch {
	case !startTimestamp.IsZero() && !endTimestamp.IsZero():
		tx = tx.Where("request_at BETWEEN ? AND ?", startTimestamp, endTimestamp)
	case !startTimestamp.IsZero():
		tx = tx.Where("request_at >= ?", startTimestamp)
	case !endTimestamp.IsZero():
		tx = tx.Where("request_at <= ?", endTimestamp)
	}

	if requestID != "" {
		tx = tx.Where("request_id = ?", requestID)
	}
	if tokenID != 0 {
		tx = tx.Where("token_id = ?", tokenID)
	}
	if channelID != 0 {
		tx = tx.Where("channel_id = ?", channelID)
	}

	switch codeType {
	case CodeTypeSuccess:
		tx = tx.Where("code = 200")
	case CodeTypeError:
		tx = tx.Where("code != 200")
	}
	if resultOnly {
		tx = tx.Where("downstream_result = true")
	}

	if ip != "" {
		tx = tx.Where("ip = ?", ip)
	}

	if mode != 0 {
		tx = tx.Where("mode = ?", mode)
	}
	if endpoint != "" {
		tx = tx.Where("endpoint = ?", endpoint)
	}

	// Handle keyword search for zero value fields
	if keyword != "" {
		var conditions []string
		var values []interface{}

		if group == "" {
			conditions = append(conditions, "group_id = ?")
			values = append(values, keyword)
		}
		if tokenName == "" {
			conditions = append(conditions, "token_name = ?")
			values = append(values, keyword)
		}
		if modelName == "" {
			conditions = append(conditions, "model = ?")
			values = append(values, keyword)
		}
		if requestID == "" {
			conditions = append(conditions, "request_id = ?")
			values = append(values, keyword)
		}

		// if num := String2Int(keyword); num != 0 {
		// 	if channelID == 0 {
		// 		conditions = append(conditions, "channel_id = ?")
		// 		values = append(values, num)
		// 	}
		// 	if mode != 0 {
		// 		conditions = append(conditions, "mode = ?")
		// 		values = append(values, num)
		// 	}
		// }

		// if ip != "" {
		// 	conditions = append(conditions, "ip = ?")
		// 	values = append(values, ip)
		// }

		// if endpoint == "" {
		// 	if common.UsingPostgreSQL {
		// 		conditions = append(conditions, "endpoint ILIKE ?")
		// 	} else {
		// 		conditions = append(conditions, "endpoint LIKE ?")
		// 	}
		// 	values = append(values, "%"+keyword+"%")
		// }

		// slow query
		// if common.UsingPostgreSQL {
		// 	conditions = append(conditions, "content ILIKE ?")
		// } else {
		// 	conditions = append(conditions, "content LIKE ?")
		// }
		// values = append(values, "%"+keyword+"%")

		if len(conditions) > 0 {
			tx = tx.Where(fmt.Sprintf("(%s)", strings.Join(conditions, " OR ")), values...)
		}
	}

	return tx
}

func searchLogs(
	group string,
	keyword string,
	endpoint string,
	requestID string,
	tokenID int,
	tokenName string,
	modelName string,
	startTimestamp time.Time,
	endTimestamp time.Time,
	channelID int,
	order string,
	mode int,
	codeType CodeType,
	withBody bool,
	ip string,
	page int,
	perPage int,
	resultOnly bool,
) (int64, []*Log, error) {
	var total int64
	var logs []*Log

	g := new(errgroup.Group)

	g.Go(func() error {
		return buildSearchLogsQuery(
			group,
			keyword,
			endpoint,
			requestID,
			tokenID,
			tokenName,
			modelName,
			startTimestamp,
			endTimestamp,
			channelID,
			mode,
			codeType,
			ip,
			resultOnly,
		).Count(&total).Error
	})

	g.Go(func() error {
		query := buildSearchLogsQuery(
			group,
			keyword,
			endpoint,
			requestID,
			tokenID,
			tokenName,
			modelName,
			startTimestamp,
			endTimestamp,
			channelID,
			mode,
			codeType,
			ip,
			resultOnly,
		)

		if withBody {
			query = query.Preload("RequestDetail")
		} else {
			query = query.Preload("RequestDetail", func(db *gorm.DB) *gorm.DB {
				return db.Select("id", "log_id")
			})
		}

		limit, offset := toLimitOffset(page, perPage)
		return query.
			Order(getLogOrder(order)).
			Limit(limit).
			Offset(offset).
			Find(&logs).Error
	})

	if err := g.Wait(); err != nil {
		return 0, nil, err
	}

	return total, logs, nil
}

func SearchLogs(
	group string,
	keyword string,
	endpoint string,
	requestID string,
	tokenID int,
	tokenName string,
	modelName string,
	startTimestamp time.Time,
	endTimestamp time.Time,
	channelID int,
	order string,
	mode int,
	codeType CodeType,
	withBody bool,
	ip string,
	page int,
	perPage int,
	resultOnly bool,
) (*GetLogsResult, error) {
	total, logs, err := searchLogs(group, keyword, endpoint, requestID, tokenID, tokenName, modelName, startTimestamp, endTimestamp, channelID, order, mode, codeType, withBody, ip, page, perPage, resultOnly)
	if err != nil {
		return nil, err
	}

	result := &GetLogsResult{
		Logs:  logs,
		Total: total,
	}

	return result, nil
}

func SearchGroupLogs(
	group string,
	keyword string,
	endpoint string,
	requestID string,
	tokenID int,
	tokenName string,
	modelName string,
	startTimestamp time.Time,
	endTimestamp time.Time,
	channelID int,
	order string,
	mode int,
	codeType CodeType,
	withBody bool,
	ip string,
	page int,
	perPage int,
	resultOnly bool,
) (*GetGroupLogsResult, error) {
	if group == "" {
		return nil, errors.New("group is required")
	}

	var (
		total      int64
		logs       []*Log
		tokenNames []string
		models     []string
	)

	g := new(errgroup.Group)

	g.Go(func() error {
		var err error
		total, logs, err = searchLogs(group, keyword, endpoint, requestID, tokenID, tokenName, modelName, startTimestamp, endTimestamp, channelID, order, mode, codeType, withBody, ip, page, perPage, resultOnly)
		return err
	})

	g.Go(func() error {
		var err error
		tokenNames, err = GetUsedTokenNames(group, startTimestamp, endTimestamp)
		return err
	})

	g.Go(func() error {
		var err error
		models, err = GetUsedModels(group, startTimestamp, endTimestamp)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	result := &GetGroupLogsResult{
		GetLogsResult: GetLogsResult{
			Logs:  logs,
			Total: total,
		},
		Models:     models,
		TokenNames: tokenNames,
	}

	return result, nil
}

func DeleteOldLog(timestamp time.Time) (int64, error) {
	result := LogDB.Where("request_at < ?", timestamp).Delete(&Log{})
	return result.RowsAffected, result.Error
}

func DeleteGroupLogs(groupID string) (int64, error) {
	if groupID == "" {
		return 0, errors.New("group is required")
	}
	result := LogDB.Where("group_id = ?", groupID).Delete(&Log{})
	return result.RowsAffected, result.Error
}

type ChartData struct {
	Timestamp           int64   `json:"timestamp"`
	RequestCount        int64   `json:"request_count"`
	UsedAmount          float64 `json:"used_amount"`
	InputTokens         int64   `json:"input_tokens"`
	OutputTokens        int64   `json:"output_tokens"`
	CachedTokens        int64   `json:"cached_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	TotalTokens         int64   `json:"total_tokens"`
	ExceptionCount      int64   `json:"exception_count"`
}

type DashboardResponse struct {
	ChartData      []*ChartData `json:"chart_data"`
	TotalCount     int64        `json:"total_count"`
	ExceptionCount int64        `json:"exception_count"`
	UsedAmount     float64      `json:"used_amount"`
	RPM            int64        `json:"rpm"`
	TPM            int64        `json:"tpm"`
}

type GroupDashboardResponse struct {
	DashboardResponse
	Models     []string `json:"models"`
	TokenNames []string `json:"token_names"`
}

type TimeSpanType string

const (
	TimeSpanDay  TimeSpanType = "day"
	TimeSpanHour TimeSpanType = "hour"
)

func getTimeSpanFormat(t TimeSpanType) string {
	switch t {
	case TimeSpanDay:
		return "timestamp_trunc_by_day"
	case TimeSpanHour:
		return "timestamp_trunc_by_hour"
	default:
		return ""
	}
}

func getChartData(group string, start, end time.Time, tokenName, modelName string, timeSpan TimeSpanType, resultOnly bool, tokenUsage bool) ([]*ChartData, error) {
	var chartData []*ChartData

	timeSpanFormat := getTimeSpanFormat(timeSpan)
	if timeSpanFormat == "" {
		return nil, errors.New("unsupported time format")
	}

	var query *gorm.DB
	if tokenUsage {
		query = LogDB.Table("logs").
			Select(timeSpanFormat + " as timestamp, count(*) as request_count, sum(used_amount) as used_amount, sum(case when code != 200 then 1 else 0 end) as exception_count, sum(input_tokens) as input_tokens, sum(output_tokens) as output_tokens, sum(cached_tokens) as cached_tokens, sum(cache_creation_tokens) as cache_creation_tokens, sum(total_tokens) as total_tokens").
			Group("timestamp").
			Order("timestamp ASC")
	} else {
		query = LogDB.Table("logs").
			Select(timeSpanFormat + " as timestamp, count(*) as request_count, sum(used_amount) as used_amount, sum(case when code != 200 then 1 else 0 end) as exception_count").
			Group("timestamp").
			Order("timestamp ASC")
	}

	if group == "" {
		query = query.Where("group_id IS NULL OR group_id = ''")
	} else if group != "*" {
		query = query.Where("group_id = ?", group)
	}

	if resultOnly {
		query = query.Where("downstream_result = true")
	}

	switch {
	case !start.IsZero() && !end.IsZero():
		query = query.Where("request_at BETWEEN ? AND ?", start, end)
	case !start.IsZero():
		query = query.Where("request_at >= ?", start)
	case !end.IsZero():
		query = query.Where("request_at <= ?", end)
	}

	if tokenName != "" {
		query = query.Where("token_name = ?", tokenName)
	}
	if modelName != "" {
		query = query.Where("model = ?", modelName)
	}

	err := query.Scan(&chartData).Error

	return chartData, err
}

func GetUsedModels(group string, start, end time.Time) ([]string, error) {
	return getLogGroupByValues[string]("model", group, start, end)
}

func GetUsedTokenNames(group string, start, end time.Time) ([]string, error) {
	return getLogGroupByValues[string]("token_name", group, start, end)
}

//nolint:unused
func getLogDistinctValues[T cmp.Ordered](field string, group string, start, end time.Time) ([]T, error) {
	var values []T
	query := LogDB.
		Model(&Log{})

	if group != "" {
		query = query.Where("group_id = ?", group)
	}

	switch {
	case !start.IsZero() && !end.IsZero():
		query = query.Where("request_at BETWEEN ? AND ?", start, end)
	case !start.IsZero():
		query = query.Where("request_at >= ?", start)
	case !end.IsZero():
		query = query.Where("request_at <= ?", end)
	}

	err := query.
		Distinct(field).
		Pluck(field, &values).Error
	if err != nil {
		return nil, err
	}
	slices.Sort(values)
	return values, nil
}

func getLogGroupByValues[T cmp.Ordered](field string, group string, start, end time.Time) ([]T, error) {
	var values []T
	query := LogDB.
		Model(&Log{})

	if group == "" {
		query = query.Where("group_id IS NULL OR group_id = ''")
	} else if group != "*" {
		query = query.Where("group_id = ?", group)
	}

	switch {
	case !start.IsZero() && !end.IsZero():
		query = query.Where("request_at BETWEEN ? AND ?", start, end)
	case !start.IsZero():
		query = query.Where("request_at >= ?", start)
	case !end.IsZero():
		query = query.Where("request_at <= ?", end)
	}

	err := query.
		Select(field).
		Group(field).
		Pluck(field, &values).Error
	if err != nil {
		return nil, err
	}
	slices.Sort(values)
	return values, nil
}

func sumTotalCount(chartData []*ChartData) int64 {
	var count int64
	for _, data := range chartData {
		count += data.RequestCount
	}
	return count
}

func sumExceptionCount(chartData []*ChartData) int64 {
	var count int64
	for _, data := range chartData {
		count += data.ExceptionCount
	}
	return count
}

func sumUsedAmount(chartData []*ChartData) float64 {
	var amount decimal.Decimal
	for _, data := range chartData {
		amount = amount.Add(decimal.NewFromFloat(data.UsedAmount))
	}
	return amount.InexactFloat64()
}

func getRPM(group string, end time.Time, tokenName, modelName string, resultOnly bool) (int64, error) {
	query := LogDB.Model(&Log{})

	if group == "" {
		query = query.Where("group_id IS NULL OR group_id = ''")
	} else if group != "*" {
		query = query.Where("group_id = ?", group)
	}
	if tokenName != "" {
		query = query.Where("token_name = ?", tokenName)
	}
	if modelName != "" {
		query = query.Where("model = ?", modelName)
	}
	if resultOnly {
		query = query.Where("downstream_result = true")
	}

	var count int64
	err := query.
		Where("request_at BETWEEN ? AND ?", end.Add(-time.Minute), end).
		Count(&count).Error
	return count, err
}

func getTPM(group string, end time.Time, tokenName, modelName string, resultOnly bool) (int64, error) {
	query := LogDB.Model(&Log{}).
		Select("COALESCE(SUM(total_tokens), 0)").
		Where("request_at >= ? AND request_at <= ?", end.Add(-time.Minute), end)

	if group == "" {
		query = query.Where("group_id IS NULL OR group_id = ''")
	} else if group != "*" {
		query = query.Where("group_id = ?", group)
	}
	if tokenName != "" {
		query = query.Where("token_name = ?", tokenName)
	}
	if modelName != "" {
		query = query.Where("model = ?", modelName)
	}
	if resultOnly {
		query = query.Where("downstream_result = true")
	}

	var tpm int64
	err := query.Scan(&tpm).Error
	return tpm, err
}

func GetDashboardData(
	group string,
	start,
	end time.Time,
	modelName string,
	timeSpan TimeSpanType,
	resultOnly bool,
	needRPM bool,
	tokenUsage bool,
) (*DashboardResponse, error) {
	if end.IsZero() {
		end = time.Now()
	} else if end.Before(start) {
		return nil, errors.New("end time is before start time")
	}

	var (
		chartData []*ChartData
		rpm       int64
		tpm       int64
	)

	g := new(errgroup.Group)

	g.Go(func() error {
		var err error
		chartData, err = getChartData(group, start, end, "", modelName, timeSpan, resultOnly, tokenUsage)
		return err
	})

	if needRPM {
		g.Go(func() error {
			var err error
			rpm, err = getRPM(group, end, "", modelName, resultOnly)
			return err
		})
	}

	g.Go(func() error {
		var err error
		tpm, err = getTPM(group, end, "", modelName, resultOnly)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	totalCount := sumTotalCount(chartData)
	exceptionCount := sumExceptionCount(chartData)
	usedAmount := sumUsedAmount(chartData)

	return &DashboardResponse{
		ChartData:      chartData,
		TotalCount:     totalCount,
		ExceptionCount: exceptionCount,
		UsedAmount:     usedAmount,
		RPM:            rpm,
		TPM:            tpm,
	}, nil
}

func GetGroupDashboardData(
	group string,
	start, end time.Time,
	tokenName string,
	modelName string,
	timeSpan TimeSpanType,
	resultOnly bool,
	needRPM bool,
	tokenUsage bool,
) (*GroupDashboardResponse, error) {
	if group == "" || group == "*" {
		return nil, errors.New("group is required")
	}

	if end.IsZero() {
		end = time.Now()
	} else if end.Before(start) {
		return nil, errors.New("end time is before start time")
	}

	var (
		chartData  []*ChartData
		tokenNames []string
		models     []string
		rpm        int64
		tpm        int64
	)

	g := new(errgroup.Group)

	g.Go(func() error {
		var err error
		chartData, err = getChartData(group, start, end, tokenName, modelName, timeSpan, resultOnly, tokenUsage)
		return err
	})

	g.Go(func() error {
		var err error
		tokenNames, err = GetUsedTokenNames(group, start, end)
		return err
	})

	g.Go(func() error {
		var err error
		models, err = GetUsedModels(group, start, end)
		return err
	})

	if needRPM {
		g.Go(func() error {
			var err error
			rpm, err = getRPM(group, end, tokenName, modelName, resultOnly)
			return err
		})
	}

	g.Go(func() error {
		var err error
		tpm, err = getTPM(group, end, tokenName, modelName, resultOnly)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	totalCount := sumTotalCount(chartData)
	exceptionCount := sumExceptionCount(chartData)
	usedAmount := sumUsedAmount(chartData)

	return &GroupDashboardResponse{
		DashboardResponse: DashboardResponse{
			ChartData:      chartData,
			TotalCount:     totalCount,
			ExceptionCount: exceptionCount,
			UsedAmount:     usedAmount,
			RPM:            rpm,
			TPM:            tpm,
		},
		Models:     models,
		TokenNames: tokenNames,
	}, nil
}

func GetGroupLastRequestTime(group string) (time.Time, error) {
	if group == "" {
		return time.Time{}, errors.New("group is required")
	}
	var log Log
	err := LogDB.Model(&Log{}).Where("group_id = ?", group).Order("request_at desc").First(&log).Error
	return log.RequestAt, err
}

func GetTokenLastRequestTime(id int) (time.Time, error) {
	var log Log
	tx := LogDB.Model(&Log{})
	err := tx.Where("token_id = ?", id).Order("request_at desc").First(&log).Error
	return log.RequestAt, err
}

func GetGroupModelTPM(group string, model string) (int64, error) {
	end := time.Now()
	start := end.Add(-time.Minute)
	var tpm int64
	err := LogDB.
		Model(&Log{}).
		Where("group_id = ? AND request_at >= ? AND request_at <= ? AND model = ?", group, start, end, model).
		Select("COALESCE(SUM(total_tokens), 0)").
		Scan(&tpm).Error
	return tpm, err
}

//nolint:revive
type ModelCostRank struct {
	Model               string  `json:"model"`
	UsedAmount          float64 `json:"used_amount"`
	InputTokens         int64   `json:"input_tokens"`
	OutputTokens        int64   `json:"output_tokens"`
	CachedTokens        int64   `json:"cached_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	TotalTokens         int64   `json:"total_tokens"`
	Total               int64   `json:"total"`
}

func GetModelCostRank(group string, start, end time.Time, tokenUsage bool) ([]*ModelCostRank, error) {
	var ranks []*ModelCostRank

	var query *gorm.DB
	if tokenUsage {
		query = LogDB.Model(&Log{}).
			Select("model, SUM(used_amount) as used_amount, SUM(input_tokens) as input_tokens, SUM(output_tokens) as output_tokens, SUM(cached_tokens) as cached_tokens, SUM(cache_creation_tokens) as cache_creation_tokens, SUM(total_tokens) as total_tokens, COUNT(*) as total").
			Group("model").
			Order("used_amount DESC")
	} else {
		query = LogDB.Model(&Log{}).
			Select("model, SUM(used_amount) as used_amount, COUNT(*) as total").
			Group("model").
			Order("used_amount DESC")
	}

	if group == "" {
		query = query.Where("group_id IS NULL OR group_id = ''")
	} else if group != "*" {
		query = query.Where("group_id = ?", group)
	}

	switch {
	case !start.IsZero() && !end.IsZero():
		query = query.Where("request_at BETWEEN ? AND ?", start, end)
	case !start.IsZero():
		query = query.Where("request_at >= ?", start)
	case !end.IsZero():
		query = query.Where("request_at <= ?", end)
	}

	err := query.Scan(&ranks).Error
	if err != nil {
		return nil, err
	}

	return ranks, nil
}

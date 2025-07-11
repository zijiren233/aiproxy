package model

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/config"
	"github.com/labring/aiproxy/core/common/conv"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ErrTokenNotFound = "token"
)

const (
	TokenStatusEnabled  = 1
	TokenStatusDisabled = 2
)

type Token struct {
	CreatedAt    time.Time       `json:"created_at"`
	ExpiredAt    time.Time       `json:"expired_at"`
	Group        *Group          `json:"-"             gorm:"foreignKey:GroupID"`
	Key          string          `json:"key"           gorm:"type:char(48);uniqueIndex"`
	Name         EmptyNullString `json:"name"          gorm:"index;uniqueIndex:idx_group_name;not null"`
	GroupID      string          `json:"group"         gorm:"index;uniqueIndex:idx_group_name"`
	Subnets      []string        `json:"subnets"       gorm:"serializer:fastjson;type:text"`
	Models       []string        `json:"models"        gorm:"serializer:fastjson;type:text"`
	Status       int             `json:"status"        gorm:"default:1;index"`
	ID           int             `json:"id"            gorm:"primaryKey"`
	Quota        float64         `json:"quota"`
	UsedAmount   float64         `json:"used_amount"   gorm:"index"`
	RequestCount int             `json:"request_count" gorm:"index"`
}

func (t *Token) BeforeCreate(_ *gorm.DB) (err error) {
	if t.Key == "" || len(t.Key) != 48 {
		t.Key = generateKey()
	}
	return
}

const (
	keyChars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func generateKey() string {
	key := make([]byte, 48)
	for i := range key {
		key[i] = keyChars[rand.IntN(len(keyChars))]
	}

	return conv.BytesToString(key)
}

func getTokenOrder(order string) string {
	prefix, suffix, _ := strings.Cut(order, "-")
	switch prefix {
	case "name", "expired_at", "group", "used_amount", "request_count", "id", "created_at":
		switch suffix {
		case "asc":
			return prefix + " asc"
		default:
			return prefix + " desc"
		}
	default:
		return "id desc"
	}
}

func InsertToken(token *Token, autoCreateGroup, ignoreExist bool) error {
	if autoCreateGroup {
		group := &Group{
			ID: token.GroupID,
		}
		if err := OnConflictDoNothing().Create(group).Error; err != nil {
			return err
		}
	}

	maxTokenNum := config.GetGroupMaxTokenNum()

	err := DB.Transaction(func(tx *gorm.DB) error {
		if maxTokenNum > 0 {
			var count int64

			err := tx.Model(&Token{}).Where("group_id = ?", token.GroupID).Count(&count).Error
			if err != nil {
				return err
			}

			if count >= maxTokenNum {
				return errors.New("group max token num reached")
			}
		}

		if ignoreExist {
			return tx.
				Where("group_id = ? and name = ?", token.GroupID, token.Name).
				FirstOrCreate(token).Error
		}

		return tx.Create(token).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			if ignoreExist {
				return nil
			}
			return errors.New("token name already exists in this group")
		}

		return err
	}

	return nil
}

func GetTokens(
	group string,
	page, perPage int,
	order string,
	status int,
) (tokens []*Token, total int64, err error) {
	tx := DB.Model(&Token{})
	if group != "" {
		tx = tx.Where("group_id = ?", group)
	}

	if status != 0 {
		tx = tx.Where("status = ?", status)
	}

	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if total <= 0 {
		return nil, 0, nil
	}

	limit, offset := toLimitOffset(page, perPage)
	err = tx.Order(getTokenOrder(order)).Limit(limit).Offset(offset).Find(&tokens).Error

	return tokens, total, err
}

func SearchTokens(
	group, keyword string,
	page, perPage int,
	order string,
	status int,
	name, key string,
) (tokens []*Token, total int64, err error) {
	tx := DB.Model(&Token{})
	if group != "" {
		tx = tx.Where("group_id = ?", group)
	}

	if status != 0 {
		tx = tx.Where("status = ?", status)
	}

	if name != "" {
		tx = tx.Where("name = ?", name)
	}

	if key != "" {
		tx = tx.Where("key = ?", key)
	}

	if keyword != "" {
		var (
			conditions []string
			values     []any
		)

		if group == "" {
			if common.UsingPostgreSQL {
				conditions = append(conditions, "group_id ILIKE ?")
			} else {
				conditions = append(conditions, "group_id LIKE ?")
			}

			values = append(values, "%"+keyword+"%")
		}

		if name == "" {
			if common.UsingPostgreSQL {
				conditions = append(conditions, "name ILIKE ?")
			} else {
				conditions = append(conditions, "name LIKE ?")
			}

			values = append(values, "%"+keyword+"%")
		}

		if key == "" {
			if common.UsingPostgreSQL {
				conditions = append(conditions, "key ILIKE ?")
			} else {
				conditions = append(conditions, "key LIKE ?")
			}

			values = append(values, "%"+keyword+"%")
		}

		if common.UsingPostgreSQL {
			conditions = append(conditions, "models ILIKE ?")
		} else {
			conditions = append(conditions, "models LIKE ?")
		}

		values = append(values, "%"+keyword+"%")

		if len(conditions) > 0 {
			tx = tx.Where(fmt.Sprintf("(%s)", strings.Join(conditions, " OR ")), values...)
		}
	}

	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if total <= 0 {
		return nil, 0, nil
	}

	limit, offset := toLimitOffset(page, perPage)
	err = tx.Order(getTokenOrder(order)).Limit(limit).Offset(offset).Find(&tokens).Error

	return tokens, total, err
}

func SearchGroupTokens(
	group, keyword string,
	page, perPage int,
	order string,
	status int,
	name, key string,
) (tokens []*Token, total int64, err error) {
	if group == "" {
		return nil, 0, errors.New("group is empty")
	}

	tx := DB.Model(&Token{}).
		Where("group_id = ?", group)
	if name != "" {
		tx = tx.Where("name = ?", name)
	}

	if key != "" {
		tx = tx.Where("key = ?", key)
	}

	if status != 0 {
		tx = tx.Where("status = ?", status)
	}

	if keyword != "" {
		var (
			conditions []string
			values     []any
		)

		if name == "" {
			if common.UsingPostgreSQL {
				conditions = append(conditions, "name ILIKE ?")
			} else {
				conditions = append(conditions, "name LIKE ?")
			}

			values = append(values, "%"+keyword+"%")
		}

		if key == "" {
			if common.UsingPostgreSQL {
				conditions = append(conditions, "key ILIKE ?")
			} else {
				conditions = append(conditions, "key LIKE ?")
			}

			values = append(values, "%"+keyword+"%")
		}

		if common.UsingPostgreSQL {
			conditions = append(conditions, "models ILIKE ?")
		} else {
			conditions = append(conditions, "models LIKE ?")
		}

		values = append(values, "%"+keyword+"%")

		if len(conditions) > 0 {
			tx = tx.Where(fmt.Sprintf("(%s)", strings.Join(conditions, " OR ")), values...)
		}
	}

	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if total <= 0 {
		return nil, 0, nil
	}

	limit, offset := toLimitOffset(page, perPage)
	err = tx.Order(getTokenOrder(order)).Limit(limit).Offset(offset).Find(&tokens).Error

	return tokens, total, err
}

func GetTokenByKey(key string) (*Token, error) {
	if key == "" {
		return nil, errors.New("key is empty")
	}

	var token Token

	err := DB.Where("key = ?", key).First(&token).Error

	return &token, HandleNotFound(err, ErrTokenNotFound)
}

func ValidateAndGetToken(key string) (token *TokenCache, err error) {
	if key == "" {
		return nil, errors.New("no token provided")
	}

	token, err = CacheGetTokenByKey(key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid token")
		}

		log.Error("get token from cache failed: " + err.Error())

		return nil, errors.New("token validation failed")
	}

	if token.Status == TokenStatusDisabled {
		return nil, fmt.Errorf("token (%s[%d]) is disabled", token.Name, token.ID)
	}

	if !time.Time(token.ExpiredAt).IsZero() && time.Time(token.ExpiredAt).Before(time.Now()) {
		return nil, fmt.Errorf("token (%s[%d]) is expired", token.Name, token.ID)
	}

	if token.Quota > 0 && token.UsedAmount >= token.Quota {
		return nil, fmt.Errorf("token (%s[%d]) quota is exhausted", token.Name, token.ID)
	}

	return token, nil
}

func GetGroupTokenByID(group string, id int) (*Token, error) {
	if id == 0 || group == "" {
		return nil, errors.New("id or group is empty")
	}

	token := Token{}
	err := DB.
		Where("id = ? and group_id = ?", id, group).
		First(&token).Error

	return &token, HandleNotFound(err, ErrTokenNotFound)
}

func GetTokenByID(id int) (*Token, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}

	token := Token{ID: id}
	err := DB.First(&token, "id = ?", id).Error

	return &token, HandleNotFound(err, ErrTokenNotFound)
}

func UpdateTokenStatus(id, status int) (err error) {
	token := Token{ID: id}
	defer func() {
		if err == nil {
			if err := CacheUpdateTokenStatus(token.Key, status); err != nil {
				log.Error("update token status in cache failed: " + err.Error())
			}
		}
	}()

	result := DB.
		Model(&token).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
			},
		}).
		Where("id = ?", id).
		Updates(
			map[string]any{
				"status": status,
			},
		)

	return HandleUpdateResult(result, ErrTokenNotFound)
}

func UpdateGroupTokenStatus(group string, id, status int) (err error) {
	if id == 0 || group == "" {
		return errors.New("id or group is empty")
	}

	token := Token{}
	defer func() {
		if err == nil {
			if err := CacheUpdateTokenStatus(token.Key, status); err != nil {
				log.Error("update token status in cache failed: " + err.Error())
			}
		}
	}()

	result := DB.
		Model(&token).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
			},
		}).
		Where("id = ? and group_id = ?", id, group).
		Updates(
			map[string]any{
				"status": status,
			},
		)

	return HandleUpdateResult(result, ErrTokenNotFound)
}

func DeleteGroupTokenByID(groupID string, id int) (err error) {
	if id == 0 || groupID == "" {
		return errors.New("id or group is empty")
	}

	token := Token{ID: id, GroupID: groupID}
	defer func() {
		if err == nil {
			if err := CacheDeleteToken(token.Key); err != nil {
				log.Error("delete token from cache failed: " + err.Error())
			}
		}
	}()

	result := DB.
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
			},
		}).
		Where(token).
		Delete(&token)

	return HandleUpdateResult(result, ErrTokenNotFound)
}

func DeleteGroupTokensByIDs(group string, ids []int) (err error) {
	if group == "" {
		return errors.New("group is empty")
	}

	if len(ids) == 0 {
		return nil
	}

	tokens := make([]Token, len(ids))
	defer func() {
		if err == nil {
			for _, token := range tokens {
				if err := CacheDeleteToken(token.Key); err != nil {
					log.Error("delete token from cache failed: " + err.Error())
				}
			}
		}
	}()

	return DB.Transaction(func(tx *gorm.DB) error {
		return tx.
			Clauses(clause.Returning{
				Columns: []clause.Column{
					{Name: "key"},
				},
			}).
			Where("group_id = ?", group).
			Where("id IN (?)", ids).
			Delete(&tokens).
			Error
	})
}

func DeleteTokenByID(id int) (err error) {
	if id == 0 {
		return errors.New("id is empty")
	}

	token := Token{ID: id}
	defer func() {
		if err == nil {
			if err := CacheDeleteToken(token.Key); err != nil {
				log.Error("delete token from cache failed: " + err.Error())
			}
		}
	}()

	result := DB.
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
			},
		}).
		Where(token).
		Delete(&token)

	return HandleUpdateResult(result, ErrTokenNotFound)
}

func DeleteTokensByIDs(ids []int) (err error) {
	if len(ids) == 0 {
		return nil
	}

	tokens := make([]Token, len(ids))
	defer func() {
		if err == nil {
			for _, token := range tokens {
				if err := CacheDeleteToken(token.Key); err != nil {
					log.Error("delete token from cache failed: " + err.Error())
				}
			}
		}
	}()

	return DB.Transaction(func(tx *gorm.DB) error {
		return tx.
			Clauses(clause.Returning{
				Columns: []clause.Column{
					{Name: "key"},
				},
			}).
			Where("id IN (?)", ids).
			Delete(&tokens).
			Error
	})
}

func UpdateToken(id int, token *Token) (err error) {
	if id == 0 {
		return errors.New("id is empty")
	}

	defer func() {
		if err == nil {
			if err := CacheDeleteToken(token.Key); err != nil {
				log.Error("delete token from cache failed: " + err.Error())
			}
		}
	}()

	selects := []string{
		"subnets",
		"quota",
		"models",
		"expired_at",
	}
	if token.Name != "" {
		selects = append(selects, "name")
	}

	if token.Status != 0 {
		selects = append(selects, "status")
	}

	result := DB.
		Select(selects).
		Where("id = ?", id).
		Clauses(clause.Returning{}).
		Updates(token)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return errors.New("token name already exists in this group")
		}
	}

	return HandleUpdateResult(result, ErrTokenNotFound)
}

func UpdateGroupToken(id int, group string, token *Token) (err error) {
	if id == 0 || group == "" {
		return errors.New("id or group is empty")
	}

	defer func() {
		if err == nil {
			if err := CacheDeleteToken(token.Key); err != nil {
				log.Error("delete token from cache failed: " + err.Error())
			}
		}
	}()

	selects := []string{
		"subnets",
		"quota",
		"models",
		"expired_at",
	}
	if token.Name != "" {
		selects = append(selects, "name")
	}

	if token.Status != 0 {
		selects = append(selects, "status")
	}

	result := DB.
		Select(selects).
		Where("id = ? and group_id = ?", id, group).
		Clauses(clause.Returning{}).
		Updates(token)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return errors.New("token name already exists in this group")
		}
	}

	return HandleUpdateResult(result, ErrTokenNotFound)
}

func UpdateTokenUsedAmount(id int, amount float64, requestCount int) (err error) {
	token := &Token{}
	defer func() {
		if amount > 0 && err == nil && token.Quota > 0 {
			if err := CacheUpdateTokenUsedAmountOnlyIncrease(token.Key, token.UsedAmount); err != nil {
				log.Error("update token used amount in cache failed: " + err.Error())
			}
		}
	}()

	result := DB.
		Model(token).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
				{Name: "quota"},
				{Name: "used_amount"},
			},
		}).
		Where("id = ?", id).
		Updates(
			map[string]any{
				"used_amount":   gorm.Expr("used_amount + ?", amount),
				"request_count": gorm.Expr("request_count + ?", requestCount),
			},
		)

	return HandleUpdateResult(result, ErrTokenNotFound)
}

func UpdateTokenName(id int, name string) (err error) {
	token := &Token{ID: id}
	defer func() {
		if err == nil {
			if err := CacheUpdateTokenName(token.Key, name); err != nil {
				log.Error("update token name in cache failed: " + err.Error())
			}
		}
	}()

	result := DB.
		Model(token).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
			},
		}).
		Where("id = ?", id).
		Update("name", name)
	if result.Error != nil && errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return errors.New("token name already exists in this group")
	}

	return HandleUpdateResult(result, ErrTokenNotFound)
}

func UpdateGroupTokenName(group string, id int, name string) (err error) {
	token := &Token{ID: id, GroupID: group}
	defer func() {
		if err == nil {
			if err := CacheUpdateTokenName(token.Key, name); err != nil {
				log.Error("update token name in cache failed: " + err.Error())
			}
		}
	}()

	result := DB.
		Model(token).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
			},
		}).
		Where("id = ? and group_id = ?", id, group).
		Update("name", name)
	if result.Error != nil && errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return errors.New("token name already exists in this group")
	}

	return HandleUpdateResult(result, ErrTokenNotFound)
}

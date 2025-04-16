package utils

import (
	"net/http"

	model "github.com/labring/aiproxy/core/relay/model"
)

func WrapErr(err error) *model.ErrorWithStatusCode {
	return &model.ErrorWithStatusCode{
		StatusCode: http.StatusInternalServerError,
		Error: model.Error{
			Message: err.Error(),
		},
	}
}

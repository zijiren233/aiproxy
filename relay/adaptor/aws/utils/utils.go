package utils

import (
	"net/http"

	model "github.com/labring/aiproxy/relay/model"
)

func WrapErr(err error) *model.ErrorWithStatusCode {
	return &model.ErrorWithStatusCode{
		StatusCode: http.StatusInternalServerError,
		Error: model.Error{
			Message: err.Error(),
		},
	}
}

package common

import (
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var fieldsPool = sync.Pool{
	New: func() any {
		return make(logrus.Fields, 6)
	},
}

func GetLogFields() logrus.Fields {
	fields, ok := fieldsPool.Get().(logrus.Fields)
	if !ok {
		panic(fmt.Sprintf("fields pool type error: %T, %v", fields, fields))
	}

	return fields
}

func PutLogFields(fields logrus.Fields) {
	clear(fields)
	fieldsPool.Put(fields)
}

func GetLogger(c *gin.Context) *logrus.Entry {
	if log, ok := c.Get("log"); ok {
		v, ok := log.(*logrus.Entry)
		if !ok {
			panic(fmt.Sprintf("log type error: %T, %v", v, v))
		}

		return v
	}

	entry := NewLogger()
	c.Set("log", entry)

	return entry
}

func NewLogger() *logrus.Entry {
	return &logrus.Entry{
		Logger: logrus.StandardLogger(),
		Data:   GetLogFields(),
	}
}

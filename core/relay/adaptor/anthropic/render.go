package anthropic

import (
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
)

func Data(c *gin.Context, data []byte) {
	if len(c.Errors) > 0 {
		return
	}
	if c.IsAborted() {
		return
	}
	c.Render(-1, &Anthropic{Data: data})
	c.Writer.Flush()
}

func ObjectData(c *gin.Context, event string, object any) error {
	if len(c.Errors) > 0 {
		return c.Errors.Last()
	}
	if c.IsAborted() {
		return errors.New("context aborted")
	}
	jsonData, err := sonic.Marshal(object)
	if err != nil {
		return fmt.Errorf("error marshalling object: %w", err)
	}
	c.Render(-1, &Anthropic{Data: jsonData})
	c.Writer.Flush()
	return nil
}

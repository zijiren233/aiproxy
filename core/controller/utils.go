package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func parsePageParams(c *gin.Context) (page, perPage int) {
	pageStr := c.Query("page")
	if pageStr == "" {
		pageStr = c.Query("p")
	}
	page, _ = strconv.Atoi(pageStr)
	perPage, _ = strconv.Atoi(c.Query("per_page"))
	return
}

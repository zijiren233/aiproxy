package common

import (
	"github.com/labring/aiproxy/core/common/env"
)

var (
	UsingSQLite     = false
	UsingPostgreSQL = false
	UsingMySQL      = false
)

var (
	SQLitePath        = env.String("SQLITE_PATH", "aiproxy.db")
	SQLiteBusyTimeout = env.Int64("SQLITE_BUSY_TIMEOUT", 3000)
)

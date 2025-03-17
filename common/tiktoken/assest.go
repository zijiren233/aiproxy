package tiktoken

import (
	"embed"
	"encoding/base64"
	"path"
	"strconv"
	"strings"

	"github.com/labring/aiproxy/common/conv"
	"github.com/pkoukk/tiktoken-go"
)

//go:embed assets
var aassets embed.FS

var (
	_                tiktoken.BpeLoader = (*embedBpeLoader)(nil)
	defaultBpeLoader                    = tiktoken.NewDefaultBpeLoader()
)

type embedBpeLoader struct{}

func (e *embedBpeLoader) LoadTiktokenBpe(tiktokenBpeFile string) (map[string]int, error) {
	embedPath := path.Join("assets", path.Base(tiktokenBpeFile))
	contents, err := aassets.ReadFile(embedPath)
	if err != nil {
		return defaultBpeLoader.LoadTiktokenBpe(tiktokenBpeFile)
	}
	bpeRanks := make(map[string]int)
	for _, line := range strings.Split(conv.BytesToString(contents), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, " ")
		token, err := base64.StdEncoding.DecodeString(parts[0])
		if err != nil {
			return nil, err
		}
		rank, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
		bpeRanks[string(token)] = rank
	}
	return bpeRanks, nil
}

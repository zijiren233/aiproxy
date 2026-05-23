package doubao

import (
	"strconv"
	"strings"
)

func firstPresent(values map[string]any, keys ...string) any {
	for _, key := range keys {
		value, ok := values[key]
		if ok && value != nil {
			return value
		}
	}

	return nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func intPtrFromAny(value any) *int {
	switch v := value.(type) {
	case int:
		return &v
	case int64:
		converted := int(v)
		return &converted
	case float64:
		converted := int(v)
		return &converted
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return nil
		}

		return &parsed
	default:
		return nil
	}
}

func boolPtrFromAny(value any) *bool {
	switch v := value.(type) {
	case bool:
		return &v
	case string:
		return boolPtrFromString(v)
	default:
		return nil
	}
}

func boolPtrFromString(value string) *bool {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return nil
	}

	return &parsed
}

func setOptionalInt(target **int, values ...string) {
	for _, value := range values {
		parsed := intPtrFromAny(value)
		if parsed != nil {
			*target = parsed
			return
		}
	}
}

func intFromPtr(value *int) int {
	if value == nil {
		return 0
	}

	return *value
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}

	return 0
}

func firstPositiveInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}

	return 0
}

func nestedURL(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		return stringFromAny(v["url"])
	default:
		return ""
	}
}

func nestedID(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		return firstNonEmptyString(stringFromAny(v["id"]), stringFromAny(v["task_id"]))
	default:
		return ""
	}
}

func openAIAudioToDoubaoURL(value any) *doubaoVideoURLContent {
	audio, ok := value.(map[string]any)
	if !ok {
		return nil
	}

	if url := stringFromAny(audio["url"]); url != "" {
		return &doubaoVideoURLContent{URL: url}
	}

	data := stringFromAny(audio["data"])
	if data == "" {
		return nil
	}

	if strings.HasPrefix(data, "data:audio/") {
		return &doubaoVideoURLContent{URL: data}
	}

	format := strings.TrimSpace(stringFromAny(audio["format"]))
	if format == "" {
		format = "wav"
	}

	return &doubaoVideoURLContent{
		URL: "data:audio/" + strings.ToLower(format) + ";base64," + data,
	}
}

func doubaoVideoResolutionFromSize(size string) string {
	size = normalizeDoubaoSize(size)
	switch size {
	case "480p", "720p", "1080p":
		return size
	}

	width, height, ok := dimensionsFromSize(size)
	if !ok {
		return ""
	}

	shortSide := min(width, height)
	switch {
	case shortSide >= 1000:
		return "1080p"
	case shortSide >= 700:
		return "720p"
	case shortSide >= 400:
		return "480p"
	default:
		return ""
	}
}

func ratioFromSize(size string) string {
	size = normalizeDoubaoSize(size)
	switch size {
	case "720p", "1080p", "480p", "":
		return ""
	}

	width, height, ok := strings.Cut(size, "x")
	if !ok {
		return ""
	}

	w, h, ok := dimensionsFromParts(width, height)
	if !ok || h == 0 {
		return ""
	}

	switch {
	case w*9 == h*16:
		return "16:9"
	case w*16 == h*9:
		return "9:16"
	case w == h:
		return "1:1"
	case w*3 == h*4:
		return "4:3"
	case w*4 == h*3:
		return "3:4"
	default:
		return ""
	}
}

func dimensionsFromSize(size string) (int, int, bool) {
	width, height, ok := strings.Cut(normalizeDoubaoSize(size), "x")
	if !ok {
		return 0, 0, false
	}

	return dimensionsFromParts(width, height)
}

func normalizeDoubaoSize(size string) string {
	size = strings.ToLower(strings.TrimSpace(size))
	size = strings.ReplaceAll(size, "×", "x")
	size = strings.ReplaceAll(size, "*", "x")

	return size
}

func dimensionsFromParts(width, height string) (int, int, bool) {
	w, err := strconv.Atoi(strings.TrimSpace(width))
	if err != nil {
		return 0, 0, false
	}

	h, err := strconv.Atoi(strings.TrimSpace(height))
	if err != nil {
		return 0, 0, false
	}

	return w, h, true
}

package controller

import (
	"slices"
	"strconv"
	"strings"

	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

type resolutionAliasFunc func(string) []string

func supportedImageResolutionMatches(
	resolution string,
	supported []string,
	fuzzy bool,
) bool {
	resolution = normalizeSupportedResolutionValue(resolution)
	if resolution == "" || len(supported) == 0 {
		return true
	}

	if !imageResolutionValue(resolution) {
		return false
	}

	supported = normalizeSupportedImageResolutionValues(supported)
	if len(supported) == 0 {
		return false
	}

	if slices.Contains(supported, resolution) {
		return true
	}

	if !fuzzy {
		return false
	}

	supportedAliases := supportedResolutionAliases(supported, supportedImageResolutionAliases)
	for _, alias := range requestImageResolutionAliases(resolution) {
		if slices.Contains(supportedAliases, alias) {
			return true
		}
	}

	return false
}

func supportedResolutionMatches(
	resolution string,
	supported []string,
	supportedAliasesFunc resolutionAliasFunc,
	requestAliasesFunc resolutionAliasFunc,
	fuzzy bool,
) bool {
	resolution = normalizeSupportedResolutionValue(resolution)
	if resolution == "" || len(supported) == 0 {
		return true
	}

	supportedExact := normalizeSupportedResolutionValues(supported)
	if len(supportedExact) == 0 {
		return false
	}

	if slices.Contains(supportedExact, resolution) {
		return true
	}

	if !fuzzy {
		return false
	}

	supportedAliases := supportedResolutionAliases(supported, supportedAliasesFunc)
	for _, alias := range requestAliasesFunc(resolution) {
		if slices.Contains(supportedAliases, alias) {
			return true
		}
	}

	return false
}

func normalizeSupportedImageResolutionValues(resolutions []string) []string {
	normalized := make([]string, 0, len(resolutions))
	for _, resolution := range resolutions {
		resolution = normalizeSupportedResolutionValue(resolution)
		if resolution != "" && imageResolutionValue(resolution) {
			normalized = append(normalized, resolution)
		}
	}

	slices.Sort(normalized)

	return slices.Compact(normalized)
}

func normalizeSupportedResolutionValues(resolutions []string) []string {
	normalized := make([]string, 0, len(resolutions))
	for _, resolution := range resolutions {
		resolution = normalizeSupportedResolutionValue(resolution)
		if resolution != "" {
			normalized = append(normalized, resolution)
		}
	}

	slices.Sort(normalized)

	return slices.Compact(normalized)
}

func normalizeSupportedResolutionValue(resolution string) string {
	resolution = strings.ToLower(strings.TrimSpace(resolution))
	resolution = strings.ReplaceAll(resolution, " ", "")
	resolution = strings.ReplaceAll(resolution, "×", "x")
	resolution = strings.ReplaceAll(resolution, "*", "x")

	return resolution
}

func validateOpenAIImageResolutionFormat(resolution string) error {
	resolution = strings.ToLower(strings.TrimSpace(resolution))
	if resolution == "" || resolution == "auto" || dimensionResolutionValue(resolution) {
		return nil
	}

	return NewBadRequestParamError("invalid image resolution `" + resolution + "`")
}

func dimensionResolutionValue(resolution string) bool {
	width, height, ok := parseOpenAIDimensions(resolution)
	return ok && width > 0 && height > 0
}

func parseOpenAIDimensions(resolution string) (int, int, bool) {
	widthText, heightText, ok := strings.Cut(strings.TrimSpace(resolution), "x")
	if !ok || widthText == "" || heightText == "" || strings.Contains(heightText, "x") {
		return 0, 0, false
	}

	width, err := strconv.Atoi(widthText)
	if err != nil {
		return 0, 0, false
	}

	height, err := strconv.Atoi(heightText)
	if err != nil {
		return 0, 0, false
	}

	return width, height, true
}

func supportedResolutionAliases(
	resolutions []string,
	aliases resolutionAliasFunc,
) []string {
	result := make([]string, 0, len(resolutions)*3)
	for _, resolution := range resolutions {
		result = append(result, aliases(normalizeSupportedResolutionValue(resolution))...)
	}

	slices.Sort(result)

	return slices.Compact(result)
}

func supportedImageResolutionAliases(resolution string) []string {
	if resolution == "" {
		return nil
	}

	result := make([]string, 0, 2)
	if imageResolutionValue(resolution) {
		result = append(result, resolution)
	}

	if imageSize := imageResolutionSizeAlias(resolution); imageSize != "" {
		result = append(result, "size:"+imageSize)
	}

	return slices.Compact(result)
}

func requestImageResolutionAliases(resolution string) []string {
	if resolution == "" {
		return nil
	}

	result := make([]string, 0, 2)
	if imageResolutionValue(resolution) {
		result = append(result, resolution)
	}

	if imageSize := imageResolutionSizeAlias(resolution); imageSize != "" {
		result = append(result, "size:"+imageSize)
	}

	return slices.Compact(result)
}

func imageResolutionValue(resolution string) bool {
	switch resolution {
	case "512", "1k", "2k", "4k":
		return true
	}

	return dimensionResolutionValue(resolution)
}

func imageResolutionSizeAlias(resolution string) string {
	switch resolution {
	case "512", "1k", "2k", "4k":
		return resolution
	}

	width, height, ok := relaymodel.ParseVideoDimensions(resolution)
	if !ok || width <= 0 || height <= 0 {
		return ""
	}

	longSide := max(width, height)
	switch {
	case longSide >= 3500:
		return "4k"
	case longSide >= 1500:
		return "2k"
	case longSide >= 900:
		return "1k"
	default:
		return "512"
	}
}

func videoResolutionAliases(resolution string) []string {
	if resolution == "" {
		return nil
	}

	result := []string{resolution}
	if width, height, ok := relaymodel.ParseVideoDimensions(resolution); ok {
		if tier := relaymodel.VideoResolutionFromDimensions(width, height); tier != "" {
			result = append(result, tier)
		}
	}

	return slices.Compact(result)
}

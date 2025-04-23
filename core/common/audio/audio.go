package audio

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/labring/aiproxy/core/common/config"
	"github.com/sirupsen/logrus"
)

var (
	ErrAudioDurationNAN = errors.New("audio duration is N/A")
	re                  = regexp.MustCompile(`time=(\d+:\d+:\d+\.\d+)`)
)

func GetAudioDuration(audio io.Reader) (float64, error) {
	logrus.Debug(config.FfmpegEnabled)
	if !config.FfmpegEnabled {
		return 0, nil
	}

	ffprobeCmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		"-i", "-",
	)
	ffprobeCmd.Stdin = audio
	output, err := ffprobeCmd.Output()
	if err != nil {
		return 0, err
	}

	str := strings.TrimSpace(string(output))

	if str == "" || str == "N/A" {
		seeker, ok := audio.(io.Seeker)
		if !ok {
			return 0, ErrAudioDurationNAN
		}
		_, err := seeker.Seek(0, io.SeekStart)
		if err != nil {
			return 0, ErrAudioDurationNAN
		}
		return getAudioDurationFallback(audio)
	}

	duration, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, err
	}
	return duration, nil
}

func getAudioDurationFallback(audio io.Reader) (float64, error) {
	if !config.FfmpegEnabled {
		return 0, nil
	}

	ffmpegCmd := exec.Command(
		"ffmpeg",
		"-i", "-",
		"-f", "null", "-",
	)
	ffmpegCmd.Stdin = audio
	var stderr bytes.Buffer
	ffmpegCmd.Stderr = &stderr
	err := ffmpegCmd.Run()
	if err != nil {
		return 0, err
	}

	logrus.Debug(stderr.String())

	// Parse the time from ffmpeg output
	// Example: size=N/A time=00:00:05.52 bitrate=N/A speed= 785x
	return parseTimeFromFfmpegOutput(stderr.String())
}

func GetAudioDurationFromFilePath(filePath string) (float64, error) {
	if !config.FfmpegEnabled {
		return 0, nil
	}

	ffprobeCmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		"-i", filePath,
	)
	output, err := ffprobeCmd.Output()
	if err != nil {
		return 0, err
	}

	str := strings.TrimSpace(string(output))

	if str == "" || str == "N/A" {
		return getAudioDurationFromFilePathFallback(filePath)
	}

	duration, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, err
	}
	return duration, nil
}

func getAudioDurationFromFilePathFallback(filePath string) (float64, error) {
	if !config.FfmpegEnabled {
		return 0, nil
	}

	ffmpegCmd := exec.Command(
		"ffmpeg",
		"-i", filePath,
		"-f", "null", "-",
	)

	var stderr bytes.Buffer
	ffmpegCmd.Stderr = &stderr
	err := ffmpegCmd.Run()
	if err != nil {
		return 0, err
	}

	// Parse the time from ffmpeg output
	return parseTimeFromFfmpegOutput(stderr.String())
}

// parseTimeFromFfmpegOutput extracts and converts time from ffmpeg output to seconds
func parseTimeFromFfmpegOutput(output string) (float64, error) {
	match := re.FindStringSubmatch(output)
	if len(match) < 2 {
		return 0, ErrAudioDurationNAN
	}

	// Convert time format HH:MM:SS.MS to seconds
	timeStr := match[1]
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0, errors.New("invalid time format")
	}

	hours, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}

	minutes, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, err
	}

	seconds, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0, err
	}

	duration := hours*3600 + minutes*60 + seconds
	return duration, nil
}

//nolint:testpackage
package controller

import (
	"reflect"
	"testing"
	"time"

	relaycontroller "github.com/labring/aiproxy/core/relay/controller"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryStateRemainingRelayDelay(t *testing.T) {
	t.Parallel()

	t.Run("uses per-channel failure count plus jitter", func(t *testing.T) {
		t.Parallel()

		state := &retryState{}
		base := time.Unix(100, 0)
		jitter := 400 * time.Millisecond

		state.recordChannelFailure(7, base)

		assert.Equal(t, 1400*time.Millisecond, state.remainingRelayDelay(7, base, jitter))
		assert.Equal(
			t,
			900*time.Millisecond,
			state.remainingRelayDelay(7, base.Add(500*time.Millisecond), jitter),
		)

		state.recordChannelFailure(7, base.Add(2*time.Second))

		assert.Equal(
			t,
			2400*time.Millisecond,
			state.remainingRelayDelay(7, base.Add(2*time.Second), jitter),
		)
		assert.Equal(
			t,
			1150*time.Millisecond,
			state.remainingRelayDelay(7, base.Add(3250*time.Millisecond), jitter),
		)
	})

	t.Run("returns zero after required wait has already elapsed", func(t *testing.T) {
		t.Parallel()

		state := &retryState{}
		base := time.Unix(200, 0)

		state.recordChannelFailure(9, base)

		assert.Zero(
			t,
			state.remainingRelayDelay(9, base.Add(1500*time.Millisecond), 400*time.Millisecond),
		)
		assert.Zero(t, state.remainingRelayDelay(9, base.Add(2*time.Second), 400*time.Millisecond))
	})

	t.Run("tracks each channel independently", func(t *testing.T) {
		t.Parallel()

		state := &retryState{}
		base := time.Unix(300, 0)

		state.recordChannelFailure(1, base)
		state.recordChannelFailure(2, base)
		state.recordChannelFailure(2, base.Add(100*time.Millisecond))

		assert.Equal(
			t,
			500*time.Millisecond,
			state.remainingRelayDelay(1, base.Add(800*time.Millisecond), 300*time.Millisecond),
		)
		assert.Equal(
			t,
			1500*time.Millisecond,
			state.remainingRelayDelay(2, base.Add(900*time.Millisecond), 300*time.Millisecond),
		)
		assert.Zero(t, state.remainingRelayDelay(3, base, 300*time.Millisecond))
	})

	t.Run("caps backoff at five seconds", func(t *testing.T) {
		t.Parallel()

		base := time.Unix(400, 0)
		state := &retryState{}

		for range 20 {
			state.recordChannelFailure(5, base)
		}

		assert.Equal(t, 5*time.Second, state.remainingRelayDelay(5, base, time.Second))
		assert.Zero(t, state.remainingRelayDelay(5, base.Add(5*time.Second), time.Second))
	})
}

func TestCalculateRelayBackoffDelay(t *testing.T) {
	t.Parallel()

	assert.Zero(t, calculateRelayBackoffDelay(0, 500*time.Millisecond))
	assert.Equal(t, 1500*time.Millisecond, calculateRelayBackoffDelay(1, 500*time.Millisecond))
	assert.Equal(t, 2500*time.Millisecond, calculateRelayBackoffDelay(2, 500*time.Millisecond))
	assert.Equal(t, 5*time.Second, calculateRelayBackoffDelay(20, time.Second))
	assert.Equal(t, 2*time.Second, calculateRelayBackoffDelay(1, time.Second))
}

func TestRelayControllerVideoModesValidateRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mode mode.Mode
		want ValidateRequest
	}{
		{
			name: "video generation jobs",
			mode: mode.VideoGenerationsJobs,
			want: relaycontroller.ValidateVideoGenerationJobRequest,
		},
		{
			name: "videos",
			mode: mode.Videos,
			want: relaycontroller.ValidateVideosRequest,
		},
		{
			name: "videos remix",
			mode: mode.VideosRemix,
			want: relaycontroller.ValidateVideosRequest,
		},
		{
			name: "gemini video",
			mode: mode.GeminiVideo,
			want: relaycontroller.ValidateGeminiVideoRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rc := relayController(tt.mode)
			require.NotNil(t, rc.ValidateRequest)
			require.Equal(
				t,
				reflect.ValueOf(tt.want).Pointer(),
				reflect.ValueOf(rc.ValidateRequest).Pointer(),
			)
		})
	}
}

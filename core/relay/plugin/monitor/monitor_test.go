//nolint:testpackage
package monitor

import (
	"net/http"
	"testing"

	"github.com/labring/aiproxy/core/common/config"
	relaymeta "github.com/labring/aiproxy/core/relay/meta"
	"github.com/stretchr/testify/require"
)

func TestGetChannelWarnErrorRateUsesChannelValueEvenWhenAutoBalanceDisabled(t *testing.T) {
	meta := &relaymeta.Meta{}
	meta.Channel.WarnErrorRate = 0.42
	meta.Channel.EnabledAutoBalanceCheck = false
	meta.Channel.MaxErrorRate = 0.95

	require.InDelta(t, 0.42, getChannelWarnErrorRate(meta), 0.0001)
	require.InDelta(t, 0.95, getChannelMaxErrorRate(meta), 0.0001)
}

func TestGetChannelWarnErrorRateFallsBackToDefault(t *testing.T) {
	previous := config.GetDefaultWarnNotifyErrorRate()
	config.SetDefaultWarnNotifyErrorRate(previous)
	t.Cleanup(func() {
		config.SetDefaultWarnNotifyErrorRate(previous)
	})

	config.SetDefaultWarnNotifyErrorRate(0.37)

	require.InDelta(t, 0.37, getChannelWarnErrorRate(&relaymeta.Meta{}), 0.0001)
}

func TestGetChannelMaxErrorRateDoesNotDependOnBalanceCheckSwitch(t *testing.T) {
	meta := &relaymeta.Meta{}
	meta.Channel.WarnErrorRate = 0.25
	meta.Channel.MaxErrorRate = 0.88

	require.InDelta(t, 0.88, getChannelMaxErrorRate(meta), 0.0001)

	meta.Channel.EnabledAutoBalanceCheck = true

	require.InDelta(t, 0.88, getChannelMaxErrorRate(meta), 0.0001)
}

func TestShouldTryBanNoPermissionRequiresChannelSwitch(t *testing.T) {
	meta := &relaymeta.Meta{}

	require.False(t, shouldTryBanNoPermission(meta, false))

	meta.Channel.EnabledNoPermissionBan = true

	require.True(t, shouldTryBanNoPermission(meta, false))
	require.False(t, shouldTryBanNoPermission(meta, true))
}

func TestChannelStatusHasPermission(t *testing.T) {
	t.Parallel()

	for _, statusCode := range []int{
		http.StatusUnauthorized,
		http.StatusPaymentRequired,
		http.StatusForbidden,
		http.StatusNotFound,
	} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			t.Parallel()

			require.False(t, ChannelStatusHasPermission(statusCode))
		})
	}

	require.True(t, ChannelStatusHasPermission(http.StatusBadRequest))
}

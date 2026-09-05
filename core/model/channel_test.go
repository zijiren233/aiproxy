//nolint:testpackage
package model

import (
	"path/filepath"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestChannelBackupOnlyPersistence(t *testing.T) {
	db, err := OpenSQLite(filepath.Join(t.TempDir(), "channels.db"))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)

	previousDB := DB
	previousCaches := LoadModelCaches()
	DB = db
	t.Cleanup(func() {
		DB = previousDB

		modelCaches.Store(previousCaches)
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(&Channel{}, &ModelConfig{}))
	require.NoError(t, db.Create(&ModelConfig{Model: "backup-test", Type: mode.Responses}).Error)

	channel := &Channel{
		Name:   "test-channel",
		Type:   ChannelTypeOpenAI,
		Models: []string{"backup-test"},
	}
	require.NoError(t, BatchInsertChannels([]*Channel{channel}))

	loaded, err := GetChannelByID(channel.ID)
	require.NoError(t, err)
	require.False(t, loaded.BackupOnly)

	for _, backupOnly := range []bool{true, false} {
		channel.BackupOnly = backupOnly
		require.NoError(t, UpdateChannel(channel))
		loaded, err = GetChannelByID(channel.ID)
		require.NoError(t, err)
		require.Equal(t, backupOnly, loaded.BackupOnly)

		infos, err := GetChannelsBasicInfoByIDs([]int{channel.ID})
		require.NoError(t, err)
		require.Len(t, infos, 1)
		require.Equal(t, backupOnly, infos[0].BackupOnly)

		mc := LoadModelCaches()
		require.Contains(t, mc.EnabledModelsBySet[ChannelDefaultSet], "backup-test")
		require.Contains(t, mc.EnabledModelConfigsMap, "backup-test")
		require.Len(t, mc.EnabledModelConfigsBySet[ChannelDefaultSet], 1)
		require.Equal(
			t,
			backupOnly,
			mc.EnabledModel2ChannelsBySet[ChannelDefaultSet]["backup-test"][0].BackupOnly,
		)
	}
}

func TestChannelBackupOnlyYAMLAndJSON(t *testing.T) {
	t.Parallel()

	var item ChannelItem
	require.NoError(t, yaml.Unmarshal([]byte("name: backup\nbackup_only: true\n"), &item))
	require.True(t, item.BackupOnly)

	data, err := sonic.Marshal(&item.Channel)
	require.NoError(t, err)

	var decoded struct {
		BackupOnly bool `json:"backup_only"`
	}
	require.NoError(t, sonic.Unmarshal(data, &decoded))
	require.True(t, decoded.BackupOnly)
}

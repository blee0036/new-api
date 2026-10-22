package model

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedFlowQuotaData(t *testing.T, quotaData QuotaData) {
	t.Helper()
	require.NoError(t, DB.Create(&quotaData).Error)
}

func seedFlowLookupData(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Create(&Channel{Id: 1, Name: "east"}).Error)
	require.NoError(t, DB.Create(&Channel{Id: 2, Name: "west"}).Error)
	require.NoError(t, DB.Create(&Token{Id: 11, UserId: 1, Key: "sk-primary", Name: "primary"}).Error)
	require.NoError(t, DB.Create(&Token{Id: 22, UserId: 2, Key: "sk-backup", Name: "backup"}).Error)
	require.NoError(t, DB.Delete(&Token{Id: 11}).Error)
}

func TestGetFlowQuotaDataUsesQuotaDataRoleSpecificDimensions(t *testing.T) {
	truncateTables(t)
	seedFlowLookupData(t)

	seedFlowQuotaData(t, QuotaData{
		UserID:    1,
		Username:  "alice",
		NodeName:  "node-a",
		TokenID:   11,
		UseGroup:  "vip",
		ModelName: "gpt-a",
		ChannelID: 1,
		CreatedAt: 1000,
		Count:     2,
		Quota:     100,
		TokenUsed: 40,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID:    1,
		Username:  "alice",
		NodeName:  "node-a",
		TokenID:   11,
		UseGroup:  "vip",
		ModelName: "gpt-a",
		ChannelID: 1,
		CreatedAt: 1100,
		Count:     1,
		Quota:     50,
		TokenUsed: 20,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID:    1,
		Username:  "alice",
		NodeName:  "node-a",
		TokenID:   11,
		UseGroup:  "vip",
		ModelName: "gpt-a",
		ChannelID: 2,
		CreatedAt: 1200,
		Count:     1,
		Quota:     25,
		TokenUsed: 10,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID:    2,
		Username:  "bob",
		NodeName:  "node-b",
		TokenID:   22,
		UseGroup:  "default",
		ModelName: "gpt-b",
		ChannelID: 1,
		CreatedAt: 1300,
		Count:     3,
		Quota:     70,
		TokenUsed: 30,
	})
	seedFlowQuotaData(t, QuotaData{
		UserID:    1,
		Username:  "alice",
		ModelName: "legacy",
		CreatedAt: 1400,
		Count:     99,
		Quota:     999,
		TokenUsed: 999,
	})

	rootRows, err := GetFlowQuotaData(900, 2000, "", 0, common.RoleRootUser)
	require.NoError(t, err)
	require.Len(t, rootRows, 3)
	// Token 11 was soft-deleted, so its name is intentionally left empty for the
	// frontend to render a localized "deleted (id)" label instead.
	require.Equal(t, FlowQuotaData{
		UserID:      1,
		Username:    "alice",
		NodeName:    "node-a",
		TokenID:     11,
		TokenName:   "",
		UseGroup:    "vip",
		ChannelID:   1,
		ChannelName: "east",
		ModelName:   "gpt-a",
		TokenUsed:   60,
		Count:       3,
		Quota:       150,
	}, *rootRows[0])
	// A token that still exists resolves to its current name.
	require.Equal(t, 22, rootRows[1].TokenID)
	require.Equal(t, "backup", rootRows[1].TokenName)

	adminRows, err := GetFlowQuotaData(900, 2000, "alice", 0, common.RoleAdminUser)
	require.NoError(t, err)
	require.Len(t, adminRows, 2)
	require.Equal(t, 0, adminRows[0].TokenID)
	require.Empty(t, adminRows[0].TokenName)
	require.Empty(t, adminRows[0].NodeName)
	require.Equal(t, "alice", adminRows[0].Username)
	require.Equal(t, "vip", adminRows[0].UseGroup)
	require.Equal(t, "east", adminRows[0].ChannelName)
	require.Equal(t, 150, adminRows[0].Quota)

	selfRows, err := GetFlowQuotaData(900, 2000, "", 1, common.RoleCommonUser)
	require.NoError(t, err)
	require.Len(t, selfRows, 1)
	require.Empty(t, selfRows[0].Username)
	require.Equal(t, 0, selfRows[0].ChannelID)
	require.Empty(t, selfRows[0].ChannelName)
	require.Empty(t, selfRows[0].TokenName)
	require.Equal(t, "vip", selfRows[0].UseGroup)
	require.Equal(t, 175, selfRows[0].Quota)
}

func TestLogQuotaDataSplitsRowsByUseGroupTokenChannelAndNode(t *testing.T) {
	truncateTables(t)
	CacheQuotaDataLock.Lock()
	CacheQuotaData = make(map[string]*QuotaData)
	CacheQuotaDataLock.Unlock()

	LogQuotaData(QuotaDataLogParams{
		UserID:    1,
		Username:  "alice",
		ModelName: "gpt-a",
		CreatedAt: 3661,
		UseGroup:  "vip",
		TokenID:   11,
		ChannelID: 1,
		NodeName:  "node-a",
		Quota:     100,
		TokenUsed: 40,
	})
	LogQuotaData(QuotaDataLogParams{
		UserID:    1,
		Username:  "alice",
		ModelName: "gpt-a",
		CreatedAt: 3700,
		UseGroup:  "vip",
		TokenID:   11,
		ChannelID: 1,
		NodeName:  "node-a",
		Quota:     50,
		TokenUsed: 20,
	})
	LogQuotaData(QuotaDataLogParams{
		UserID:    1,
		Username:  "alice",
		ModelName: "gpt-a",
		CreatedAt: 3700,
		UseGroup:  "default",
		TokenID:   11,
		ChannelID: 1,
		NodeName:  "node-a",
		Quota:     25,
		TokenUsed: 10,
	})

	SaveQuotaDataCache()

	var rows []QuotaData
	require.NoError(t, DB.Order("quota DESC").Find(&rows).Error)
	require.Len(t, rows, 2)
	require.Equal(t, int64(3600), rows[0].CreatedAt)
	require.Equal(t, "vip", rows[0].UseGroup)
	require.Equal(t, 11, rows[0].TokenID)
	require.Equal(t, 1, rows[0].ChannelID)
	require.Equal(t, "node-a", rows[0].NodeName)
	require.Equal(t, 2, rows[0].Count)
	require.Equal(t, 150, rows[0].Quota)
	require.Equal(t, 60, rows[0].TokenUsed)
	require.Equal(t, "default", rows[1].UseGroup)
	require.Equal(t, 25, rows[1].Quota)
}

func TestRecordConsumeLogPreservesCacheUsageAcrossWriteModes(t *testing.T) {
	for _, mode := range []string{"immediate", "batched", "full queue"} {
		t.Run(mode, func(t *testing.T) {
			truncateTables(t)
			previousQueue := consumeLogQueue
			previousExport, previousConsume, previousNode := common.DataExportEnabled, common.LogConsumeEnabled, common.NodeName
			CacheQuotaDataLock.Lock()
			previousCache := CacheQuotaData
			CacheQuotaData = make(map[string]*QuotaData)
			CacheQuotaDataLock.Unlock()
			t.Cleanup(func() {
				consumeLogQueue = previousQueue
				common.DataExportEnabled, common.LogConsumeEnabled, common.NodeName = previousExport, previousConsume, previousNode
				CacheQuotaDataLock.Lock()
				CacheQuotaData = previousCache
				CacheQuotaDataLock.Unlock()
			})
			common.DataExportEnabled, common.LogConsumeEnabled, common.NodeName = true, true, "node-cache"
			consumeLogQueue = nil
			switch mode {
			case "batched":
				consumeLogQueue = make(chan consumeLogBatchItem, 1)
			case "full queue":
				consumeLogQueue = make(chan consumeLogBatchItem)
			}

			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Set("username", "alice")
			common.SetContextKey(ctx, constant.ContextKeyUserSetting, dto.UserSetting{})
			other := NewLogOther()
			other.MergePublic(map[string]any{
				"cache_tokens":             30,
				"cache_write_tokens":       70,
				"cache_creation_tokens":    999,
				"cache_creation_tokens_5m": 20,
				"cache_creation_tokens_1h": 50,
			})
			RecordConsumeLog(ctx, 7, RecordConsumeLogParams{
				ModelName: "cache-model", PromptTokens: 200, CompletionTokens: 10,
				Quota: 100, Group: "vip", TokenId: 11, ChannelId: 2, Other: other,
			})
			if mode == "batched" {
				require.Len(t, consumeLogQueue, 1)
				writeConsumeLogBatch([]consumeLogBatchItem{<-consumeLogQueue})
			}
			SaveQuotaDataCache()

			var logs []Log
			require.NoError(t, LOG_DB.Find(&logs).Error)
			require.Len(t, logs, 1)
			assert.Equal(t, 100, logs[0].Quota)
			var rows []QuotaData
			require.NoError(t, DB.Find(&rows).Error)
			require.Len(t, rows, 1)
			row := rows[0]
			assert.Equal(t, 1, row.Count)
			assert.Equal(t, 100, row.Quota)
			assert.Equal(t, 210, row.TokenUsed)
			assert.Equal(t, int64(30), row.CacheReadTokens)
			assert.Equal(t, int64(70), row.CacheCreationTokens)
			assert.Equal(t, 7, row.UserID)
			assert.Equal(t, "alice", row.Username)
			assert.Equal(t, "cache-model", row.ModelName)
			assert.Equal(t, "vip", row.UseGroup)
			assert.Equal(t, 11, row.TokenID)
			assert.Equal(t, 2, row.ChannelID)
			assert.Equal(t, "node-cache", row.NodeName)
		})
	}
}

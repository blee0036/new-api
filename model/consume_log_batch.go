package model

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/bytedance/gopkg/util/gopool"
)

type consumeLogBatchItem struct {
	log        *Log
	dataExport *consumeLogDataExport
}

type consumeLogDataExport struct {
	userId    int
	username  string
	modelName string
	quota     int
	createdAt int64
	tokenUsed int
}

var (
	consumeLogBatchOnce sync.Once
	consumeLogQueue     chan consumeLogBatchItem
)

const (
	defaultConsumeLogQueueSize = 10000
	defaultConsumeLogBatchSize = 200
)

func InitConsumeLogBatcher() {
	if !common.BatchUpdateEnabled {
		return
	}
	consumeLogBatchOnce.Do(func() {
		intervalSeconds := common.BatchUpdateInterval
		if intervalSeconds <= 0 {
			intervalSeconds = 5
		}

		consumeLogQueue = make(chan consumeLogBatchItem, defaultConsumeLogQueueSize)
		gopool.Go(func() {
			consumeLogBatchWorker(defaultConsumeLogBatchSize, time.Duration(intervalSeconds)*time.Second)
		})
		common.SysLog(fmt.Sprintf("consume log batch enabled: queue=%d, batch=%d, interval=%ds", defaultConsumeLogQueueSize, defaultConsumeLogBatchSize, intervalSeconds))
	})
}

func enqueueConsumeLog(item consumeLogBatchItem) bool {
	if consumeLogQueue == nil || item.log == nil {
		return false
	}
	select {
	case consumeLogQueue <- item:
		return true
	default:
		return false
	}
}

func consumeLogBatchWorker(batchSize int, flushInterval time.Duration) {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	batch := make([]consumeLogBatchItem, 0, batchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		writeConsumeLogBatch(batch)
		batch = batch[:0]
	}

	for {
		select {
		case item := <-consumeLogQueue:
			if item.log == nil {
				continue
			}
			batch = append(batch, item)
			if len(batch) >= batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func writeConsumeLogBatch(items []consumeLogBatchItem) {
	logs := make([]*Log, 0, len(items))
	for _, item := range items {
		if item.log != nil {
			logs = append(logs, item.log)
		}
	}
	if len(logs) == 0 {
		return
	}

	if err := LOG_DB.CreateInBatches(logs, len(logs)).Error; err != nil {
		common.SysLog("failed to record consume log batch: " + err.Error())
		for _, logItem := range logs {
			if err := LOG_DB.Create(logItem).Error; err != nil {
				common.SysLog("failed to record consume log: " + err.Error())
			}
		}
	}

	for _, item := range items {
		recordConsumeLogDataExport(item.dataExport)
	}
}

func writeConsumeLog(item consumeLogBatchItem) {
	if item.log == nil {
		return
	}
	if err := LOG_DB.Create(item.log).Error; err != nil {
		common.SysLog("failed to record consume log: " + err.Error())
	}
	recordConsumeLogDataExport(item.dataExport)
}

func recordConsumeLogDataExport(data *consumeLogDataExport) {
	if data == nil || !common.DataExportEnabled {
		return
	}
	LogQuotaData(data.userId, data.username, data.modelName, data.quota, data.createdAt, data.tokenUsed)
}

package service

import (
	"testing"
	"time"
)

func TestSyncService(t *testing.T) {
	// 创建同步服务实例
	syncService := NewSyncService()

	// 检查初始状态
	if syncService.IsRunning() {
		t.Error("初始状态应该是未运行")
	}

	stats := syncService.GetSyncStats()
	if stats.TotalPins != 0 {
		t.Error("初始总pin数应该为0")
	}

	t.Log("同步服务初始化测试通过")
}

func TestSyncPinsByTimeRange(t *testing.T) {
	// 这个测试需要真实的区块链连接和数据库，所以跳过
	t.Skip("跳过需要真实区块链连接和数据库的测试")

	syncService := NewSyncService()

	// 设置测试时间范围 (最近1小时)
	now := time.Now()
	endTs := now.Unix()
	startTs := now.Add(-1 * time.Hour).Unix()

	t.Logf("测试时间范围: %d - %d", startTs, endTs)

	// 测试同步pin数据
	err := syncService.SyncPinsByTimeRangeWithBatch(startTs, endTs, nil, nil, 3600)
	if err != nil {
		t.Errorf("启动同步失败: %v", err)
		return
	}

	// 检查是否开始运行
	if !syncService.IsRunning() {
		t.Error("同步服务应该正在运行")
	}

	// 等待一段时间查看进度
	time.Sleep(5 * time.Second)

	stats := syncService.GetSyncStats()
	t.Logf("同步状态: 总数 %d, 已处理 %d, 成功 %d, 失败 %d",
		stats.TotalPins, stats.ProcessedPins, stats.SuccessPins, stats.FailedPins)

	// 等待同步完成
	for syncService.IsRunning() {
		time.Sleep(1 * time.Second)
		stats := syncService.GetSyncStats()
		t.Logf("进度: %d/%d (%.1f%%)",
			stats.ProcessedPins, stats.TotalPins,
			float64(stats.ProcessedPins)/float64(stats.TotalPins)*100)
	}

	finalStats := syncService.GetSyncStats()
	t.Logf("最终结果: 总数 %d, 成功 %d, 失败 %d, 完成时间: %v",
		finalStats.TotalPins, finalStats.SuccessPins, finalStats.FailedPins, finalStats.EndTime)
}

func TestSyncPinsByTimeRangeWithBatch(t *testing.T) {
	// 这个测试需要真实的区块链连接和数据库，所以跳过
	t.Skip("跳过需要真实区块链连接和数据库的测试")

	syncService := NewSyncService()

	// 设置测试时间范围 (最近3小时)
	now := time.Now()
	endTs := now.Unix()
	startTs := now.Add(-3 * time.Hour).Unix()

	t.Logf("测试时间范围: %d - %d", startTs, endTs)

	// 测试分批同步pin数据，每批30分钟
	batchDuration := int64(30 * 60) // 30分钟
	err := syncService.SyncPinsByTimeRangeWithBatch(startTs, endTs, nil, nil, batchDuration)
	if err != nil {
		t.Errorf("启动分批同步失败: %v", err)
		return
	}

	// 检查是否开始运行
	if !syncService.IsRunning() {
		t.Error("同步服务应该正在运行")
	}

	// 等待一段时间查看进度
	time.Sleep(5 * time.Second)

	stats := syncService.GetSyncStats()
	t.Logf("分批同步状态: 总数 %d, 已处理 %d, 成功 %d, 失败 %d",
		stats.TotalPins, stats.ProcessedPins, stats.SuccessPins, stats.FailedPins)

	// 等待同步完成
	for syncService.IsRunning() {
		time.Sleep(2 * time.Second)
		stats := syncService.GetSyncStats()
		t.Logf("分批进度: %d/%d (%.1f%%)",
			stats.ProcessedPins, stats.TotalPins,
			float64(stats.ProcessedPins)/float64(stats.TotalPins)*100)
	}

	finalStats := syncService.GetSyncStats()
	t.Logf("分批最终结果: 总数 %d, 成功 %d, 失败 %d, 完成时间: %v",
		finalStats.TotalPins, finalStats.SuccessPins, finalStats.FailedPins, finalStats.EndTime)
}

func TestConcurrentSync(t *testing.T) {
	syncService := NewSyncService()

	// 测试重复启动同步
	now := time.Now()
	endTs := now.Unix()
	startTs := now.Add(-1 * time.Hour).Unix()

	// 第一次启动
	err1 := syncService.SyncPinsByTimeRangeWithBatch(startTs, endTs, nil, nil, 3600)
	if err1 != nil {
		t.Logf("第一次启动结果: %v", err1)
	}

	// 立即尝试第二次启动，应该失败
	err2 := syncService.SyncPinsByTimeRangeWithBatch(startTs, endTs, nil, nil, 3600)
	if err2 == nil {
		t.Error("第二次启动应该失败")
	} else {
		t.Logf("第二次启动正确失败: %v", err2)
	}
}

func TestStopSync(t *testing.T) {
	syncService := NewSyncService()

	// 测试在未运行状态下停止
	err := syncService.StopSync()
	if err == nil {
		t.Error("未运行状态下停止应该失败")
	} else {
		t.Logf("未运行状态下停止正确失败: %v", err)
	}

	// 测试在运行状态下停止
	now := time.Now()
	endTs := now.Unix()
	startTs := now.Add(-1 * time.Hour).Unix()

	// 启动同步
	err = syncService.SyncPinsByTimeRangeWithBatch(startTs, endTs, nil, nil, 3600)
	if err != nil {
		t.Logf("启动同步结果: %v", err)
	}

	// 等待一小段时间
	time.Sleep(100 * time.Millisecond)

	// 检查是否在运行
	if !syncService.IsRunning() {
		t.Error("同步服务应该正在运行")
	}

	// 停止同步
	err = syncService.StopSync()
	if err != nil {
		t.Errorf("停止同步失败: %v", err)
	}

	// 等待停止完成
	time.Sleep(100 * time.Millisecond)

	// 检查是否已停止
	if syncService.IsRunning() {
		t.Error("同步服务应该已停止")
	}

	t.Log("停止同步测试完成")
}

func TestGetSyncStats(t *testing.T) {
	syncService := NewSyncService()

	// 测试获取统计信息
	stats := syncService.GetSyncStats()

	if stats.TotalPins != 0 {
		t.Error("初始总pin数应该为0")
	}

	if stats.ProcessedPins != 0 {
		t.Error("初始已处理pin数应该为0")
	}

	if stats.IsCompleted != false {
		t.Error("初始完成状态应该为false")
	}

	t.Log("统计信息测试通过")
}

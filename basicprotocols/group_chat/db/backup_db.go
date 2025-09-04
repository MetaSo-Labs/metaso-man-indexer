package db

import (
	"fmt"
	"log"
	"manindexer/common"
	"os"
	"path/filepath"
	"time"

	"github.com/cockroachdb/pebble"
)

// BackupDB handles database backup operations
type BackupDB struct {
	backupDir       string
	backupHour      int
	backupRetention int
	stopChan        chan bool
}

// NewBackupDB creates a new BackupDB instance
func NewBackupDB(backupDir string) *BackupDB {
	// Get backup configuration from config
	backupHour := 3
	backupRetention := 7

	if common.Config != nil {
		backupHour = common.Config.GroupChat.BackupHour
		backupRetention = common.Config.GroupChat.BackupRetention
	}

	// Set default values if not configured
	if backupHour < 0 || backupHour > 23 {
		backupHour = 3
	}
	if backupRetention <= 0 {
		backupRetention = 7
	}

	return &BackupDB{
		backupDir:       backupDir,
		backupHour:      backupHour,
		backupRetention: backupRetention,
		stopChan:        make(chan bool),
	}
}

// InitBackupDB initializes and starts the backup system
// This method should be called after database initialization
func InitBackupDB() error {
	var dbPath string
	if common.Config != nil && common.Config.Pebble.Dir != "" {
		dbPath = common.Config.Pebble.Dir
	} else {
		// Use default path
		dbPath = "./data"
	}
	// Create backup directory
	backupDir := filepath.Join(dbPath, "group_chat_backup")
	err := os.MkdirAll(backupDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	// Create backup DB instance
	backupDB := NewBackupDB(backupDir)

	// Start backup scheduler
	go backupDB.startBackupScheduler()

	log.Printf("[backup_db] Database backup system initialized, backup directory: %s", backupDir)
	return nil
}

// startBackupScheduler starts the backup scheduler that runs at configured time daily
func (bdb *BackupDB) startBackupScheduler() {
	log.Printf("[backup_db] Starting backup scheduler...")

	for {
		// Calculate next backup time using configured hour and minute
		now := time.Now()
		nextBackup := time.Date(now.Year(), now.Month(), now.Day(), bdb.backupHour, 0, 0, 0, now.Location())

		// If it's already past the configured time today, schedule for tomorrow
		if now.After(nextBackup) {
			nextBackup = nextBackup.Add(24 * time.Hour)
		}

		// Wait until next backup time
		waitDuration := nextBackup.Sub(now)
		log.Printf("[backup_db] Next backup scheduled at: %s (in %v)", nextBackup.Format("2006-01-02 15:04:05"), waitDuration)

		select {
		case <-time.After(waitDuration):
			// Perform backup
			err := bdb.performBackup()
			if err != nil {
				log.Printf("[backup_db] Backup failed: %v", err)
			} else {
				log.Printf("[backup_db] Backup completed successfully")
			}
		case <-bdb.stopChan:
			log.Printf("[backup_db] Backup scheduler stopped")
			return
		}
	}
}

// performBackup performs a complete backup of all group chat collections
func (bdb *BackupDB) performBackup() error {
	log.Printf("[backup_db] Starting database backup...")

	// Create backup timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupPath := filepath.Join(bdb.backupDir, "backup_"+timestamp)

	// Create backup directory
	err := os.MkdirAll(backupPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	// Define all collections to backup
	collections := []string{
		// Community related collections
		TalkCommunityVersionInfoCollection,
		TalkCommunityInfoCollection,
		TalkCommunityAddressCollection,
		TalkCommunityJoinCollection,
		TalkCommunityPersonCollection,

		// Group related collections
		TalkGroupInfoCollection,
		TalkGroupVersionInfoCollection,
		TalkGroupCommunityCollection,
		TalkGroupMetaIdJoinCollection,
		TalkGroupJoinCollection,
		TalkGroupPersonCollection,
		TalkMetaIdContextListCollection,
		TalkGroupLatestChatCollection,
		TalkGroupRemoveUserCollection,

		// Message queue collections
		TalkGroupChatQueueCollection,
		TalkGroupOpenLuckyBagQueueCollection,
		TalkGroupResidueLuckyBagQueueCollection,

		// Chat related collections
		TalkGroupChatPinCollection,
		TalkGroupLuckyBagPinCollection,
		TalkGroupOpenLuckyBagPinCollection,
		TalkGroupResidueLuckyBagPinCollection,
		TalkGroupOpenLuckyBagListCollection,
		TalkGroupResidueLuckyBagListCollection,
		TalkGroupChatTimestampCollection,
		TalkGroupChatTimestampOutCollection,
		TalkGroupChatTimestamp2Collection,
		TalkGroupChatTimestamp2OutCollection,

		// Lucky bag residue related collections
		TalkGroupLuckyBagPinPendingCollection,
		TalkGroupLuckyBagPinCompletedCollection,
		TalkGroupLuckyBagPinTimeoutResidueCollection,
		TalkGroupLuckyBagPinErrPendingCollection,
		TalkGroupLuckyBagPinErrTimeoutResidueCollection,
		TalkGroupLuckyBagCodeAddressKeyCollection,
		TalkGroupLuckyBagCodeAddressKeyCompletedCollection,
		TalkPrivateChatBlockPinCollection,

		// Private chat collections
		TalkPrivateChatPinCollection,
		TalkPrivateChatTimestampCollection,
		TalkPrivateChatTimestampOutCollection,
		TalkPrivateChatQueueCollection,
		TalkPrivateChatMetaIdBlockListCollection,
		TalkPrivateChatBlockPinCollection,

		// Version info collection
		TalkVersionInfoCollection,

		// Index collections
		TalkGroupChatIndexCollection,
		TalkPrivateChatIndexCollection,

		// User info collections
		TalkUserAddressChatPublicKeyCollection,
		TalkUserMetaIdChatPublicKeyCollection,
	}

	// Backup each collection
	successCount := 0
	totalCount := len(collections)

	for _, collectionName := range collections {
		err := bdb.backupCollection(collectionName, backupPath)
		if err != nil {
			log.Printf("[backup_db] Failed to backup collection %s: %v", collectionName, err)
		} else {
			successCount++
			log.Printf("[backup_db] Successfully backed up collection: %s", collectionName)
		}
	}

	// Create backup summary
	err = bdb.createBackupSummary(backupPath, timestamp, successCount, totalCount)
	if err != nil {
		log.Printf("[backup_db] Failed to create backup summary: %v", err)
	}

	// Clean up old backups based on configured retention period
	err = bdb.cleanupOldBackups()
	if err != nil {
		log.Printf("[backup_db] Failed to cleanup old backups: %v", err)
	}

	log.Printf("[backup_db] Backup completed: %d/%d collections backed up to %s", successCount, totalCount, backupPath)
	return nil
}

// backupCollection backs up a single collection
func (bdb *BackupDB) backupCollection(collectionName, backupPath string) error {
	// Get source database
	sourceDB, exists := Pb[collectionName]
	if !exists {
		return fmt.Errorf("collection %s does not exist", collectionName)
	}

	// Create backup database path
	backupDBPath := filepath.Join(backupPath, collectionName)

	// Create backup database
	backupDB, err := pebble.Open(backupDBPath, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to create backup database for %s: %v", collectionName, err)
	}
	defer backupDB.Close()

	// Create iterator for source database
	iter, err := sourceDB.NewIter(nil)
	if err != nil {
		return fmt.Errorf("failed to create iterator for %s: %v", collectionName, err)
	}
	defer iter.Close()

	// Create batch for backup database
	batch := backupDB.NewBatch()
	defer batch.Close()

	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Write to backup database
		err := batch.Set(key, value, nil)
		if err != nil {
			return fmt.Errorf("failed to write to backup database: %v", err)
		}

		count++

		// Commit batch every 1000 records
		if count%1000 == 0 {
			err = batch.Commit(pebble.Sync)
			if err != nil {
				return fmt.Errorf("failed to commit batch: %v", err)
			}
			batch = backupDB.NewBatch()
		}
	}

	// Commit remaining batch
	if count%1000 != 0 {
		err = batch.Commit(pebble.Sync)
		if err != nil {
			return fmt.Errorf("failed to commit final batch: %v", err)
		}
	}

	log.Printf("[backup_db] Backed up %d records from collection %s", count, collectionName)
	return nil
}

// createBackupSummary creates a summary file for the backup
func (bdb *BackupDB) createBackupSummary(backupPath, timestamp string, successCount, totalCount int) error {
	summaryPath := filepath.Join(backupPath, "backup_summary.txt")

	summary := fmt.Sprintf(`Group Chat Database Backup Summary
==========================================
Backup Time: %s
Backup Path: %s
Collections Backed Up: %d/%d
Status: %s

Collections:
- TalkCommunityVersionInfoCollection
- TalkCommunityInfoCollection
- TalkCommunityAddressCollection
- TalkCommunityJoinCollection
- TalkCommunityPersonCollection
- TalkGroupInfoCollection
- TalkGroupVersionInfoCollection
- TalkGroupCommunityCollection
- TalkGroupMetaIdJoinCollection
- TalkGroupJoinCollection
- TalkGroupPersonCollection
- TalkMetaIdContextListCollection
- TalkGroupLatestChatCollection
- TalkGroupChatQueueCollection
- TalkGroupOpenLuckyBagQueueCollection
- TalkGroupResidueLuckyBagQueueCollection
- TalkGroupChatPinCollection
- TalkGroupLuckyBagPinCollection
- TalkGroupOpenLuckyBagPinCollection
- TalkGroupResidueLuckyBagPinCollection
- TalkGroupOpenLuckyBagListCollection
- TalkGroupResidueLuckyBagListCollection
- TalkGroupChatTimestampCollection
- TalkGroupChatTimestampOutCollection
- TalkGroupChatTimestamp2Collection
- TalkGroupChatTimestamp2OutCollection
- TalkPrivateChatPinCollection
- TalkPrivateChatTimestampCollection
- TalkPrivateChatTimestampOutCollection
- TalkPrivateChatQueueCollection
- TalkPrivateChatMetaIdBlockListCollection
- TalkVersionInfoCollection
- TalkGroupLuckyBagCodeAddressKeyCollection
- TalkGroupLuckyBagCodeAddressKeyCompletedCollection

Backup completed at: %s
`,
		timestamp,
		backupPath,
		successCount,
		totalCount,
		func() string {
			if successCount == totalCount {
				return "SUCCESS"
			}
			return "PARTIAL"
		}(),
		time.Now().Format("2006-01-02 15:04:05"))

	return os.WriteFile(summaryPath, []byte(summary), 0644)
}

// cleanupOldBackups removes backups older than configured retention period
func (bdb *BackupDB) cleanupOldBackups() error {
	entries, err := os.ReadDir(bdb.backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %v", err)
	}

	cutoffTime := time.Now().AddDate(0, 0, -bdb.backupRetention) // configured days ago
	removedCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if it's a backup directory
		if !isBackupDirectory(entry.Name()) {
			continue
		}

		// Get directory info
		info, err := entry.Info()
		if err != nil {
			log.Printf("[backup_db] Failed to get info for %s: %v", entry.Name(), err)
			continue
		}

		// Check if backup is older than configured retention period
		if info.ModTime().Before(cutoffTime) {
			backupPath := filepath.Join(bdb.backupDir, entry.Name())
			err := os.RemoveAll(backupPath)
			if err != nil {
				log.Printf("[backup_db] Failed to remove old backup %s: %v", entry.Name(), err)
			} else {
				log.Printf("[backup_db] Removed old backup: %s", entry.Name())
				removedCount++
			}
		}
	}

	if removedCount > 0 {
		log.Printf("[backup_db] Cleaned up %d old backups", removedCount)
	}

	return nil
}

// isBackupDirectory checks if a directory name follows the backup naming pattern
func isBackupDirectory(name string) bool {
	return len(name) > 7 && name[:7] == "backup_"
}

// StopBackup stops the backup scheduler
func (bdb *BackupDB) StopBackup() {
	close(bdb.stopChan)
	log.Printf("[backup_db] Backup system stopped")
}

// GetBackupStatus returns the current backup status
func (bdb *BackupDB) GetBackupStatus() map[string]interface{} {
	// Count existing backups
	entries, err := os.ReadDir(bdb.backupDir)
	backupCount := 0
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() && isBackupDirectory(entry.Name()) {
				backupCount++
			}
		}
	}

	return map[string]interface{}{
		"backup_dir":   bdb.backupDir,
		"backup_count": backupCount,
		"next_backup":  getNextBackupTime(),
	}
}

// getNextBackupTime calculates the next backup time
func getNextBackupTime() string {
	now := time.Now()
	// Use default values if config is not available
	backupHour := 3
	backupMinute := 0

	if common.Config != nil {
		backupHour = common.Config.GroupChat.BackupHour
	}

	nextBackup := time.Date(now.Year(), now.Month(), now.Day(), backupHour, backupMinute, 0, 0, now.Location())

	if now.After(nextBackup) {
		nextBackup = nextBackup.Add(24 * time.Hour)
	}

	return nextBackup.Format("2006-01-02 15:04:05")
}

// ManualBackup performs a manual backup (can be called via API)
func (bdb *BackupDB) ManualBackup() error {
	log.Printf("[backup_db] Manual backup requested")
	return bdb.performBackup()
}

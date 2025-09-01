package db

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/pebble"
)

const (
	LatestTargetVersion = 5
)

// Version migration record structure
type VersionMigration struct {
	Version     int    `json:"version"`
	Description string `json:"description"`
	AppliedAt   int64  `json:"applied_at"`
}

// Migration operation interface
type Migration interface {
	Version() int
	Description() string
	Execute() error
}

// Migration from version 1 to version 2: Copy timestamp collection data
type MigrationV1ToV2 struct{}

func (m *MigrationV1ToV2) Version() int {
	return 2
}

func (m *MigrationV1ToV2) Description() string {
	return "Copy TalkGroupChatTimestampCollection data to TalkGroupChatTimestamp2Collection, and TalkGroupChatTimestampOutCollection data to TalkGroupChatTimestamp2OutCollection"
}

func (m *MigrationV1ToV2) Execute() error {
	log.Printf("[migrate_db] Starting version 2 migration: %s", m.Description())

	// Copy TalkGroupChatTimestampCollection to TalkGroupChatTimestamp2Collection
	err := copyCollectionData(TalkGroupChatTimestampCollection, TalkGroupChatTimestamp2Collection)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to copy TalkGroupChatTimestampCollection: %v", err)
	}

	// Copy TalkGroupChatTimestampOutCollection to TalkGroupChatTimestamp2OutCollection
	err = copyCollectionData(TalkGroupChatTimestampOutCollection, TalkGroupChatTimestamp2OutCollection)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to copy TalkGroupChatTimestampOutCollection: %v", err)
	}

	log.Printf("[migrate_db] Version 2 migration completed")
	return nil
}

// Migration from version 2 to version 3: Rebuild timestamp2 collections with new key format
type MigrationV2ToV3 struct{}

func (m *MigrationV2ToV3) Version() int {
	return 3
}

func (m *MigrationV2ToV3) Description() string {
	return "Rebuild TalkGroupChatTimestamp2Collection and TalkGroupChatTimestamp2OutCollection with new key format (groupId_timestamp+number(6)) from original collections"
}

func (m *MigrationV2ToV3) Execute() error {
	log.Printf("[migrate_db] Starting version 3 migration: %s", m.Description())

	// Clear existing TalkGroupChatTimestamp2Collection and rebuild from TalkGroupChatTimestampCollection
	err := rebuildTimestamp2Collection(TalkGroupChatTimestampCollection, TalkGroupChatTimestamp2Collection)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to rebuild TalkGroupChatTimestamp2Collection: %v", err)
	}

	// Clear existing TalkGroupChatTimestamp2OutCollection and rebuild from TalkGroupChatTimestampOutCollection
	err = rebuildTimestamp2Collection(TalkGroupChatTimestampOutCollection, TalkGroupChatTimestamp2OutCollection)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to rebuild TalkGroupChatTimestamp2OutCollection: %v", err)
	}

	log.Printf("[migrate_db] Version 3 migration completed")
	return nil
}

// Migration from version 3 to version 4: Rebuild index collections from timestamp collections
type MigrationV3ToV4 struct{}

func (m *MigrationV3ToV4) Version() int {
	return 4
}

func (m *MigrationV3ToV4) Description() string {
	return "Rebuild TalkGroupChatIndexCollection and TalkPrivateChatIndexCollection from timestamp collections"
}

func (m *MigrationV3ToV4) Execute() error {
	log.Printf("[migrate_db] Starting version 4 migration: %s", m.Description())

	// Rebuild group chat index collection from TalkGroupChatTimestamp2Collection
	err := rebuildGroupChatIndexCollection()
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to rebuild TalkGroupChatIndexCollection: %v", err)
	}

	// Rebuild private chat index collection from TalkPrivateChatTimestampCollection
	err = rebuildPrivateChatIndexCollection()
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to rebuild TalkPrivateChatIndexCollection: %v", err)
	}

	log.Printf("[migrate_db] Version 4 migration completed")
	return nil
}

// Migration from version 4 to version 5: Rebuild index collections with zero-padded format
type MigrationV4ToV5 struct{}

func (m *MigrationV4ToV5) Version() int {
	return 5
}

func (m *MigrationV4ToV5) Description() string {
	return "Rebuild TalkGroupChatIndexCollection and TalkPrivateChatIndexCollection with zero-padded index format for proper sorting"
}

func (m *MigrationV4ToV5) Execute() error {
	log.Printf("[migrate_db] Starting version 5 migration: %s", m.Description())

	// Rebuild group chat index collection with zero-padded format
	err := rebuildGroupChatIndexCollectionV5()
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to rebuild TalkGroupChatIndexCollection: %v", err)
	}

	// Rebuild private chat index collection with zero-padded format
	err = rebuildPrivateChatIndexCollectionV5()
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to rebuild TalkPrivateChatIndexCollection: %v", err)
	}

	log.Printf("[migrate_db] Version 5 migration completed")
	return nil
}

// Generic function to copy collection data
func copyCollectionData(sourceCollection, targetCollection string) error {
	sourceDB, exists := Pb[sourceCollection]
	if !exists {
		return fmt.Errorf("[migrate_db] source collection %s does not exist", sourceCollection)
	}

	targetDB, exists := Pb[targetCollection]
	if !exists {
		return fmt.Errorf("[migrate_db] target collection %s does not exist", targetCollection)
	}

	// Use iterator to traverse all data in source collection
	iter, err := sourceDB.NewIter(nil)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to create source collection iterator: %v", err)
	}
	defer iter.Close()

	batch := targetDB.NewBatch()
	defer batch.Close()

	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Write data to target collection
		err := batch.Set(key, value, nil)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to write to target collection: %v", err)
		}

		count++

		// Commit batch every 1000 records
		if count%1000 == 0 {
			err = batch.Commit(pebble.Sync)
			if err != nil {
				return fmt.Errorf("[migrate_db] failed to commit batch: %v", err)
			}
			batch = targetDB.NewBatch()
			log.Printf("Copied %d records from %s to %s", count, sourceCollection, targetCollection)
		}
	}

	// Commit remaining batch
	if count%1000 != 0 {
		err := batch.Commit(pebble.Sync)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to commit final batch: %v", err)
		}
	}

	log.Printf("[migrate_db] Successfully copied %d records from %s to %s", count, sourceCollection, targetCollection)
	return nil
}

// Rebuild timestamp2 collection with new key format from original collection
func rebuildTimestamp2Collection(sourceCollection, targetCollection string) error {
	sourceDB, exists := Pb[sourceCollection]
	if !exists {
		return fmt.Errorf("[migrate_db] source collection %s does not exist", sourceCollection)
	}

	targetDB, exists := Pb[targetCollection]
	if !exists {
		return fmt.Errorf("[migrate_db] target collection %s does not exist", targetCollection)
	}

	// Clear target collection first
	log.Printf("[migrate_db] Clearing target collection %s", targetCollection)
	err := clearCollection(targetDB)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to clear target collection: %v", err)
	}

	// Use iterator to traverse all data in source collection
	iter, err := sourceDB.NewIter(nil)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to create source collection iterator: %v", err)
	}
	defer iter.Close()

	batch := targetDB.NewBatch()
	defer batch.Close()

	count := 0
	// timestampCounter := make(map[string]int) // Track counter for each timestamp

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Parse original key: groupId_timestamp
		keyStr := string(key)
		keyParts := strings.Split(keyStr, "_")
		if len(keyParts) < 2 {
			log.Printf("[migrate_db] Skipping invalid key format: %s", keyStr)
			continue
		}

		groupId := keyParts[0]
		timestampStr := keyParts[1]

		// Parse original value: pinId_chatType_timestamp
		valueStr := string(value)
		valueParts := strings.Split(valueStr, "_")
		if len(valueParts) < 3 {
			log.Printf("[migrate_db] Skipping invalid value format: %s", valueStr)
			continue
		}

		pinId := valueParts[0]
		chatType := valueParts[1]
		timestamp := valueParts[2]

		// Generate counter for this timestamp
		// timestampKey := groupId + "_" + timestampStr
		// timestampCounter[timestampKey]++
		//counter := timestampCounter[timestampKey]

		// Generate 6-digit random number (using counter for consistency)
		randomNum := generateRandomNumber(6)

		// Construct new key: groupId_timestamp+number(6)
		newKey := groupId + "_" + timestampStr + randomNum

		// Construct new value: pinId_chatType_timestamp_number
		newValue := pinId + "_" + chatType + "_" + timestamp + "_" + randomNum

		// Write data to target collection
		err := batch.Set([]byte(newKey), []byte(newValue), nil)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to write to target collection: %v", err)
		}

		count++

		// Commit batch every 1000 records
		if count%1000 == 0 {
			err = batch.Commit(pebble.Sync)
			if err != nil {
				return fmt.Errorf("[migrate_db] failed to commit batch: %v", err)
			}
			batch = targetDB.NewBatch()
			log.Printf("[migrate_db] Rebuilt %d records from %s to %s", count, sourceCollection, targetCollection)
		}
	}

	// Commit remaining batch
	if count%1000 != 0 {
		err := batch.Commit(pebble.Sync)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to commit final batch: %v", err)
		}
	}

	log.Printf("[migrate_db] Successfully rebuilt %d records from %s to %s", count, sourceCollection, targetCollection)
	return nil
}

// Clear all data from a collection
func clearCollection(db *pebble.DB) error {
	iter, err := db.NewIter(nil)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to create iterator for clearing: %v", err)
	}
	defer iter.Close()

	batch := db.NewBatch()
	defer batch.Close()

	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		err := batch.Delete(key, nil)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to delete key: %v", err)
		}
		count++

		// Commit batch every 1000 deletions
		if count%1000 == 0 {
			err = batch.Commit(pebble.Sync)
			if err != nil {
				return fmt.Errorf("[migrate_db] failed to commit deletion batch: %v", err)
			}
			batch = db.NewBatch()
		}
	}

	// Commit remaining deletions
	if count%1000 != 0 {
		err = batch.Commit(pebble.Sync)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to commit final deletion batch: %v", err)
		}
	}

	log.Printf("[migrate_db] Cleared %d records from collection", count)
	return nil
}

// Get current database version
func GetCurrentVersion() (int, error) {
	db, exists := Pb[TalkVersionInfoCollection]
	if !exists {
		return 0, fmt.Errorf("version info collection does not exist")
	}

	// Find the latest version record
	iter, err := db.NewIter(nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create version info iterator: %v", err)
	}
	defer iter.Close()

	var maxVersion int
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		versionStr := string(key)

		// Try to parse version number
		if version, err := strconv.Atoi(versionStr); err == nil {
			if version > maxVersion {
				maxVersion = version
			}
		}
	}

	return maxVersion, nil
}

// Set database version
func SetVersion(version int) error {
	db, exists := Pb[TalkVersionInfoCollection]
	if !exists {
		return fmt.Errorf("version info collection does not exist")
	}

	versionStr := strconv.Itoa(version)
	versionBytes := []byte(versionStr)

	err := db.Set(versionBytes, versionBytes, pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to set version: %v", err)
	}

	log.Printf("[migrate_db] Database version set to: %d", version)
	return nil
}

// Record migration history
func RecordMigration(migration Migration, timestamp int64) error {
	db, exists := Pb[TalkVersionInfoCollection]
	if !exists {
		return fmt.Errorf("version info collection does not exist")
	}

	migrationRecord := VersionMigration{
		Version:     migration.Version(),
		Description: migration.Description(),
		AppliedAt:   timestamp,
	}

	recordBytes, err := json.Marshal(migrationRecord)
	if err != nil {
		return fmt.Errorf("failed to serialize migration record: %v", err)
	}

	key := fmt.Sprintf("migration_%d", migration.Version())
	err = db.Set([]byte(key), recordBytes, pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to record migration history: %v", err)
	}

	log.Printf("[migrate_db] Migration history recorded: version %d", migration.Version())
	return nil
}

// Get all migration records
func GetMigrationHistory() ([]VersionMigration, error) {
	db, exists := Pb[TalkVersionInfoCollection]
	if !exists {
		return nil, fmt.Errorf("version info collection does not exist")
	}

	var migrations []VersionMigration
	iter, err := db.NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration history iterator: %v", err)
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := string(iter.Key())
		if strings.HasPrefix(key, "migration_") {
			var migration VersionMigration
			err := json.Unmarshal(iter.Value(), &migration)
			if err != nil {
				log.Printf("Failed to parse migration record: %v", err)
				continue
			}
			migrations = append(migrations, migration)
		}
	}

	return migrations, nil
}

// Execute database migration
func MigrateDatabase(targetVersion int) error {
	currentVersion, err := GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %v", err)
	}

	log.Printf("[migrate_db] Current database version: %d, target version: %d", currentVersion, targetVersion)

	if currentVersion >= targetVersion {
		log.Printf("[migrate_db] Database is already at latest version, no migration needed")
		return nil
	}

	// Define all migration operations
	migrations := []Migration{
		&MigrationV1ToV2{},
		&MigrationV2ToV3{},
		&MigrationV3ToV4{},
		&MigrationV4ToV5{},
		// Add more migration operations here
		// Note: When adding new migrations, also update the LatestTargetVersion constant
	}

	// Execute migrations in version order
	for _, migration := range migrations {
		if migration.Version() > currentVersion && migration.Version() <= targetVersion {
			log.Printf("[migrate_db] Starting migration to version %d", migration.Version())

			err := migration.Execute()
			if err != nil {
				return fmt.Errorf("failed to execute version %d migration: %v", migration.Version(), err)
			}

			// Record migration history
			timestamp := getCurrentTimestamp()
			err = RecordMigration(migration, timestamp)
			if err != nil {
				log.Printf("[migrate_db] Failed to record migration history: %v", err)
			}

			// Update current version
			currentVersion = migration.Version()
			err = SetVersion(currentVersion)
			if err != nil {
				return fmt.Errorf("failed to update version number: %v", err)
			}

			log.Printf("[migrate_db] Successfully migrated to version %d", migration.Version())
		}
	}

	log.Printf("[migrate_db] Database migration completed, current version: %d", currentVersion)
	return nil
}

// Get current timestamp (milliseconds)
func getCurrentTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// Initialize database version (if database is brand new)
func InitializeDatabaseVersion() error {
	currentVersion, err := GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %v", err)
	}

	if currentVersion == 0 {
		// Database is brand new, set to version 1
		err = SetVersion(1)
		if err != nil {
			return fmt.Errorf("failed to initialize database version: %v", err)
		}
		log.Printf("[migrate_db] Database version initialized to: 1")
	}

	return nil
}

// Check and migrate database to latest version
// This method should be called at application startup to automatically check current version and execute necessary migrations
func CheckAndMigrateDatabase() error {
	// First validate migration configuration
	err := validateMigrationConfig()
	if err != nil {
		return fmt.Errorf("migration configuration validation failed: %v", err)
	}

	// Initialize database version (if it's a new database)
	err = InitializeDatabaseVersion()
	if err != nil {
		return fmt.Errorf("failed to initialize database version: %v", err)
	}

	// Get current version
	currentVersion, err := GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %v", err)
	}

	// Use configured target version
	targetVersion := LatestTargetVersion

	log.Printf("[migrate_db] Checking database migration: current version %d, target version %d", currentVersion, targetVersion)

	// If current version is already at latest, no migration needed
	if currentVersion >= targetVersion {
		log.Printf("[migrate_db] Database is already at latest version, no migration needed")
		return nil
	}

	// Execute migration
	err = MigrateDatabase(targetVersion)
	if err != nil {
		return fmt.Errorf("database migration failed: %v", err)
	}

	log.Printf("[migrate_db] Database migration completed, upgraded to version %d", targetVersion)
	return nil
}

// Validate migration configuration consistency
func validateMigrationConfig() error {
	// Define all migration operations
	migrations := []Migration{
		&MigrationV1ToV2{},
		&MigrationV2ToV3{},
		&MigrationV3ToV4{},
		&MigrationV4ToV5{},
		// Add more migration operations here
	}

	// // Check if all migration version numbers do not exceed LatestTargetVersion
	// for _, migration := range migrations {
	// 	if migration.Version() > LatestTargetVersion {
	// 		return fmt.Errorf("migration version %d exceeds configured target version %d", migration.Version(), LatestTargetVersion)
	// 	}
	// }

	// Check for duplicate version numbers
	versionMap := make(map[int]bool)
	for _, migration := range migrations {
		if versionMap[migration.Version()] {
			return fmt.Errorf("duplicate migration version number found: %d", migration.Version())
		}
		versionMap[migration.Version()] = true
	}

	log.Printf("[migrate_db] Migration configuration validation passed, supported version range: 1-%d", LatestTargetVersion)
	return nil
}

// MigrationInfo represents the comprehensive migration information
type MigrationInfo struct {
	CurrentStatus struct {
		CurrentVersion    int    `json:"current_version"`
		TargetVersion     int    `json:"target_version"`
		MigrationNeeded   bool   `json:"migration_needed"`
		RecommendedAction string `json:"recommended_action,omitempty"`
	} `json:"current_status"`
	SupportedMigrations []struct {
		Version     int    `json:"version"`
		Description string `json:"description"`
	} `json:"supported_migrations"`
	MigrationHistory []struct {
		Version     int    `json:"version"`
		Description string `json:"description"`
		AppliedAt   string `json:"applied_at"`
	} `json:"migration_history"`
}

// GetMigrationInfo returns comprehensive database migration information
func GetMigrationInfo() (*MigrationInfo, error) {
	info := &MigrationInfo{}

	// Get current status
	currentVersion, err := GetCurrentVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get current version: %v", err)
	}

	targetVersion := LatestTargetVersion
	info.CurrentStatus.CurrentVersion = currentVersion
	info.CurrentStatus.TargetVersion = targetVersion
	info.CurrentStatus.MigrationNeeded = currentVersion < targetVersion

	if currentVersion < targetVersion {
		info.CurrentStatus.RecommendedAction = "CheckAndMigrateDatabase()"
	}

	// Get supported migrations
	migrations := []Migration{
		&MigrationV1ToV2{},
		&MigrationV2ToV3{},
		&MigrationV3ToV4{},
		&MigrationV4ToV5{},
		// Add more migration operations here
	}

	info.SupportedMigrations = make([]struct {
		Version     int    `json:"version"`
		Description string `json:"description"`
	}, len(migrations))

	for i, migration := range migrations {
		info.SupportedMigrations[i].Version = migration.Version()
		info.SupportedMigrations[i].Description = migration.Description()
	}

	// Get migration history
	history, err := GetMigrationHistory()
	if err != nil {
		return nil, fmt.Errorf("failed to get migration history: %v", err)
	}

	info.MigrationHistory = make([]struct {
		Version     int    `json:"version"`
		Description string `json:"description"`
		AppliedAt   string `json:"applied_at"`
	}, len(history))

	for i, migration := range history {
		timestamp := time.Unix(migration.AppliedAt/1000, (migration.AppliedAt%1000)*1000000)
		info.MigrationHistory[i].Version = migration.Version
		info.MigrationHistory[i].Description = migration.Description
		info.MigrationHistory[i].AppliedAt = timestamp.Format("2006-01-02 15:04:05")
	}

	return info, nil
}

// Show comprehensive database migration information
// This function displays current status, supported migrations, and migration history
func ShowMigrationInfo() {
	log.Printf("[migrate_db] ===== Database Migration Information =====")

	// Show current status
	currentVersion, err := GetCurrentVersion()
	if err != nil {
		log.Printf("[migrate_db] Failed to get current version: %v", err)
		return
	}

	targetVersion := LatestTargetVersion
	log.Printf("[migrate_db] Current Status:")
	log.Printf("[migrate_db]   Current version: %d", currentVersion)
	log.Printf("[migrate_db]   Target version: %d", targetVersion)

	if currentVersion < targetVersion {
		log.Printf("[migrate_db]   Migration needed: Yes")
		log.Printf("[migrate_db]   Recommended action: CheckAndMigrateDatabase()")
	} else {
		log.Printf("[migrate_db]   Migration needed: No")
	}

	// Show supported migrations
	log.Printf("[migrate_db] Supported Migrations:")
	migrations := []Migration{
		&MigrationV1ToV2{},
		&MigrationV2ToV3{},
		&MigrationV3ToV4{},
		&MigrationV4ToV5{},
		// Add more migration operations here
	}
	log.Printf("[migrate_db]   Total migrations: %d", len(migrations))
	for _, migration := range migrations {
		log.Printf("[migrate_db]   Version %d: %s", migration.Version(), migration.Description())
	}

	// Show migration history
	log.Printf("[migrate_db] Migration History:")
	history, err := GetMigrationHistory()
	if err != nil {
		log.Printf("[migrate_db] Failed to get migration history: %v", err)
		return
	}

	if len(history) == 0 {
		log.Printf("[migrate_db]   No migration history")
	} else {
		for _, migration := range history {
			timestamp := time.Unix(migration.AppliedAt/1000, (migration.AppliedAt%1000)*1000000)
			log.Printf("[migrate_db]   Version %d: %s (Applied at: %s)",
				migration.Version,
				migration.Description,
				timestamp.Format("2006-01-02 15:04:05"))
		}
	}

	log.Printf("[migrate_db] ===========================================")
}

// Rebuild group chat index collection from TalkGroupChatTimestamp2Collection
func rebuildGroupChatIndexCollection() error {
	log.Printf("[migrate_db] Starting to rebuild TalkGroupChatIndexCollection")

	sourceDB, exists := Pb[TalkGroupChatTimestamp2Collection]
	if !exists {
		return fmt.Errorf("[migrate_db] source collection %s does not exist", TalkGroupChatTimestamp2Collection)
	}

	targetDB, exists := Pb[TalkGroupChatIndexCollection]
	if !exists {
		return fmt.Errorf("[migrate_db] target collection %s does not exist", TalkGroupChatIndexCollection)
	}

	// Clear target collection first
	log.Printf("[migrate_db] Clearing TalkGroupChatIndexCollection")
	err := clearCollection(targetDB)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to clear TalkGroupChatIndexCollection: %v", err)
	}

	// Use iterator to traverse all data in source collection
	iter, err := sourceDB.NewIter(nil)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to create source collection iterator: %v", err)
	}
	defer iter.Close()

	batch := targetDB.NewBatch()
	defer batch.Close()

	// Track index for each group
	groupIndexCounter := make(map[string]int64)
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Parse source key: groupId_timestamp+number(6)
		keyStr := string(key)
		keyParts := strings.Split(keyStr, "_")
		if len(keyParts) < 2 {
			log.Printf("[migrate_db] Skipping invalid key format: %s", keyStr)
			continue
		}

		groupId := keyParts[0]

		// Parse source value: pinId_chatType_timestamp_number
		valueStr := string(value)
		valueParts := strings.Split(valueStr, "_")
		if len(valueParts) < 1 {
			log.Printf("[migrate_db] Skipping invalid value format: %s", valueStr)
			continue
		}

		pinId := valueParts[0]

		// Get next index for this group
		groupIndexCounter[groupId]++
		nextIndex := groupIndexCounter[groupId]

		// Construct new index key: groupId_index
		indexKey := groupId + "_" + strconv.FormatInt(nextIndex, 10)

		// Construct new index value: pinId_chatType_timestamp_isSet
		indexValue := valueStr + "_1" // Add isSet flag

		// Write data to target collection
		err := batch.Set([]byte(indexKey), []byte(indexValue), nil)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to write to TalkGroupChatIndexCollection: %v", err)
		}

		// Update chat message with index
		err = updateChatMessageIndex(pinId, nextIndex)
		if err != nil {
			log.Printf("[migrate_db] Warning: failed to update chat message index for pinId %s: %v", pinId, err)
		}

		count++

		// Commit batch every 1000 records
		if count%1000 == 0 {
			err = batch.Commit(pebble.Sync)
			if err != nil {
				return fmt.Errorf("[migrate_db] failed to commit batch: %v", err)
			}
			batch = targetDB.NewBatch()
			log.Printf("[migrate_db] Rebuilt %d records for TalkGroupChatIndexCollection", count)
		}
	}

	// Commit remaining batch
	if count%1000 != 0 {
		err := batch.Commit(pebble.Sync)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to commit final batch: %v", err)
		}
	}

	log.Printf("[migrate_db] Successfully rebuilt %d records for TalkGroupChatIndexCollection", count)
	return nil
}

// Rebuild private chat index collection from TalkPrivateChatTimestampCollection
func rebuildPrivateChatIndexCollection() error {
	log.Printf("[migrate_db] Starting to rebuild TalkPrivateChatIndexCollection")

	sourceDB, exists := Pb[TalkPrivateChatTimestampCollection]
	if !exists {
		return fmt.Errorf("[migrate_db] source collection %s does not exist", TalkPrivateChatTimestampCollection)
	}

	targetDB, exists := Pb[TalkPrivateChatIndexCollection]
	if !exists {
		return fmt.Errorf("[migrate_db] target collection %s does not exist", TalkPrivateChatIndexCollection)
	}

	// Clear target collection first
	log.Printf("[migrate_db] Clearing TalkPrivateChatIndexCollection")
	err := clearCollection(targetDB)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to clear TalkPrivateChatIndexCollection: %v", err)
	}

	// Use iterator to traverse all data in source collection
	iter, err := sourceDB.NewIter(nil)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to create source collection iterator: %v", err)
	}
	defer iter.Close()

	batch := targetDB.NewBatch()
	defer batch.Close()

	// Track index for each conversation (from_to)
	conversationIndexCounter := make(map[string]int64)
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Parse source key: from_to_timestamp+number(6) or to_from_timestamp+number(6)
		keyStr := string(key)
		keyParts := strings.Split(keyStr, "_")
		if len(keyParts) < 3 {
			log.Printf("[migrate_db] Skipping invalid key format: %s", keyStr)
			continue
		}

		fromMetaId := keyParts[0]
		toMetaId := keyParts[1]

		// Parse source value: pinId_chatType_timestamp_number
		valueStr := string(value)
		valueParts := strings.Split(valueStr, "_")
		if len(valueParts) < 1 {
			log.Printf("[migrate_db] Skipping invalid value format: %s", valueStr)
			continue
		}

		pinId := valueParts[0]

		// Create conversation key (from_to)
		conversationKey := fromMetaId + "_" + toMetaId

		// Get next index for this conversation
		conversationIndexCounter[conversationKey]++
		nextIndex := conversationIndexCounter[conversationKey]

		// Construct new index key: fromMetaId_toMetaId_index
		indexKey1 := fromMetaId + "_" + toMetaId + "_" + strconv.FormatInt(nextIndex, 10)

		indexKey2 := toMetaId + "_" + fromMetaId + "_" + strconv.FormatInt(nextIndex, 10)

		// Construct new index value: pinId_chatType_timestamp_isSet
		indexValue := valueStr + "_1" // Add isSet flag

		// Write data to target collection
		err := batch.Set([]byte(indexKey1), []byte(indexValue), nil)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to write to TalkPrivateChatIndexCollection: %v", err)
		}

		err = batch.Set([]byte(indexKey2), []byte(indexValue), nil)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to write to TalkPrivateChatIndexCollection: %v", err)
		}

		// Update private chat message with index
		err = updatePrivateChatMessageIndex(pinId, nextIndex)
		if err != nil {
			log.Printf("[migrate_db] Warning: failed to update private chat message index for pinId %s: %v", pinId, err)
		}

		count++

		// Commit batch every 1000 records
		if count%1000 == 0 {
			err = batch.Commit(pebble.Sync)
			if err != nil {
				return fmt.Errorf("[migrate_db] failed to commit batch: %v", err)
			}
			batch = targetDB.NewBatch()
			log.Printf("[migrate_db] Rebuilt %d records for TalkPrivateChatIndexCollection", count)
		}
	}

	// Commit remaining batch
	if count%1000 != 0 {
		err := batch.Commit(pebble.Sync)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to commit final batch: %v", err)
		}
	}

	log.Printf("[migrate_db] Successfully rebuilt %d records for TalkPrivateChatIndexCollection", count)
	return nil
}

// Update chat message with index
func updateChatMessageIndex(pinId string, index int64) error {
	chatDB, exists := Pb[TalkGroupChatPinCollection]
	if !exists {
		return fmt.Errorf("TalkGroupChatPinCollection does not exist")
	}

	// Get chat message
	key := []byte(pinId)
	value, closer, err := chatDB.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return fmt.Errorf("chat message not found for pinId: %s", pinId)
		}
		return err
	}
	defer closer.Close()

	// Parse chat message
	var chat map[string]interface{}
	err = json.Unmarshal(value, &chat)
	if err != nil {
		return fmt.Errorf("failed to parse chat message: %v", err)
	}

	// Update index
	chat["index"] = index

	// Save updated chat message
	updatedValue, err := json.Marshal(chat)
	if err != nil {
		return fmt.Errorf("failed to marshal updated chat message: %v", err)
	}

	return chatDB.Set(key, updatedValue, pebble.Sync)
}

// Rebuild group chat index collection with zero-padded format (version 5)
func rebuildGroupChatIndexCollectionV5() error {
	log.Printf("[migrate_db] Starting to rebuild TalkGroupChatIndexCollection with zero-padded format")

	sourceDB, exists := Pb[TalkGroupChatIndexCollection]
	if !exists {
		return fmt.Errorf("[migrate_db] source collection %s does not exist", TalkGroupChatIndexCollection)
	}

	// Clear target collection first (same as source)
	log.Printf("[migrate_db] Clearing TalkGroupChatIndexCollection")
	err := clearCollection(sourceDB)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to clear TalkGroupChatIndexCollection: %v", err)
	}

	// Get data from TalkGroupChatTimestamp2Collection to rebuild with proper order
	timestampDB, exists := Pb[TalkGroupChatTimestamp2Collection]
	if !exists {
		return fmt.Errorf("[migrate_db] source collection %s does not exist", TalkGroupChatTimestamp2Collection)
	}

	// Use iterator to traverse all data in timestamp collection
	iter, err := timestampDB.NewIter(nil)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to create timestamp collection iterator: %v", err)
	}
	defer iter.Close()

	batch := sourceDB.NewBatch()
	defer batch.Close()

	// Track index for each group
	groupIndexCounter := make(map[string]int64)
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Parse source key: groupId_timestamp+number(6)
		keyStr := string(key)
		keyParts := strings.Split(keyStr, "_")
		if len(keyParts) < 2 {
			log.Printf("[migrate_db] Skipping invalid key format: %s", keyStr)
			continue
		}

		groupId := keyParts[0]

		// Parse source value: pinId_chatType_timestamp_number
		valueStr := string(value)
		valueParts := strings.Split(valueStr, "_")
		if len(valueParts) < 1 {
			log.Printf("[migrate_db] Skipping invalid value format: %s", valueStr)
			continue
		}

		pinId := valueParts[0]

		// Get next index for this group
		groupIndexCounter[groupId]++
		nextIndex := groupIndexCounter[groupId]

		// Construct new index key with zero-padded format: groupId_00000000000000000001
		indexKey := groupId + "_" + fmt.Sprintf("%040d", nextIndex)

		// Construct new index value: pinId_chatType_timestamp_isSet
		indexValue := valueStr + "_1" // Add isSet flag

		// Write data to target collection
		err := batch.Set([]byte(indexKey), []byte(indexValue), nil)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to write to TalkGroupChatIndexCollection: %v", err)
		}

		// Update chat message with index
		err = updateChatMessageIndex(pinId, nextIndex)
		if err != nil {
			log.Printf("[migrate_db] Warning: failed to update chat message index for pinId %s: %v", pinId, err)
		}

		count++

		// Commit batch every 1000 records
		if count%1000 == 0 {
			err = batch.Commit(pebble.Sync)
			if err != nil {
				return fmt.Errorf("[migrate_db] failed to commit batch: %v", err)
			}
			batch = sourceDB.NewBatch()
			log.Printf("[migrate_db] Rebuilt %d records for TalkGroupChatIndexCollection with zero-padded format", count)
		}
	}

	// Commit remaining batch
	if count%1000 != 0 {
		err := batch.Commit(pebble.Sync)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to commit final batch: %v", err)
		}
	}

	log.Printf("[migrate_db] Successfully rebuilt %d records for TalkGroupChatIndexCollection with zero-padded format", count)
	return nil
}

// Rebuild private chat index collection with zero-padded format (version 5)
func rebuildPrivateChatIndexCollectionV5() error {
	log.Printf("[migrate_db] Starting to rebuild TalkPrivateChatIndexCollection with zero-padded format")

	sourceDB, exists := Pb[TalkPrivateChatIndexCollection]
	if !exists {
		return fmt.Errorf("[migrate_db] source collection %s does not exist", TalkPrivateChatIndexCollection)
	}

	// Clear target collection first (same as source)
	log.Printf("[migrate_db] Clearing TalkPrivateChatIndexCollection")
	err := clearCollection(sourceDB)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to clear TalkPrivateChatIndexCollection: %v", err)
	}

	// Get data from TalkPrivateChatTimestampCollection to rebuild with proper order
	timestampDB, exists := Pb[TalkPrivateChatTimestampCollection]
	if !exists {
		return fmt.Errorf("[migrate_db] source collection %s does not exist", TalkPrivateChatTimestampCollection)
	}

	// Use iterator to traverse all data in timestamp collection
	iter, err := timestampDB.NewIter(nil)
	if err != nil {
		return fmt.Errorf("[migrate_db] failed to create timestamp collection iterator: %v", err)
	}
	defer iter.Close()

	batch := sourceDB.NewBatch()
	defer batch.Close()

	// Track index for each conversation (from_to)
	conversationIndexCounter := make(map[string]int64)
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Parse source key: from_to_timestamp+number(6) or to_from_timestamp+number(6)
		keyStr := string(key)
		keyParts := strings.Split(keyStr, "_")
		if len(keyParts) < 3 {
			log.Printf("[migrate_db] Skipping invalid key format: %s", keyStr)
			continue
		}

		fromMetaId := keyParts[0]
		toMetaId := keyParts[1]

		// Parse source value: pinId_chatType_timestamp_number
		valueStr := string(value)
		valueParts := strings.Split(valueStr, "_")
		if len(valueParts) < 1 {
			log.Printf("[migrate_db] Skipping invalid value format: %s", valueStr)
			continue
		}

		pinId := valueParts[0]

		// Create conversation key (from_to)
		conversationKey := fromMetaId + "_" + toMetaId

		// Get next index for this conversation
		conversationIndexCounter[conversationKey]++
		nextIndex := conversationIndexCounter[conversationKey]

		// Construct new index key with zero-padded format: fromMetaId_toMetaId_00000000000000000001
		indexKey1 := fromMetaId + "_" + toMetaId + "_" + fmt.Sprintf("%040d", nextIndex)
		indexKey2 := toMetaId + "_" + fromMetaId + "_" + fmt.Sprintf("%040d", nextIndex)

		// Construct new index value: pinId_chatType_timestamp_isSet
		indexValue := valueStr + "_1" // Add isSet flag

		// Write data to target collection
		err := batch.Set([]byte(indexKey1), []byte(indexValue), nil)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to write to TalkPrivateChatIndexCollection: %v", err)
		}

		err = batch.Set([]byte(indexKey2), []byte(indexValue), nil)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to write to TalkPrivateChatIndexCollection: %v", err)
		}

		// Update private chat message with index
		err = updatePrivateChatMessageIndex(pinId, nextIndex)
		if err != nil {
			log.Printf("[migrate_db] Warning: failed to update private chat message index for pinId %s: %v", pinId, err)
		}

		count++

		// Commit batch every 1000 records
		if count%1000 == 0 {
			err = batch.Commit(pebble.Sync)
			if err != nil {
				return fmt.Errorf("[migrate_db] failed to commit batch: %v", err)
			}
			batch = sourceDB.NewBatch()
			log.Printf("[migrate_db] Rebuilt %d records for TalkPrivateChatIndexCollection with zero-padded format", count)
		}
	}

	// Commit remaining batch
	if count%1000 != 0 {
		err := batch.Commit(pebble.Sync)
		if err != nil {
			return fmt.Errorf("[migrate_db] failed to commit final batch: %v", err)
		}
	}

	log.Printf("[migrate_db] Successfully rebuilt %d records for TalkPrivateChatIndexCollection with zero-padded format", count)
	return nil
}

// Update private chat message with index
func updatePrivateChatMessageIndex(pinId string, index int64) error {
	chatDB, exists := Pb[TalkPrivateChatPinCollection]
	if !exists {
		return fmt.Errorf("TalkPrivateChatPinCollection does not exist")
	}

	// Get private chat message
	key := []byte(pinId)
	value, closer, err := chatDB.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return fmt.Errorf("private chat message not found for pinId: %s", pinId)
		}
		return err
	}
	defer closer.Close()

	// Parse private chat message
	var chat map[string]interface{}
	err = json.Unmarshal(value, &chat)
	if err != nil {
		return fmt.Errorf("failed to parse private chat message: %v", err)
	}

	// Update index
	chat["index"] = index

	// Save updated private chat message
	updatedValue, err := json.Marshal(chat)
	if err != nil {
		return fmt.Errorf("failed to marshal updated private chat message: %v", err)
	}

	return chatDB.Set(key, updatedValue, pebble.Sync)
}

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
	LatestTargetVersion = 2
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

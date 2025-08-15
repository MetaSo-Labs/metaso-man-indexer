package indexer

import (
	"log"
	"manindexer/basicprotocols/group_chat/db"
	"manindexer/basicprotocols/group_chat/protocols"
	"manindexer/pin"
	"strings"
)

// GroupChatIndexer Group chat indexer
type GroupChatIndexer struct {
	communityDB *db.CommunityDB
	groupDB     *db.GroupDB
	chatDB      *db.ChatDB
	privateDB   *db.PrivateChatDB
	pb          *db.Pebble
}

// NewGroupChatIndexer Create new group chat indexer
func NewGroupChatIndexer() (*GroupChatIndexer, error) {
	pb := &db.Pebble{}

	// Initialize database
	err := pb.InitDatabase()
	if err != nil {
		log.Printf("Failed to initialize database: %v", err)
		return nil, err
	}

	return &GroupChatIndexer{
		communityDB: db.NewCommunityDB(pb),
		groupDB:     db.NewGroupDB(pb),
		chatDB:      db.NewChatDB(pb),
		privateDB:   db.NewPrivateChatDB(pb),
		pb:          pb,
	}, nil
}

// Start Start group chat indexer
func (gci *GroupChatIndexer) Start() error {
	log.Println("Starting Group Chat Indexer...")

	// Start chat queue processor
	gci.chatDB.StartQueueProcessor(gci.groupDB)

	// Start private chat queue processor
	gci.privateDB.StartPrivateQueueProcessor()

	log.Println("Group Chat Indexer started successfully")
	return nil
}

// Stop Stop group chat indexer
func (gci *GroupChatIndexer) Stop() error {
	log.Println("Stopping Group Chat Indexer...")

	// Close all database connections
	gci.pb.CloseAll()

	log.Println("Group Chat Indexer stopped successfully")
	return nil
}

// ProcessPin Process single Pin
func (gci *GroupChatIndexer) ProcessPin(pin *pin.PinInscription, tx interface{}) error {
	if pin == nil {
		return nil
	}

	// Distribute processing based on protocol path
	protocol := gci.extractProtocol(pin.Path)

	switch strings.ToLower(protocol) {
	case strings.ToLower(protocols.MonitorSimpleCommunity), strings.ToLower(protocols.MonitorSimpleCommunityJoin):
		log.Printf("Community protocol: %s", pin.Path)
		// Community related protocols
		return gci.communityDB.ProcessCommunityPin(pin)
	case strings.ToLower(protocols.MonitorSimpleGroupCreate), strings.ToLower(protocols.MonitorSimpleGroupJoin):
		log.Printf("Group protocol: %s", pin.Path)
		// Group related protocols
		return gci.groupDB.ProcessGroupPin(pin)
	case strings.ToLower(protocols.MonitorSimpleGroupChat),
		strings.ToLower(protocols.MonitorSimpleFileGroupChat),
		strings.ToLower(protocols.MonitorSimpleGroupLuckyBag),
		strings.ToLower(protocols.MonitorSimpleGroupOpenLuckyBag),
		strings.ToLower(protocols.MonitorSimpleGroupResidueLuckyBag):
		log.Printf("Chat protocol: %s", pin.Path)
		// Chat related protocols
		return gci.chatDB.ProcessGroupChatPin(pin, tx)
	case strings.ToLower(protocols.MonitorSimpleMsg), strings.ToLower(protocols.MonitorSimpleFileMsg):
		log.Printf("Private chat protocol: %s", pin.Path)
		// Private chat related protocols
		return gci.privateDB.ProcessPrivateChatPin(pin)
	default:
		log.Printf("Unknown protocol: %s", protocol)
		return nil
	}
}

// extractProtocol Extract protocol name from path
func (gci *GroupChatIndexer) extractProtocol(path string) string {
	// Remove "/protocols/" prefix
	protocol := strings.Replace(path, "/protocols/", "", -1)
	return protocol
}

// GetCommunityDB Get community database instance
func (gci *GroupChatIndexer) GetCommunityDB() *db.CommunityDB {
	return gci.communityDB
}

// GetGroupDB Get group database instance
func (gci *GroupChatIndexer) GetGroupDB() *db.GroupDB {
	return gci.groupDB
}

// GetChatDB Get chat database instance
func (gci *GroupChatIndexer) GetChatDB() *db.ChatDB {
	return gci.chatDB
}

// GetPrivateDB Get private chat database instance
func (gci *GroupChatIndexer) GetPrivateDB() *db.PrivateChatDB {
	return gci.privateDB
}

// GetPebble Get Pebble database instance
func (gci *GroupChatIndexer) GetPebble() *db.Pebble {
	return gci.pb
}

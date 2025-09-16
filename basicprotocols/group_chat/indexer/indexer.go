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
	communityDB   *db.CommunityDB
	groupDB       *db.GroupDB
	chatDB        *db.ChatDB
	privateDB     *db.PrivateChatDB
	userDB        *db.UserInfoDB
	globalBlockDB *db.GlobalBlockDB
	socketInfoDB  *db.SocketInfoDB
	pb            *db.Pebble
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

	ch := db.NewChatDB(pb)
	gdb := db.NewGroupDB(pb, ch)
	ch.SetGdb(gdb)

	return &GroupChatIndexer{
		communityDB:   db.NewCommunityDB(pb),
		groupDB:       gdb,
		chatDB:        ch,
		privateDB:     db.NewPrivateChatDB(pb),
		userDB:        db.NewUserInfoDB(pb),
		globalBlockDB: db.NewGlobalBlockDB(pb),
		socketInfoDB:  db.NewSocketInfoDB(pb),
		pb:            pb,
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

	// Check if the address is globally blocked
	isBlocked, _ := gci.globalBlockDB.IsAddressGloballyBlocked(pin.CreateAddress)
	// if err != nil {
	// 	log.Printf("Failed to check if address is globally blocked: %v", err)
	// 	return err
	// }
	if isBlocked {
		log.Printf("[%s]Address is globally blocked: %s", pin.ChainName, pin.CreateAddress)
		return nil
	}

	// ProcessUserInfoPin
	infoNode, ok := gci.extractInfo(pin.Path)
	if ok {
		if strings.ToLower(infoNode) == strings.ToLower(protocols.MonitorInfoChatpubkey) {
			log.Printf("[%s]ProcessUserInfoPin: %s", pin.ChainName, pin.Path)
			return gci.userDB.ProcessUserInfoPin(pin)
		}
	}

	// Distribute processing based on protocol path
	protocol := gci.extractProtocol(pin.Path)

	switch strings.ToLower(protocol) {
	case strings.ToLower(protocols.MonitorSimpleCommunity), strings.ToLower(protocols.MonitorSimpleCommunityJoin):
		log.Printf("[%s]Community protocol: %s", pin.ChainName, pin.Path)
		// Community related protocols
		return gci.communityDB.ProcessCommunityPin(pin)
	case strings.ToLower(protocols.MonitorSimpleGroupCreate),
		strings.ToLower(protocols.MonitorSimpleGroupChannel),
		strings.ToLower(protocols.MonitorSimpleGroupJoin),
		strings.ToLower(protocols.MonitorSimpleGroupRemoveUser),
		strings.ToLower(protocols.MonitorSimpleGroupAdmin),
		strings.ToLower(protocols.MonitorSimpleGroupBlock),
		strings.ToLower(protocols.MonitorSimpleGroupWhitelist):
		log.Printf("[%s]Group protocol: %s", pin.ChainName, pin.Path)
		// Group related protocols
		return gci.groupDB.ProcessGroupPin(pin)
	case strings.ToLower(protocols.MonitorSimpleGroupChat),
		strings.ToLower(protocols.MonitorSimpleFileGroupChat),
		strings.ToLower(protocols.MonitorSimpleGroupLuckyBag),
		strings.ToLower(protocols.MonitorSimpleGroupOpenLuckyBag),
		strings.ToLower(protocols.MonitorSimpleGroupResidueLuckyBag):
		log.Printf("[%s]Chat protocol: %s", pin.ChainName, pin.Path)
		// Chat related protocols
		return gci.chatDB.ProcessGroupChatPin(pin, tx)
	case strings.ToLower(protocols.MonitorSimpleMsg),
		strings.ToLower(protocols.MonitorSimpleFileMsg),
		strings.ToLower(protocols.MonitorSimplePrivateBlock):
		log.Printf("[%s]Private chat protocol: %s", pin.ChainName, pin.Path)
		// Private chat related protocols
		return gci.privateDB.ProcessPrivateChatPin(pin)
	default:
		log.Printf("[%s]Unknown protocol: %s", pin.ChainName, protocol)
		return nil
	}
}

// extractProtocol Extract protocol name from path
func (gci *GroupChatIndexer) extractProtocol(path string) string {
	// Remove "/protocols/" prefix
	protocol := strings.Replace(path, "/protocols/", "", -1)
	return protocol
}

func (gci *GroupChatIndexer) extractInfo(path string) (string, bool) {
	if !strings.HasPrefix(path, "/info/") {
		return "", false
	}
	// Remove "/info/" prefix
	infoNode := strings.Replace(path, "/info/", "", -1)
	return infoNode, true
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

// GetUserInfoDB Get user info database instance
func (gci *GroupChatIndexer) GetUserInfoDB() *db.UserInfoDB {
	return gci.userDB
}

// GetSocketInfoDB Get socket info database instance
func (gci *GroupChatIndexer) GetSocketInfoDB() *db.SocketInfoDB {
	return gci.socketInfoDB
}

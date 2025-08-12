package indexer

import (
	"log"
	"manindexer/basicprotocols/group_chat/db"
	"manindexer/basicprotocols/group_chat/protocols"
	"manindexer/pin"
	"strings"
)

// GroupChatIndexer 群聊索引器
type GroupChatIndexer struct {
	communityDB *db.CommunityDB
	groupDB     *db.GroupDB
	chatDB      *db.ChatDB
	privateDB   *db.PrivateChatDB
	pb          *db.Pebble
}

// NewGroupChatIndexer 创建新的群聊索引器
func NewGroupChatIndexer() (*GroupChatIndexer, error) {
	pb := &db.Pebble{}

	// 初始化数据库
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

// Start 启动群聊索引器
func (gci *GroupChatIndexer) Start() error {
	log.Println("Starting Group Chat Indexer...")

	// 启动聊天队列处理器
	gci.chatDB.StartQueueProcessor(gci.groupDB)

	// 启动私聊队列处理器
	gci.privateDB.StartPrivateQueueProcessor()

	log.Println("Group Chat Indexer started successfully")
	return nil
}

// Stop 停止群聊索引器
func (gci *GroupChatIndexer) Stop() error {
	log.Println("Stopping Group Chat Indexer...")

	// 关闭所有数据库连接
	gci.pb.CloseAll()

	log.Println("Group Chat Indexer stopped successfully")
	return nil
}

// ProcessPin 处理单个 Pin
func (gci *GroupChatIndexer) ProcessPin(pin *pin.PinInscription) error {
	if pin == nil {
		return nil
	}

	// 根据协议路径分发处理
	protocol := gci.extractProtocol(pin.Path)

	switch strings.ToLower(protocol) {
	case strings.ToLower(protocols.MonitorSimpleCommunity), strings.ToLower(protocols.MonitorSimpleCommunityJoin):
		log.Printf("Community protocol: %s", pin.Path)
		// 社区相关协议
		return gci.communityDB.ProcessCommunityPin(pin)
	case strings.ToLower(protocols.MonitorSimpleGroupCreate), strings.ToLower(protocols.MonitorSimpleGroupJoin):
		log.Printf("Group protocol: %s", pin.Path)
		// 群组相关协议
		return gci.groupDB.ProcessGroupPin(pin)
	case strings.ToLower(protocols.MonitorSimpleGroupChat), strings.ToLower(protocols.MonitorSimpleFileGroupChat):
		log.Printf("Chat protocol: %s", pin.Path)
		// 聊天相关协议
		return gci.chatDB.ProcessGroupChatPin(pin)
	case strings.ToLower(protocols.MonitorSimpleMsg), strings.ToLower(protocols.MonitorSimpleFileMsg):
		log.Printf("Private chat protocol: %s", pin.Path)
		// 私聊相关协议
		return gci.privateDB.ProcessPrivateChatPin(pin)
	default:
		log.Printf("Unknown protocol: %s", protocol)
		return nil
	}
}

// extractProtocol 从路径中提取协议名称
func (gci *GroupChatIndexer) extractProtocol(path string) string {
	// 移除 "/protocols/" 前缀
	protocol := strings.Replace(path, "/protocols/", "", -1)
	return protocol
}

// GetCommunityDB 获取社区数据库实例
func (gci *GroupChatIndexer) GetCommunityDB() *db.CommunityDB {
	return gci.communityDB
}

// GetGroupDB 获取群组数据库实例
func (gci *GroupChatIndexer) GetGroupDB() *db.GroupDB {
	return gci.groupDB
}

// GetChatDB 获取聊天数据库实例
func (gci *GroupChatIndexer) GetChatDB() *db.ChatDB {
	return gci.chatDB
}

// GetPrivateDB 获取私聊数据库实例
func (gci *GroupChatIndexer) GetPrivateDB() *db.PrivateChatDB {
	return gci.privateDB
}

// GetPebble 获取 Pebble 数据库实例
func (gci *GroupChatIndexer) GetPebble() *db.Pebble {
	return gci.pb
}

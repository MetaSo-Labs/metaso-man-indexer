package indexer

import (
	"log"
	"manindexer/pin"
)

// ExampleUsage 展示如何使用 GroupChatIndexer 的示例
func ExampleUsage() {
	// 创建索引器
	indexer, err := NewGroupChatIndexer()
	if err != nil {
		log.Fatalf("Failed to create GroupChatIndexer: %v", err)
	}

	// 启动索引器
	err = indexer.Start()
	if err != nil {
		log.Fatalf("Failed to start GroupChatIndexer: %v", err)
	}

	// 示例：处理社区创建 Pin
	communityPin := &pin.PinInscription{
		Id:            "pin123",
		Path:          "/protocols/SimpleCommunity",
		Operation:     "create",
		CreateMetaId:  "metaId123",
		CreateAddress: "address123",
		ContentBody:   []byte(`{"communityId":"hash123","name":"测试社区","description":"这是一个测试社区"}`),
		Timestamp:     1234567890,
	}

	err = indexer.ProcessPin(communityPin, nil)
	if err != nil {
		log.Printf("Failed to process community pin: %v", err)
	}

	// 示例：处理群组创建 Pin
	groupPin := &pin.PinInscription{
		Id:            "pin456",
		Path:          "/protocols/SimpleGroupCreate",
		Operation:     "create",
		CreateMetaId:  "metaId456",
		CreateAddress: "address456",
		ContentBody:   []byte(`{"groupId":"group123","communityId":"hash123","groupName":"测试群组"}`),
		Timestamp:     1234567890,
	}

	err = indexer.ProcessPin(groupPin, nil)
	if err != nil {
		log.Printf("Failed to process group pin: %v", err)
	}

	// 示例：处理聊天消息 Pin
	chatPin := &pin.PinInscription{
		Id:            "pin789",
		Path:          "/protocols/SimpleGroupChat",
		Operation:     "create",
		CreateMetaId:  "metaId789",
		CreateAddress: "address789",
		ContentBody:   []byte(`{"groupId":"group123","content":"Hello World!","contentType":"text"}`),
		Timestamp:     1234567890,
	}

	err = indexer.ProcessPin(chatPin, nil)
	if err != nil {
		log.Printf("Failed to process chat pin: %v", err)
	}

	// 示例：处理群组加入 Pin
	joinPin := &pin.PinInscription{
		Id:            "pin101",
		Path:          "/protocols/SimpleGroupJoin",
		Operation:     "join",
		CreateMetaId:  "metaId101",
		CreateAddress: "address101",
		ContentBody:   []byte(`{"groupId":"group123","state":1}`),
		Timestamp:     1234567890,
	}

	err = indexer.ProcessPin(joinPin, nil)
	if err != nil {
		log.Printf("Failed to process join pin: %v", err)
	}

	// 停止索引器
	err = indexer.Stop()
	if err != nil {
		log.Printf("Failed to stop GroupChatIndexer: %v", err)
	}
}

package models

type RoomState int64

const (
	RoomStateIn  RoomState = 1
	RoomStateOut RoomState = -1
)

type ChatType int64

const (
	ChatTypeMsg             ChatType = 0  // Group chat - Message type
	ChatTypeEmoji           ChatType = 6  // Group chat - Emoji type
	ChatTypeLuckyBag        ChatType = 1  // Group chat - Lucky bag type
	ChatTypeOpenLuckyBag    ChatType = 2  // Group chat - Grab lucky bag type
	ChatTypeRecycleLuckyBag ChatType = 22 // Group chat - Reclaim lucky bag type
	ChatTypeLuckyBagV2      ChatType = 23 // Group chat - Lucky bag v2 type
	ChatTypeFile            ChatType = 3  // Group chat - File type
	ChatTypeJoin            ChatType = 4  // Group chat - Join type
	ChatTypeLeave           ChatType = 5  // Group chat - Leave type
	ChatTypeRemove          ChatType = 7  // Group chat - Remove user type
)

const (
	PrivateBlockStateBlock   int64 = 1  // block
	PrivateBlockStateUnblock int64 = -1 // unblock
)

package request

// SyncPinsByTimeRangeRequest Sync request structure
type SyncPinsByTimeRangeRequest struct {
	StartTs              int64 `form:"startTs"`              // Start timestamp
	EndTs                int64 `form:"endTs"`                // End timestamp
	BatchDurationSeconds int64 `form:"batchDurationSeconds"` // Batch duration in seconds
	// Host                 []string `form:"host,omitempty"`       // Host filter conditions
	// Protocol             []string `form:"protocol,omitempty"`   // Protocol filter conditions
}

// GetBlockHeightByTimestampRequest Block height query request structure by timestamp
type GetBlockHeightByTimestampRequest struct {
	Timestamp int64 `form:"timestamp"` // Target timestamp
}

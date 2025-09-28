package respond

import "time"

// SyncStatsResponse Sync statistics response
type SyncStatsResponse struct {
	TotalPins     int       `json:"totalPins"`     // Total pin count
	ProcessedPins int       `json:"processedPins"` // Processed pin count
	SuccessPins   int       `json:"successPins"`   // Successfully processed pin count
	FailedPins    int       `json:"failedPins"`    // Failed pin count
	StartTime     time.Time `json:"startTime"`     // Start time
	EndTime       time.Time `json:"endTime"`       // End time
	IsCompleted   bool      `json:"isCompleted"`   // Whether completed
	ErrorMessage  string    `json:"errorMessage"`  // Error message
}

// SyncProgressResponse Sync progress response
type SyncProgressResponse struct {
	Progress      float64 `json:"progress"`      // Progress percentage
	Status        string  `json:"status"`        // Status (waiting/in progress/completed)
	ProcessedPins int     `json:"processedPins"` // Processed pin count
	TotalPins     int     `json:"totalPins"`     // Total pin count
	SuccessPins   int     `json:"successPins"`   // Successfully processed pin count
	FailedPins    int     `json:"failedPins"`    // Failed pin count
}

// SyncStatusResponse Sync status response
type SyncStatusResponse struct {
	IsRunning bool `json:"isRunning"` // Whether running
}

// BlockHeightResultResponse Block height query result response
type BlockHeightResultResponse struct {
	ChainName string `json:"chainName"` // Chain name (mvc or btc)
	Height    int64  `json:"height"`    // Block height
	Timestamp int64  `json:"timestamp"` // Block timestamp
}

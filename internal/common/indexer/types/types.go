package types

import (
	"sync"
	"time"
)

const (
	// Indexer common logic
	RPCCallTimeout          = 13 * time.Second
	AfterFailedRetryTimeout = 10 * time.Second
	CatchingUpSleepDuration = 100 * time.Millisecond
	DefaultSleepDuration    = 5 * time.Second
	UnHealthSleep           = 10 * time.Second

	// SQL Durations
	SQLQueryMaxDuration         = 10 * time.Second // common query timeout
	RetentionQuerySleepDuration = 1 * time.Hour    // retention logic

	// Fetching Height Logic
	FetchTimeout                  = 5 * time.Second
	FetchSleepDuration            = 5 * time.Second
	AfterFailedFetchSleepDuration = 10 * time.Second

	BatchSyncLimit = 10
)

// types for indexer packages
type ValidatorIDMap map[string]int64
type ValidatorAddressMap map[int64]string
type MonikerIDMap map[int64]bool

type LatestHeightCache struct {
	Mutex        sync.RWMutex
	LatestHeight int64
}

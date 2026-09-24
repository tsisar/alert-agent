package webhook

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// dedupKey returns a short hex digest of the raw dedup key so that
// special characters (quotes, braces, slashes) don't cause issues
// when used as map keys or Redis keys.
func dedupKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// DedupStore is the interface for alert deduplication.
type DedupStore interface {
	ShouldProcess(key string) bool
	Clear(key string)
}

// Deduplicator tracks recently processed alerts by their dedup key
// and suppresses duplicates within the configured cooldown period.
// When a resolved alert is received, the entry is removed so the
// next firing of the same alert triggers a new investigation.
type Deduplicator struct {
	mu       sync.Mutex
	seen     map[string]time.Time
	cooldown time.Duration
}

func NewDeduplicator(cooldown time.Duration) *Deduplicator {
	return &Deduplicator{
		seen:     make(map[string]time.Time),
		cooldown: cooldown,
	}
}

// ShouldProcess returns true if the alert should be investigated.
// It records the current time for the given key.
func (d *Deduplicator) ShouldProcess(key string) bool {
	key = dedupKey(key)
	d.mu.Lock()
	defer d.mu.Unlock()

	if last, ok := d.seen[key]; ok {
		if time.Since(last) < d.cooldown {
			return false
		}
	}
	d.seen[key] = time.Now()
	return true
}

// Clear removes the key from the map, allowing the next
// occurrence to be processed immediately.
func (d *Deduplicator) Clear(key string) {
	key = dedupKey(key)
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.seen, key)
}

package storage

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/tsisar/extended-log-go/log"
	"gorm.io/gorm"
)

type promptRepo struct {
	db      *gorm.DB
	mu      sync.RWMutex
	cache   *PromptSet
	version atomic.Int64
	loaded  atomic.Int64
}

// NewPromptRepository creates a cached prompt repository backed by GORM.
func NewPromptRepository(db *gorm.DB) PromptRepository {
	r := &promptRepo{db: db}
	r.version.Store(1)
	r.loaded.Store(0)
	return r
}

func (r *promptRepo) ensureLoaded() {
	if r.loaded.Load() == r.version.Load() {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.loaded.Load() == r.version.Load() {
		return
	}
	var prompts []Prompt
	if err := r.db.Find(&prompts).Error; err != nil {
		log.Errorf("failed to load prompts from DB: %v", err)
		return
	}
	ps := &PromptSet{}
	for _, p := range prompts {
		switch p.Key {
		case "system":
			ps.System = p.Value
		case "summary":
			ps.Summary = p.Value
		case "resolved":
			ps.Resolved = p.Value
		case "paused":
			ps.Paused = p.Value
		}
	}
	r.cache = ps
	r.loaded.Store(r.version.Load())
}

func (r *promptRepo) Get(key string) (string, error) {
	r.ensureLoaded()
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.cache == nil {
		return "", fmt.Errorf("prompts not loaded")
	}
	switch key {
	case "system":
		return r.cache.System, nil
	case "summary":
		return r.cache.Summary, nil
	case "resolved":
		return r.cache.Resolved, nil
	case "paused":
		return r.cache.Paused, nil
	default:
		return "", fmt.Errorf("unknown prompt key: %s", key)
	}
}

func (r *promptRepo) GetAll() (*PromptSet, error) {
	r.ensureLoaded()
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.cache == nil {
		return nil, fmt.Errorf("prompts not loaded")
	}
	// Return a copy to prevent external mutation.
	cp := *r.cache
	return &cp, nil
}

func (r *promptRepo) Set(key, value string) error {
	result := r.db.Where(Prompt{Key: key}).Assign(Prompt{Value: value}).FirstOrCreate(&Prompt{})
	if result.Error != nil {
		return result.Error
	}
	// If the record existed, update it.
	if result.RowsAffected == 0 {
		if err := r.db.Model(&Prompt{}).Where("key = ?", key).Update("value", value).Error; err != nil {
			return err
		}
	}
	r.Invalidate()
	return nil
}

func (r *promptRepo) Invalidate() {
	r.version.Add(1)
}

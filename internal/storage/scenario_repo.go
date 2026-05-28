package storage

import (
	"sync"
	"sync/atomic"

	"github.com/tsisar/alert-agent/internal/scenario"
	"github.com/tsisar/extended-log-go/log"
	"gorm.io/gorm"
)

type scenarioRepo struct {
	db      *gorm.DB
	mu      sync.RWMutex
	cache   []scenario.Scenario
	version atomic.Int64
	loaded  atomic.Int64
}

// NewScenarioRepository creates a cached scenario repository backed by GORM.
func NewScenarioRepository(db *gorm.DB) ScenarioRepository {
	r := &scenarioRepo{db: db}
	r.version.Store(1)
	r.loaded.Store(0)
	return r
}

func (r *scenarioRepo) ensureLoaded() {
	if r.loaded.Load() == r.version.Load() {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.loaded.Load() == r.version.Load() {
		return
	}
	var models []Scenario
	if err := r.db.Order("order_index asc").Find(&models).Error; err != nil {
		log.Errorf("failed to load scenarios from DB: %v", err)
		return
	}
	r.cache = make([]scenario.Scenario, len(models))
	for i, m := range models {
		r.cache[i] = m.ToDomain()
	}
	r.loaded.Store(r.version.Load())
}

func (r *scenarioRepo) Match(alertLabels map[string]string) *scenario.Scenario {
	r.ensureLoaded()
	r.mu.RLock()
	defer r.mu.RUnlock()

	var catchAll *scenario.Scenario
	for i := range r.cache {
		s := &r.cache[i]
		if len(s.Match) == 0 {
			catchAll = s
			continue
		}
		if scenario.LabelsMatch(s.Match, alertLabels) {
			return s
		}
	}
	return catchAll
}

func (r *scenarioRepo) FindByName(name string) *scenario.Scenario {
	r.ensureLoaded()
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := range r.cache {
		if r.cache[i].Name == name {
			return &r.cache[i]
		}
	}
	return nil
}

func (r *scenarioRepo) List() ([]Scenario, error) {
	var models []Scenario
	if err := r.db.Order("order_index asc").Find(&models).Error; err != nil {
		return nil, err
	}
	return models, nil
}

func (r *scenarioRepo) Create(s *Scenario) error {
	if err := r.db.Create(s).Error; err != nil {
		return err
	}
	r.Invalidate()
	return nil
}

func (r *scenarioRepo) Update(s *Scenario) error {
	if err := r.db.Save(s).Error; err != nil {
		return err
	}
	r.Invalidate()
	return nil
}

func (r *scenarioRepo) Delete(id uint) error {
	// Hard delete: scenarios have a unique index on name and no restore
	// feature, so a soft-deleted row would only block recreating a scenario
	// with the same name.
	if err := r.db.Unscoped().Delete(&Scenario{}, id).Error; err != nil {
		return err
	}
	r.Invalidate()
	return nil
}

func (r *scenarioRepo) Invalidate() {
	r.version.Add(1)
}

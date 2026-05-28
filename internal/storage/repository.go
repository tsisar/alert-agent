package storage

import "github.com/tsisar/alert-agent/internal/scenario"

// PromptSet holds all prompt values for convenient access.
type PromptSet struct {
	System   string
	Summary  string
	Resolved string
	Paused   string
}

// ScenarioRepository provides cached access to scenarios.
type ScenarioRepository interface {
	Match(alertLabels map[string]string) *scenario.Scenario
	FindByName(name string) *scenario.Scenario
	List() ([]Scenario, error)
	Create(s *Scenario) error
	Update(s *Scenario) error
	Delete(id uint) error
	Invalidate()
}

// PromptRepository provides cached access to prompts.
type PromptRepository interface {
	Get(key string) (string, error)
	GetAll() (*PromptSet, error)
	Set(key, value string) error
	Invalidate()
}

package storage

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

//go:embed seed/scenarios.yaml
var seedScenariosYAML []byte

//go:embed seed/prompts.yaml
var seedPromptsYAML []byte

type seedScenario struct {
	Name     string            `yaml:"name"`
	Match    map[string]string `yaml:"match"`
	Prompt   string            `yaml:"prompt"`
	Tools    []string          `yaml:"tools"`
	Channels struct {
		Telegram string `yaml:"telegram"`
		Slack    string `yaml:"slack"`
	} `yaml:"channels"`
	Timeout    string `yaml:"timeout"`
	Priority   string `yaml:"priority"`
	SendImages bool   `yaml:"send_images"`
}

type seedScenariosFile struct {
	Scenarios []seedScenario `yaml:"scenarios"`
}

type seedPromptsFile struct {
	System   string `yaml:"system"`
	Summary  string `yaml:"summary"`
	Resolved string `yaml:"resolved"`
	Paused   string `yaml:"paused"`
}

// Seed populates the database with default scenarios and prompts
// if the tables are empty. It is idempotent.
func Seed(db *gorm.DB) error {
	if err := seedScenarios(db); err != nil {
		return fmt.Errorf("seed scenarios: %w", err)
	}
	if err := seedPrompts(db); err != nil {
		return fmt.Errorf("seed prompts: %w", err)
	}
	return nil
}

func seedScenarios(db *gorm.DB) error {
	var count int64
	if err := db.Model(&Scenario{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	var file seedScenariosFile
	if err := yaml.Unmarshal(seedScenariosYAML, &file); err != nil {
		return fmt.Errorf("parse seed scenarios: %w", err)
	}

	for i, s := range file.Scenarios {
		timeout := s.Timeout
		if timeout == "" {
			timeout = "2m"
		}
		priority := s.Priority
		if priority == "" {
			priority = "normal"
		}

		model := Scenario{
			Name:            s.Name,
			OrderIndex:      i * 10,
			Match:           JSONMap(s.Match),
			Prompt:          s.Prompt,
			Tools:           JSONStringSlice(s.Tools),
			ChannelTelegram: s.Channels.Telegram,
			ChannelSlack:    s.Channels.Slack,
			Timeout:         timeout,
			Priority:        priority,
			SendImages:      s.SendImages,
		}
		if err := db.Create(&model).Error; err != nil {
			return fmt.Errorf("create scenario %q: %w", s.Name, err)
		}
	}
	return nil
}

func seedPrompts(db *gorm.DB) error {
	var count int64
	if err := db.Model(&Prompt{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	var file seedPromptsFile
	if err := yaml.Unmarshal(seedPromptsYAML, &file); err != nil {
		return fmt.Errorf("parse seed prompts: %w", err)
	}

	prompts := []Prompt{
		{Key: "system", Value: file.System},
		{Key: "summary", Value: file.Summary},
		{Key: "resolved", Value: file.Resolved},
		{Key: "paused", Value: file.Paused},
	}
	for _, p := range prompts {
		if err := db.Create(&p).Error; err != nil {
			return fmt.Errorf("create prompt %q: %w", p.Key, err)
		}
	}
	return nil
}

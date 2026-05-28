package scenario

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Scenario defines an alert investigation scenario.
type Scenario struct {
	Name       string            `yaml:"name"`
	Match      map[string]string `yaml:"match"`
	Prompt     string            `yaml:"prompt"`
	Tools      []string          `yaml:"tools"`
	Channels   Channels          `yaml:"channels"`
	Timeout    Duration          `yaml:"timeout"`
	Priority   string            `yaml:"priority"`
	SendImages bool              `yaml:"send_images"`
}

// Channels holds notification channel destinations.
type Channels struct {
	Telegram string `yaml:"telegram"`
	Slack    string `yaml:"slack"`
}

// Duration wraps time.Duration for YAML unmarshalling (e.g. "3m", "30s").
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	parsed, err := time.ParseDuration(value.Value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", value.Value, err)
	}
	d.Duration = parsed
	return nil
}

func (d Duration) MarshalYAML() (any, error) {
	return d.String(), nil
}

// LabelsMatch checks if all required labels are present in actual with matching values.
func LabelsMatch(required, actual map[string]string) bool {
	for k, v := range required {
		if actual[k] != v {
			return false
		}
	}
	return true
}

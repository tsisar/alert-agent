package storage

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/tsisar/alert-agent/internal/scenario"
	"github.com/tsisar/extended-log-go/log"
	"gorm.io/gorm"
)

const defaultTimeout = 2 * time.Minute

// JSONMap stores map[string]string as a JSON column.
type JSONMap map[string]string

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	return string(b), err
}

func (m *JSONMap) Scan(value any) error {
	if value == nil {
		*m = JSONMap{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return json.Unmarshal([]byte("{}"), m)
	}
	return json.Unmarshal(bytes, m)
}

// JSONStringSlice stores []string as a JSON column.
type JSONStringSlice []string

func (s JSONStringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *JSONStringSlice) Scan(value any) error {
	if value == nil {
		*s = JSONStringSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return json.Unmarshal([]byte("[]"), s)
	}
	return json.Unmarshal(bytes, s)
}

// Scenario is the GORM model for alert investigation scenarios.
type Scenario struct {
	gorm.Model
	Name            string          `gorm:"uniqueIndex;not null"`
	OrderIndex      int             `gorm:"not null;default:0"`
	Match           JSONMap         `gorm:"type:text"`
	Prompt          string          `gorm:"type:text;not null"`
	Tools           JSONStringSlice `gorm:"type:text"`
	ChannelTelegram string
	ChannelSlack    string
	Timeout         string `gorm:"not null;default:'2m'"`
	Priority        string `gorm:"not null;default:'normal'"`
	SendImages      bool   `gorm:"not null;default:false"`
}

// ToDomain converts the GORM model to the domain Scenario type.
func (s *Scenario) ToDomain() scenario.Scenario {
	d, err := time.ParseDuration(s.Timeout)
	if err != nil {
		log.Warnf("scenario %q has invalid timeout %q, using default %s", s.Name, s.Timeout, defaultTimeout)
		d = defaultTimeout
	}
	return scenario.Scenario{
		Name:       s.Name,
		Match:      map[string]string(s.Match),
		Prompt:     s.Prompt,
		Tools:      []string(s.Tools),
		Channels:   scenario.Channels{Telegram: s.ChannelTelegram, Slack: s.ChannelSlack},
		Timeout:    scenario.Duration{Duration: d},
		Priority:   s.Priority,
		SendImages: s.SendImages,
	}
}

// Prompt is the GORM model for LLM prompt templates.
type Prompt struct {
	gorm.Model
	Key   string `gorm:"uniqueIndex;not null"`
	Value string `gorm:"type:text;not null"`
}

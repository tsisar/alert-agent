package web

import (
	"testing"

	"github.com/tsisar/alert-agent/internal/storage"
)

func scenarioFixture(id uint, name string, order int, match storage.JSONMap) storage.Scenario {
	s := storage.Scenario{Name: name, OrderIndex: order, Match: match}
	s.ID = id
	return s
}

func labelled(id uint, name string, order int) storage.Scenario {
	return scenarioFixture(id, name, order, storage.JSONMap{"alertname": name})
}

func TestReorderScenarios(t *testing.T) {
	list := []storage.Scenario{
		scenarioFixture(9, "fallback", 0, nil),
		labelled(1, "a", 0),
		labelled(2, "b", 0),
		labelled(3, "c", 5),
	}

	changed := reorderScenarios(list, 3, true)
	got := map[string]int{}
	for _, s := range changed {
		got[s.Name] = s.OrderIndex
	}
	want := map[string]int{"a": 10, "c": 20, "b": 30, "fallback": 40}
	for name, idx := range want {
		if got[name] != idx {
			t.Errorf("%s: order_index = %d, want %d (changed=%v)", name, got[name], idx, got)
		}
	}

	if out := reorderScenarios(list, 1, true); out != nil {
		t.Errorf("moving the first scenario up should be a no-op, got %v", out)
	}
	if out := reorderScenarios(list, 9, false); out != nil {
		t.Errorf("a fallback scenario has no position to move, got %v", out)
	}
}

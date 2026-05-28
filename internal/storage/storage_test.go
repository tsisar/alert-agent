package storage

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&Scenario{}, &Prompt{}); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}
	return db
}

// --- Scenario Repository Tests ---

func TestScenarioRepo_MatchExact(t *testing.T) {
	db := testDB(t)
	db.Create(&Scenario{Name: "pg", OrderIndex: 0, Match: JSONMap{"alertname": "PGDown"}, Prompt: "check pg", Timeout: "1m", Priority: "high"})
	db.Create(&Scenario{Name: "default", OrderIndex: 10, Match: JSONMap{}, Prompt: "default", Timeout: "1m", Priority: "normal"})

	repo := NewScenarioRepository(db)
	sc := repo.Match(map[string]string{"alertname": "PGDown", "instance": "db-1"})
	if sc == nil || sc.Name != "pg" {
		t.Fatalf("expected 'pg', got %v", sc)
	}
}

func TestScenarioRepo_MatchCatchAll(t *testing.T) {
	db := testDB(t)
	db.Create(&Scenario{Name: "specific", OrderIndex: 0, Match: JSONMap{"alertname": "OnlyThis"}, Prompt: "x", Timeout: "1m", Priority: "normal"})
	db.Create(&Scenario{Name: "default", OrderIndex: 10, Match: JSONMap{}, Prompt: "catch all", Timeout: "1m", Priority: "normal"})

	repo := NewScenarioRepository(db)
	sc := repo.Match(map[string]string{"alertname": "Other"})
	if sc == nil || sc.Name != "default" {
		t.Fatalf("expected 'default', got %v", sc)
	}
}

func TestScenarioRepo_MatchNone(t *testing.T) {
	db := testDB(t)
	db.Create(&Scenario{Name: "specific", OrderIndex: 0, Match: JSONMap{"alertname": "OnlyThis"}, Prompt: "x", Timeout: "1m", Priority: "normal"})

	repo := NewScenarioRepository(db)
	sc := repo.Match(map[string]string{"alertname": "Other"})
	if sc != nil {
		t.Fatalf("expected nil, got %v", sc)
	}
}

func TestScenarioRepo_MatchMultipleLabels(t *testing.T) {
	db := testDB(t)
	db.Create(&Scenario{Name: "cpu-crit", OrderIndex: 0, Match: JSONMap{"alertname": "CPU", "severity": "critical"}, Prompt: "x", Timeout: "1m", Priority: "high"})
	db.Create(&Scenario{Name: "default", OrderIndex: 10, Match: JSONMap{}, Prompt: "default", Timeout: "1m", Priority: "normal"})

	repo := NewScenarioRepository(db)

	// Partial match should fall through to default.
	sc := repo.Match(map[string]string{"alertname": "CPU", "severity": "warning"})
	if sc == nil || sc.Name != "default" {
		t.Fatalf("expected 'default', got %v", sc)
	}

	// Full match.
	sc = repo.Match(map[string]string{"alertname": "CPU", "severity": "critical"})
	if sc == nil || sc.Name != "cpu-crit" {
		t.Fatalf("expected 'cpu-crit', got %v", sc)
	}
}

func TestScenarioRepo_OrderIndex(t *testing.T) {
	db := testDB(t)
	// Higher index scenario first in DB, but lower index should match first.
	db.Create(&Scenario{Name: "second", OrderIndex: 20, Match: JSONMap{"alertname": "CPU"}, Prompt: "x", Timeout: "1m", Priority: "normal"})
	db.Create(&Scenario{Name: "first", OrderIndex: 10, Match: JSONMap{"alertname": "CPU"}, Prompt: "x", Timeout: "1m", Priority: "high"})

	repo := NewScenarioRepository(db)
	sc := repo.Match(map[string]string{"alertname": "CPU"})
	if sc == nil || sc.Name != "first" {
		t.Fatalf("expected 'first' (lower OrderIndex), got %v", sc)
	}
}

func TestScenarioRepo_FindByName(t *testing.T) {
	db := testDB(t)
	db.Create(&Scenario{Name: "test-sc", OrderIndex: 0, Match: JSONMap{}, Prompt: "x", Timeout: "2m", Priority: "normal"})

	repo := NewScenarioRepository(db)
	sc := repo.FindByName("test-sc")
	if sc == nil || sc.Name != "test-sc" {
		t.Fatal("expected to find scenario by name")
	}

	sc = repo.FindByName("nonexistent")
	if sc != nil {
		t.Fatal("expected nil for nonexistent name")
	}
}

func TestScenarioRepo_CRUD(t *testing.T) {
	db := testDB(t)
	repo := NewScenarioRepository(db)

	// Create
	s := &Scenario{Name: "new", OrderIndex: 0, Match: JSONMap{"a": "b"}, Prompt: "test", Timeout: "1m", Priority: "normal"}
	if err := repo.Create(s); err != nil {
		t.Fatalf("create: %v", err)
	}

	// List
	list, err := repo.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Name != "new" {
		t.Fatalf("expected 1 scenario 'new', got %v", list)
	}

	// Update
	list[0].Priority = "high"
	if err := repo.Update(&list[0]); err != nil {
		t.Fatalf("update: %v", err)
	}
	sc := repo.FindByName("new")
	if sc == nil || sc.Priority != "high" {
		t.Fatal("expected updated priority")
	}

	// Delete
	if err := repo.Delete(list[0].ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	sc = repo.FindByName("new")
	if sc != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestScenarioRepo_CacheInvalidation(t *testing.T) {
	db := testDB(t)
	repo := NewScenarioRepository(db)

	// Initial state: no scenarios.
	sc := repo.Match(map[string]string{"alertname": "X"})
	if sc != nil {
		t.Fatal("expected nil initially")
	}

	// Insert directly into DB (bypassing repo).
	db.Create(&Scenario{Name: "direct", OrderIndex: 0, Match: JSONMap{"alertname": "X"}, Prompt: "x", Timeout: "1m", Priority: "normal"})

	// Without invalidation, cache is stale.
	sc = repo.Match(map[string]string{"alertname": "X"})
	if sc != nil {
		t.Fatal("expected nil before invalidation")
	}

	// After invalidation, should find the new scenario.
	repo.Invalidate()
	sc = repo.Match(map[string]string{"alertname": "X"})
	if sc == nil || sc.Name != "direct" {
		t.Fatalf("expected 'direct' after invalidation, got %v", sc)
	}
}

func TestScenarioRepo_ToDomain(t *testing.T) {
	db := testDB(t)
	db.Create(&Scenario{
		Name:            "full",
		OrderIndex:      5,
		Match:           JSONMap{"alertname": "Test"},
		Prompt:          "investigate",
		Tools:           JSONStringSlice{"query_prometheus", "get_panel_image"},
		ChannelTelegram: "-100",
		ChannelSlack:    "#ops",
		Timeout:         "3m",
		Priority:        "high",
		SendImages:      true,
	})

	repo := NewScenarioRepository(db)
	sc := repo.FindByName("full")
	if sc == nil {
		t.Fatal("expected scenario")
	}
	if sc.Channels.Telegram != "-100" || sc.Channels.Slack != "#ops" {
		t.Fatalf("channels mismatch: %v", sc.Channels)
	}
	if len(sc.Tools) != 2 || sc.Tools[0] != "query_prometheus" {
		t.Fatalf("tools mismatch: %v", sc.Tools)
	}
	if sc.Timeout.String() != "3m0s" {
		t.Fatalf("timeout mismatch: %v", sc.Timeout)
	}
	if !sc.SendImages {
		t.Fatal("expected send_images=true")
	}
}

// --- Prompt Repository Tests ---

func TestPromptRepo_GetAll(t *testing.T) {
	db := testDB(t)
	db.Create(&Prompt{Key: "system", Value: "sys"})
	db.Create(&Prompt{Key: "summary", Value: "sum"})
	db.Create(&Prompt{Key: "resolved", Value: "res"})
	db.Create(&Prompt{Key: "paused", Value: "pau"})

	repo := NewPromptRepository(db)
	ps, err := repo.GetAll()
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if ps.System != "sys" || ps.Summary != "sum" || ps.Resolved != "res" || ps.Paused != "pau" {
		t.Fatalf("unexpected prompts: %+v", ps)
	}
}

func TestPromptRepo_Get(t *testing.T) {
	db := testDB(t)
	db.Create(&Prompt{Key: "system", Value: "test-system"})

	repo := NewPromptRepository(db)
	v, err := repo.Get("system")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if v != "test-system" {
		t.Fatalf("expected 'test-system', got %q", v)
	}

	_, err = repo.Get("unknown")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestPromptRepo_Set(t *testing.T) {
	db := testDB(t)
	repo := NewPromptRepository(db)

	if err := repo.Set("system", "v1"); err != nil {
		t.Fatalf("set: %v", err)
	}
	v, _ := repo.Get("system")
	if v != "v1" {
		t.Fatalf("expected 'v1', got %q", v)
	}

	// Update existing.
	if err := repo.Set("system", "v2"); err != nil {
		t.Fatalf("set update: %v", err)
	}
	v, _ = repo.Get("system")
	if v != "v2" {
		t.Fatalf("expected 'v2', got %q", v)
	}
}

func TestPromptRepo_CacheInvalidation(t *testing.T) {
	db := testDB(t)
	db.Create(&Prompt{Key: "system", Value: "original"})

	repo := NewPromptRepository(db)
	v, _ := repo.Get("system")
	if v != "original" {
		t.Fatalf("expected 'original', got %q", v)
	}

	// Direct DB update (bypassing repo).
	db.Model(&Prompt{}).Where("key = ?", "system").Update("value", "changed")

	// Stale cache.
	v, _ = repo.Get("system")
	if v != "original" {
		t.Fatal("expected stale cache")
	}

	// After invalidation.
	repo.Invalidate()
	v, _ = repo.Get("system")
	if v != "changed" {
		t.Fatalf("expected 'changed' after invalidation, got %q", v)
	}
}

// --- Seed Tests ---

func TestSeed(t *testing.T) {
	db := testDB(t)

	if err := Seed(db); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var scenarioCount int64
	db.Model(&Scenario{}).Count(&scenarioCount)
	if scenarioCount == 0 {
		t.Fatal("expected scenarios to be seeded")
	}

	var promptCount int64
	db.Model(&Prompt{}).Count(&promptCount)
	if promptCount != 4 {
		t.Fatalf("expected 4 prompts, got %d", promptCount)
	}
}

func TestSeed_Idempotent(t *testing.T) {
	db := testDB(t)

	if err := Seed(db); err != nil {
		t.Fatalf("first seed: %v", err)
	}

	var count1 int64
	db.Model(&Scenario{}).Count(&count1)

	if err := Seed(db); err != nil {
		t.Fatalf("second seed: %v", err)
	}

	var count2 int64
	db.Model(&Scenario{}).Count(&count2)

	if count1 != count2 {
		t.Fatalf("seed not idempotent: %d -> %d", count1, count2)
	}
}

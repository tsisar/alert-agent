package scenario

import (
	"testing"
)

func TestLabelsMatch_ExactMatch(t *testing.T) {
	required := map[string]string{"alertname": "HighCPU"}
	actual := map[string]string{"alertname": "HighCPU", "instance": "web-1"}

	if !LabelsMatch(required, actual) {
		t.Fatal("expected match")
	}
}

func TestLabelsMatch_MultipleLabels(t *testing.T) {
	required := map[string]string{"alertname": "HighCPU", "severity": "critical"}
	actual := map[string]string{"alertname": "HighCPU", "severity": "critical", "node": "worker-1"}

	if !LabelsMatch(required, actual) {
		t.Fatal("expected match")
	}
}

func TestLabelsMatch_Mismatch(t *testing.T) {
	required := map[string]string{"alertname": "HighCPU", "severity": "critical"}
	actual := map[string]string{"alertname": "HighCPU", "severity": "warning"}

	if LabelsMatch(required, actual) {
		t.Fatal("expected no match")
	}
}

func TestLabelsMatch_MissingLabel(t *testing.T) {
	required := map[string]string{"alertname": "HighCPU", "severity": "critical"}
	actual := map[string]string{"alertname": "HighCPU"}

	if LabelsMatch(required, actual) {
		t.Fatal("expected no match for missing label")
	}
}

func TestLabelsMatch_EmptyRequired(t *testing.T) {
	required := map[string]string{}
	actual := map[string]string{"alertname": "HighCPU"}

	if !LabelsMatch(required, actual) {
		t.Fatal("empty required should match anything")
	}
}

func TestLabelsMatch_EmptyBoth(t *testing.T) {
	if !LabelsMatch(map[string]string{}, map[string]string{}) {
		t.Fatal("empty both should match")
	}
}

package territory

import (
	"testing"

	"github.com/terrarun/backend/internal/domain"
)

func TestAddHP(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		maxHP    int
		expected int
	}{
		{"below cap", 3, 10, 4},
		{"at cap", 10, 10, 10},
		{"over cap", 12, 10, 10},
		{"zero current", 0, 10, 1},
		{"empty, max 1", 0, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AddHP(tt.current, tt.maxHP)
			if got != tt.expected {
				t.Errorf("AddHP(%d, %d) = %d, want %d", tt.current, tt.maxHP, got, tt.expected)
			}
		})
	}
}

func TestRemoveHP(t *testing.T) {
	tests := []struct {
		name        string
		current     int
		wantHP      int
		wantFlipped bool
	}{
		{"normal remove", 5, 4, false},
		{"at 2", 2, 1, false},
		{"at 1, flips", 1, 1, true},
		{"at 0, flips", 0, 1, true},
		{"at 10", 10, 9, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newHP, flipped := RemoveHP(tt.current)
			if newHP != tt.wantHP || flipped != tt.wantFlipped {
				t.Errorf("RemoveHP(%d) = (%d, %v), want (%d, %v)",
					tt.current, newHP, flipped, tt.wantHP, tt.wantFlipped)
			}
		})
	}
}

func TestDailyDecay(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		expected int
	}{
		{"normal decay", 5, 4},
		{"at 2", 2, 1},
		{"at 1, floor", 1, 1},
		{"at 0, floor", 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DailyDecay(tt.current)
			if got != tt.expected {
				t.Errorf("DailyDecay(%d) = %d, want %d", tt.current, got, tt.expected)
			}
		})
	}
}

func TestComputeCaptureTransition(t *testing.T) {
	neon := domain.FactionNeon
	umbra := domain.FactionUmbra
	cfg := CaptureConfig{BasePoints: 10, MaxHP: 10, StealBonusPct: 0.5}

	t.Run("unclaimed hex", func(t *testing.T) {
		hex := &domain.Hex{HP: 0, OwnedBy: nil}
		tr := ComputeCaptureTransition(hex, neon, cfg)
		if !tr.Flipped || tr.ToHP != 1 || tr.FromHP != 0 || tr.TerritoryPoints != 10 {
			t.Errorf("unexpected transition: %+v", tr)
		}
	})

	t.Run("same faction, below cap", func(t *testing.T) {
		hex := &domain.Hex{HP: 3, OwnedBy: &neon}
		tr := ComputeCaptureTransition(hex, neon, cfg)
		if tr.Flipped || tr.ToHP != 4 || tr.FromHP != 3 || tr.TerritoryPoints != 10 {
			t.Errorf("unexpected transition: %+v", tr)
		}
	})

	t.Run("same faction, at cap", func(t *testing.T) {
		hex := &domain.Hex{HP: 10, OwnedBy: &neon}
		tr := ComputeCaptureTransition(hex, neon, cfg)
		if tr.Flipped || tr.ToHP != 10 || tr.FromHP != 10 || tr.TerritoryPoints != 10 {
			t.Errorf("unexpected transition: %+v", tr)
		}
	})

	t.Run("opponent faction, no flip", func(t *testing.T) {
		hex := &domain.Hex{HP: 5, OwnedBy: &umbra}
		tr := ComputeCaptureTransition(hex, neon, cfg)
		if tr.Flipped || tr.ToHP != 4 || tr.FromHP != 5 || tr.TerritoryPoints != 10 {
			t.Errorf("unexpected transition: %+v", tr)
		}
	})

	t.Run("opponent faction, flip from 1", func(t *testing.T) {
		hex := &domain.Hex{HP: 1, OwnedBy: &umbra}
		tr := ComputeCaptureTransition(hex, neon, cfg)
		if !tr.Flipped || tr.ToHP != 1 || tr.FromHP != 1 {
			t.Errorf("unexpected transition: %+v", tr)
		}
		if tr.TerritoryPoints != 15 {
			t.Errorf("TerritoryPoints = %d, want 15 (base 10 + 50%% steal bonus)", tr.TerritoryPoints)
		}
	})

	t.Run("opponent faction, flip from 0", func(t *testing.T) {
		hex := &domain.Hex{HP: 0, OwnedBy: &umbra}
		tr := ComputeCaptureTransition(hex, neon, cfg)
		if !tr.Flipped || tr.ToHP != 1 || tr.FromHP != 0 {
			t.Errorf("unexpected transition: %+v", tr)
		}
		if tr.TerritoryPoints != 15 {
			t.Errorf("TerritoryPoints = %d, want 15", tr.TerritoryPoints)
		}
	})
}

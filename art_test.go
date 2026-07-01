package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestConditionForCode(t *testing.T) {
	cases := []struct {
		code int
		want condition
	}{
		{0, condClear}, {1, condPartlyCloudy}, {2, condPartlyCloudy}, {3, condOvercast},
		{45, condFog}, {51, condDrizzle}, {56, condFreezingRain}, {61, condRain},
		{66, condFreezingRain}, {73, condSnow}, {77, condSnowGrains},
		{81, condRainShowers}, {85, condSnowShowers}, {95, condThunderstorm}, {99, condThunderstormHail},
	}
	for _, c := range cases {
		if got := conditionForCode(c.code); got != c.want {
			t.Errorf("conditionForCode(%d) = %d, want %d", c.code, got, c.want)
		}
	}
}

func TestConditionGroups(t *testing.T) {
	// every gallery condition must have a label, slug, and one of the 4 groups
	groups := map[string]bool{"Clear Skies": true, "Precipitation": true, "Snow": true, "Storms": true}
	if len(allConditions) != 14 {
		t.Fatalf("expected 14 conditions, got %d", len(allConditions))
	}
	for _, c := range allConditions {
		if c.label() == "" || c.slug() == "" {
			t.Errorf("condition %d missing label/slug", c)
		}
		if !groups[c.group()] {
			t.Errorf("condition %q has unknown group %q", c.slug(), c.group())
		}
		if idx, ok := demoIndexFor(c.slug()); !ok || allConditions[idx] != c {
			t.Errorf("demoIndexFor(%q) failed to round-trip", c.slug())
		}
	}
}

// canvas run-coalescing should not drop characters.
func TestCanvasString(t *testing.T) {
	cv := newCanvas(3, 2)
	cv.set(0, 0, 'a', "1", "")
	cv.set(1, 0, 'b', "1", "")
	cv.set(2, 0, 'c', "2", "")
	got := cv.String()
	for _, r := range "abc" {
		if !contains(got, r) {
			t.Errorf("canvas output missing %q", r)
		}
	}
	if !contains(got, '\n') {
		t.Errorf("canvas output missing row separator")
	}
}

func contains(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

func TestDemoCycle(t *testing.T) {
	if _, ok := demoIndexFor("thunderstorm-hail"); !ok {
		t.Fatal("slug 'thunderstorm-hail' should resolve")
	}
	m := initialModel()
	m.width, m.height = 80, 24
	mi, _ := m.Update(keyPress("s"))
	m = mi.(model)
	if !m.demo || m.demoIdx != 0 || m.cond != condClear {
		t.Fatalf("first 's' should enter demo at Clear, got demo=%v idx=%d", m.demo, m.demoIdx)
	}
	seen := map[condition]bool{condClear: true}
	for i := 0; i < len(allConditions)-1; i++ {
		mi, _ = m.Update(keyPress("s"))
		m = mi.(model)
		seen[m.cond] = true
	}
	if len(seen) != len(allConditions) {
		t.Fatalf("cycling did not visit all %d conditions, saw %d", len(allConditions), len(seen))
	}
	mi, _ = m.Update(keyPress("s"))
	m = mi.(model)
	if m.demoIdx != 0 {
		t.Fatalf("'s' should wrap to 0, got %d", m.demoIdx)
	}
	mi, _ = m.Update(keyPress("n"))
	m = mi.(model)
	if !m.demoNight || m.w.isDay {
		t.Fatalf("'n' should switch to night, got demoNight=%v isDay=%v", m.demoNight, m.w.isDay)
	}
}

func keyPress(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

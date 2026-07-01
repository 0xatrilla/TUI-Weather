package main

import (
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	loading state = iota
	ready
	failed
)

type frameMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second/12, func(t time.Time) tea.Msg { return frameMsg(t) })
}

type model struct {
	state         state
	editing       bool
	err           string
	place         string
	lat, lon      float64
	w             weatherMsg
	cond          condition
	world         int
	input         textinput.Model
	width, height int

	// demo/simulation mode
	demo      bool
	demoIdx   int
	demoNight bool

	// animation state
	rng       *rand.Rand
	frame     int
	precip    []particle
	splashes  []splash
	fog       []particle
	clouds    []cloud
	stars     []particle
	birds     []bird
	smoke     []particle
	fireflies []particle
	plane     planeState
	bolt      []boltSeg
	boltLife  int
	flash     int
}

// demoWeather returns plausible sample data for a simulated condition.
func demoWeather(c condition) (temp, wind, precip float64, hum int) {
	switch c {
	case condClear:
		return 24, 6, 0, 40
	case condPartlyCloudy:
		return 21, 10, 0, 50
	case condCloudy:
		return 18, 12, 0, 62
	case condOvercast:
		return 15, 14, 0, 72
	case condFog:
		return 9, 4, 0, 96
	case condDrizzle:
		return 12, 8, 0.3, 88
	case condRain:
		return 11, 16, 2.4, 90
	case condFreezingRain:
		return 0, 18, 3.1, 93
	case condRainShowers:
		return 14, 20, 4.0, 84
	case condSnow:
		return -2, 12, 1.4, 92
	case condSnowGrains:
		return -1, 6, 0.4, 89
	case condSnowShowers:
		return -3, 18, 1.0, 90
	case condThunderstorm:
		return 19, 26, 6.5, 82
	default: // thunderstorm & hail
		return 17, 32, 8.0, 80
	}
}

// demoIndexFor maps a --simulate name (weathr slug) to a gallery index.
func demoIndexFor(name string) (int, bool) {
	want := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(name, "_", "-")))
	alias := map[string]string{
		"partly": "partly-cloudy", "thunder": "thunderstorm",
		"storm": "thunderstorm", "hail": "thunderstorm-hail",
		"showers": "rain-showers", "grains": "snow-grains",
	}
	if a, ok := alias[want]; ok {
		want = a
	}
	for i, c := range allConditions {
		if c.slug() == want {
			return i, true
		}
	}
	return 0, false
}

// applyDemo loads the current gallery condition as the displayed weather.
func (m *model) applyDemo() {
	c := allConditions[m.demoIdx]
	temp, wind, precip, hum := demoWeather(c)
	nowMin := 720 // midday
	if m.demoNight {
		nowMin = 60
	}
	code := demoCode[c]
	m.w = weatherMsg{
		code: code, temp: temp, feels: temp - 1, wind: wind, precip: precip,
		humidity: hum, hi: temp + 3, lo: temp - 4, isDay: !m.demoNight,
		nowMin: nowMin, sunriseMin: 360, sunsetMin: 1080,
	}
	m.cond = c
	m.place = c.group() + " — " + c.label()
	m.precip, m.clouds = nil, nil // rebuild pools for the new condition
	m.state = ready
	m.demo = true
}

// demoCode is a representative WMO code per condition (for the HUD/data path).
var demoCode = map[condition]int{
	condClear: 0, condPartlyCloudy: 2, condCloudy: 3, condOvercast: 3,
	condFog: 45, condDrizzle: 51, condRain: 63, condFreezingRain: 66, condRainShowers: 81,
	condSnow: 73, condSnowGrains: 77, condSnowShowers: 85,
	condThunderstorm: 95, condThunderstormHail: 96,
}

func initialModel() model {
	in := textinput.New()
	in.Placeholder = "city name"
	in.CharLimit = 60
	in.Prompt = "city ⇢ "
	return model{
		state: loading,
		input: in,
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (m model) Init() tea.Cmd {
	if m.demo {
		return tick() // started in simulation mode: no network fetch
	}
	return tea.Batch(fetchIPLocation, tick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		// rebuild size-dependent pools for the new dimensions
		m.stars, m.birds, m.smoke, m.fireflies = nil, nil, nil, nil
		m.precip, m.clouds, m.fog, m.splashes = nil, nil, nil, nil
		return m, nil

	case frameMsg:
		m.advanceAnim()
		return m, tick()

	case tea.KeyMsg:
		if m.editing {
			switch msg.String() {
			case "enter":
				name := strings.TrimSpace(m.input.Value())
				m.editing, m.input = false, blur(m.input)
				if name == "" {
					return m, nil
				}
				m.state = loading
				return m, geocodeCity(name)
			case "esc":
				m.editing, m.input = false, blur(m.input)
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "s":
			// enter simulation mode, or advance to the next condition
			if m.demo {
				m.demoIdx = (m.demoIdx + 1) % len(allConditions)
			} else {
				m.demoIdx = 0
			}
			m.applyDemo()
			return m, nil
		case "n":
			// toggle day/night (enters simulation so the change is visible)
			m.demoNight = !m.demoNight
			m.applyDemo()
			return m, nil
		case "w":
			// cycle scenery: countryside → city → beach
			m.world = (m.world + 1) % len(worldNames)
			return m, nil
		case "r":
			// return to live weather
			m.demo = false
			m.state = loading
			if m.lat == 0 && m.lon == 0 {
				return m, fetchIPLocation
			}
			return m, fetchWeather(m.lat, m.lon)
		case "e", "/":
			m.editing = true
			return m, m.input.Focus()
		}

	case locationMsg:
		m.place, m.lat, m.lon = msg.place, msg.lat, msg.lon
		return m, fetchWeather(msg.lat, msg.lon)

	case weatherMsg:
		m.w = msg
		m.cond = conditionForCode(msg.code)
		m.precip, m.clouds = nil, nil // rebuild pools for new condition
		m.state = ready
		m.demo = false // real data supersedes any simulation
		return m, nil

	case errMsg:
		m.state = failed
		m.err = msg.Error()
		return m, nil
	}
	return m, nil
}

func blur(in textinput.Model) textinput.Model {
	in.Blur()
	in.SetValue("")
	return in
}

var footerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

func (m model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "starting…"
	}
	var scene string
	switch m.state {
	case ready:
		scene = m.renderScene()
	case failed:
		scene = center(m.width, m.height-1,
			lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render("⚠  "+m.err)+
				"\n\npress r to retry · e for a city")
	default:
		scene = center(m.width, m.height-1, "fetching weather…")
	}

	hint := " s simulate · w scene · e city · r refresh · q quit"
	if m.demo {
		hint = " s next · n day/night · w scene · r live · q quit"
	}
	footer := footerStyle.Render(hint)
	if m.editing {
		footer = " " + m.input.View()
	}
	return scene + "\n" + footer
}

func center(w, h int, s string) string {
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, s)
}

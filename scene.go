package main

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// ---------- canvas: a colored character frame buffer ----------

type cell struct {
	r      rune
	fg, bg string
}

type canvas struct {
	w, h int
	c    []cell
}

func newCanvas(w, h int) *canvas {
	cv := &canvas{w: w, h: h, c: make([]cell, w*h)}
	for i := range cv.c {
		cv.c[i] = cell{r: ' '}
	}
	return cv
}

// set writes a rune at (x,y). Empty fg/bg leave existing color untouched.
func (cv *canvas) set(x, y int, r rune, fg, bg string) {
	if x < 0 || y < 0 || x >= cv.w || y >= cv.h {
		return
	}
	p := &cv.c[y*cv.w+x]
	p.r = r
	if fg != "" {
		p.fg = fg
	}
	if bg != "" {
		p.bg = bg
	}
}

// blit draws non-space runes of an art block at (x,y) (spaces are transparent).
func (cv *canvas) blit(x, y int, lines []string, fg, bg string) {
	for dy, line := range lines {
		dx := 0
		for _, r := range line {
			if r != ' ' {
				cv.set(x+dx, y+dy, r, fg, bg)
			}
			dx++
		}
	}
}

// writeOpaque draws every rune including spaces, covering what's behind.
func (cv *canvas) writeOpaque(x, y int, s, fg, bg string) {
	dx := 0
	for _, r := range s {
		cv.set(x+dx, y, r, fg, bg)
		dx++
	}
}

var styleCache = map[string]lipgloss.Style{}

func styled(fg, bg string) lipgloss.Style {
	key := fg + "|" + bg
	if s, ok := styleCache[key]; ok {
		return s
	}
	s := lipgloss.NewStyle()
	if fg != "" {
		s = s.Foreground(lipgloss.Color(fg))
	}
	if bg != "" {
		s = s.Background(lipgloss.Color(bg))
	}
	styleCache[key] = s
	return s
}

func (cv *canvas) String() string {
	var b strings.Builder
	for y := 0; y < cv.h; y++ {
		x := 0
		for x < cv.w {
			fg, bg := cv.c[y*cv.w+x].fg, cv.c[y*cv.w+x].bg
			var run []rune
			for x < cv.w {
				p := cv.c[y*cv.w+x]
				if p.fg != fg || p.bg != bg {
					break
				}
				run = append(run, p.r)
				x++
			}
			seg := string(run)
			if fg == "" && bg == "" {
				b.WriteString(seg)
			} else {
				b.WriteString(styled(fg, bg).Render(seg))
			}
		}
		if y < cv.h-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// ---------- animation state ----------

type particle struct{ x, y float64 }
type cloud struct {
	x, y    float64
	variant int
}
type splash struct {
	x, y, t int
}
type bird struct{ x, y, dir float64 }
type planeState struct {
	x      float64
	y      int
	active bool
}
type boltSeg struct {
	x, y int
	r    rune
}

var leafColors = []string{"208", "166", "172", "136", "94"}

func rainGlyphs(intensity int) []rune {
	switch intensity {
	case 0:
		return []rune{'.', ','}
	case 1:
		return []rune{'|', ':', '.'}
	case 2:
		return []rune{'|', ':'}
	default:
		return []rune{'/', '\\'}
	}
}

func snowGlyphs(intensity int) []rune {
	switch intensity {
	case 0:
		return []rune{'.', '·'}
	case 1:
		return []rune{'.', '·', '*'}
	default:
		return []rune{'*', '.', '·'}
	}
}

// precipTarget mirrors weathr's per-intensity particle counts.
func (m *model) precipTarget() int {
	w := m.width
	if m.cond.isRaining() {
		switch m.cond.rainIntensity() {
		case 0:
			return w / 4
		case 1:
			return w / 2
		case 2:
			return w
		default:
			return w * 3 / 2
		}
	}
	if m.cond.isSnowing() {
		switch m.cond.snowIntensity() {
		case 0:
			return w / 4
		case 1:
			return w / 2
		default:
			return w
		}
	}
	if m.cond == condClear {
		return 10 // drifting leaves
	}
	return 0
}

// ensureParticles (re)builds each particle pool as size or condition changes.
func (m *model) ensureParticles() {
	w, ground := m.width, m.groundTop()
	if w <= 0 || ground <= 0 {
		return
	}
	fw, fg := float64(w), float64(ground)

	if want := m.precipTarget(); len(m.precip) != want {
		m.precip = make([]particle, want)
		for i := range m.precip {
			m.precip[i] = particle{x: m.rng.Float64() * fw, y: -m.rng.Float64() * fg}
		}
	}

	if cwant := m.cond.cloudCount(); len(m.clouds) != cwant {
		m.clouds = make([]cloud, cwant)
		for i := range m.clouds {
			m.clouds[i] = cloud{
				x:       m.rng.Float64() * fw,
				y:       float64(1 + m.rng.Intn(max(1, ground/3))),
				variant: m.rng.Intn(len(cloudArts)),
			}
		}
	}

	fogWant := 0
	if m.cond.isFoggy() {
		fogWant = w * ground / 22
	}
	if len(m.fog) != fogWant {
		m.fog = make([]particle, fogWant)
		for i := range m.fog {
			m.fog[i] = particle{x: m.rng.Float64() * fw, y: fg*0.45 + m.rng.Float64()*fg*0.5}
		}
	}

	if m.stars == nil {
		n := w * ground / 40
		m.stars = make([]particle, n)
		for i := range m.stars {
			m.stars[i] = particle{x: float64(m.rng.Intn(w)), y: float64(m.rng.Intn(max(1, ground*2/3)))}
		}
	}
	if m.birds == nil {
		m.birds = make([]bird, 4)
		for i := range m.birds {
			m.birds[i] = bird{x: m.rng.Float64() * fw, y: float64(2 + m.rng.Intn(max(1, ground/3))), dir: 0.45}
		}
	}
	if m.smoke == nil {
		m.smoke = make([]particle, 8)
		for i := range m.smoke {
			m.smoke[i] = particle{x: 0, y: -1}
		}
	}
	if m.fireflies == nil {
		m.fireflies = make([]particle, 10)
		for i := range m.fireflies {
			m.fireflies[i] = particle{x: m.rng.Float64() * fw, y: fg - 1 - m.rng.Float64()*fg/4}
		}
	}
}

func (m *model) advanceAnim() {
	if m.width <= 0 {
		return
	}
	m.frame++
	m.ensureParticles()
	w := float64(m.width)
	ground := m.groundTop()
	fground := float64(ground)

	// precipitation
	switch {
	case m.cond.isRaining():
		in := m.cond.rainIntensity()
		speed := []float64{0.35, 0.6, 0.9, 1.7}[in]
		drift := 0.1 + m.w.wind/45
		if in == 3 {
			drift += 0.5
		}
		for i := range m.precip {
			m.precip[i].x += drift
			m.precip[i].y += speed
			if m.precip[i].y >= fground {
				if i%2 == 0 && m.rng.Intn(3) == 0 && len(m.splashes) < 120 {
					m.splashes = append(m.splashes, splash{x: int(m.precip[i].x), y: ground, t: 0})
				}
				m.precip[i] = particle{x: m.rng.Float64() * w, y: 0}
			} else if m.precip[i].x >= w {
				m.precip[i] = particle{x: 0, y: 0}
			}
		}
	case m.cond.isSnowing():
		speed := []float64{0.15, 0.25, 0.4}[m.cond.snowIntensity()]
		for i := range m.precip {
			m.precip[i].x += math.Sin(float64(m.frame)*0.06+float64(i))*0.3 + m.w.wind/120
			m.precip[i].y += speed
			if m.precip[i].y >= fground {
				m.precip[i] = particle{x: m.rng.Float64() * w, y: 0}
			}
		}
	case m.cond == condClear:
		for i := range m.precip {
			m.precip[i].x += math.Sin(float64(m.frame)*0.05+float64(i))*0.3 + 0.15
			m.precip[i].y += 0.2
			if m.precip[i].y >= fground || m.precip[i].x >= w {
				m.precip[i] = particle{x: m.rng.Float64() * w * 0.5, y: 0}
			}
		}
	}

	// splashes age out (. o O)
	for i := 0; i < len(m.splashes); {
		m.splashes[i].t++
		if m.splashes[i].t > 2 {
			m.splashes = append(m.splashes[:i], m.splashes[i+1:]...)
		} else {
			i++
		}
	}

	// fog wisps drift
	for i := range m.fog {
		m.fog[i].x += math.Sin(float64(m.frame)*0.02+float64(i))*0.12 + 0.04
		if m.fog[i].x >= w {
			m.fog[i].x = 0
		}
	}

	// clouds drift and wrap
	for i := range m.clouds {
		m.clouds[i].x += 0.12
		if m.clouds[i].x > w+8 {
			m.clouds[i].x = -8
		}
	}

	// lightning
	if m.flash > 0 {
		m.flash--
	}
	if m.boltLife > 0 {
		m.boltLife--
		if m.boltLife == 0 {
			m.bolt = nil
		}
	} else if m.cond.isThunder() && m.rng.Intn(45) == 0 {
		m.bolt = m.genBolt(ground)
		m.boltLife = 8
		m.flash = 2
	}

	// airplane
	if m.plane.active {
		m.plane.x += 0.8
		if m.plane.x > w+8 {
			m.plane.active = false
		}
	} else if m.rng.Intn(240) == 0 {
		m.plane = planeState{x: -8, y: 1 + m.rng.Intn(max(1, ground/3)), active: true}
	}

	// birds
	for i := range m.birds {
		m.birds[i].x += m.birds[i].dir
		m.birds[i].y += math.Sin(float64(m.frame)*0.1+float64(i)) * 0.06
		if m.birds[i].x > w+3 {
			m.birds[i].x = -3
			m.birds[i].y = float64(2 + m.rng.Intn(max(1, ground/3)))
		}
	}

	// chimney smoke (countryside only)
	if m.world == worldCountryside {
		cx, tipY := m.chimneyPos()
		if m.frame%5 == 0 {
			for i := range m.smoke {
				if m.smoke[i].y < 0 {
					m.smoke[i] = particle{x: float64(cx), y: float64(tipY - 1)}
					break
				}
			}
		}
		for i := range m.smoke {
			if m.smoke[i].y < 0 {
				continue
			}
			m.smoke[i].y -= 0.4
			m.smoke[i].x += 0.12
			if m.smoke[i].y < float64(tipY-9) {
				m.smoke[i].y = -1
			}
		}
	}

	// fireflies
	lo := fground - fground/4
	for i := range m.fireflies {
		m.fireflies[i].x += math.Sin(float64(m.frame)*0.05+float64(i*2)) * 0.25
		m.fireflies[i].y += math.Cos(float64(m.frame)*0.04+float64(i)) * 0.12
		if m.fireflies[i].y < lo {
			m.fireflies[i].y = lo
		}
		if m.fireflies[i].y > fground-1 {
			m.fireflies[i].y = fground - 1
		}
		if m.fireflies[i].x < 0 {
			m.fireflies[i].x += w
		}
		if m.fireflies[i].x >= w {
			m.fireflies[i].x -= w
		}
	}
}

// genBolt builds a jagged lightning bolt from near the top toward the ground.
func (m *model) genBolt(ground int) []boltSeg {
	if m.width < 12 {
		return nil
	}
	x := 5 + m.rng.Intn(max(1, m.width-10))
	y := 2
	yEnd := ground - 2
	segs := []boltSeg{{x, y, '+'}}
	for y < yEnd {
		dir := m.rng.Intn(3) - 1 // -1,0,1
		x += dir
		y++
		if x < 2 {
			x = 2
		}
		if x > m.width-3 {
			x = m.width - 3
		}
		r := '|'
		if dir < 0 {
			r = '/'
		} else if dir > 0 {
			r = '\\'
		}
		segs = append(segs, boltSeg{x, y, r})
		if m.rng.Intn(5) == 0 { // branch
			bx, by := x, y+1
			for k := 0; k < 3 && by < ground-1; k++ {
				segs = append(segs, boltSeg{bx, by, '\\'})
				bx++
				by++
			}
		}
	}
	return segs
}

// ---------- scene composition ----------

func drawGround(cv *canvas, ground, h int, st sceneStyle) {
	w := cv.w
	for x := 0; x < w; x++ {
		// grass tuft line
		switch r := hash2(x, ground) % 100; {
		case r < 5:
			cv.set(x, ground, '*', st.flowers[(x)%len(st.flowers)], "")
		case r < 15:
			cv.set(x, ground, ',', st.grassSec, "")
		default:
			cv.set(x, ground, '^', st.grass, "")
		}
		// soil below
		for y := ground + 1; y < h; y++ {
			switch r := hash2(x, y) % 100; {
			case r < 20:
				cv.set(x, y, '~', st.soil, "")
			case r < 25:
				cv.set(x, y, '.', st.soil, "")
			}
		}
	}
}

func hash2(x, y int) uint32 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	return h ^ (h >> 16)
}

func (m *model) groundTop() int {
	band := m.height / 3
	if band < 6 {
		band = 6
	}
	g := m.height - band
	if g < 3 {
		g = m.height - 2
	}
	if g < 1 {
		g = 1
	}
	return g
}

// sceneStyle holds the day/night color scheme (mirrors weathr's WorldSceneStyle).
type sceneStyle struct {
	grass, grassSec, soil string
	flowers               []string
	roof, window, trim    string
	wood, door            string
	tree, fence           string
}

func styleFor(tod string) sceneStyle {
	if tod == "night" {
		return sceneStyle{
			grass: "22", grassSec: "236", soil: "58",
			flowers: []string{"53", "88", "19", "58"},
			roof:    "90", window: "226", trim: "240",
			wood: "138", door: "94", tree: "236", fence: "247",
		}
	}
	return sceneStyle{
		grass: "34", grassSec: "22", soil: "94",
		flowers: []string{"13", "9", "51", "226"},
		roof:    "88", window: "51", trim: "240",
		wood: "180", door: "130", tree: "22", fence: "15",
	}
}

func arcPos(frac float64, w, skyH int) (int, int) {
	frac = math.Max(0, math.Min(1, frac))
	x := int(frac * float64(w-1))
	y := int(float64(skyH-2) - math.Sin(frac*math.Pi)*float64(skyH-3))
	if y < 0 {
		y = 0
	}
	return x, y
}

func (m *model) timeOfDay() string {
	w := m.w
	if w.nowMin < 0 || w.sunriseMin < 0 || w.sunsetMin < 0 {
		if w.isDay {
			return "day"
		}
		return "night"
	}
	near := func(a, b int) bool { d := a - b; return d < 35 && d > -35 }
	if near(w.nowMin, w.sunriseMin) || near(w.nowMin, w.sunsetMin) {
		return "dusk"
	}
	if w.nowMin >= w.sunriseMin && w.nowMin < w.sunsetMin {
		return "day"
	}
	return "night"
}

func (m *model) chimneyPos() (x, tipY int) {
	hx := m.houseX()
	roofBase := m.groundTop() - len(houseArt) + 3
	return hx + 8, roofBase - 1
}

func (m *model) houseX() int {
	hx := m.width/2 - len(houseArt[len(houseArt)-1])/2 - 4
	if hx < 1 {
		hx = 1
	}
	return hx
}

func (m *model) renderScene() string {
	w, h := m.width, m.height-1 // reserve last row for footer
	if w <= 0 || h <= 2 {
		return "loading…"
	}
	ground := m.groundTop()
	if ground >= h {
		ground = h - 1
	}
	tod := m.timeOfDay()
	st := styleFor(tod)
	cv := newCanvas(w, h)

	// no sky fill — everything is foreground ASCII on the terminal background.
	// the selected world paints the ground band (grass, street, or ocean+sand).
	m.drawWorldGround(cv, ground, h, st, tod)

	// stars at night
	if tod == "night" {
		for i, s := range m.stars {
			if (m.frame/6+i)%5 != 0 {
				cv.set(int(s.x), int(s.y), '·', "250", "")
			}
		}
	}

	// sun (two-frame shimmer) in fair weather, or moon at night
	if m.w.sunriseMin >= 0 && m.w.sunsetMin > m.w.sunriseMin {
		if tod == "night" {
			cx, cy := arcPos(0.5, w, ground)
			cv.blit(cx-5, cy, moonArt, "15", "")
		} else if !m.cond.isRaining() && !m.cond.isSnowing() && !m.cond.isThunder() &&
			!m.cond.isFoggy() && m.cond != condOvercast {
			frac := float64(m.w.nowMin-m.w.sunriseMin) / float64(m.w.sunsetMin-m.w.sunriseMin)
			cx, cy := arcPos(frac, w, ground)
			cv.blit(cx-10, cy, sunFrames[(m.frame/10)%2], "226", "")
		}
	}

	// clouds (color by condition, mirroring weathr)
	cloudFG := "240"
	switch {
	case m.cond == condClear:
		cloudFG = "15"
	case m.cond == condPartlyCloudy:
		cloudFG = "250"
	}
	for _, c := range m.clouds {
		cv.blit(int(c.x), int(c.y), cloudArts[c.variant], cloudFG, "")
	}

	// birds in fair daylight
	if tod != "night" && !m.cond.isRaining() && !m.cond.isSnowing() && !m.cond.isThunder() {
		for i, b := range m.birds {
			g := '⌒'
			if (m.frame/4+i)%2 == 0 {
				g = 'v'
			}
			cv.set(int(b.x), int(b.y), g, "226", "")
		}
	}

	// airplane
	if m.plane.active {
		px := int(m.plane.x)
		for t := 1; t <= 5; t++ {
			cv.set(px-t*2, m.plane.y, '·', "250", "")
		}
		cv.set(px, m.plane.y, '✈', "15", "")
	}

	// foreground scenery for the selected world (house/city/beach)
	m.drawWorldStructures(cv, ground, h, st, tod)

	// precipitation / effects overlay
	switch {
	case m.cond.isRaining():
		glyphs := rainGlyphs(m.cond.rainIntensity())
		for i, p := range m.precip {
			g := glyphs[i%len(glyphs)]
			fg := "240" // dim background layer
			if i%2 == 0 {
				fg = rainBrightColor(m.cond.rainIntensity())
			}
			cv.set(int(p.x), int(p.y), g, fg, "")
		}
		for _, s := range m.splashes {
			ch := []rune{'.', 'o', 'O'}[s.t]
			cv.set(s.x, s.y, ch, "15", "")
		}
	case m.cond.isSnowing():
		glyphs := snowGlyphs(m.cond.snowIntensity())
		for i, p := range m.precip {
			fg := "240"
			if i%2 == 0 {
				fg = "15"
			}
			cv.set(int(p.x), int(p.y), glyphs[i%len(glyphs)], fg, "")
		}
	case m.cond.isFoggy():
		wisps := []rune{'.', ',', '-', '~'}
		for i, p := range m.fog {
			fg := "250"
			if i%3 == 0 {
				fg = "240"
			}
			cv.set(int(p.x), int(p.y), wisps[i%len(wisps)], fg, "")
		}
	case m.cond == condClear && tod != "night":
		for i, p := range m.precip {
			cv.set(int(p.x), int(p.y), '❧', leafColors[i%len(leafColors)], "")
		}
	}

	// lightning
	if len(m.bolt) > 0 {
		fg := "226"
		if m.flash > 0 {
			fg = "15"
		}
		for _, s := range m.bolt {
			cv.set(s.x, s.y, s.r, fg, "")
		}
	}

	// fireflies on calm nights (not in the city)
	if tod == "night" && m.world != worldCity && (m.cond == condClear || m.cond.isCloudy()) {
		for i, f := range m.fireflies {
			if (m.frame/6+i)%3 == 0 {
				continue
			}
			cv.set(int(f.x), int(f.y), '·', "226", "")
		}
	}

	m.drawHUD(cv)
	return cv.String()
}

func rainBrightColor(intensity int) string {
	switch intensity {
	case 0:
		return "51" // cyan drizzle
	case 2:
		return "51" // cyan heavy
	default:
		return "15" // white light / storm
	}
}

// drawHouse renders the mansion with per-character colors and a long fence.
func (m *model) drawHouse(cv *canvas, ground int, st sceneStyle) {
	hx := m.houseX()
	topY := ground - len(houseArt)
	for i, line := range houseArt {
		row := topY + i
		dx := 0
		for _, ch := range line {
			if ch != ' ' {
				var fg string
				switch {
				case i <= 4:
					fg = st.roof
				case ch == '[' || ch == ']':
					fg = st.window
				case ch == '(' || ch == ')':
					fg = st.door
				case ch == '=':
					fg = st.trim
				default:
					fg = st.wood
				}
				cv.set(hx+dx, row, ch, fg, "")
			}
			dx++
		}
	}
	// extend the fence to the right across the yard
	fenceRow := topY + len(houseArt) - 1
	topRow := fenceRow - 1
	for x := hx + len(houseArt[len(houseArt)-1]); x < cv.w-1; x++ {
		if (x-hx)%2 == 0 {
			cv.set(x, fenceRow, '|', st.trim, "")
			cv.set(x, topRow, '_', st.wood, "")
		} else {
			cv.set(x, fenceRow, '=', st.trim, "")
			cv.set(x, topRow, '.', st.wood, "")
		}
	}
}

func (m *model) drawHUD(cv *canvas) {
	c := m.cond
	lines := []string{
		fmt.Sprintf(" %.0f°C  %s ", m.w.temp, c.label()),
		fmt.Sprintf(" feels %.0f°  hi %.0f° lo %.0f° ", m.w.feels, m.w.hi, m.w.lo),
		fmt.Sprintf(" wind %.0f km/h  hum %d%% ", m.w.wind, m.w.humidity),
	}
	title := truncate(m.place, 30)
	rc := utf8.RuneCountInString
	width := rc(title) + 2
	for _, l := range lines {
		if rc(l) > width {
			width = rc(l)
		}
	}
	fg, bg := "231", "236"
	top := "┌─ " + title + " " + strings.Repeat("─", maxInt(0, width-rc(title)-3)) + "┐"
	cv.writeOpaque(1, 1, top, fg, bg)
	for i, l := range lines {
		row := "│" + l + strings.Repeat(" ", maxInt(0, width-rc(l))) + "│"
		cv.writeOpaque(1, 2+i, row, fg, bg)
	}
	bottom := "└" + strings.Repeat("─", width) + "┘"
	cv.writeOpaque(1, 2+len(lines), bottom, fg, bg)
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

package main

import "math"

const (
	worldCountryside = iota
	worldCity
	worldBeach
)

var worldNames = []string{"countryside", "city", "beach"}

func worldIndex(name string) (int, bool) {
	for i, n := range worldNames {
		if n == name {
			return i, true
		}
	}
	return 0, false
}

// drawWorldGround paints the ground band for the current world.
func (m *model) drawWorldGround(cv *canvas, ground, h int, st sceneStyle, tod string) {
	switch m.world {
	case worldCity:
		m.drawStreet(cv, ground, h, tod)
	case worldBeach:
		m.drawBeach(cv, ground, h, tod)
	default:
		drawGround(cv, ground, h, st)
	}
}

// drawWorldStructures paints the foreground scenery for the current world.
func (m *model) drawWorldStructures(cv *canvas, ground, h int, st sceneStyle, tod string) {
	switch m.world {
	case worldCity:
		m.drawSkyline(cv, ground, tod)
	case worldBeach:
		m.drawBeachDecor(cv, ground)
	default:
		m.drawCountryside(cv, ground, st, tod)
	}
}

// ---------- countryside ----------

func (m *model) drawCountryside(cv *canvas, ground int, st sceneStyle, tod string) {
	cv.blit(2, ground-len(pineArt), pineArt, st.tree, "")
	cv.blit(cv.w-14, ground-len(treeArt), treeArt, st.tree, "")
	m.drawHouse(cv, ground, st)

	for _, s := range m.smoke {
		if s.y < 0 {
			continue
		}
		_, tipY := m.chimneyPos()
		g := 'o'
		switch rise := tipY - int(s.y); {
		case rise > 6:
			g = '˙'
		case rise > 3:
			g = '°'
		}
		fg := "15"
		if tod == "night" {
			fg = "245"
		}
		cv.set(int(s.x), int(s.y), g, fg, "")
	}
}

// ---------- city ----------

// drawStreet paints the asphalt with a dashed centre line.
func (m *model) drawStreet(cv *canvas, ground, h int, tod string) {
	w := cv.w
	mid := ground + (h-ground)/2
	for x := 0; x < w; x++ {
		cv.set(x, ground, '_', "244", "") // kerb / pavement
		for y := ground + 1; y < h; y++ {
			switch {
			case y == mid && x%4 < 2:
				cv.set(x, y, '-', "226", "") // lane markings
			case hash2(x, y)%100 < 12:
				cv.set(x, y, '.', "238", "") // asphalt speckle
			}
		}
	}
}

// drawSkyline renders a row of tower blocks of varying height with lit windows.
// Building sizes/window lighting are hashed from x so they stay stable per frame.
func (m *model) drawSkyline(cv *canvas, ground int, tod string) {
	w := cv.w
	night := tod == "night"
	for bx := 0; bx < w-3; {
		bw := 6 + int(hash2(bx, 7)%7) // 6..12 wide
		if bx+bw > w {
			bw = w - bx
		}
		if bw < 4 {
			break
		}
		// height varies: some short, some skyscrapers
		maxH := ground - 1
		bh := 3 + int(hash2(bx, 3)%uint32(max(1, maxH-3)))
		if bh > maxH {
			bh = maxH
		}
		ty := ground - bh
		drawBuilding(cv, bx, ty, bw, ground, night)
		bx += bw + 1 // 1-col gap between towers
	}
}

func drawBuilding(cv *canvas, bx, ty, bw, ground int, night bool) {
	concrete := "245"
	if night {
		concrete = "240"
	}
	// solid-ish silhouette with a light shade block, then windows over it
	for y := ty; y < ground; y++ {
		for x := bx; x < bx+bw; x++ {
			cv.set(x, y, '░', concrete, "")
		}
	}
	// antenna on tall towers
	if ground-ty > ground/2 {
		cv.set(bx+bw/2, ty-1, '|', concrete, "")
	}
	// windows on a 2x2 grid
	for y := ty + 1; y < ground-1; y += 2 {
		for x := bx + 1; x < bx+bw-1; x += 2 {
			lit := hash2(x, y)%3 == 0
			ch, col := '▪', "236"
			if night {
				if lit {
					col = "227" // warm glow
				} else {
					col = "234"
				}
			} else if lit {
				col = "117" // reflective glass
			} else {
				col = "239"
			}
			cv.set(x, y, ch, col, "")
		}
	}
}

// ---------- beach ----------

func (m *model) drawBeach(cv *canvas, ground, h int, tod string) {
	w := cv.w
	oceanDepth := (h - ground) / 2
	if oceanDepth < 2 {
		oceanDepth = 2
	}
	surf := ground + oceanDepth
	// animated ocean
	for y := ground; y < surf && y < h; y++ {
		depth := y - ground
		base := "45" // cyan near surf
		if depth == 1 {
			base = "39"
		} else if depth >= 2 {
			base = "26"
		}
		for x := 0; x < w; x++ {
			v := math.Sin(float64(x)*0.3+float64(m.frame)*0.12+float64(y)*0.6) +
				0.5*math.Sin(float64(x)*0.11-float64(m.frame)*0.05+float64(y))
			switch {
			case v > 1.25:
				cv.set(x, y, '·', "15", "") // whitecap
			case v > -0.2:
				cv.set(x, y, '~', base, "")
			}
		}
	}
	// wet-sand shoreline + dry sand
	for y := surf; y < h; y++ {
		for x := 0; x < w; x++ {
			r := hash2(x, y) % 100
			switch {
			case y == surf && (x+m.frame/3)%9 < 3:
				cv.set(x, y, '~', "251", "") // foam line
			case r < 12:
				cv.set(x, y, '.', "179", "")
			case r < 18:
				cv.set(x, y, ',', "137", "")
			case r < 22:
				cv.set(x, y, '·', "180", "")
			}
		}
	}
}

func (m *model) drawBeachDecor(cv *canvas, ground int) {
	surf := ground + (m.height-1-ground)/2
	if surf < ground+1 {
		surf = ground + 1
	}
	// palm tree on the left of the sand
	px := 4
	cv.blit(px, surf, palmFronds, "34", "")
	cv.blit(px, surf+len(palmFronds), palmTrunk, "94", "")
	// beach umbrella toward the right
	ux := cv.w - 16
	cv.blit(ux, surf, umbrellaCanopy, "196", "")
	cv.blit(ux, surf+len(umbrellaCanopy), umbrellaPole, "250", "")
}

var palmFronds = []string{
	`  \\ | //  `,
	` \_\|/_/ `,
	`    |    `,
}

var palmTrunk = []string{
	`   |   `,
	`   )   `,
	`   |   `,
}

var umbrellaCanopy = []string{
	`  _.-._  `,
	` /_/_\_\ `,
}

var umbrellaPole = []string{
	`    |    `,
	`    |    `,
}

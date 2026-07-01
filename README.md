# TUI Weather (`tuiwthr`)

A cute terminal weather app. It finds your location, fetches the current
conditions, and draws them as a living ASCII scene — a rayed sun and drifting
clouds, falling rain and snow, rolling fog, forked lightning, a crescent moon
with twinkling stars, birds, a chimney-smoking cottage, and more.

All weather data comes from [Open-Meteo](https://open-meteo.com/) and location
from [ip-api.com](https://ip-api.com/) — **no API keys required.**

```
┌─ London, England ────────┐        \   |   /
│ 18°C  Partly cloudy      │      '  .---.  '        .--.
│ feels 18°  hi 24° lo 17° │       ,'     ',      .-(    ).
│ wind 9 km/h  hum 70%     │   -- (         ) --  (___.__)_)
└──────────────────────────┘       ',     ,'
```

---

## Install

### Quick install (curl)

```bash
curl -fsSL https://raw.githubusercontent.com/0xatrilla/TUI-Weather/main/install.sh | bash
```

This builds from source and installs the `tuiwthr` command into a directory on
your `PATH` (`~/.local/bin`, `~/bin`, or `/usr/local/bin`). It requires
[Go](https://go.dev/dl/) and `git`. If the install directory isn't on your
`PATH`, the script tells you what to add to your shell profile.

Then just run:

```bash
tuiwthr
```

### Build & run from source

```bash
git clone https://github.com/0xatrilla/TUI-Weather.git
cd TUI-Weather
go build -o tuiwthr .   # produces the ./tuiwthr binary
./tuiwthr

# or run without building a binary:
go run .
```

### With `go install`

```bash
go install github.com/0xatrilla/TUI-Weather@latest
# installs a binary named "TUI-Weather" into $(go env GOPATH)/bin
```

---

## Features

- **Auto location** via IP, with a manual city override.
- **Live ASCII weather scene** at ~12 fps that reflects the real conditions.
- **Day / night cycle** driven by the location's actual sunrise & sunset:
  a sun that arcs across the sky by time of day, or a crescent moon with a
  twinkling star field at night.
- **14 weather conditions** in 4 groups, each animated:
  - **Clear Skies** — clear, partly-cloudy, cloudy, overcast
  - **Precipitation** — fog, drizzle, rain, freezing-rain, rain-showers
  - **Snow** — snow, snow-grains, snow-showers
  - **Storms** — thunderstorm, thunderstorm-hail
- **Intensity-aware effects** — drizzle vs. heavy rain vs. slanting storm, light
  flurries vs. heavy snow, ground splashes, drifting fog wisps, and branching
  lightning with a bright flash.
- **Ambient life** — drifting clouds, an occasional airplane, a flock of birds
  by day, fireflies on calm nights, autumn leaves on clear days, and rising
  chimney smoke.
- **Three sceneries** (press `w` to cycle):
  - **countryside** — a windowed cottage, pine & oak trees, grass and soil
  - **city** — a skyline of tower blocks with windows that glow at night
  - **beach** — animated ocean waves with a palm tree and a beach umbrella
- **Simulation / gallery mode** to preview every condition without waiting for
  the weather to change.

---

## Keyboard controls

| Key | Action |
|-----|--------|
| `s` | Enter **simulation mode** / advance to the next of the 14 conditions |
| `n` | Toggle **day / night** (in simulation mode) |
| `w` | Cycle **scenery**: countryside → city → beach |
| `e` or `/` | Enter a **city name** to look up |
| `r` | Return to **live** weather / refresh |
| `q` or `Ctrl-C` | Quit |

---

## Command-line flags

```
tuiwthr [flags]
```

| Flag | Description |
|------|-------------|
| `--simulate <condition>` | Start in simulation mode showing a condition (see list below) |
| `--night` | Simulate night (moon, stars, fireflies) |
| `--scene <name>` | Start in a scenery: `countryside`, `city`, or `beach` |

**Conditions** for `--simulate`:
`clear`, `partly-cloudy`, `cloudy`, `overcast`, `fog`, `drizzle`, `rain`,
`freezing-rain`, `rain-showers`, `snow`, `snow-grains`, `snow-showers`,
`thunderstorm`, `thunderstorm-hail`.

Examples:

```bash
tuiwthr --simulate thunderstorm-hail       # storm gallery
tuiwthr --simulate snow --scene city --night
tuiwthr --scene beach                       # live weather on a beach
```

---

## How it works

- **Location:** `http://ip-api.com/json/` for IP geolocation, and Open-Meteo's
  geocoding API for the manual city override.
- **Weather:** Open-Meteo's forecast API (`current` conditions, today's hi/lo,
  and sunrise/sunset for the day-night cycle). WMO weather codes are mapped to
  the 14 conditions.
- **Rendering:** a small character frame buffer is composed each tick and
  colored per element — no solid sky or ground fills, just foreground ASCII on
  your terminal background.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Lip Gloss](https://github.com/charmbracelet/lipgloss).

---

## Development

```bash
go test ./...   # condition mapping, canvas, and simulation-cycle tests
go vet ./...
go run .
```

---

## Credits & inspiration

- Inspired by [**veirt/weathr**](https://github.com/veirt/weathr) (GPL-3.0) —
  this is an independent Go reimplementation of the same idea and aesthetic.
- ASCII art in the style of [asciiart.eu](https://www.asciiart.eu/)
  (house & moon after Joan G. Stark).
- Weather data by [Open-Meteo](https://open-meteo.com/)
  ([CC BY 4.0](https://creativecommons.org/licenses/by/4.0/)).
- Geolocation by [ip-api.com](https://ip-api.com/).

## License

MIT — see [LICENSE](LICENSE).

package main

// Weather conditions mirror veirt/weathr's model (14 conditions in 4 groups).
// ASCII art is in the style of asciiart.eu (house/moon after Joan G. Stark).
type condition int

const (
	condClear condition = iota
	condPartlyCloudy
	condCloudy
	condOvercast
	condFog
	condDrizzle
	condRain
	condFreezingRain
	condRainShowers
	condSnow
	condSnowGrains
	condSnowShowers
	condThunderstorm
	condThunderstormHail
)

// allConditions is the gallery order used by simulation mode.
var allConditions = []condition{
	condClear, condPartlyCloudy, condCloudy, condOvercast,
	condFog, condDrizzle, condRain, condFreezingRain, condRainShowers,
	condSnow, condSnowGrains, condSnowShowers,
	condThunderstorm, condThunderstormHail,
}

// conditionForCode maps a WMO weather code to a condition (matches weathr).
func conditionForCode(code int) condition {
	switch code {
	case 0:
		return condClear
	case 1, 2:
		return condPartlyCloudy
	case 3:
		return condOvercast
	case 45, 48:
		return condFog
	case 51, 53, 55:
		return condDrizzle
	case 56, 57, 66, 67:
		return condFreezingRain
	case 61, 63, 65:
		return condRain
	case 71, 73, 75:
		return condSnow
	case 77:
		return condSnowGrains
	case 80, 81, 82:
		return condRainShowers
	case 85, 86:
		return condSnowShowers
	case 95:
		return condThunderstorm
	case 96, 99:
		return condThunderstormHail
	default:
		return condClear
	}
}

func (c condition) label() string {
	return [...]string{
		"Clear", "Partly cloudy", "Cloudy", "Overcast", "Fog",
		"Drizzle", "Rain", "Freezing rain", "Rain showers",
		"Snow", "Snow grains", "Snow showers",
		"Thunderstorm", "Thunderstorm & hail",
	}[c]
}

func (c condition) slug() string {
	return [...]string{
		"clear", "partly-cloudy", "cloudy", "overcast", "fog",
		"drizzle", "rain", "freezing-rain", "rain-showers",
		"snow", "snow-grains", "snow-showers",
		"thunderstorm", "thunderstorm-hail",
	}[c]
}

func (c condition) group() string {
	switch c {
	case condClear, condPartlyCloudy, condCloudy, condOvercast:
		return "Clear Skies"
	case condFog, condDrizzle, condRain, condFreezingRain, condRainShowers:
		return "Precipitation"
	case condSnow, condSnowGrains, condSnowShowers:
		return "Snow"
	default:
		return "Storms"
	}
}

func (c condition) isThunder() bool {
	return c == condThunderstorm || c == condThunderstormHail
}

func (c condition) isRaining() bool {
	switch c {
	case condDrizzle, condRain, condRainShowers, condFreezingRain, condThunderstorm, condThunderstormHail:
		return true
	}
	return false
}

func (c condition) isSnowing() bool {
	return c == condSnow || c == condSnowGrains || c == condSnowShowers
}

func (c condition) isFoggy() bool { return c == condFog }

func (c condition) isCloudy() bool {
	return c == condPartlyCloudy || c == condCloudy || c == condOvercast
}

// rain intensity: 0 drizzle, 1 light, 2 heavy, 3 storm
func (c condition) rainIntensity() int {
	switch c {
	case condDrizzle:
		return 0
	case condFreezingRain, condThunderstorm:
		return 2
	case condThunderstormHail:
		return 3
	default:
		return 1 // rain, rain-showers
	}
}

// snow intensity: 0 grains (light), 1 showers (medium), 2 snow (heavy)
func (c condition) snowIntensity() int {
	switch c {
	case condSnowGrains:
		return 0
	case condSnowShowers:
		return 1
	default:
		return 2
	}
}

// cloudCount is how many clouds drift for this condition.
func (c condition) cloudCount() int {
	switch c {
	case condClear:
		return 1
	case condPartlyCloudy:
		return 3
	case condCloudy:
		return 5
	case condOvercast, condRain, condDrizzle, condRainShowers, condFreezingRain, condFog:
		return 6
	case condSnow, condSnowGrains, condSnowShowers:
		return 5
	default: // storms
		return 6
	}
}

// --- ASCII art assets (asciiart.eu style) ---

// A grand two-storey house with a windowed facade and a door.
var houseArt = []string{
	`        _   _._          `,
	`       |_|-'_~_` + "`" + `-._      `,
	`    _.-'-_~_-~-_-~-` + "`" + `-._  `,
	` _.-'_~-_~-_-~-_~_~-_~-_` + "`" + ``,
	`~~~~~~~~~~~~~~~~~~~~~~~~~~`,
	`  |  []  []   []   [] |  `,
	`  |          __   ___ |  `,
	`._|  []  []  |.|  [__]|_ `,
	`|=|________()|__|()___|=|`,
}

// Two sun frames (subtle shimmer between them). After a classic asciiart sun.
var sunFrames = [2][]string{
	{
		`      ;   :   ;      `,
		`   .   \_,!,_/   ,   `,
		"    `.,'     `.,'    ",
		`     /         \     `,
		`~ -- :         : -- ~`,
		`     \         /     `,
		"    ,'`._   _.'`.    ",
		"   '   / `!` \\   `   ",
		`      ;   :   ;      `,
	},
	{
		`      .   |   .      `,
		`   ;   \_,|,_/   ;   `,
		"    `.,'     `.,'    ",
		`     /         \     `,
		`~ -- |         | -- ~`,
		`     \         /     `,
		"    ,'`._   _.'`.    ",
		"   ;   / `|` \\   ;   ",
		`      .   |   .      `,
	},
}

// Four cloud shapes of varying size.
var cloudArts = [][]string{
	{
		`   .--.   `,
		` .-(    ). `,
		`(___.__)_)`,
	},
	{
		`      _  _   `,
		"    ( `   )_ ",
		"   (    )    `)",
		`    \_  (___  )`,
	},
	{
		`     .--.    `,
		`  .-(    ).  `,
		` (___.__)__) `,
	},
	{
		`   _  _   `,
		"  ( `   )_ ",
		"  (    )   `)",
		`  ` + "`" + `--'     `,
	},
}

// A full moon with craters (after Joan G. Stark).
var moonArt = []string{
	`   _..._   `,
	"  .'~o~~~`. ",
	`  :~~~~~o~~:`,
	`  :~o~~~~.~:`,
	"  `.~~~~~o.'",
	"    `-...-' ",
}

var treeArt = []string{
	`   ####   `,
	` ######## `,
	`##########`,
	` ######## `,
	`   _||_   `,
}

var pineArt = []string{
	`    *    `,
	`   ***   `,
	`  *****  `,
	` ******* `,
	`   |||   `,
}

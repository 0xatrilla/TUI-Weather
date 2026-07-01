package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	names := make([]string, len(allConditions))
	for i, c := range allConditions {
		names[i] = c.slug()
	}
	sim := flag.String("simulate", "", "start in simulation mode showing a condition: "+strings.Join(names, ", "))
	night := flag.Bool("night", false, "simulate night (moon, stars, fireflies)")
	scene := flag.String("scene", "", "scenery: "+strings.Join(worldNames, ", "))
	flag.Parse()

	m := initialModel()
	if *scene != "" {
		idx, ok := worldIndex(*scene)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown scene %q; try one of: %s\n", *scene, strings.Join(worldNames, ", "))
			os.Exit(2)
		}
		m.world = idx
	}
	if *sim != "" || *night {
		m.demoNight = *night
		if *sim != "" {
			idx, ok := demoIndexFor(*sim)
			if !ok {
				fmt.Fprintf(os.Stderr, "unknown condition %q; try one of: %s\n", *sim, strings.Join(names, ", "))
				os.Exit(2)
			}
			m.demoIdx = idx
		}
		m.applyDemo()
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

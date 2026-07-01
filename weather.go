package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var client = &http.Client{Timeout: 8 * time.Second}

// --- messages ---

type locationMsg struct {
	lat, lon float64
	place    string
}

type weatherMsg struct {
	code        int
	temp, feels float64
	wind        float64
	precip      float64
	humidity    int
	hi, lo      float64
	isDay       bool
	nowMin      int // local minutes since midnight
	sunriseMin  int
	sunsetMin   int
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func getJSON(u string, v any) error {
	resp, err := client.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %s", u, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

// fetchIPLocation auto-detects approximate location from the caller's IP.
func fetchIPLocation() tea.Msg {
	var r struct {
		Status     string  `json:"status"`
		City       string  `json:"city"`
		RegionName string  `json:"regionName"`
		Country    string  `json:"country"`
		Lat        float64 `json:"lat"`
		Lon        float64 `json:"lon"`
	}
	if err := getJSON("http://ip-api.com/json/", &r); err != nil {
		return errMsg{err}
	}
	if r.Status != "success" {
		return errMsg{fmt.Errorf("ip lookup failed")}
	}
	return locationMsg{lat: r.Lat, lon: r.Lon, place: placeName(r.City, r.RegionName, r.Country)}
}

// geocodeCity resolves a typed city name to coordinates via Open-Meteo geocoding.
func geocodeCity(name string) tea.Cmd {
	return func() tea.Msg {
		var r struct {
			Results []struct {
				Name      string  `json:"name"`
				Admin1    string  `json:"admin1"`
				Country   string  `json:"country"`
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			} `json:"results"`
		}
		u := "https://geocoding-api.open-meteo.com/v1/search?count=1&name=" + url.QueryEscape(name)
		if err := getJSON(u, &r); err != nil {
			return errMsg{err}
		}
		if len(r.Results) == 0 {
			return errMsg{fmt.Errorf("no match for %q", name)}
		}
		c := r.Results[0]
		return locationMsg{lat: c.Latitude, lon: c.Longitude, place: placeName(c.Name, c.Admin1, c.Country)}
	}
}

// fetchWeather pulls current conditions, today's hi/lo, and sun times for a coordinate.
func fetchWeather(lat, lon float64) tea.Cmd {
	return func() tea.Msg {
		var r struct {
			Current struct {
				Time     string  `json:"time"`
				Temp     float64 `json:"temperature_2m"`
				Feels    float64 `json:"apparent_temperature"`
				Code     int     `json:"weather_code"`
				Wind     float64 `json:"wind_speed_10m"`
				Humidity int     `json:"relative_humidity_2m"`
				Precip   float64 `json:"precipitation"`
				IsDay    int     `json:"is_day"`
			} `json:"current"`
			Daily struct {
				Max     []float64 `json:"temperature_2m_max"`
				Min     []float64 `json:"temperature_2m_min"`
				Sunrise []string  `json:"sunrise"`
				Sunset  []string  `json:"sunset"`
			} `json:"daily"`
		}
		u := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f"+
			"&current=temperature_2m,apparent_temperature,weather_code,wind_speed_10m,relative_humidity_2m,precipitation,is_day"+
			"&daily=temperature_2m_max,temperature_2m_min,sunrise,sunset&timezone=auto", lat, lon)
		if err := getJSON(u, &r); err != nil {
			return errMsg{err}
		}
		m := weatherMsg{
			code:     r.Current.Code,
			temp:     r.Current.Temp,
			feels:    r.Current.Feels,
			wind:     r.Current.Wind,
			precip:   r.Current.Precip,
			humidity: r.Current.Humidity,
			isDay:    r.Current.IsDay == 1,
			nowMin:   hm(r.Current.Time),
		}
		if len(r.Daily.Max) > 0 {
			m.hi = r.Daily.Max[0]
		}
		if len(r.Daily.Min) > 0 {
			m.lo = r.Daily.Min[0]
		}
		if len(r.Daily.Sunrise) > 0 {
			m.sunriseMin = hm(r.Daily.Sunrise[0])
		}
		if len(r.Daily.Sunset) > 0 {
			m.sunsetMin = hm(r.Daily.Sunset[0])
		}
		return m
	}
}

// hm parses an Open-Meteo local time ("2006-01-02T15:04") to minutes since midnight.
func hm(s string) int {
	t, err := time.Parse("2006-01-02T15:04", s)
	if err != nil {
		return -1
	}
	return t.Hour()*60 + t.Minute()
}

// placeName joins the non-empty location parts into "City, Region, Country".
func placeName(parts ...string) string {
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if out != "" {
			out += ", "
		}
		out += p
	}
	return out
}

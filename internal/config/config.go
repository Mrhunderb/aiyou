// Package config loads and validates user preferences independently of the
// audio backend and TUI. A missing default file uses built-in defaults.
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// Duration is a human-readable JSON duration such as "2s" or "500ms".
type Duration time.Duration

func (d Duration) Duration() time.Duration      { return time.Duration(d) }
func (d Duration) MarshalJSON() ([]byte, error) { return json.Marshal(time.Duration(d).String()) }
func (d *Duration) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("duration must be a string: %w", err)
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", value, err)
	}
	*d = Duration(parsed)
	return nil
}

// Config is the complete set of supported user-facing runtime preferences.
// Values are resolved and validated once at startup; no implicit file writes.
type Config struct {
	RefreshInterval Duration `json:"refresh_interval"`
	VolumeStep      int      `json:"volume_step"`
	Layout          Layout   `json:"layout"`
	Theme           Theme    `json:"theme"`
	Backend         Backend  `json:"backend"`
}
type Layout struct {
	DashboardMinWidth   int `json:"dashboard_min_width"`
	DashboardMinHeight  int `json:"dashboard_min_height"`
	VolumeBarWidth      int `json:"volume_bar_width"`
	CompactBarWidth     int `json:"compact_bar_width"`
	CompactBarThreshold int `json:"compact_bar_threshold"`
}
type Theme struct {
	Accent              string `json:"accent"`
	Dim                 string `json:"dim"`
	Active              string `json:"active"`
	Volume              string `json:"volume"`
	Muted               string `json:"muted"`
	SelectionForeground string `json:"selection_foreground"`
	SelectionBackground string `json:"selection_background"`
}
type Backend struct {
	PactlTimeout     Duration `json:"pactl_timeout"`
	BluetoothTimeout Duration `json:"bluetooth_timeout"`
}

func Default() Config {
	return Config{
		RefreshInterval: Duration(2 * time.Second), VolumeStep: 5,
		Layout: Layout{
			DashboardMinWidth: 70, DashboardMinHeight: 20,
			VolumeBarWidth: 10, CompactBarWidth: 5, CompactBarThreshold: 42,
		},
		Theme: Theme{
			Accent: "#81A1C1", Dim: "#6B7486", Active: "#88C0D0",
			Volume: "#A3BE8C", Muted: "#BF616A",
			SelectionForeground: "#2E3440", SelectionBackground: "#88C0D0",
		},
		Backend: Backend{PactlTimeout: Duration(5 * time.Second), BluetoothTimeout: Duration(time.Second)},
	}
}

// DefaultPath follows os.UserConfigDir: on Linux it is
// $XDG_CONFIG_HOME/ayou/config.json or ~/.config/ayou/config.json.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config directory: %w", err)
	}
	return filepath.Join(dir, "ayou", "config.json"), nil
}

// Load uses the default path when path is empty. A missing default file is
// optional; a missing explicit file, unreadable file or invalid file is an
// error. Unknown keys are rejected instead of silently misspelling settings.
func Load(path string) (Config, error) {
	cfg := Default()
	explicit := path != ""
	if !explicit {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return cfg, err
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !explicit && os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}
	if trimmed := bytes.TrimSpace(data); len(trimmed) == 0 || trimmed[0] != '{' {
		return cfg, fmt.Errorf("parse config %s: expected a JSON object", path)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("unexpected content after JSON object")
		}
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return cfg, nil
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (c Config) Validate() error {
	checkDuration := func(name string, d Duration, min, max time.Duration) error {
		if d.Duration() < min || d.Duration() > max {
			return fmt.Errorf("%s must be between %s and %s", name, min, max)
		}
		return nil
	}
	if err := checkDuration("refresh_interval", c.RefreshInterval, 100*time.Millisecond, time.Minute); err != nil {
		return err
	}
	if c.VolumeStep < 1 || c.VolumeStep > 100 {
		return fmt.Errorf("volume_step must be between 1 and 100")
	}
	l := c.Layout
	if l.DashboardMinWidth < 20 || l.DashboardMinWidth > 300 {
		return fmt.Errorf("layout.dashboard_min_width must be between 20 and 300")
	}
	if l.DashboardMinHeight < 16 || l.DashboardMinHeight > 100 {
		return fmt.Errorf("layout.dashboard_min_height must be between 16 and 100")
	}
	if l.VolumeBarWidth < 1 || l.VolumeBarWidth > 30 {
		return fmt.Errorf("layout.volume_bar_width must be between 1 and 30")
	}
	if l.CompactBarWidth < 1 || l.CompactBarWidth > l.VolumeBarWidth {
		return fmt.Errorf("layout.compact_bar_width must be between 1 and volume_bar_width")
	}
	if l.CompactBarThreshold < 20 || l.CompactBarThreshold > 300 {
		return fmt.Errorf("layout.compact_bar_threshold must be between 20 and 300")
	}
	for _, color := range []struct{ name, value string }{
		{"accent", c.Theme.Accent}, {"dim", c.Theme.Dim}, {"active", c.Theme.Active},
		{"volume", c.Theme.Volume}, {"muted", c.Theme.Muted},
		{"selection_foreground", c.Theme.SelectionForeground}, {"selection_background", c.Theme.SelectionBackground},
	} {
		if !hexColor.MatchString(color.value) {
			return fmt.Errorf("theme.%s must be a #RRGGBB color", color.name)
		}
	}
	if err := checkDuration("backend.pactl_timeout", c.Backend.PactlTimeout, 100*time.Millisecond, 30*time.Second); err != nil {
		return err
	}
	if err := checkDuration("backend.bluetooth_timeout", c.Backend.BluetoothTimeout, 100*time.Millisecond, 30*time.Second); err != nil {
		return err
	}
	return nil
}

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfigAndPath(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := DefaultPath()
	if err != nil || !strings.HasSuffix(path, filepath.Join("ayou", "config.json")) {
		t.Fatalf("path=%q err=%v", path, err)
	}
	got, err := Load("")
	if err != nil || got != Default() {
		t.Fatalf("missing default file: %+v, %v", got, err)
	}
}
func TestDefaultLocationAndExplicitOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	defaultPath, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(defaultPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(defaultPath, []byte(`{"volume_step":7}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := Load("")
	if err != nil || got.VolumeStep != 7 {
		t.Fatalf("default location: %+v, %v", got, err)
	}
	explicit := filepath.Join(t.TempDir(), "custom.json")
	if err := os.WriteFile(explicit, []byte(`{"volume_step":9}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err = Load(explicit)
	if err != nil || got.VolumeStep != 9 {
		t.Fatalf("explicit override: %+v, %v", got, err)
	}
	t.Setenv("XDG_CONFIG_HOME", "relative/config")
	if _, err := DefaultPath(); err == nil {
		t.Fatal("relative XDG_CONFIG_HOME should be rejected")
	}
}

func TestLoadPartialConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	data := `{"refresh_interval":"500ms", "volume_step":12, "theme":{"active":"#112233"}, "layout":{"dashboard_min_width":85}, "backend":{"pactl_timeout":"3s"}}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.RefreshInterval.Duration() != 500*time.Millisecond || got.VolumeStep != 12 || got.Theme.Active != "#112233" || got.Layout.DashboardMinWidth != 85 || got.Backend.PactlTimeout.Duration() != 3*time.Second {
		t.Fatalf("overrides: %+v", got)
	}
	if got.Theme.Dim != Default().Theme.Dim || got.Layout.CompactBarWidth != Default().Layout.CompactBarWidth {
		t.Fatal("partial nested values lost defaults")
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"refresh_interval":"500ms"`) {
		t.Fatalf("duration encoding: %s", encoded)
	}
}
func TestExampleMatchesDefaults(t *testing.T) {
	got, err := Load(filepath.Join("..", "..", "config.example.json"))
	if err != nil || got != Default() {
		t.Fatalf("example differs from defaults: %+v, %v", got, err)
	}
}

func TestInvalidConfig(t *testing.T) {
	for _, tc := range []struct{ name, content, want string }{
		{"unknown", `{"volum_step":5}`, "unknown field"},
		{"nested unknown", `{"theme":{"active_colour":"#112233"}}`, "unknown field"},
		{"invalid duration", `{"refresh_interval":"soon"}`, "duration"},
		{"numeric duration", `{"refresh_interval":2}`, "duration must be a string"},
		{"zero step", `{"volume_step":0}`, "volume_step"},
		{"invalid color", `{"theme":{"active":"rainbow"}}`, "theme.active"},
		{"bad layout", `{"layout":{"compact_bar_width":100}}`, "compact_bar_width"},
		{"too short dashboard", `{"layout":{"dashboard_min_height":9}}`, "dashboard_min_height"},
		{"bad timeout", `{"backend":{"pactl_timeout":"0s"}}`, "pactl_timeout"},
		{"trailing JSON", `{} {}`, "unexpected content"},
		{"null", `null`, "expected a JSON object"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v; want %q", err, tc.want)
			}
		})
	}
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("missing explicit file was silently ignored")
	}
}

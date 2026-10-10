# aiyou

A keyboard-driven terminal audio control center for PulseAudio and PipeWire (via `pipewire-pulse`). Inspired by [bottom](https://github.com/ClementTsang/bottom)'s focused widgets and dashboard layout, and by pavucontrol's audio controls. It uses `pactl` as its backend.

![aiyou dashboard (earlier four-widget layout)](docs/assets/aiyou-dashboard.png)

*Screenshot shows an earlier four-widget layout; the current dashboard also includes Configuration.*

## Requirements

- Linux with PulseAudio or PipeWire's PulseAudio compatibility server
- `pactl` available in `PATH`
- Optional: `bluetoothctl` for the Bluetooth alias when PulseAudio reports `(null)`
- Go 1.24+ to build from source

## Run

```sh
go run .
# or
go build -o aiyou . && ./aiyou
```

## Layout

On a wide terminal, all five bordered widgets share one dashboard. The left column contains **1 Output Devices** (speakers/headphones) above **2 Input Devices** (microphones/monitor sources). The right column contains **3 Playback** (application audio), **4 Recording** (applications capturing audio), and **5 Configuration** (card profiles), top to bottom. A focused widget has a cyan border and highlighted selection. On narrow terminals or in expanded mode, only the focused widget is shown. `Tab` and keys `1`–`5` follow this numbered order; directional focus uses the widgets' screen positions.

## Keys

| Keys | Action |
| --- | --- |
| `Tab` / `Shift-Tab`, `1`–`5` | Switch widgets |
| `Ctrl`/`Shift` + arrows, uppercase `H`/`J`/`K`/`L` or `W`/`A`/`S`/`D` | Move focus to the widget above/left/below/right (no wrap at edges) |
| `↑`/`↓` or `j`/`k` | Select an item |
| `Home`/`End`, `g`/`G`, `PgUp`/`PgDn` | Navigate a list |
| `←`/`→`, `h`/`l`, `-`/`+` | Adjust selected volume by 5% |
| `m` | Toggle mute for a selected stream or device |
| `d` | Set selected output/input device as default |
| `Enter` or `p` | Choose destination for a stream, port for a device, or profile for a card |
| `e` | Expand or collapse selected widget |
| `/`, `n`/`N` | Find an item by name, then jump to next/previous match |
| `Esc` | Close menu/search/help or collapse expanded view |
| `?` | Show help |
| `r` | Refresh now (also refreshes every two seconds) |
| `q` or `Ctrl-C` | Quit |

Selecting a card profile can change which inputs and outputs are available. Monitor sources are listed alongside microphones. Volume display averages channels; controls currently change all channels together. This is not a full replacement for pavucontrol: per-channel volume, balance, and advanced device settings are not supported.

## Project structure

- `main.go` — composition root: checks for `pactl`, wires the backend and starts Bubble Tea.
- `internal/audio/` — audio snapshot types, operations and backend interface; no CLI or TUI dependencies.
- `internal/pactl/` — adapter for `pactl` JSON, commands and optional Bluetooth names.
- `internal/ui/` — Bubble Tea state and key handling (`model.go`, `navigation.go`, `actions.go`, `update.go`), rendering (`view.go`) and styles (`theme.go`).

The UI depends on the `audio.Backend` interface, not directly on `pactl`, so tests use a fake backend without touching system audio. Go's `internal` packages keep implementation details private to this module.

## Tests

```sh
go test ./...
go vet ./...
```

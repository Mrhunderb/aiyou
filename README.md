# audiotui

A minimal Go TUI for controlling PulseAudio-compatible servers (including PipeWire via `pipewire-pulse`). It uses `pactl` as its backend.

## Requirements

- Linux with PulseAudio or PipeWire's PulseAudio compatibility server
- `pactl` available in `PATH`
- Optional: `bluetoothctl` to show the Bluetooth alias when `pactl` reports `(null)` for an output name
- Go 1.24+

## Run

```sh
go run .
```

Or build an executable with `go build -o audiotui .`.

## Keys

- `↑` / `↓` or `j` / `k`: select an output or application stream
- `Tab`: switch focus between output devices and application streams
- `d` / `Enter`: set selected output as default; on a stream, move it to the selected output
- `+` / `-`: adjust selected output volume by 5%
- `m`: toggle mute for selected output or stream
- `r`: refresh
- `q` / `Ctrl-C`: quit

The view refreshes every two seconds. This MVP intentionally omits input devices, profiles, ports, and PipeWire-native controls.

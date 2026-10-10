package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSinkDisplayName(t *testing.T) {
	lookup := func(address string) string {
		if address != "41:42:D1:97:77:91" {
			t.Fatalf("unexpected address: %q", address)
		}
		return "觅声-Polaris"
	}
	device := Sink{
		Name: "bluez_output.41_42_D1_97_77_91.1", Description: "(null)",
		Properties: map[string]string{"device.bus": "bluetooth", "device.alias": "(null)", "api.bluez5.address": "41:42:D1:97:77:91"},
	}
	if got := sinkDisplayName(device, lookup); got != "觅声-Polaris" {
		t.Fatalf("bluetooth fallback = %q", got)
	}
	device.Description = "Speakers"
	if got := sinkDisplayName(device, func(string) string { t.Fatal("unnecessary lookup"); return "" }); got != "Speakers" {
		t.Fatalf("valid description = %q", got)
	}
	device.Description = "(null)"
	if got := sinkDisplayName(device, func(string) string { return "" }); got != "Bluetooth 41:42:D1:97:77:91" {
		t.Fatalf("unavailable bluetooth lookup = %q", got)
	}
	device.Properties = map[string]string{}
	if got := sinkDisplayName(device, lookup); got != device.Name {
		t.Fatalf("missing properties = %q", got)
	}
}

func TestParseBluetoothName(t *testing.T) {
	if got := parseBluetoothName("\tName: 旧名\n\tAlias: 觅声-Polaris\n"); got != "觅声-Polaris" {
		t.Fatalf("alias = %q", got)
	}
	if got := parseBluetoothName("\tName: 中文耳机\n"); got != "中文耳机" {
		t.Fatalf("name = %q", got)
	}
	if got := parseBluetoothName("\tAlias: (null)\n"); got != "" {
		t.Fatalf("null alias = %q", got)
	}
}

func TestTruncateUnicode(t *testing.T) {
	got := truncate(strings.Repeat("中文", 25), 42)
	if !utf8.ValidString(got) || !strings.HasSuffix(got, "…") {
		t.Fatalf("invalid truncated display: %q", got)
	}
}

package audio

import "testing"

func TestAverageVolume(t *testing.T) {
	p, label, _ := AverageVolume(map[string]ChannelVolume{
		"left": {ValuePercent: "30%"}, "right": {ValuePercent: "50%"},
	})
	if p != 40 || label != "40%" {
		t.Fatalf("average = %d, %q", p, label)
	}
	if p, label, _ := AverageVolume(nil); p != 0 || label != "0%" {
		t.Fatalf("empty = %d, %q", p, label)
	}
}

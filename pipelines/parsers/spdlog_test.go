package parsers

import (
	"testing"
	"time"
)

const line = "[2026-11-16 12:00:00.250] [info] [probe] Msg log: 7, cluster: 0xabc, relay: 127.0.0.1:1"

func TestParseInLocation(t *testing.T) {
	want := time.Date(2026, 11, 16, 12, 0, 0, 250000000, time.UTC)

	for _, c := range []struct {
		loc    *time.Location
		offset time.Duration
	}{
		{time.UTC, 0},
		// machine at UTC+5:30 logs 12:00 when it is 06:30 UTC
		{time.FixedZone("ahead", 5*3600+1800), 5*time.Hour + 30*time.Minute},
		// machine at UTC-8 logs 12:00 when it is 20:00 UTC
		{time.FixedZone("behind", -8*3600), -8 * time.Hour},
	} {
		ts, level, location, msg, err := NewSpdlogParser(c.loc).Parse(line)
		if err != nil {
			t.Fatal(err)
		}
		if !ts.Equal(want.Add(-c.offset)) {
			t.Fatalf("%v: got %v, want %v", c.loc, ts.UTC(), want.Add(-c.offset))
		}
		if level != "info" || location != "probe" || msg != "Msg log: 7, cluster: 0xabc, relay: 127.0.0.1:1" {
			t.Fatalf("got %q %q %q", level, location, msg)
		}
	}
}

func TestParseLocalByDefault(t *testing.T) {
	ts, _, _, _, err := NewSpdlogParser(nil).Parse(line)
	if err != nil {
		t.Fatal(err)
	}
	if !ts.Equal(time.Date(2026, 11, 16, 12, 0, 0, 250000000, time.Local)) {
		t.Fatalf("got %v", ts)
	}
}

func TestParseError(t *testing.T) {
	if _, _, _, _, err := NewSpdlogParser(time.UTC).Parse("not a log line"); err == nil {
		t.Fatal("expected an error")
	}
	if _, _, _, _, err := NewSpdlogParser(time.UTC).Parse("[yesterday] [info] [probe] msg"); err == nil {
		t.Fatal("expected an error for a time that can't be read")
	}
}

package buildinfo

import "testing"

func TestCurrent(t *testing.T) {
	oldVersion := Version
	oldDate := Date
	oldCommit := Commit

	t.Cleanup(func() {
		Version = oldVersion
		Date = oldDate
		Commit = oldCommit
	})

	Version = "v1.2.3"
	Date = "2026-04-08"
	Commit = "abcdef1"

	got := Current()

	if got.Version != "v1.2.3" {
		t.Fatalf("unexpected version: got %q, want %q", got.Version, "v1.2.3")
	}
	if got.Date != "2026-04-08" {
		t.Fatalf("unexpected date: got %q, want %q", got.Date, "2026-04-08")
	}
	if got.Commit != "abcdef1" {
		t.Fatalf("unexpected commit: got %q, want %q", got.Commit, "abcdef1")
	}
}

func TestInfoString(t *testing.T) {
	info := Info{
		Version: "v1.2.3",
		Date:    "2026-04-08",
		Commit:  "abcdef1",
	}

	got := info.String()
	want := "version: v1.2.3\ndate: 2026-04-08\ncommit: abcdef1"

	if got != want {
		t.Fatalf("unexpected string:\n got: %q\nwant: %q", got, want)
	}
}
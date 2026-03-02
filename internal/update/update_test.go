package update

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v1.2.3", "v1.2.3", 0},
		{"v1.2.4", "v1.2.3", 1},
		{"v2.0.0", "v1.9.9", 1},
		{"v1.10.0", "v1.9.9", 1},
		{"v1.2.3", "v1.2.4", -1},
		{"v1.2.3", "v1.2.3-beta.1", 1},
		{"v1.2.3-beta.2", "v1.2.3-beta.1", 1},
	}

	for _, tc := range cases {
		got := compareVersions(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestClassifyChange(t *testing.T) {
	cases := []struct {
		current string
		latest  string
		want    string
	}{
		{"v1.2.3", "v2.0.0", "major"},
		{"v1.2.3", "v1.3.0", "minor"},
		{"v1.2.3", "v1.2.4", "patch"},
		{"v1.2.3-beta.1", "v1.2.3", "prerelease"},
	}

	for _, tc := range cases {
		got := classifyChange(tc.current, tc.latest)
		if got != tc.want {
			t.Errorf("classifyChange(%q, %q) = %q, want %q", tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestReleaseURL(t *testing.T) {
	got, err := releaseURL("v1.2.3")
	if err != nil {
		t.Fatalf("releaseURL error: %v", err)
	}
	if got == "" {
		t.Fatal("releaseURL returned empty URL")
	}
}

func TestApprovalPrompt(t *testing.T) {
	cases := []struct {
		change string
		want   string
	}{
		{"major", "Major update v2.0.0 is available. Upgrade now? [y/N]"},
		{"minor", "Minor update v2.0.0 is available. Upgrade now? [y/N]"},
		{"patch", "Patch update v2.0.0 is available. Upgrade now? [y/N]"},
		{"prerelease", "Update v2.0.0 is available. Upgrade now? [y/N]"},
	}

	for _, tc := range cases {
		got := approvalPrompt(tc.change, "v2.0.0")
		if got != tc.want {
			t.Errorf("approvalPrompt(%q) = %q, want %q", tc.change, got, tc.want)
		}
	}
}

func TestMaybeAutoCheckAndPrompt_SkipsWhenWithinTTL(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("WTW_NO_UPDATE_CHECK", "")

	origNow := now
	origInterval := checkInterval
	origClient := http.DefaultClient
	origInteractive := isInteractiveFn
	t.Cleanup(func() {
		now = origNow
		checkInterval = origInterval
		http.DefaultClient = origClient
		isInteractiveFn = origInteractive
	})

	fixedNow := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	now = func() time.Time { return fixedNow }
	checkInterval = 24 * time.Hour
	isInteractiveFn = func() bool { return false }

	path, err := stateFilePath()
	if err != nil {
		t.Fatal(err)
	}
	saveState(path, cacheState{
		LastCheckedAt:       fixedNow.Add(-1 * time.Hour),
		LatestSeenVersion:   "v1.0.1",
		LastNotifiedVersion: "v1.0.1",
	})

	http.DefaultClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("unexpected network call within TTL window")
			return nil, nil
		}),
	}

	MaybeAutoCheckAndPrompt("v1.0.0")

	st := loadState(path)
	if !st.LastCheckedAt.Equal(fixedNow.Add(-1 * time.Hour)) {
		t.Fatalf("LastCheckedAt changed: got %v", st.LastCheckedAt)
	}
	if st.LatestSeenVersion != "v1.0.1" || st.LastNotifiedVersion != "v1.0.1" {
		t.Fatalf("state changed unexpectedly: %+v", st)
	}
}

func TestMaybeAutoCheckAndPrompt_ChecksAfterTTL(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("WTW_NO_UPDATE_CHECK", "")

	origNow := now
	origInterval := checkInterval
	origClient := http.DefaultClient
	origInteractive := isInteractiveFn
	t.Cleanup(func() {
		now = origNow
		checkInterval = origInterval
		http.DefaultClient = origClient
		isInteractiveFn = origInteractive
	})

	fixedNow := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	now = func() time.Time { return fixedNow }
	checkInterval = 24 * time.Hour
	isInteractiveFn = func() bool { return false }

	path, err := stateFilePath()
	if err != nil {
		t.Fatal(err)
	}
	saveState(path, cacheState{
		LastCheckedAt: fixedNow.Add(-25 * time.Hour),
	})

	http.DefaultClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			body := `{"tag_name":"v1.2.0"}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	MaybeAutoCheckAndPrompt("v1.0.0")

	st := loadState(path)
	if !st.LastCheckedAt.Equal(fixedNow) {
		t.Fatalf("LastCheckedAt = %v, want %v", st.LastCheckedAt, fixedNow)
	}
	if st.LatestSeenVersion != "v1.2.0" {
		t.Fatalf("LatestSeenVersion = %q, want %q", st.LatestSeenVersion, "v1.2.0")
	}
	if st.LastNotifiedVersion != "v1.2.0" {
		t.Fatalf("LastNotifiedVersion = %q, want %q", st.LastNotifiedVersion, "v1.2.0")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

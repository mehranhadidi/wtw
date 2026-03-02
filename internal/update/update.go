package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"wtw/internal/ui"
)

const (
	repoOwner  = "mehranhadidi"
	repoName   = "wtw"
	binaryName = "wtw"
)

var (
	checkInterval      = 24 * time.Hour
	checkTimeout       = 1500 * time.Millisecond
	downloadTimeout    = 30 * time.Second
	now                = time.Now
	executablePathFunc = os.Executable
	isInteractiveFn    = isInteractive
)

type cacheState struct {
	LastCheckedAt       time.Time `json:"last_checked_at"`
	LatestSeenVersion   string    `json:"latest_seen_version"`
	LastNotifiedVersion string    `json:"last_notified_version"`
}

type release struct {
	TagName string `json:"tag_name"`
}

type version struct {
	major int
	minor int
	patch int
	pre   string
}

func MaybeAutoCheckAndPrompt(currentVersion string) {
	if skipUpdateCheck(currentVersion) {
		return
	}

	statePath, err := stateFilePath()
	if err != nil {
		return
	}

	state := loadState(statePath)
	if now().Sub(state.LastCheckedAt) < checkInterval {
		return
	}

	state.LastCheckedAt = now()
	latest, err := latestVersionWithTimeout(checkTimeout)
	if err != nil {
		saveState(statePath, state)
		return
	}

	state.LatestSeenVersion = latest
	if !isNewer(latest, currentVersion) {
		saveState(statePath, state)
		return
	}

	if state.LastNotifiedVersion == latest {
		saveState(statePath, state)
		return
	}

	change := classifyChange(currentVersion, latest)
	fmt.Printf("\nUpdate available: %s (%s update from %s)\n", latest, change, currentVersion)
	fmt.Printf("Run 'wtw update' anytime to install manually.\n")

	if isInteractiveFn() {
		prompt := approvalPrompt(change, latest)
		if ui.Confirm(prompt, "N") {
			if err := installVersion(latest); err != nil {
				ui.Error(fmt.Sprintf("auto-update failed: %v", err))
				fmt.Println("Install manually: curl -fsSL https://raw.githubusercontent.com/mehranhadidi/wtw/main/install.sh | bash")
			} else {
				ui.Success(fmt.Sprintf("updated to %s", latest))
			}
		}
	}

	state.LastNotifiedVersion = latest
	saveState(statePath, state)
}

func ManualUpdate(currentVersion string, autoApprove bool) error {
	if skipUpdateCheck(currentVersion) {
		return fmt.Errorf("cannot check updates for local dev build (%s)", currentVersion)
	}

	latest, err := latestVersionWithTimeout(5 * time.Second)
	if err != nil {
		return err
	}

	if !isNewer(latest, currentVersion) {
		fmt.Printf("wtw is up to date (%s).\n", currentVersion)
		return nil
	}

	change := classifyChange(currentVersion, latest)
	fmt.Printf("Update available: %s (%s update from %s)\n", latest, change, currentVersion)

	approve := autoApprove
	if !approve {
		if !isInteractiveFn() {
			fmt.Println("Re-run with --yes to install non-interactively.")
			return nil
		}
		approve = ui.Confirm(approvalPrompt(change, latest), "N")
	}

	if !approve {
		fmt.Println("Skipped update.")
		return nil
	}

	if err := installVersion(latest); err != nil {
		return err
	}

	fmt.Printf("Updated wtw to %s.\n", latest)
	return nil
}

func latestVersionWithTimeout(timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return fetchLatestVersion(ctx)
}

func fetchLatestVersion(ctx context.Context) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "wtw-update-check")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("update check failed: %s", resp.Status)
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", errors.New("latest release tag is empty")
	}
	return normalizeTag(rel.TagName), nil
}

func installVersion(ver string) error {
	exePath, err := executablePathFunc()
	if err != nil {
		return err
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		// Fallback to raw executable path when symlink resolution fails.
		exePath, err = executablePathFunc()
		if err != nil {
			return err
		}
	}

	url, err := releaseURL(ver)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "wtw-self-update")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	bin, err := extractBinary(resp.Body)
	if err != nil {
		return err
	}

	dir := filepath.Dir(exePath)
	tmp, err := os.CreateTemp(dir, "wtw-update-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(bin); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		return fmt.Errorf("install failed (try running with elevated permissions): %w", err)
	}
	return nil
}

func releaseURL(ver string) (string, error) {
	osName := runtime.GOOS
	switch osName {
	case "darwin", "linux":
	default:
		return "", fmt.Errorf("unsupported OS: %s", osName)
	}

	arch := runtime.GOARCH
	switch arch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("unsupported architecture: %s", arch)
	}

	return fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/%s/%s_%s_%s.tar.gz",
		repoOwner,
		repoName,
		normalizeTag(ver),
		binaryName,
		osName,
		arch,
	), nil
}

func extractBinary(r io.Reader) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if filepath.Base(hdr.Name) == binaryName {
			return io.ReadAll(tr)
		}
	}

	return nil, errors.New("binary not found in release archive")
}

func skipUpdateCheck(currentVersion string) bool {
	if os.Getenv("WTW_NO_UPDATE_CHECK") == "1" {
		return true
	}
	if currentVersion == "" || currentVersion == "dev" {
		return true
	}
	_, err := parseVersion(currentVersion)
	return err != nil
}

func isNewer(latest, current string) bool {
	return compareVersions(latest, current) > 0
}

func classifyChange(current, latest string) string {
	cur, errCur := parseVersion(current)
	lat, errLat := parseVersion(latest)
	if errCur != nil || errLat != nil {
		return "version"
	}
	if lat.major > cur.major {
		return "major"
	}
	if lat.minor > cur.minor {
		return "minor"
	}
	if lat.patch > cur.patch {
		return "patch"
	}
	if lat.pre != cur.pre {
		return "prerelease"
	}
	return "version"
}

func compareVersions(a, b string) int {
	av, errA := parseVersion(a)
	bv, errB := parseVersion(b)
	if errA != nil || errB != nil {
		return strings.Compare(normalizeTag(a), normalizeTag(b))
	}

	if av.major != bv.major {
		if av.major > bv.major {
			return 1
		}
		return -1
	}
	if av.minor != bv.minor {
		if av.minor > bv.minor {
			return 1
		}
		return -1
	}
	if av.patch != bv.patch {
		if av.patch > bv.patch {
			return 1
		}
		return -1
	}

	if av.pre == bv.pre {
		return 0
	}
	if av.pre == "" {
		return 1
	}
	if bv.pre == "" {
		return -1
	}
	return strings.Compare(av.pre, bv.pre)
}

func parseVersion(raw string) (version, error) {
	raw = normalizeTag(raw)
	if raw == "" {
		return version{}, errors.New("empty version")
	}
	parts := strings.SplitN(raw, "-", 2)
	core := parts[0]
	if len(core) < 2 || core[0] != 'v' {
		return version{}, fmt.Errorf("invalid version: %s", raw)
	}
	pre := ""
	if len(parts) == 2 {
		pre = parts[1]
	}

	nums := strings.Split(core[1:], ".")
	if len(nums) < 2 || len(nums) > 3 {
		return version{}, fmt.Errorf("invalid version: %s", raw)
	}

	major, err := strconv.Atoi(nums[0])
	if err != nil {
		return version{}, err
	}
	minor, err := strconv.Atoi(nums[1])
	if err != nil {
		return version{}, err
	}
	patch := 0
	if len(nums) == 3 {
		patch, err = strconv.Atoi(nums[2])
		if err != nil {
			return version{}, err
		}
	}

	return version{major: major, minor: minor, patch: patch, pre: pre}, nil
}

func normalizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	tag = strings.TrimPrefix(tag, "v")
	if tag == "" {
		return ""
	}
	return "v" + tag
}

func approvalPrompt(changeType, latest string) string {
	prefix := "Update"
	switch changeType {
	case "major", "minor", "patch":
		prefix = strings.ToUpper(changeType[:1]) + changeType[1:] + " update"
	}
	return fmt.Sprintf("%s %s is available. Upgrade now? [y/N]", prefix, latest)
}

func stateFilePath() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "wtw")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "update.json"), nil
}

func loadState(path string) cacheState {
	data, err := os.ReadFile(path)
	if err != nil {
		return cacheState{}
	}
	var st cacheState
	if err := json.Unmarshal(data, &st); err != nil {
		return cacheState{}
	}
	return st
}

func saveState(path string, st cacheState) {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

func isInteractive() bool {
	st, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (st.Mode() & os.ModeCharDevice) != 0
}

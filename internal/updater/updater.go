package updater

import (
	"archive/tar"
	"bytes"
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
)

const (
	repo         = "imadys/diffmate"
	installHelp  = "run: curl -fsSL https://diffmate.imadys.dev/install.sh | sh"
	binaryName   = "diffmate"
	requestAgent = "diffmate"
)

var httpClient = &http.Client{Timeout: 45 * time.Second}

type Result struct {
	Current string
	Latest  string
	Updated bool
	Message string
}

func CheckAndInstall(ctx context.Context, current string) (Result, error) {
	latest, err := Latest(ctx)
	if err != nil {
		return Result{Current: current}, err
	}
	result := Result{Current: current, Latest: latest}
	if !IsNewer(current, latest) {
		result.Message = "diffmate is up to date (" + latest + ")"
		return result, nil
	}
	if err := installRelease(ctx, latest); err != nil {
		return result, fmt.Errorf("%w; %s", err, installHelp)
	}
	result.Updated = true
	result.Message = "updated diffmate to " + latest
	return result, nil
}

func Latest(ctx context.Context) (string, error) {
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := getJSON(ctx, "https://api.github.com/repos/"+repo+"/releases/latest", &payload); err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.TagName) == "" {
		return "", errors.New("latest release did not include a tag")
	}
	return payload.TagName, nil
}

func IsNewer(current, latest string) bool {
	currentVersion, currentOK := parseVersion(current)
	latestVersion, latestOK := parseVersion(latest)
	if !latestOK {
		return false
	}
	if !currentOK {
		return true
	}
	for i := range latestVersion {
		if latestVersion[i] != currentVersion[i] {
			return latestVersion[i] > currentVersion[i]
		}
	}
	return false
}

func installRelease(ctx context.Context, tag string) error {
	asset, err := assetName()
	if err != nil {
		return err
	}
	url := "https://github.com/" + repo + "/releases/download/" + tag + "/" + asset
	archive, err := download(ctx, url)
	if err != nil {
		return err
	}
	binary, err := extractBinary(archive)
	if err != nil {
		return err
	}
	return replaceExecutable(binary)
}

func assetName() (string, error) {
	osName := runtime.GOOS
	if osName != "darwin" && osName != "linux" {
		return "", fmt.Errorf("unsupported OS: %s", osName)
	}
	arch := runtime.GOARCH
	if arch != "amd64" && arch != "arm64" {
		return "", fmt.Errorf("unsupported architecture: %s", arch)
	}
	return fmt.Sprintf("diffmate-%s-%s.tar.gz", osName, arch), nil
}

func download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", requestAgent)
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %s", res.Status)
	}
	return io.ReadAll(res.Body)
}

func getJSON(ctx context.Context, url string, target any) error {
	body, err := download(ctx, url)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func extractBinary(archive []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(header.Name) != binaryName || header.Typeflag != tar.TypeReg {
			continue
		}
		return io.ReadAll(tr)
	}
	return nil, errors.New("release archive did not contain diffmate")
}

func replaceExecutable(binary []byte) error {
	current, err := os.Executable()
	if err != nil {
		return err
	}
	current, err = filepath.EvalSymlinks(current)
	if err != nil {
		return err
	}
	dir := filepath.Dir(current)
	temp, err := os.CreateTemp(dir, ".diffmate-update-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()
	if _, err := temp.Write(binary); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Chmod(0o755); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, current); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func parseVersion(value string) ([3]int, bool) {
	var version [3]int
	value = strings.TrimSpace(strings.TrimPrefix(value, "v"))
	if value == "" || value == "dev" {
		return version, false
	}
	if index := strings.IndexAny(value, "-+"); index >= 0 {
		value = value[:index]
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return version, false
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return version, false
		}
		version[i] = n
	}
	return version, true
}

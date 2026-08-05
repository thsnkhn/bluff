// Package updater installs verified Bluff release archives and relaunches the client.
package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/thsnkhn/bluff/internal/api"
)

const maxArchiveBytes = 64 << 20
const maxChecksumsBytes = 1 << 20

const releaseBaseURL = "https://github.com/thsnkhn/bluff/releases/download/"

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Installer struct {
	http HTTPClient
}

func New() *Installer {
	return &Installer{http: &http.Client{Timeout: 30 * time.Second}}
}

func (installer *Installer) Install(ctx context.Context, release api.ClientRelease) error {
	if release.Version == "" {
		return errors.New("release version is missing")
	}
	if release.AssetName == "" || release.DownloadURL == "" || release.SHA256 == "" {
		resolved, err := installer.resolveRelease(ctx, release.Version)
		if err != nil {
			return err
		}
		release = resolved
	}
	if err := validateRelease(release); err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find Bluff executable: %w", err)
	}
	resolvedExecutable, err := filepath.EvalSymlinks(executable)
	if err == nil {
		executable = resolvedExecutable
	}
	if isHomebrewManaged(executable) {
		upgraded, err := upgradeHomebrew(ctx)
		if err != nil {
			return err
		}
		if !upgraded {
			return errors.New("Homebrew did not install a newer Bluff release")
		}
		return restartCurrentProcess()
	}
	if runtime.GOOS == "windows" {
		// TODO: add a small Windows replacement helper; a running .exe cannot be renamed safely.
		return errors.New("automatic updates are not supported for Windows binaries yet")
	}

	temporaryDirectory, err := os.MkdirTemp("", "bluff-update-")
	if err != nil {
		return fmt.Errorf("create update workspace: %w", err)
	}
	defer func() { _ = os.RemoveAll(temporaryDirectory) }()
	archivePath := filepath.Join(temporaryDirectory, release.AssetName)
	if err := download(ctx, installer.http, release.DownloadURL, archivePath); err != nil {
		return err
	}
	if err := verifySHA256(archivePath, release.SHA256); err != nil {
		return err
	}
	binaryPath, err := extractBinary(archivePath, release.AssetName, temporaryDirectory)
	if err != nil {
		return err
	}
	if err := replaceExecutable(executable, binaryPath); err != nil {
		return err
	}
	return restartCurrentProcess()
}

func (installer *Installer) resolveRelease(ctx context.Context, version string) (api.ClientRelease, error) {
	version = strings.TrimSpace(version)
	if !validReleaseVersion(version) {
		return api.ClientRelease{}, errors.New("release version is invalid")
	}
	extension := ".tar.gz"
	if runtime.GOOS == "windows" {
		extension = ".zip"
	}
	assetName := fmt.Sprintf("bluff_%s_%s_%s%s", version, runtime.GOOS, runtime.GOARCH, extension)
	releaseURL := releaseBaseURL + version
	checksum, err := installer.releaseChecksum(ctx, releaseURL+"/checksums.txt", assetName)
	if err != nil {
		return api.ClientRelease{}, err
	}
	return api.ClientRelease{
		Version:     version,
		ReleaseURL:  releaseURL,
		AssetName:   assetName,
		DownloadURL: releaseURL + "/" + assetName,
		SHA256:      checksum,
	}, nil
}

func (installer *Installer) releaseChecksum(ctx context.Context, url, assetName string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create release metadata request: %w", err)
	}
	response, err := installer.http.Do(request)
	if err != nil {
		return "", fmt.Errorf("download release metadata: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("download release metadata: %s", response.Status)
	}
	contents, err := io.ReadAll(io.LimitReader(response.Body, maxChecksumsBytes+1))
	if err != nil {
		return "", fmt.Errorf("read release metadata: %w", err)
	}
	if len(contents) > maxChecksumsBytes {
		return "", errors.New("release metadata is too large")
	}
	for _, line := range strings.Split(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		filename := filepath.Base(strings.TrimPrefix(fields[len(fields)-1], "*"))
		if filename != assetName {
			continue
		}
		checksum := strings.ToLower(fields[0])
		if len(checksum) != sha256.Size*2 {
			return "", errors.New("release checksum is invalid")
		}
		if _, err := hex.DecodeString(checksum); err != nil {
			return "", errors.New("release checksum is invalid")
		}
		return checksum, nil
	}
	return "", fmt.Errorf("release metadata does not contain %s", assetName)
}

func validReleaseVersion(version string) bool {
	if !strings.HasPrefix(version, "v") || len(version) < 6 {
		return false
	}
	for _, character := range version[1:] {
		if (character >= '0' && character <= '9') || character == '.' {
			continue
		}
		return false
	}
	return true
}

func validateRelease(release api.ClientRelease) error {
	if release.Version == "" || release.AssetName == "" || release.DownloadURL == "" || len(release.SHA256) != sha256.Size*2 {
		return errors.New("release metadata is incomplete")
	}
	if !strings.HasPrefix(release.DownloadURL, "https://") {
		return errors.New("release download must use HTTPS")
	}
	if _, err := hex.DecodeString(release.SHA256); err != nil {
		return errors.New("release checksum is invalid")
	}
	return nil
}

func download(ctx context.Context, client HTTPClient, url, destination string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create update download: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("download update: %s", response.Status)
	}
	file, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create update archive: %w", err)
	}
	defer func() { _ = file.Close() }()
	written, err := io.Copy(file, io.LimitReader(response.Body, maxArchiveBytes+1))
	if err != nil {
		return fmt.Errorf("save update archive: %w", err)
	}
	if written > maxArchiveBytes {
		return errors.New("update archive is too large")
	}
	return nil
}

func verifySHA256(path, expected string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open update archive: %w", err)
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("hash update archive: %w", err)
	}
	actual := hash.Sum(nil)
	want, _ := hex.DecodeString(expected)
	if subtle.ConstantTimeCompare(actual, want) != 1 {
		return errors.New("update checksum does not match")
	}
	return nil
}

func extractBinary(archivePath, assetName, destination string) (string, error) {
	binaryName := "bluff"
	if strings.HasSuffix(assetName, ".zip") {
		binaryName = "bluff.exe"
	}
	binaryPath := filepath.Join(destination, binaryName)
	if strings.HasSuffix(assetName, ".zip") {
		return binaryPath, extractZip(archivePath, binaryName, binaryPath)
	}
	return binaryPath, extractTarGz(archivePath, binaryName, binaryPath)
}

func extractTarGz(archivePath, binaryName, destination string) error {
	archive, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open update archive: %w", err)
	}
	defer func() { _ = archive.Close() }()
	compressed, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("read update archive: %w", err)
	}
	defer func() { _ = compressed.Close() }()
	reader := tar.NewReader(compressed)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read update files: %w", err)
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != binaryName {
			continue
		}
		return writeBinary(destination, reader)
	}
	return fmt.Errorf("update archive does not contain %s", binaryName)
}

func extractZip(archivePath, binaryName, destination string) error {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("read update archive: %w", err)
	}
	defer func() { _ = archive.Close() }()
	for _, file := range archive.File {
		if filepath.Base(file.Name) != binaryName || file.FileInfo().IsDir() {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return fmt.Errorf("read update binary: %w", err)
		}
		err = writeBinary(destination, reader)
		_ = reader.Close()
		return err
	}
	return fmt.Errorf("update archive does not contain %s", binaryName)
}

func writeBinary(destination string, reader io.Reader) error {
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("create updated Bluff binary: %w", err)
	}
	defer func() { _ = file.Close() }()
	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("extract updated Bluff binary: %w", err)
	}
	return file.Chmod(0o755)
}

func replaceExecutable(executable, binaryPath string) error {
	directory := filepath.Dir(executable)
	temporary, err := os.CreateTemp(directory, ".bluff-update-*")
	if err != nil {
		return fmt.Errorf("prepare executable replacement: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	source, err := os.Open(binaryPath)
	if err != nil {
		_ = temporary.Close()
		return fmt.Errorf("open updated Bluff binary: %w", err)
	}
	_, copyErr := io.Copy(temporary, source)
	_ = source.Close()
	if copyErr == nil {
		copyErr = temporary.Chmod(0o755)
	}
	closeErr := temporary.Close()
	if copyErr != nil {
		return fmt.Errorf("prepare executable replacement: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close executable replacement: %w", closeErr)
	}
	if err := os.Rename(temporaryPath, executable); err != nil {
		return fmt.Errorf("install updated Bluff binary: %w", err)
	}
	return nil
}

func isHomebrewManaged(executable string) bool {
	path := filepath.ToSlash(executable)
	return strings.Contains(path, "/Cellar/bluff/") || strings.Contains(path, "/homebrew/Cellar/bluff/")
}

func upgradeHomebrew(ctx context.Context) (bool, error) {
	before, err := installedHomebrewVersion(ctx)
	if err != nil {
		return false, err
	}
	command := exec.CommandContext(ctx, "brew", "upgrade", "bluff")
	command.Stdin = nil
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		return false, fmt.Errorf("upgrade Bluff with Homebrew: %w", err)
	}
	after, err := installedHomebrewVersion(ctx)
	if err != nil {
		return false, err
	}
	return before != "" && after != "" && before != after, nil
}

func installedHomebrewVersion(ctx context.Context) (string, error) {
	output, err := exec.CommandContext(ctx, "brew", "list", "--versions", "bluff").Output()
	if err != nil {
		return "", fmt.Errorf("read installed Bluff version from Homebrew: %w", err)
	}
	fields := strings.Fields(string(output))
	if len(fields) < 2 {
		return "", nil
	}
	return fields[len(fields)-1], nil
}

func restartCurrentProcess() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find updated Bluff executable: %w", err)
	}
	if isHomebrewManaged(executable) {
		if path, err := exec.LookPath("bluff"); err == nil {
			executable = path
		}
	}
	command := exec.Command(executable, os.Args[1:]...)
	command.Env = os.Environ()
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		return fmt.Errorf("restart Bluff: %w", err)
	}
	return nil
}

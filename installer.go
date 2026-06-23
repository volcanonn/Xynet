package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func (a *App) InstallDae() error {
	a.EmitLog("installer", "Fetching latest dae release from GitHub...")
	
	resp, err := http.Get("https://api.github.com/repos/daeuniverse/dae/releases/latest")
	if err != nil {
		return fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to parse release JSON: %w", err)
	}

	// Assuming linux-x86_64 structure for now
	targetAsset := "dae-linux-x86_64.zip"
	if runtime.GOARCH == "arm64" {
		targetAsset = "dae-linux-arm64.zip"
	}

	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == targetAsset {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("could not find asset %s in release %s", targetAsset, release.TagName)
	}

	a.EmitLog("installer", fmt.Sprintf("Downloading %s...", downloadURL))

	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return err
	}
	// Some assets are blocked without a User-Agent
	req.Header.Set("User-Agent", "Xynet-Installer")

	dlResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != 200 {
		return fmt.Errorf("download failed with status: %d", dlResp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(dlResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read downloaded bytes: %w", err)
	}

	a.EmitLog("installer", "Extracting dae...")

	zipReader, err := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	binDir := filepath.Join(configDir, "xynet", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	expectedBinName := strings.TrimSuffix(targetAsset, ".zip")

	var daeContent []byte
	for _, file := range zipReader.File {
		baseName := filepath.Base(file.Name)
		if baseName == expectedBinName && !file.FileInfo().IsDir() {
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("failed to open dae inside zip: %w", err)
			}
			daeContent, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return fmt.Errorf("failed to read dae inside zip: %w", err)
			}
			break
		}
	}

	if len(daeContent) == 0 {
		return fmt.Errorf("could not find dae binary inside zip")
	}

	daePath := filepath.Join(binDir, "dae")
	if err := os.WriteFile(daePath, daeContent, 0755); err != nil {
		return fmt.Errorf("failed to write dae binary: %w", err)
	}

	a.EmitLog("installer", fmt.Sprintf("Successfully installed dae %s to %s", release.TagName, daePath))
	return nil
}

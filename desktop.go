package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type DesktopApp struct {
	Name        string `json:"name"`
	Exec        string `json:"exec"`
	Icon        string `json:"icon"`
	ProcessName string `json:"processName"`
}

func (a *App) ListDesktopApps() ([]DesktopApp, error) {
	dirs := []string{"/usr/share/applications"}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local", "share", "applications"))
	}

	seen := make(map[string]bool)
	var apps []DesktopApp

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".desktop") {
				continue
			}
			app, err := parseDesktopFile(filepath.Join(dir, entry.Name()))
			if err != nil || app == nil {
				continue
			}
			if seen[app.Name] {
				continue
			}
			seen[app.Name] = true
			apps = append(apps, *app)
		}
	}

	sort.Slice(apps, func(i, j int) bool {
		return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name)
	})

	return apps, nil
}

var fieldCodeRegex = regexp.MustCompile(`\s*%[a-zA-Z]`)

func parseDesktopFile(path string) (*DesktopApp, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var name, execLine, icon, appType string
	noDisplay := false
	hidden := false
	inDesktopEntry := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "[Desktop Entry]" {
			inDesktopEntry = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			if inDesktopEntry {
				break
			}
			continue
		}
		if !inDesktopEntry {
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		switch key {
		case "Name":
			if name == "" {
				name = val
			}
		case "Exec":
			execLine = val
		case "Icon":
			icon = val
		case "Type":
			appType = val
		case "NoDisplay":
			noDisplay = strings.EqualFold(val, "true")
		case "Hidden":
			hidden = strings.EqualFold(val, "true")
		}
	}

	if appType != "Application" || noDisplay || hidden || execLine == "" || name == "" {
		return nil, nil
	}

	processName := extractProcessName(execLine)
	if processName == "" {
		return nil, nil
	}

	iconDataURI := resolveIcon(icon)

	return &DesktopApp{
		Name:        name,
		Exec:        execLine,
		Icon:        iconDataURI,
		ProcessName: processName,
	}, nil
}

func extractProcessName(execLine string) string {
	cleaned := fieldCodeRegex.ReplaceAllString(execLine, "")
	cleaned = strings.TrimSpace(cleaned)

	parts := strings.Fields(cleaned)
	if len(parts) == 0 {
		return ""
	}

	bin := parts[0]
	if strings.Contains(bin, "=") {
		for _, p := range parts[1:] {
			if !strings.Contains(p, "=") {
				bin = p
				break
			}
		}
	}

	return filepath.Base(bin)
}

func resolveIcon(icon string) string {
	if icon == "" {
		return ""
	}

	if filepath.IsAbs(icon) {
		return readIconAsDataURI(icon)
	}

	searchPaths := []string{
		fmt.Sprintf("/usr/share/icons/hicolor/48x48/apps/%s.png", icon),
		fmt.Sprintf("/usr/share/icons/hicolor/64x64/apps/%s.png", icon),
		fmt.Sprintf("/usr/share/icons/hicolor/128x128/apps/%s.png", icon),
		fmt.Sprintf("/usr/share/icons/hicolor/scalable/apps/%s.svg", icon),
		fmt.Sprintf("/usr/share/pixmaps/%s.png", icon),
		fmt.Sprintf("/usr/share/pixmaps/%s.svg", icon),
		fmt.Sprintf("/usr/share/pixmaps/%s.xpm", icon),
	}

	for _, p := range searchPaths {
		if uri := readIconAsDataURI(p); uri != "" {
			return uri
		}
	}

	return ""
}

func readIconAsDataURI(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var mime string
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		mime = "image/png"
	case ".svg":
		mime = "image/svg+xml"
	case ".xpm":
		return ""
	default:
		return ""
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mime, encoded)
}

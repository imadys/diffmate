package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	Editor      string          `json:"editor"`
	Agent       string          `json:"agent"`
	Tabs        map[string]bool `json:"tabs"`
	SeenWelcome bool            `json:"seen_welcome"`
}

func Defaults() Settings {
	return Settings{
		Editor: "nvim",
		Agent:  "codex",
		Tabs: map[string]bool{
			"changes":  true,
			"branches": true,
			"commits":  true,
			"stash":    true,
		},
	}
}

func Load() (Settings, error) {
	path, err := Path()
	if err != nil {
		return Defaults(), err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Defaults(), nil
		}
		return Defaults(), err
	}

	settings := Defaults()
	if err := json.Unmarshal(data, &settings); err != nil {
		return Defaults(), err
	}
	settings.normalize()
	return settings, nil
}

func Save(settings Settings) error {
	settings.normalize()

	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "diffmate", "config.json"), nil
}

func (s *Settings) normalize() {
	defaults := Defaults()
	if s.Editor == "" {
		s.Editor = defaults.Editor
	}
	if s.Agent == "" {
		s.Agent = defaults.Agent
	}
	if s.Tabs == nil {
		s.Tabs = defaults.Tabs
	}
	for name, enabled := range defaults.Tabs {
		if _, ok := s.Tabs[name]; !ok {
			s.Tabs[name] = enabled
		}
	}

	any := false
	for _, enabled := range s.Tabs {
		if enabled {
			any = true
			break
		}
	}
	if !any {
		s.Tabs["changes"] = true
	}
}

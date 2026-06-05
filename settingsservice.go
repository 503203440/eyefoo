package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Settings struct {
	WorkInterval  int  `json:"work_interval"`
	BreakDuration int  `json:"break_duration"`
	StrictMode    bool `json:"strict_mode"`
	SoundEnabled  bool `json:"sound_enabled"`
	WarmLevel     int  `json:"warm_level"`
}

var defaultSettings = Settings{
	WorkInterval:  45,
	BreakDuration: 5,
	StrictMode:    false,
	SoundEnabled:  true,
	WarmLevel:     0,
}

type SettingsStore struct {
	mu       sync.RWMutex
	path     string
	settings Settings
}

func NewSettingsStore(configDir string) (*SettingsStore, error) {
	s := &SettingsStore{
		path:     filepath.Join(configDir, "settings.json"),
		settings: defaultSettings,
	}

		data, err := os.ReadFile(s.path)
	if err == nil {
		var loaded Settings
		if json.Unmarshal(data, &loaded) == nil {
			if loaded.WorkInterval >= 1 && loaded.WorkInterval <= 180 {
				s.settings.WorkInterval = loaded.WorkInterval
			}
			if loaded.BreakDuration >= 1 && loaded.BreakDuration <= 30 {
				s.settings.BreakDuration = loaded.BreakDuration
			}
			s.settings.StrictMode = loaded.StrictMode
			s.settings.SoundEnabled = loaded.SoundEnabled
			if loaded.WarmLevel >= 0 && loaded.WarmLevel <= 100 {
				s.settings.WarmLevel = loaded.WarmLevel
			}
		}
	}

	return s, s.save()
}

func (s *SettingsStore) GetSettings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

func (s *SettingsStore) SaveSettings(st Settings) error {
	if st.WorkInterval < 1 {
		st.WorkInterval = 1
	}
	if st.WorkInterval > 180 {
		st.WorkInterval = 180
	}
	if st.BreakDuration < 1 {
		st.BreakDuration = 1
	}
	if st.BreakDuration > 30 {
		st.BreakDuration = 30
	}
	if st.WarmLevel < 0 {
		st.WarmLevel = 0
	}
	if st.WarmLevel > 100 {
		st.WarmLevel = 100
	}

	s.mu.Lock()
	s.settings = st
	s.mu.Unlock()

	err := s.save()
	if err != nil {
		return err
	}

	ApplyColorTemperature(st.WarmLevel)
	return nil
}

func (s *SettingsStore) save() error {
	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *SettingsStore) Get() Settings {
	return s.GetSettings()
}

func (s *SettingsStore) Set(st Settings) error {
	return s.SaveSettings(st)
}

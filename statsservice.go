package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type DailyStats struct {
	Date      string `json:"date"`
	WorkSecs  int    `json:"work_secs"`
	BreakSecs int    `json:"break_secs"`
	Breaks    int    `json:"breaks"`
}

type StatsStore struct {
	mu    sync.Mutex
	path  string
	daily map[string]*DailyStats
}

func NewStatsStore(configDir string) (*StatsStore, error) {
	s := &StatsStore{
		path:  filepath.Join(configDir, "stats.json"),
		daily: make(map[string]*DailyStats),
	}

	data, err := os.ReadFile(s.path)
	if err == nil {
		var records []DailyStats
		if json.Unmarshal(data, &records) == nil {
			for i := range records {
				s.daily[records[i].Date] = &records[i]
			}
		}
	}

	return s, nil
}

func (s *StatsStore) AddWork(secs int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	ds, ok := s.daily[today]
	if !ok {
		ds = &DailyStats{Date: today}
		s.daily[today] = ds
	}
	ds.WorkSecs += secs
	go s.save()
	return nil
}

func (s *StatsStore) AddBreak(count int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	ds, ok := s.daily[today]
	if !ok {
		ds = &DailyStats{Date: today}
		s.daily[today] = ds
	}
	ds.Breaks += count
	go s.save()
	return nil
}

func (s *StatsStore) GetToday() *DailyStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	ds := s.daily[today]
	if ds == nil {
		return &DailyStats{Date: today}
	}
	cp := *ds
	return &cp
}

func (s *StatsStore) GetRecent(days int) []DailyStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []DailyStats
	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		if ds, ok := s.daily[date]; ok {
			result = append(result, *ds)
		} else {
			result = append(result, DailyStats{Date: date})
		}
	}
	return result
}

func (s *StatsStore) save() {
	s.mu.Lock()
	var records []DailyStats
	for _, ds := range s.daily {
		records = append(records, *ds)
	}
	s.mu.Unlock()

	data, _ := json.MarshalIndent(records, "", "  ")
	os.WriteFile(s.path, data, 0644)
}

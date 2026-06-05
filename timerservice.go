package main

import (
	"context"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type TimerPhase int

const (
	TimerIdle    TimerPhase = 0
	TimerWorking TimerPhase = 1
	TimerBreak   TimerPhase = 2
)

type TimerState struct {
	Phase     TimerPhase `json:"phase"`
	Elapsed   int        `json:"elapsed"`
	Total     int        `json:"total"`
	TotalWork int        `json:"total_work"`
	Paused    bool       `json:"paused"`
}

type TimerService struct {
	settings *SettingsStore
	stats    *StatsStore
	window   application.Window
	app      *application.App

	mu    sync.Mutex
	phase TimerPhase

	elapsed   int
	paused    bool
	workTotal int
	workGoal  int
	breakGoal int

	cancel context.CancelFunc
}

func NewTimerService(settings *SettingsStore, stats *StatsStore) *TimerService {
	t := &TimerService{
		phase:    TimerIdle,
		settings: settings,
		stats:    stats,
	}
	t.updateGoals()
	return t
}

func (t *TimerService) setWindow(w application.Window, app *application.App) {
	t.window = w
	t.app = app
}

func (t *TimerService) ShowSettings() {
	if t.window == nil || t.app == nil {
		return
	}
	state := t.GetState()
	s := t.settings.Get()
	t.app.Event.Emit("timer:tick", state)
	t.app.Event.Emit("settings:loaded", s)
	t.window.Show().Focus()
}

func (t *TimerService) Toggle() {
	t.mu.Lock()
	phase := t.phase
	t.mu.Unlock()

	if phase == TimerIdle {
		t.Start()
	} else {
		t.PauseResume()
	}
}

func (t *TimerService) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.phase != TimerIdle {
		return
	}

	t.updateGoals()
	t.phase = TimerWorking
	t.elapsed = 0
	t.paused = false

	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	go t.loop(ctx)
	t.emit()
}

func (t *TimerService) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
	t.phase = TimerIdle
	t.elapsed = 0
	t.paused = false
	t.emit()
}

// RefreshGoals re-reads settings and updates timer targets without restarting.
func (t *TimerService) RefreshGoals() {
	t.mu.Lock()
	t.updateGoals()
	// If already past new goal, cap elapsed
	if t.phase == TimerWorking && t.elapsed > t.workGoal {
		t.elapsed = t.workGoal
	}
	if t.phase == TimerBreak && t.elapsed > t.breakGoal {
		t.elapsed = t.breakGoal
	}
	t.mu.Unlock()
	t.emit()
}

func (t *TimerService) PauseResume() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.paused = !t.paused
	t.emit()
}

func (t *TimerService) SkipBreak() string {
	t.mu.Lock()
	settings := t.settings.Get()
	phase := t.phase
	t.mu.Unlock()

	if settings.StrictMode {
		return "strict_mode_enabled"
	}
	if phase != TimerBreak {
		return "not_in_break"
	}

	t.mu.Lock()
	if t.cancel != nil {
		t.cancel()
	}
	t.updateGoals()
	t.phase = TimerWorking
	t.elapsed = 0
	t.paused = false

	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	go t.loop(ctx)
	t.emit()
	t.mu.Unlock()

	if t.window != nil {
		t.window.UnFullscreen()
		t.window.SetAlwaysOnTop(false)
		t.window.Hide()
	}

	return "ok"
}

func (t *TimerService) GetState() TimerState {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lockedState()
}

// lockedState must be called with t.mu held.
func (t *TimerService) lockedState() TimerState {
	total := t.workGoal
	if t.phase == TimerBreak {
		total = t.breakGoal
	}
	return TimerState{
		Phase:     t.phase,
		Elapsed:   t.elapsed,
		Total:     total,
		TotalWork: t.workTotal,
		Paused:    t.paused,
	}
}

func (t *TimerService) updateGoals() {
	s := t.settings.Get()
	t.workGoal = s.WorkInterval * 60
	t.breakGoal = s.BreakDuration * 60
}

func (t *TimerService) emit() {
	state := t.lockedState()
	if t.app != nil {
		t.app.Event.Emit("timer:tick", state)
	}
}

func (t *TimerService) loop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.mu.Lock()
			if t.paused {
				t.mu.Unlock()
				continue
			}

			t.elapsed++

			if t.phase == TimerWorking {
				t.workTotal++
				if t.elapsed >= t.workGoal {
					t.phase = TimerBreak
					t.elapsed = 0
					t.mu.Unlock()

					if t.app != nil {
						t.app.Event.Emit("timer:phase", "break")
					}
					if t.window != nil {
						t.window.UnMinimise()
						t.window.Show()
						t.window.SetAlwaysOnTop(true)
						t.window.Fullscreen()
					}
					_ = t.stats.AddBreak(1)
					continue
				}
				t.emit()
			} else if t.phase == TimerBreak {
				t.workTotal++
				if t.elapsed >= t.breakGoal {
					t.updateGoals()
					t.phase = TimerWorking
					t.elapsed = 0
					t.mu.Unlock()

					if t.app != nil {
						t.app.Event.Emit("timer:phase", "work")
					}
					if t.window != nil {
						t.window.UnFullscreen()
						t.window.SetAlwaysOnTop(false)
						t.window.Hide()
					}
					_ = t.stats.AddWork(t.workGoal)
					continue
				}
				t.emit()
			}
			t.mu.Unlock()
		}
	}
}

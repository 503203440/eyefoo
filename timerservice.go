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
	TimerWaiting TimerPhase = 3
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

	cancel       context.CancelFunc
	waitCancel   context.CancelFunc
	waitingSince time.Time

	phaseStart time.Time
	pausedAt   time.Time
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

// lockFullscreenChrome disables the window's fullscreen (zoom) button while
// a break is running so the user has no way to dismiss the mask via the
// macOS title bar controls.
func (t *TimerService) lockFullscreenChrome(locked bool) {
	if t.window == nil {
		return
	}
	if locked {
		t.window.SetFullscreenButtonState(application.ButtonDisabled)
		t.window.SetMaximiseButtonState(application.ButtonDisabled)
	} else {
		t.window.SetFullscreenButtonState(application.ButtonEnabled)
		t.window.SetMaximiseButtonState(application.ButtonEnabled)
	}
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

	switch phase {
	case TimerIdle:
		t.Start()
	case TimerWaiting:
		t.ForceResumeWork()
	default:
		t.PauseResume()
	}
}

func (t *TimerService) Start() {
	t.mu.Lock()
	if t.phase != TimerIdle {
		t.mu.Unlock()
		return
	}
	t.mu.Unlock()

	if err := StartActivityTap(); err != nil {
		println("activity tap unavailable, falling back to legacy behavior:", err.Error())
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.phase != TimerIdle {
		return
	}

	t.updateGoals()
	t.phase = TimerWorking
	t.elapsed = 0
	t.paused = false
	t.phaseStart = time.Now()
	t.pausedAt = time.Time{}

	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	go t.loop(ctx)
	t.emit()
}

func (t *TimerService) Stop() {
	t.mu.Lock()
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
	if t.waitCancel != nil {
		t.waitCancel()
		t.waitCancel = nil
	}
	t.phase = TimerIdle
	t.elapsed = 0
	t.paused = false
	t.waitingSince = time.Time{}
	t.phaseStart = time.Time{}
	t.pausedAt = time.Time{}
	t.mu.Unlock()
	StopActivityTap()
	ExitKiosk()
	if t.window != nil {
		t.lockFullscreenChrome(false)
	}
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
	if t.paused {
		t.phaseStart = t.phaseStart.Add(time.Since(t.pausedAt))
		t.paused = false
		t.pausedAt = time.Time{}
	} else {
		t.pausedAt = time.Now()
		t.paused = true
	}
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
	if phase != TimerBreak && phase != TimerWaiting {
		return "not_in_break"
	}

	t.mu.Lock()
	if phase == TimerBreak && t.cancel != nil {
		t.cancel()
	}
	if phase == TimerWaiting && t.waitCancel != nil {
		t.waitCancel()
		t.waitCancel = nil
	}
	t.updateGoals()
	t.phase = TimerWorking
	t.elapsed = 0
	t.paused = false
	t.waitingSince = time.Time{}
	t.phaseStart = time.Now()
	t.pausedAt = time.Time{}

	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	go t.loop(ctx)
	t.emit()
	t.mu.Unlock()

	ExitKiosk()
	if t.window != nil {
		t.window.UnFullscreen()
		t.window.SetAlwaysOnTop(false)
		t.window.Hide()
		t.lockFullscreenChrome(false)
	}

	if t.app != nil {
		t.app.Event.Emit("timer:phase", "work")
	}

	return "ok"
}

// ForceResumeWork starts the work phase immediately, ignoring the
// activity-wait gate. Used by Toggle() while in TimerWaiting.
func (t *TimerService) ForceResumeWork() {
	t.mu.Lock()
	if t.phase != TimerWaiting {
		t.mu.Unlock()
		return
	}
	if t.waitCancel != nil {
		t.waitCancel()
		t.waitCancel = nil
	}
	t.updateGoals()
	t.phase = TimerWorking
	t.elapsed = 0
	t.paused = false
	t.waitingSince = time.Time{}
	t.phaseStart = time.Now()
	t.pausedAt = time.Time{}
	t.mu.Unlock()

	if t.app != nil {
		t.app.Event.Emit("timer:phase", "work")
	}
	t.emit()

	ctx, cancel := context.WithCancel(context.Background())
	t.mu.Lock()
	t.cancel = cancel
	t.mu.Unlock()
	go t.loop(ctx)
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
	if t.phase == TimerWaiting {
		total = t.workGoal
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
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.mu.Lock()
			if t.phase != TimerWorking && t.phase != TimerBreak {
				t.mu.Unlock()
				continue
			}
			if t.paused {
				t.mu.Unlock()
				continue
			}

			newElapsed := int(time.Since(t.phaseStart).Seconds())
			if newElapsed == t.elapsed {
				t.mu.Unlock()
				continue
			}
			prev := t.elapsed
			t.elapsed = newElapsed
			t.workTotal += newElapsed - prev

			if t.phase == TimerWorking {
				if newElapsed >= t.workGoal {
					t.phase = TimerBreak
					t.elapsed = 0
					t.phaseStart = time.Now()
					t.mu.Unlock()

					if t.app != nil {
						t.app.Event.Emit("timer:phase", "break")
					}
					if t.window != nil {
						t.window.UnMinimise()
						t.window.Show()
						t.window.SetAlwaysOnTop(true)
						t.window.Fullscreen()
						t.lockFullscreenChrome(true)
					}
					EnterKiosk()
					_ = t.stats.AddBreak(1)
					continue
				}
				t.emit()
				t.mu.Unlock()
			} else if t.phase == TimerBreak {
				if newElapsed >= t.breakGoal {
					_ = t.stats.AddWork(t.workGoal)
					if !IsActivityTapEnabled() {
						t.phase = TimerWorking
						t.elapsed = 0
						t.phaseStart = time.Now()
						t.mu.Unlock()

						ExitKiosk()
						if t.app != nil {
							t.app.Event.Emit("timer:phase", "work")
						}
						if t.window != nil {
							t.window.UnFullscreen()
							t.window.SetAlwaysOnTop(false)
							t.window.Hide()
							t.lockFullscreenChrome(false)
						}
						continue
					}

					t.phase = TimerWaiting
					t.elapsed = 0
					t.phaseStart = time.Time{}
					t.waitingSince = time.Now()
					waitCtx, waitCancel := context.WithCancel(context.Background())
					t.waitCancel = waitCancel
					t.mu.Unlock()

					ExitKiosk()
					if t.app != nil {
						t.app.Event.Emit("timer:phase", "waiting")
					}
					if t.window != nil {
						t.window.UnFullscreen()
						t.window.SetAlwaysOnTop(false)
						t.window.Hide()
						t.lockFullscreenChrome(false)
					}
					notify("休息结束", "请敲一下键盘或移动鼠标开始工作")
					go t.waitLoop(waitCtx)
					continue
				}
				t.emit()
				t.mu.Unlock()
			} else {
				t.mu.Unlock()
			}
		}
	}
}

// waitLoop polls LastActivity() once per second. As soon as user activity
// is detected after waitingSince, it transitions the timer into the
// working phase.
func (t *TimerService) waitLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.mu.Lock()
			if t.phase != TimerWaiting || t.paused {
				t.mu.Unlock()
				return
			}
			if !LastActivity().After(t.waitingSince) {
				t.mu.Unlock()
				continue
			}
			t.updateGoals()
			t.phase = TimerWorking
			t.elapsed = 0
			t.paused = false
			t.waitingSince = time.Time{}
			t.phaseStart = time.Now()
			t.pausedAt = time.Time{}
			t.waitCancel = nil
			t.mu.Unlock()

			if t.app != nil {
				t.app.Event.Emit("timer:phase", "work")
			}
			t.emit()

			loopCtx, cancel := context.WithCancel(context.Background())
			t.mu.Lock()
			t.cancel = cancel
			t.mu.Unlock()
			go t.loop(loopCtx)
			return
		}
	}
}

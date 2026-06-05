import {Events} from "@wailsio/runtime";
import {SettingsStore, TimerService, StatsStore} from "../bindings/eyefoo";

// ---- DOM elements ----
const settingsView = document.getElementById('settings-view');
const breakView = document.getElementById('break-view');

const timerPhaseEl = document.getElementById('timer-phase');
const timerTimeEl = document.getElementById('timer-time');
const timerLabelEl = document.getElementById('timer-label');
const timerBarEl = document.getElementById('timer-bar');
const timerIconEl = document.getElementById('timer-icon');
const btnToggle = document.getElementById('btn-toggle');
const btnSkip = document.getElementById('btn-skip');
const breakCountdownEl = document.getElementById('break-countdown');
const btnSkipBreak = document.getElementById('btn-skip-break');
const todayWorkEl = document.getElementById('today-work-time');
const eyeInstructionEl = document.getElementById('eye-instruction');
const eyeDotEl = document.getElementById('eye-dot');

// settings inputs
const setWorkEl = document.getElementById('set-work');
const setBreakEl = document.getElementById('set-break');
const setStrictEl = document.getElementById('set-strict');
const setSoundEl = document.getElementById('set-sound');
const setWarmEl = document.getElementById('set-warm');
const warmValueEl = document.getElementById('warm-value');

// ---- State ----
let currentPhase = 0; // 0=idle, 1=work, 2=break
let currentStrict = false;
let eyeAnimationTimer = null;
let exerciseStep = 0;

const exercises = [
    {text: '上下转动眼球', x: 50, y: [10, 90]},
    {text: '左右转动眼球', x: [10, 90], y: 50},
    {text: '左上到右下', x: [10, 90], y: [10, 90]},
    {text: '右上到左下', x: [90, 10], y: [10, 90]},
    {text: '画圈转动', x: 50, y: 50},
];

// ---- Global functions (called from HTML onclick) ----
window.toggleTimer = toggleTimer;
window.skipBreak = skipBreak;
window.switchTab = switchTab;
window.saveSettings = saveSettings;
window.updateWarmDisplay = updateWarmDisplay;

async function toggleTimer() {
    try {
        await TimerService.Toggle();
    } catch (e) {
        console.error(e);
    }
}

async function skipBreak() {
    try {
        const result = await TimerService.SkipBreak();
        if (result === 'strict_mode_enabled') {
            return; // silently ignore in strict mode
        }
    } catch (e) {
        console.error(e);
    }
}

async function saveSettings() {
    try {
        await SettingsStore.SaveSettings({
            work_interval: parseInt(setWorkEl.value) || 45,
            break_duration: parseInt(setBreakEl.value) || 5,
            strict_mode: setStrictEl.checked,
            sound_enabled: setSoundEl.checked,
            warm_level: parseInt(setWarmEl.value) || 0,
        });
        await TimerService.RefreshGoals();
    } catch (e) {
        console.error(e);
    }
}

function updateWarmDisplay() {
    warmValueEl.textContent = setWarmEl.value + '%';
}

function switchTab(name) {
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    document.querySelector(`.tab[data-tab="${name}"]`)?.classList.add('active');
    document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
    const tabEl = document.getElementById(`tab-${name}`);
    if (tabEl) tabEl.classList.add('active');

    if (name === 'stats') {
        loadStats();
    }
}

// ---- Timer state update ----
function updateTimerUI(state) {
    currentPhase = state.phase;

    const mins = Math.floor(state.elapsed / 60);
    const secs = state.elapsed % 60;
    const totalMins = Math.floor(state.total / 60);
    const timeStr = String(mins).padStart(2, '0') + ':' + String(secs).padStart(2, '0');
    const totalStr = String(totalMins).padStart(2, '0') + ':00';
    const pct = state.total > 0 ? Math.min(100, (state.elapsed / state.total) * 100) : 0;

    timerTimeEl.textContent = timeStr;
    timerLabelEl.textContent = '/ ' + totalStr;
    timerBarEl.style.width = pct + '%';

    const workMins = Math.round(state.total_work / 60);
    todayWorkEl.textContent = workMins + ' 分钟';

    if (state.phase === 0) { // idle
        timerPhaseEl.textContent = '已暂停';
        timerIconEl.innerHTML = '&#x23f8;';
        btnToggle.textContent = '开始计时';
        btnToggle.disabled = false;
        btnSkip.style.display = 'none';
        btnSkip.textContent = '跳过休息';
    } else if (state.phase === 1) { // working
        timerPhaseEl.textContent = state.paused ? '已暂停' : '工作中...';
        timerIconEl.innerHTML = state.paused ? '&#x23f8;' : '&#x1f4bb;';
        btnToggle.textContent = state.paused ? '继续' : '暂停';
        btnToggle.disabled = false;
        btnSkip.style.display = 'none';
        btnSkip.textContent = '跳过休息';
    } else if (state.phase === 2) { // break
        timerPhaseEl.textContent = '休息中...';
        timerIconEl.innerHTML = '&#x1f441;';
        btnToggle.textContent = '暂停';
        btnToggle.disabled = false;
        btnSkip.style.display = currentStrict ? 'none' : 'inline-block';
        btnSkip.textContent = '跳过休息';
    } else if (state.phase === 3) { // waiting
        timerPhaseEl.textContent = '请敲键盘 / 移动鼠标开始工作';
        timerIconEl.innerHTML = '&#x1f4a4;';
        timerTimeEl.textContent = '00:00';
        timerBarEl.style.width = '0%';
        btnToggle.textContent = '等待中...';
        btnToggle.disabled = true;
        btnSkip.style.display = currentStrict ? 'none' : 'inline-block';
        btnSkip.textContent = '开始工作';
    }
}

// ---- Break overlay ----
function updateBreakUI(state) {
    const remaining = state.total - state.elapsed;
    const mins = Math.floor(Math.max(0, remaining) / 60);
    const secs = Math.max(0, remaining) % 60;
    breakCountdownEl.textContent = String(mins).padStart(2, '0') + ':' + String(secs).padStart(2, '0');

    if (currentStrict) {
        btnSkipBreak.style.display = 'none';
    } else {
        btnSkipBreak.style.display = 'inline-block';
    }
}

function showBreakView() {
    settingsView.style.display = 'none';
    breakView.style.display = 'flex';
    document.body.classList.add('break-mode');
    startEyeExercise();
}

function showSettingsView() {
    breakView.style.display = 'none';
    settingsView.style.display = 'block';
    document.body.classList.remove('break-mode');
    stopEyeExercise();
}

// ---- Eye exercise animation ----
function startEyeExercise() {
    exerciseStep = 0;
    runExercise();
    eyeAnimationTimer = setInterval(runExercise, 3000);
}

function stopEyeExercise() {
    if (eyeAnimationTimer) {
        clearInterval(eyeAnimationTimer);
        eyeAnimationTimer = null;
    }
    eyeDotEl.style.transition = 'none';
    eyeDotEl.style.left = '50%';
    eyeDotEl.style.top = '50%';
}

function runExercise() {
    const ex = exercises[exerciseStep % exercises.length];
    eyeInstructionEl.textContent = ex.text;

    // Apply animation
    eyeDotEl.style.transition = 'all 1.5s ease-in-out';

    if (Array.isArray(ex.x)) {
        // Moving between two points
        const phase = (exerciseStep % 4) < 2;
        eyeDotEl.style.left = (phase ? ex.x[0] : ex.x[1]) + '%';
        eyeDotEl.style.top = (Array.isArray(ex.y) ? (phase ? ex.y[0] : ex.y[1]) : ex.y) + '%';
    } else if (Array.isArray(ex.y)) {
        const phase = (exerciseStep % 4) < 2;
        eyeDotEl.style.left = ex.x + '%';
        eyeDotEl.style.top = (phase ? ex.y[0] : ex.y[1]) + '%';
    } else {
        eyeDotEl.style.left = ex.x + '%';
        eyeDotEl.style.top = ex.y + '%';
    }

    exerciseStep++;
}

// ---- Stats ----
async function loadStats() {
    try {
        const stats = await StatsStore.GetRecent(7);
        const list = document.getElementById('stats-list');
        list.innerHTML = stats.map(s => {
            const workMin = Math.round(s.work_secs / 60);
            const breakMin = Math.round(s.break_secs / 60);
            return `<div class="stat-row">
                <span class="stat-date">${s.date}</span>
                <span class="stat-work">工作 ${workMin}分</span>
                <span class="stat-break">休息 ${s.breaks}次</span>
            </div>`;
        }).join('');
    } catch (e) {
        console.error(e);
    }
}

// ---- Events ----
Events.On('timer:tick', (evt) => {
    const state = evt.data;
    if (state.phase === 2) {
        updateBreakUI(state);
    }
    updateTimerUI(state);
});

Events.On('timer:phase', (evt) => {
    const phase = evt.data;
    if (phase === 'break') {
        showBreakView();
    } else if (phase === 'work') {
        showSettingsView();
    } else if (phase === 'waiting') {
        showSettingsView();
    }
});

// Load settings on 'settings:loaded' event
Events.On('settings:loaded', (evt) => {
    const s = evt.data;
    setWorkEl.value = s.work_interval;
    setBreakEl.value = s.break_duration;
    setStrictEl.checked = s.strict_mode;
    setSoundEl.checked = s.sound_enabled;
    setWarmEl.value = s.warm_level;
    warmValueEl.textContent = s.warm_level + '%';
    currentStrict = s.strict_mode;
});

// ---- Initialize ----
async function init() {
    try {
        const [state, settings, stats] = await Promise.all([
            TimerService.GetState(),
            SettingsStore.GetSettings(),
            StatsStore.GetToday(),
        ]);

        setWorkEl.value = settings.work_interval;
        setBreakEl.value = settings.break_duration;
        setStrictEl.checked = settings.strict_mode;
        setSoundEl.checked = settings.sound_enabled;
        setWarmEl.value = settings.warm_level;
        warmValueEl.textContent = settings.warm_level + '%';
        currentStrict = settings.strict_mode;

        updateTimerUI(state);
        if (state.phase === 2) {
            showBreakView();
        } else {
            showSettingsView();
        }
        if (state.phase === 3) {
            btnToggle.disabled = true;
        }

        todayWorkEl.textContent = Math.round(stats.work_secs / 60) + ' 分钟';
    } catch (e) {
        console.error('Init error:', e);
    }
}

init();

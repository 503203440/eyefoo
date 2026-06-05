# 眼睛护士 Eyefoo

基于 Wails v3 构建的跨平台护眼提醒软件，致敬经典的 eyefoo 眼睛护士。

## 功能特性

| 功能 | 说明 |
|------|------|
| 系统托盘 | macOS 菜单栏图标，顶栏常驻，右键菜单操作 |
| 定时提醒 | 工作 x 分钟后弹出全屏休息遮罩（默认 45 分钟） |
| 强制休息 | 全屏暗色遮罩 + Always on Top + 倒计时 |
| 眼保健操 | 休息时播放动画引导（上下/左右/对角/画圈转动眼球） |
| 灵活设置 | 工作时长、休息时长、强制模式、声音开关，修改即时生效 |
| 使用统计 | 每日工作时长、休息次数，保留近 7 天记录 |
| 严格模式 | 开启后无法跳过休息，强制护眼 |
| Focus 自动隐藏 | 设置窗口失焦自动收起，与 macOS 原生 popover 体验一致 |

## 技术栈

| 层级 | 技术 |
|------|------|
| 框架 | [Wails v3](https://v3.wails.io/) (alpha.98) |
| 后端 | Go 1.25+ |
| 前端 | 原生 HTML / CSS / JavaScript (无框架) |
| 构建工具 | Vite + Taskfile |
| 平台 | macOS / Windows / Linux (当前主要适配 macOS) |

## 环境要求

- **Go** ≥ 1.25
- **Node.js** ≥ 18 (含 npm)
- **Wails v3 CLI**:
  ```bash
  go install github.com/wailsapp/wails/v3/cmd/wails3@latest
  ```
- **Xcode Command Line Tools** (仅 macOS)

验证环境:
```bash
wails3 doctor
```

## 项目结构

```
eyefoo/
├── main.go              # 入口，系统托盘 + 窗口 + 事件注册
├── settingsservice.go   # 设置 CRUD（JSON 持久化）
├── timerservice.go      # 定时器引擎（工作/休息循环）
├── statsservice.go      # 使用统计
├── frontend/
│   ├── index.html       # UI 主页面（设置页 + 休息遮罩 + 眼保健操）
│   ├── src/main.js      # 前端逻辑（事件监听、UI 切换）
│   ├── public/style.css # 深色主题样式
│   └── bindings/        # 自动生成的 Go ↔ JS 绑定（勿手动编辑）
├── build/               # 构建配置、图标、Info.plist 等
├── Taskfile.yml         # 构建任务定义
└── go.mod / go.sum      # Go 模块依赖
```

## 快速开始

```bash
# 1. 克隆项目
git clone https://github.com/503203440/eyefoo.git
cd eyefoo

# 2. 安装前端依赖
cd frontend && npm install && cd ..

# 3. 开发模式（热重载）
wails3 dev

# 4. 生产构建 + 打包为 .app
wails3 package
# 产物在 bin/eyefoo.app
```

## 配置文件

设置和统计数据存储在 `~/.config/eyefoo/`:

| 文件 | 内容 |
|------|------|
| `settings.json` | 工作时长、休息时长、严格模式、声音开关 |
| `stats.json` | 每日工作时长、休息次数历史 |

## 自定义图标

1. 准备一张 1024×1024 的 PNG 图片
2. 替换 `build/appicon.png`
3. 重新执行 `wails3 build && wails3 package`

## License

MIT

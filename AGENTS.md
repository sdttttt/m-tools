# AGENTS.md — M-Tools

## 项目定位

个人工具箱桌面应用，目标平台 **macOS** 和 **Windows**。产出单一 Go 二进制，体积小、内存占用低。功能以"日常高频小工具"为单位逐步增加（功能清单由用户安排，不要擅自增加业务功能）。

## 技术栈（不要替换）

| 层            | 选型                                                               | 说明                                                       |
| ------------- | ------------------------------------------------------------------ | ---------------------------------------------------------- |
| 桌面框架      | [MyGO](https://github.com/egoist/mygo)（`github.com/egoist/mygo`） | **只用 webview 模式**，不使用其原生 GoUI（package `ui`）   |
| 前端 UI       | SolidJS + TypeScript                                               | 运行在系统 webview（macOS: WKWebView / Windows: WebView2） |
| 包管理        | bun                                                                | 前端依赖一律用 `bun add`，不用 npm/pnpm/yarn               |
| 构建器        | vite + vite-plugin-solid                                           |                                                            |
| 原子化 CSS    | [UnoCSS](https://unocss.dev)（v66，presetWind3）                   | 组件样式一律用原子类                                       |
| Lint / 格式化 | oxlint / oxfmt                                                     | 不引入 eslint / prettier / biome                           |
| 后端          | Go（纯 Go，无 cgo）                                                | 系统能力、复杂逻辑                                         |
| 可选 WASM     | Rust                                                               | 仅在边界收益明确时使用，见下文决策规则                     |

## 架构与数据流

```
┌─ webview ──────────────┐        ┌─ Go 进程 ─────────────┐
│ SolidJS (webview/src)  │  IPC   │ cmd/m-tools + services│──▶ 系统 API
│  import { X } from     │◀──────▶│ mygo.Bind(X{})        │
│  "./mygo"  (生成物)     │ 类型化  │ mygo.NewEvent[T]      │
└────────────────────────┘        └───────────────────────┘
```

- Go 侧导出服务结构体，`cmd/m-tools/main.go` 中 `mygo.Bind(X{})` 注册；方法第一个字母大写即可被前端调用。
- `bun run generate` 从 Go 代码生成类型化客户端 **`webview/src/mygo.ts`**（生成物，**禁止手改**）。方法名转 camelCase，字段名遵循 Go struct 的 json tag（用 camelCase tag）。
- 前端调用是 async：`const info = await System.info()`。
- 事件（Go → 前端推送流）：`mygo.NewEvent[T]("name")` 声明，`Emit(win, v)` 发送；前端 `import { events } from "./mygo"` 后 `events.name.on(...)`。
- MyGo 配置唯一来源是 `mygo.config.ts`（devUrl、devCommand、buildCommand、frontendDist、bindings、out、identifier 等）。

## 目录结构（Go 标准布局）

```
m-tools/
├── cmd/
│   └── m-tools/
│       └── main.go       # 应用入口：Bind 服务 + 窗口创建（保持薄）
├── app.go                # 根包 mtools：应用元数据；go:embed 无法引用上级路径，
├── assets_ffmpeg.go      #   所以 with_ffmpeg 的嵌入 hook（//go:build with_ffmpeg）留在根
├── go.mod                # module github.com/sdttttt/m-tools（main: ./cmd/m-tools）
├── mygo.config.ts        # MyGo 唯一配置（main/devCommand/buildCommand/bindings/out）
├── package.json          # 根只放编排脚本（dev/build/generate）+ mygo-cli
├── webview/              # ⭐ 前端整体住这里
│   ├── package.json      # 前端依赖与脚本（dev/build/typecheck/lint/format）
│   ├── bun.lock
│   ├── index.html        # 前端入口 HTML
│   ├── vite.config.ts    # vite + solid 插件；dev 端口 5173（strictPort）
│   ├── tsconfig.json     # strict；jsx preserve + jsxImportSource solid-js
│   ├── uno.config.ts     # UnoCSS：presetWind3，dark:"media"，主题色映射 CSS 变量
│   ├── .oxfmtrc.json
│   └── src/
│       ├── index.tsx     # render(App)；先 import virtual:uno.css
│       ├── App.tsx       # 根组件：首页工具网格 + 视图切换
│       ├── tools/        # 各工具页面（每个工具一个文件，如 Mp4Gif.tsx）
│       └── mygo.ts       # ⚠️ 生成物：bun run generate（根目录跑）
├── ffmpeg/               # ⭐ 独立 gomod（github.com/sdttttt/m-tools/ffmpeg）
│   │                     #   封装全部 ffmpeg/ffprobe 交互，见下方桥接口
│   ├── client.go         # Client + New()
│   ├── lookup.go         # 工具查找（env → bundled → bin/ → PATH）
│   ├── probe.go          # Probe + VideoInfo
│   └── gif.go            # ConvertGIF + GIFRequest/GIFResult/GIFProgress
├── resources/
│   ├── icon.png
│   └── bin/              # with_ffmpeg 构建时捆绑的 ffmpeg（gitignore，留 README）
├── build/                # mygo build 产物（gitignore）
├── .agents/skills/mygo-maintenance/  # MyGo 维护 skill（脚手架自带）
└── AGENTS.md
```

布局规则：Go 入口在 `cmd/<app>/`；前端代码只进 `webview/`（组件、样式、配置、依赖）；ffmpeg 交互只进 `ffmpeg/` 模块；仓库根只留 go.mod、编排配置和必须位于根的文件（embed hook、mygo.config.ts）。

## 常用命令

| 命令                                                | 作用                                                    |
| --------------------------------------------------- | ------------------------------------------------------- |
| `bun run dev`（根目录）                             | 桌面开发：启动 vite dev server 并打开应用窗口（热更新） |
| `cd webview && bun run dev`                         | 只起前端 dev server（浏览器里调 UI，无 Go 桥）          |
| `bun run generate`（根目录）                        | Go 服务变更后重新生成 `webview/src/mygo.ts`             |
| `bun run build`（根目录）                           | 打包桌面应用到 `build/`（内嵌 webview/dist）            |
| `cd webview && bun run typecheck/lint/format`       | 前端检查（tsc / oxlint / oxfmt，作用域限 webview/）     |
| `go vet ./...` / `go build ./...` / `go test ./...` | Go 侧检查（含 cmd/m-tools 与 ffmpeg 模块）              |
| `go build -tags with_ffmpeg ./cmd/m-tools`          | 捆绑 ffmpeg 的构建变体                                  |
| `bunx mygo vet .`（根目录）                         | MyGo 静态检查（goroutine 捕获等）                       |

## 开发约定

### 前端（webview/src/）

- 组件文件 PascalCase（`Foo.tsx`），入口/工具文件小写。
- SolidJS 响应式红线：不要解构 props、不要用对象展开丢失响应性、粒度用 signal/memo 而非大状态对象。
- **禁止手写 .css 文件**。样式一律用 UnoCSS 原子类（presetWind3）。语义色 token 定义在 `uno.config.ts` 的 `theme.colors`，映射 preflight 里的 CSS 变量：`bg-bg`、`text-fg`、`text-muted`、`bg-card`、`border-line`、`text-accent`。新增语义色：在 uno.config.ts 的 preflights 加变量（浅/深两套）→ theme.colors 映射 → 组件里用类名。
- 基础 token 与 reset（CSS 变量、body 字体/背景、color-scheme）全部放在 `uno.config.ts` 的 `preflights` 中，由 `virtual:uno.css` 注入；页面级特例也优先用原子类解决。
- `dark:` 变体配置为 `media` 模式（跟随系统），与 CSS 变量的翻转机制一致；大部分场景用语义 token 即可，不需要写 dark: 变体。
- 必须同时适配浅色/深色。**窗口 `BackgroundColor`（cmd/m-tools/main.go）与 `--bg` 保持一致**，避免启动闪白/闪黑。
- `webview/src/mygo.ts` 的类型即 IPC 契约：Go 返回类型变更后先 `bun run generate` 再写前端。

### Go

- 服务方法必须有 doc comment——它会出现在生成的 TS 客户端文档里。
- struct 字段 json tag 用 camelCase。
- `cmd/m-tools/main.go` 只做 Bind 与窗口装配；具体工具的实现放 `internal/<工具域名>/`（如 `internal/clipboard/`）。
- 平台相关代码用 build tag 分文件（`foo_windows.go` / `foo_darwin.go`），接口在无后缀文件中声明。

### 代码风格

- 提交前：`cd webview && bun run format` + `bun run lint`；Go 用 gofmt（`gofmt -w`）。
- 生成物（`webview/src/mygo.ts`、两份 `bun.lock`）正常提交，不进 .gitignore。

## 功能实现边界（前端 / Go / WASM 决策）

按顺序判断，取第一个满足的：

1. **纯前端**：Web/WebView API 能覆盖且无性能问题（表单、纯计算、DOM 操作、fetch 等）→ 只写 SolidJS。
2. **Go 服务**：需要 OS 能力——文件系统、进程、剪贴板/窗口控制、注册表/plist、原始网络、后台定时任务等 → Go 服务 + IPC。**默认倾向这里**。
3. **WASM（Rust）**：仅当 ① 计算密集且明显受益（解析/压缩/图像等），② Rust 生态有成熟现成库而 Go 没有或质量差距大。引入前先与 Go 方案对比二进制体积、加载延迟、维护成本，向用户说明收益再动手。

原则：新增 OS 依赖优先 Go；不为"用 Rust"而用 WASM；每个工具保持小而独立。

## MP4 → GIF 工具要点（mp4gif）

- 依赖外部 **ffmpeg/ffprobe**（纯 Go 无成熟视频解码器，见选型记录）。
- **ffmpeg 封装原则**：所有 ffmpeg 交互（可执行文件查找、CLI 参数、滤镜链、输出解析）封装在独立 gomod `ffmpeg/` 中，主程序只见桥接口，禁止在模块外拼装命令或引用可执行路径：
  - `ffmpeg.New() (*Client, error)` —— 解析工具并返回客户端
  - `client.Probe(ctx, path) (VideoInfo, error)` —— 元数据
  - `client.ConvertGIF(ctx, GIFRequest, onProgress) (GIFResult, error)` —— 转换 + 输出校验；请求里传 `SourceFPS`/`SourceWidth`，上限收敛在模块内做
  - `ffmpeg.UseBundledDir(func() (dir, error))` —— with_ffmpeg 构建的注入点（main 包 init 里注册）
  - `ffmpeg.EnvDir`（`MTOOLS_FFMPEG_DIR`）—— 开发/测试用环境变量覆盖
- 查找顺序：`$MTOOLS_FFMPEG_DIR` → `with_ffmpeg` 构建内嵌包（解压到用户缓存目录）→ 可执行文件旁的 `bin/` → PATH。
- **构建变体**：`go build`（默认，依赖系统/环境 PATH 的 ffmpeg）；`go build -tags with_ffmpeg`（捆绑 `resources/bin/` 下的 ffmpeg/ffprobe 进单二进制，运行时首次解压）。
- 转换策略：两遍 palettegen/paletteuse（GIF 质量关键），FPS/宽度不超过源视频；进度经 `-progress pipe:1` 解析后通过 `mp4gifProgress` 事件推给前端（palette 占 30% / encode 占 70%）。
- mygo 服务包装（Bind/事件/对话框/取消/预览/保存）在 `cmd/m-tools/main.go`；一次只跑一个转换任务（`CancelActive`）。
- 模块测试：ffmpeg 模块含真 ffmpeg 端到端集成测试（lavfi 合成测试视频→转换→GIF 解码校验），无 ffmpeg 时自动 skip。

## 版本与构建

- 版本号三处同步：`mygo.config.ts` 的 `version`、`cmd/m-tools/main.go` 的 `Version` 常量；发版时一并改。
- `mygo build` 会把 `frontendDist`（`dist/`）嵌入二进制；`build/` 下产物按平台输出（macOS app bundle / Windows exe）。
- 目标平台只有 darwin (amd64/arm64) 和 windows (amd64)；构建/签名/公证脚本将来单独安排。

## 验证清单（提交前 / 大改后）

1. `cd webview && bun run format` → `bun run lint` → `bun run typecheck`
2. `go vet ./...` && `go build ./...`
3. 改过 Go 服务：`bun run generate`，检查 `webview/src/mygo.ts` 的 diff
4. UI 或 IPC 改动：`bun run dev` 实际跑一次，确认窗口、深浅色、IPC 调用正常
5. MyGo 生命周期/并发改动：`go tool mygo vet .`

## Git

- 远程：`origin` = `git@github.com:sdttttt/m-tools.git`，主分支 `master`。
- Commit message 用英文祈使句、小写开头，如 `add clipboard history tool`；一次提交聚焦一件事。

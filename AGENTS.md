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
| 组件库        | [solid-ui](https://www.solid-ui.com)（Kobalte + Tailwind v3）      | shadcn 风格，组件代码复制进 webview/src/components/ui      |
| Lint / 格式化 | oxlint / oxfmt                                                     | 不引入 eslint / prettier / biome                           |
| 后端          | Go（纯 Go，无 cgo）                                                | 系统能力、复杂逻辑                                         |
| 可选 WASM     | Rust                                                               | 仅在边界收益明确时使用，见下文决策规则                     |

## 架构与数据流

```
┌─ webview ──────────────┐        ┌─ Go 进程 ─────────────┐
│ SolidJS (webview/src)  │  IPC   │ cmd/m-tools + internal│──▶ 系统 API
│  import { X } from     │◀──────▶│ mygo.BindAs("X", ...) │
│  "./mygo"  (生成物)     │ 类型化  │ mygo.NewEvent[T]      │
└────────────────────────┘        └───────────────────────┘
```

- Go 侧在 `internal/<工具域名>/` 导出服务结构体（名字统一用 `Service`），在 `cmd/m-tools/main.go` 用 `mygo.BindAs("X", ...)` 注册；**`BindAs` 的名字就是前端看到的服务名**（生成物契约），方法第一个字母大写即可被前端调用。
- `bun run generate` 从 Go 代码生成类型化客户端 **`webview/src/mygo.ts`**（生成物，**禁止手改**）。方法名转 camelCase，字段名遵循 Go struct 的 json tag（用 camelCase tag）。
- 前端调用是 async：`const info = await System.info()`。
- 事件（Go → 前端推送流）：`mygo.NewEvent[T]("name")` 声明，`Emit(win, v)` 发送；前端 `import { events } from "./mygo"` 后 `events.name.on(...)`。
- MyGo 配置唯一来源是 `mygo.config.ts`（devUrl、devCommand、buildCommand、frontendDist、bindings、out、identifier 等）。

## 目录结构（Go 标准布局）

```
m-tools/
├── cmd/
│   └── m-tools/
│       └── main.go       # 应用入口：BindAs 服务 + 窗口创建（保持薄）
├── internal/             # ⭐ 各工具的后端实现（一个工具一个包）
│   ├── system/           # System 服务：主机信息 + 窗口主题（system.go / theme.go）
│   ├── mp4gif/           # MP4→GIF 服务：service / wire / media / files / events
│   ├── imgfmt/           # 图片工具共享层：格式表 + probe + decode→resize→flatten→encode 管线
│   ├── imgconv/          # 图片格式转换服务：service / wire / files（纯转格式，不改尺寸）
│   ├── imgresize/        # 图片大小服务：service / wire / files（比例/精确尺寸，可放大）
│   ├── imgopt/           # 图片质量优化服务：optimize（各格式无损策略）/ wire / service / files
│   ├── imgfile/          # 图片工具共享：选图/保存对话框、复制另存、打开目录
│   └── reveal/           # 共享：在系统文件管理器中打开目录（darwin/windows/other 三份）
├── app.go                # 根包 mtools：AppName/Version 等应用元数据；go:embed 无法引用上级路径，
├── assets_ffmpeg.go      #   所以 with_ffmpeg 的嵌入 hook（//go:build with_ffmpeg）留在根
├── go.mod                # module github.com/sdttttt/m-tools（main: ./cmd/m-tools）
├── mygo.config.ts        # MyGo 唯一配置（main/devCommand/buildCommand/bindings/out）
├── package.json          # 根只放编排脚本（dev/build/generate）+ mygo-cli
├── webview/              # ⭐ 前端整体住这里
│   ├── package.json      # 前端依赖与脚本（dev/build/typecheck/lint/format）
│   ├── bun.lock
│   ├── index.html        # 前端入口 HTML
│   ├── vite.config.ts    # vite + solid 插件 + ~ 别名；dev 端口 5173（strictPort）
│   ├── tsconfig.json     # strict；jsx preserve；paths ~/* → src（TS7 无 baseUrl）
│   ├── tailwind.config.ts # darkMode 走 .dark 类；颜色映射 hsl(var(--x)) token
│   │                      #   postcss 内联在 vite.config.ts（无 postcss.config.*）
│   ├── ui.config.json   # solidui-cli 配置（别名 ~/components/ui、~/lib/utils）
│   ├── index.html        # 前端入口 HTML
│   ├── .oxfmtrc.json
│   └── src/
│       ├── index.tsx     # render(App)；import index.css
│       ├── App.tsx       # 根组件：首页工具网格 + 视图切换 + ThemeToggle
│       ├── tools/        # 各工具页面（每个工具一个文件，如 Mp4Gif.tsx）
│       ├── components/ui/ # solid-ui 组件（从 registry 复制，勿手写样式进这里）
│       ├── components/ThemeToggle.tsx
│       ├── components/TrimTimeline.tsx # 胶片时间轴（双拖拽手柄 + 键盘微调）
│       ├── lib/theme.ts  # 主题唯一来源（light|dark|system，持久化 + 窗口同步）
│       ├── lib/utils.ts  # cn()
│       ├── index.css     # 唯一 CSS：@tailwind 指令 + HSL token + body 基础
│       └── mygo.ts       # ⚠️ 生成物：bun run generate（根目录跑）
├── ffmpeg/               # ⭐ 独立 gomod（github.com/sdttttt/m-tools/ffmpeg）
│   │                     #   封装全部 ffmpeg/ffprobe 交互，见下方桥接口
│   ├── client.go         # Client + New()
│   ├── lookup.go         # 工具查找（env → bundled → bin/ → PATH）
│   ├── probe.go          # Probe + VideoInfo
│   └── gif.go            # ConvertGIF + GIFRequest/GIFResult/GIFProgress
├── webp/                 # 独立 gomod：WebP 编解码桥（with_webp 构建才有真实现）
│   ├── real.go / stub.go #   //go:build with_webp 分界：chai2010/webp vs ErrUnsupported
│   └── webp.go           #   ErrUnsupported 哨兵 + 包文档
├── resources/
│   ├── icon.png
│   └── bin/              # with_ffmpeg 构建时捆绑的 ffmpeg（gitignore，留 README）
├── build/                # mygo build 产物（gitignore）
├── README.md             # 面向使用者的简介（功能 / 安装 / 开发 / 许可）
├── LICENSE               # 0BSD：只覆盖本仓库原创代码
├── THIRD-PARTY-NOTICES.md # 分发产物内第三方组件的许可与版权声明
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
| `go test -tags with_webp ./internal/imgconv/` 等    | WebP 变体检查（CI 同款，见 CI 小节）                    |
| `bunx mygo vet .`（根目录）                         | MyGo 静态检查（goroutine 捕获等）                       |

## 开发约定

### 前端（webview/src/）

- 组件文件 PascalCase（`Foo.tsx`），入口/工具文件小写。
- SolidJS 响应式红线：不要解构 props、不要用对象展开丢失响应性、粒度用 signal/memo 而非大状态对象。
- 样式一律 Tailwind 工具类（solid-ui 体系）。语义色是 HSL token：唯一 CSS 文件 `webview/src/index.css`（@tailwind 指令 + `:root`/`.dark` 两套变量 + body 基础样式），`tailwind.config.cjs` 把 `bg-primary`、`text-muted-foreground` 等映射到 `hsl(var(--x))`。新增语义色：index.css 加变量（浅/深两套）→ 组件里直接用类名，tailwind 配置不用动。
- 组件来自 solid-ui registry：`bunx solidui-cli add <name>`（CLI 在 Windows/bun 下装依赖有 bug——给 bun 传了空参数；绕过方式：抓 `https://www.solid-ui.com/registry/ui/<name>.json`，把 `files[].content` 落盘到 `src/components/ui/`，并把内容里的 `~/registry/ui` 改写为 `~/components/ui`）。
- **主题机制**：`lib/theme.ts` 是唯一来源。模式 `light|dark|system` 持久化在 localStorage(`mtools-theme`)，默认跟随系统；解析结果以 `.dark` 类挂到 `<html>`（tailwind darkMode variant），并经 `System.setWindowTheme` IPC 同步原生窗口 chrome（背景色常量在 `internal/system/theme.go`，与 index.css 的 `--background` 保持一致）。UI 切换器在 `components/ThemeToggle.tsx`。
- Kobalte 表单注意：`TextField` 的 `value`/`onChange` 挂在 **Root** 上（onChange 收字符串）；`Select` 要传 `itemComponent` 且 `<SelectContent />` 必须渲染，否则下拉不显示；`ToggleGroup` 单选模式 value 是字符串不是数组。
- `~/*` 别名指向 `webview/src`（tsconfig paths + vite alias；TS7 已移除 baseUrl，paths 直接相对 tsconfig 解析）。
- 必须同时适配浅色/深色。**窗口 `BackgroundColor`（`system.WindowBackground()`，颜色常量在 `internal/system/theme.go`）与 `--bg` 保持一致**，避免启动闪白/闪黑。
- `webview/src/mygo.ts` 的类型即 IPC 契约：Go 返回类型变更后先 `bun run generate` 再写前端。

### Go

- 服务方法必须有 doc comment——它会出现在生成的 TS 客户端文档里。
- struct 字段 json tag 用 camelCase。
- `cmd/m-tools/main.go` 只做 `mygo.BindAs` 与窗口装配；具体工具的实现放 `internal/<工具域名>/`（如 `internal/clipboard/`），服务类型统一命名 `Service`，绑定名在 `BindAs` 里显式给出。
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
  - `client.Thumbnails(ctx, input, count, width, durationSec) ([][]byte, error)` —— 时间轴胶片缩略图（JPEG，按全片均匀采样）
  - `ffmpeg.UseBundledDir(func() (dir, error))` —— with_ffmpeg 构建的注入点（main 包 init 里注册）
  - `ffmpeg.EnvDir`（`MTOOLS_FFMPEG_DIR`）—— 开发/测试用环境变量覆盖
- 查找顺序：`$MTOOLS_FFMPEG_DIR` → `with_ffmpeg` 构建内嵌包（解压到用户缓存目录）→ 可执行文件旁的 `bin/` → PATH。
- Windows 上 ffmpeg/ffprobe 是 console 程序，GUI 版 exe（`-H=windowsgui`）直接启动会弹黑窗口；模块内所有子进程启动都调用 `hideConsole`（`exec_windows.go`，`CREATE_NO_WINDOW`），新增 exec 调用点必须记得调用（`exec_other.go` 为非 Windows 的空实现）。
- **构建变体**：`go build`（默认，依赖系统/环境 PATH 的 ffmpeg）；`go build -tags with_ffmpeg`（捆绑 `resources/bin/` 下的 ffmpeg/ffprobe 进单二进制，运行时首次解压）。图片工具同理：默认构建无 WebP；`go build -tags with_webp`（静态链接 chai2010/webp + libwebp，需本机 C 工具链；Windows 变体需 mingw-w64 交叉编译，未接入 CI）。两个 tag 可叠加。
- 转换策略：两遍 palettegen/paletteuse（GIF 质量关键），FPS/宽度不超过源视频；进度经 `-progress pipe:1` 解析后通过 `mp4gifProgress` 事件推给前端（palette 占 30% / encode 占 70%）。
- 截取范围：前端 `components/TrimTimeline.tsx` 是胶片时间轴（拖起点/终点手柄，拖选中段整体平移，方向键 ±0.1s、Shift ±1s），起点/终点数值输入框与之同步；缩略图经 `Mp4Gif.Thumbnails`（12 帧、160px 宽、data URL）加载，失败时退化为纯色轨道仍可拖拽。
- 片段预览：「预览片段」按钮调 `Mp4Gif.Preview`（前 3 秒、宽度 ≤320、fps ≤15，质量沿用当前选择）生成小 GIF 以 data URL 内联展示并自动删除临时文件；取消与转换共用 `CancelActive`，两种任务互斥（`begin`/`end` 守卫）；裁剪或参数变化后旧预览自动失效。
- mygo 服务包装在 `internal/mp4gif/`（`service.go` 生命周期与取消守卫、`wire.go` IPC 类型、`media.go` ffmpeg 桥接、`files.go` 对话框/保存/打开目录、`events.go` 进度事件）；一次只跑一个任务（转换或预览，`CancelActive` 取消）。打开文件夹走共享包 `internal/reveal`（`reveal.Open`，darwin/windows/other 三份平台文件）。
- 模块测试：ffmpeg 模块含真 ffmpeg 端到端集成测试（lavfi 合成测试视频→转换→GIF 解码校验、时间轴缩略图抽取），无 ffmpeg 时自动 skip。

## 图片工具要点（imgconv / imgresize / imgopt / imgfile）

- 各图片工具共用 `internal/imgfmt`（格式注册表、magic 嗅探 probe、decode→resize→flatten→encode 管线）与 `internal/imgfile`（选图/保存对话框、复制另存、reveal），格式知识、缩放/白底合成与文件交互都在共享层，服务层只做守卫与 IPC 粘合：
  - **图片格式转换（imgconv）**：只转格式，不改尺寸（无任何缩放参数）；`Convert` 保持源像素大小。
  - **图片大小（imgresize）**：按比例（1–400%，可放大）或精确尺寸（width/height，`lock=true` 时单轴按源比例推导另一轴）调整；`Resize` 里 `resolveSize` 算目标像素后交给同一管线；输出格式可顺带切换（含保持原格式）。
  - **图片质量优化（imgopt）**：像素级无损压缩，输出带 `-min` 后缀的副本（`imgfmt.OutputFor` 防冲突），**绝不覆盖源文件、绝不写出更大的文件**（产物 ≥ 源时标「已是最优」不写盘）。按格式分发（`optimize.go`）：
    - PNG：全量解码 → 唯一色 ≤256 且 alpha 只取全透明/不透明 → 索引色重编码（逐像素 16-bit 通道校验等值，失败退回全色 `png.BestCompression`）；APNG（嗅探 `acTL` chunk）跳过
    - JPEG：**字节级剥离 APPn/COM 元数据段，零解码零编码**（DQT/SOS 后的熵编码数据原样保留）；无段可剥即「已是最优」
    - TIFF：解码 → Deflate 重编码；静态 GIF：重编码（`gif.DecodeAll` 判帧数，多帧跳过）；WebP：VP8L 无损重编码（仅 with_webp，`webp.EncodeLossless`；VP8X 动画位判动图）；BMP：整格式跳过（无压缩空间）
    - 动图（APNG/动 GIF/动 WebP）一律跳过——「动图取首帧」的管线语义对优化工具是破坏性的
  - 图片服务文件交互（`PickImages`/`SaveAs`/`OpenFolder`）统一走 `internal/imgfile`，新增图片工具不要复制这份逻辑。
  - 三个图片服务各自绑定 `ImageConv` / `ImageResize` / `ImageOptimize`（`BindAs` 指针），API 形状一致：`Supported/Probe/PickImages/<Convert|Resize|Optimize>/SaveAs/OpenFolder/CancelActive`（imgconv/imgresize 另有 `TargetNames`；imgopt 无参数无选项）。
- 纯 Go（默认构建）：标准库覆盖 PNG/JPEG/GIF，`golang.org/x/image` 覆盖 BMP/TIFF 与缩放（`draw.CatmullRom`）；**WebP 编解码只在 `-tags with_webp` 构建**（见下方构建变体）。
- **WebP 封装原则**（镜像 ffmpeg 模块的边界）：cgo 依赖 `chai2010/webp`（内嵌 libwebp C 源码，无需系统 libwebp）只存在于独立 gomod `webp/` 中：
  - `webp.Supported() bool` —— 当前构建是否有编解码能力
  - `webp.Decode/DecodeConfig/Encode` —— with_webp 构建为真实现；否则返回哨兵 `webp.ErrUnsupported`
  - `webp.Register()` —— with_webp 构建在两个图片服务的 init 里调用，向 `image` 包注册 webp 魔数；stub 构建是 no-op（保证 `image.Decode` 不误认 webp 字节）
- **流水线全在内存**（`bytes.Buffer`，无临时文件）：`image.Decode`（动图取首帧）→ `imgfmt.Resize`（两轴独立，0=保持该轴；放大允许，钳制策略归 imgresize）→ 带 alpha 转 JPEG 合白底 → 编码写盘到源文件旁（同名冲突加 `-1` 后缀）。
- 语义约束：画质（high=92/balanced=82/compact=62）只影响 JPEG/WebP——无损目标下前端把画质框禁用；GIF 由标准库自动 256 色量化；EXIF 元数据丢弃、EXIF 方向不处理（已知限制）；HEIC/AVIF 不支持。imgopt 与此相反：目标是**保元数据以外的一切**（JPEG 剥元数据是它的优化手段，属预期行为）。
- **批量 = 前端逐张循环调单张 API**（单张 IPC 毫秒级，前端逐行更新状态/失败不阻塞/取消即停止发起下一张），后端不做批量 API，只有单任务互斥守卫。`Probe` 永不报错——不可识别/不支持的源通过 `skipReason` 让前端标"跳过"。
- 前端共享 `webview/src/lib/img.ts`（格式/画质标签表、probeAll、行状态——六态含 `optimizing`）；页面 `tools/ImageConv.tsx`、`tools/ImageResize.tsx`、`tools/ImageOptimize.tsx`：挂载先调 `Supported()`，stub 构建下 WebP 选项直接从下拉里剔除并在选图卡注明需 `with_webp` 构建。**Select 的 options/value/onChange 必须同域**（都用标签字符串），且选项字符串必须与 value **逐字相等**——任何展示性修饰（如给当前项加"（当前）"后缀）都会让 Kobalte 匹配失败、触发框空白；需要额外信息用 InfoTip 或页面文案表达，不要改选项文本（Kobalte 按 `value ∈ options` 回显）。
- 测试：`internal/imgfmt` 合成图 round-trip（矩阵/白底合成/首帧/上下采样/部分轴/命名冲突/取消 + stub webp 行为）；`internal/imgresize` 的 `resolveSize`（百分比/锁比例推导/取整/非法输入）；`internal/imgopt` 各格式策略（噪声 PNG 收缩+像素等值、palette 路径、半透明不走 palette、JPEG 剥段保 DQT/EOI、动/静 GIF、BMP/未知跳过、APNG、守卫不写更大文件）；`webp/` 模块在两种 tag 下都有测试。

## 版本与构建

- 版本号同步：`mygo.config.ts` 的 `version` 与根包 `app.go` 的 `Version` 常量；发版时一并改。格式为日期版本 `YYYY.MM.DD`（CalVer），如 `2026.11.05`。
- `mygo build` 会把 `frontendDist`（`dist/`）嵌入二进制；`build/` 下产物按平台输出（macOS app bundle / Windows exe）。
- 目标平台只有 darwin (amd64/arm64) 和 windows (amd64)；签名/公证脚本将来单独安排。

## CI 与发布（.github/workflows/ci.yml）

- 路径过滤（不触发 CI 的改动）：仅 `.md`、`LICENSE`、`THIRD-PARTY-NOTICES.md`、`.gitignore`、`.agents/**`。PR 用触发级 `paths-ignore` 直接跳过；master push 由 `changes` job（`dorny/paths-filter@v3`，白名单在 `ci.yml` 的 `changes` job 里维护）判定，纯文档/工作流改动跳过 `test`/`build`；tag push 无条件全跑（否则 release 会被路径过滤误杀，不要给 push 加触发级 `paths`/`paths-ignore`）。
- push/PR：`test` 跑 gofmt、版本一致性、生成物漂移检查（`bun run generate` 后 `git diff` 为空）、Go vet/build/test、webp 变体检查（`go build/vet -tags with_webp ./...` + `webp/` 与 `internal/imgconv` 双 tag 测试）、ffmpeg 模块集成测试、`mygo vet` 与前端 format/lint/typecheck；通过后 `build` 在 macos-14 构建 `darwin/universal`、在 windows-latest 构建 `windows/amd64`，安装包经 `upload-artifact` 传递。
- 打 tag `vX.Y.Z`：额外触发 `release`，用 `gh release create/upload` 发布 `build/darwin-universal/*.dmg` 与 `build/windows-amd64/*.exe`（含 NSIS setup）；tag 必须与 `mygo.config.ts`/`app.go` 的版本一致，否则 `test` 直接失败。
- 发布构建未带 `with_ffmpeg` 标签（运行时依赖系统 ffmpeg），也未带 `with_webp` 标签（WebP 选项在 UI 中不可用）；要出捆绑版需先备好对应资源（ffmpeg 放进 `resources/bin` / 本机配好 C 工具链）再调整构建参数（见上文构建变体）。
- 签名/公证未接入：macOS 目前是 ad-hoc 签名；需要 Developer ID 时配 `macos.signingIdentity` 并注入证书（后续单独做）。

## 验证清单（提交前 / 大改后）

1. `cd webview && bun run format` → `bun run lint` → `bun run typecheck`
2. `go vet ./...` && `go build ./...`
3. 改过 Go 服务：`bun run generate`，检查 `webview/src/mygo.ts` 的 diff
4. UI 或 IPC 改动：`bun run dev` 实际跑一次，确认窗口、深浅色、IPC 调用正常
5. MyGo 生命周期/并发改动：`go tool mygo vet .`

## 许可

- 原创代码以 **0BSD** 发布（`LICENSE`，版权行 `Copyright (C) 2026 sdttttt`）：零条件、无署名要求，也不含商标授权。
- 分发产物里的第三方组件**不是** 0BSD：Go 侧 mygo（MIT）、purego（Apache-2.0）、`golang.org/x/*`（含 `x/image`）与标准库（BSD-3-Clause）；前端 bundle 多数为 MIT，另有 `class-variance-authority`、`@internationalized/number`（Apache-2.0）。完整清单、版权行与许可全文在 `THIRD-PARTY-NOTICES.md`。
- `THIRD-PARTY-NOTICES.md` 通过 `mygo.config.ts` 的 `resources` 随应用复制到可执行文件旁（macOS 为 `Contents/Resources`）；改名或删掉该配置会让 MIT / Apache-2.0 的声明缺失。
- `webview/src/components/ui/` 与 `webview/src/lib/utils.ts` 是复制的 solid-ui / shadcn 代码，保留 MIT 与原作者版权，不能按 0BSD 重标；新增 registry 组件时同样处理。
- 新增任何会进入二进制或前端 bundle 的依赖后，先确认无 copyleft（GPL / AGPL / MPL / EPL），再把版权行补进 `THIRD-PARTY-NOTICES.md`；仅构建期依赖列在单独小节，不进产物。
- `-tags with_ffmpeg` 捆绑 ffmpeg 等于分发第三方二进制：发行包需附 ffmpeg 自身许可与源码获取方式，且只用 `--disable-gpl --disable-nonfree` 的 LGPL 构建。
- `-tags with_webp` 把 chai2010/webp（BSD-3）与内嵌 libwebp（BSD-3 + PATENTS）静态链接进二进制：默认构建不含、不链接，THIRD-PARTY-NOTICES.md 第 6 节即为该变体的声明保留；引入时已核查无 copyleft。

## Git

- 远程：`origin` = `git@github.com:sdttttt/m-tools.git`，主分支 `master`。
- Commit message 必须符合 Conventional Commits 规范，四部分齐全：action（type）、scope、subject、body，格式为 `<type>(<scope>): <subject>` + 空行 + `<body>`：
  - **type**（action）：`feat` / `fix` / `docs` / `refactor` / `test` / `chore` 等标准类型；
  - **scope**：改动所在模块或工具域名，如 `mp4gif`、`ffmpeg`、`webview`、`theme`；跨多个模块时可省略括号部分，写 `<type>: <subject>`；
  - **subject**：英文祈使句、小写开头、结尾不加句号，一句话说清改了什么；
  - **body**：必填，空一行后写，说明动机与关键实现点（为什么改、怎么改的），英文、每行不超过 ~72 字符。
- 示例：

  ```
  fix(ffmpeg): remove temp gif output when conversion fails

  ConvertGIF created its temp output before ffmpeg ran but returned no
  path on failure, leaving orphaned m-tools-*.gif files in the system
  temp dir. Track self-created temp outputs and remove them on every
  error path; caller-supplied outputs are still left untouched.
  ```

- 一次提交聚焦一件事；scope 即该件事的边界，不要在 body 里夹带无关改动。

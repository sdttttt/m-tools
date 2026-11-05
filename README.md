# M-Tools

macOS / Windows 桌面小工具箱。单个 Go 二进制 + 系统 webview，体积小、内存占用低。

## 功能

- **MP4 → GIF**：选文件、胶片时间轴裁剪片段、调节质量 / FPS / 宽度、预览片段、实时进度、保存并定位输出文件
- 浅色 / 深色 / 跟随系统主题

## 使用

从 [Releases](https://github.com/sdttttt/m-tools/releases) 下载安装包（macOS universal / Windows x64）。

MP4 → GIF 需要本机有 `ffmpeg` / `ffprobe`：放在 `PATH` 里，或用 `MTOOLS_FFMPEG_DIR` 指向它们所在目录。

## 开发

需要 Go 1.27+ 和 [Bun](https://bun.sh)。`ffmpeg` 同上，仅测试转换功能时需要。

```bash
bun install
cd webview && bun install && cd ..

bun run dev       # 开发窗口，前端热更新
bun run build     # 打包到 build/
bun run generate  # 改过 Go 服务后重新生成 webview/src/mygo.ts（生成物，勿手改）
```

技术栈：MyGO（webview 模式）+ SolidJS + TypeScript + Tailwind CSS，服务经 IPC 暴露给前端；构建配置见 `mygo.config.ts`。

## 许可

原创代码以 [0BSD](LICENSE) 发布；随产物分发的第三方组件及其许可见 [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md)。

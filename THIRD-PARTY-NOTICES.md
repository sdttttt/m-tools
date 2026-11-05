# Third-party notices

本仓库的原创代码以 Zero-Clause BSD（0BSD）发布，全文见 [`LICENSE`](LICENSE)。

分发的桌面应用是单个 Go 二进制，内部静态链接下表第 1 节的 Go 模块，并内嵌打包后的前端
bundle（第 2 节）与从 registry 复制的源码（第 3 节）。这些第三方组件各自保留原有许可，
0BSD 不适用于它们；本文件即为 MIT / Apache-2.0 / BSD-3-Clause 要求的声明保留。

新增任何会被编译进二进制或被打进前端 bundle 的依赖时，请同步更新本文件。

## 1. Go 二进制中静态链接的模块

| 模块                        | 版本     | 许可         | 版权声明                              |
| --------------------------- | -------- | ------------ | ------------------------------------- |
| `github.com/egoist/mygo`    | v0.7.4   | MIT          | Copyright (c) 2026 MyGo contributors  |
| `github.com/ebitengine/purego` | v0.11.3 | Apache-2.0 | 见下文 Apache License 2.0             |
| `golang.org/x/image`        | v0.47.0  | BSD-3-Clause | Copyright 2009 The Go Authors         |
| `golang.org/x/net`          | v0.62.0  | BSD-3-Clause | Copyright 2009 The Go Authors         |
| `golang.org/x/text`         | v0.43.0  | BSD-3-Clause | Copyright 2009 The Go Authors         |
| Go 标准库（含 vendored `golang.org/x/crypto`、`golang.org/x/sys`） | go1.27.3 | BSD-3-Clause | Copyright 2009 The Go Authors |

（模块图中存在但未被链接进本应用的依赖，例如 MyGo 原生 GoUI 渲染器与音频解码库，不在此列。
`-tags with_webp` 构建额外静态链接 `github.com/chai2010/webp` 及其内嵌的 libwebp C 源码，
见下文第 6 节；默认构建不包含。）

## 2. 内嵌前端 bundle（`webview/dist`，运行于系统 webview）

### MIT

| 包                                                                                         | 版本              | 版权声明                                         |
| ------------------------------------------------------------------------------------------ | ----------------- | ------------------------------------------------ |
| `solid-js`                                                                                 | 1.9.18            | Copyright (c) 2016-2025 Ryan Carniato             |
| `@kobalte/core`                                                                            | 0.13.14           | Copyright (c) 2024 jer3m01                       |
| `@kobalte/utils`                                                                           | 0.9.2             | Copyright (c) 2022 Fabien Marie-Louise           |
| `@corvu/utils`                                                                             | 0.4.2             | Copyright (c) 2023-2024 Jasmin Noetzli           |
| `@floating-ui/core`、`@floating-ui/dom`、`@floating-ui/utils`                              | 1.8.0 / 1.8.0 / 0.2.12 | Copyright (c) 2021-present Floating UI contributors |
| `solid-presence`                                                                           | 0.2.0             | Copyright (c) 2023-2025 Jasmin Noetzli           |
| `solid-prevent-scroll`                                                                      | 0.1.11            | Copyright (c) 2023-2025 Jasmin Noetzli           |
| `@solid-primitives/*`（event-listener、keyed、map、media、props、refs、resize-observer、rootless、static-store、trigger、utils） | 2.4.6 / 1.5.3 / 0.4.13 / 2.3.6 / 3.2.4 / 1.1.4 / 2.2.0 / 1.5.4 / 0.1.4 / 1.2.4 / 6.4.1 | Copyright (c) 2021 Solid Primitives Working Group |
| `clsx`                                                                                     | 2.1.1             | Copyright (c) Luke Edwards                       |
| `tailwind-merge`                                                                           | 3.7.0             | Copyright (c) 2021 Dany Castillo                 |
| `mygo-runtime`                                                                             | 0.7.4             | Copyright (c) 2026 MyGo contributors             |

### Apache-2.0

| 包                            | 版本   | 上游                                          |
| ----------------------------- | ------ | --------------------------------------------- |
| `class-variance-authority`    | 0.7.1  | https://cva.style （作者 Joe Bell）           |
| `@internationalized/number`   | 3.6.9  | Adobe react-spectrum（https://github.com/adobe/react-spectrum） |

## 3. 仓库内复制的第三方源码（保留 MIT 许可）

- `webview/src/components/ui/*.tsx` —— 从 [solid-ui](https://www.solid-ui.com) registry 复制的组件，
  MIT License，版权归 solid-ui / shadcn 各自作者。
- `webview/src/lib/utils.ts` —— `cn()` 组合工具，源自 shadcn/solid-ui 的同类实现，MIT License。

这些文件的版权与许可归原作者，不因本仓库采用 0BSD 而变更。

## 4. 仅构建期使用（不随产物分发）

| 包                                                                                       | 许可         |
| ---------------------------------------------------------------------------------------- | ------------ |
| `typescript`                                                                            | Apache-2.0   |
| `mygo-cli`（含各平台二进制）                                                             | MIT          |
| `vite`、`rolldown`、`vite-plugin-solid`、`oxlint`、`oxfmt`、`postcss`、`autoprefixer`、`tailwindcss`、`tailwindcss-animate`、`@babel/*`、`babel-preset-solid`、`babel-plugin-jsx-dom-expressions` | MIT          |
| `lightningcss`                                                                          | MPL-2.0      |
| `caniuse-lite`                                                                          | CC-BY-4.0    |
| `baseline-browser-mapping`                                                              | Apache-2.0   |
| `@types/node`                                                                           | MIT          |

`lightningcss`（MPL-2.0）仅作为 vite 的构建期 CSS 工具使用，未修改其源码；MPL 的文件级
互惠义务不及于本项目自身代码与其构建产出（`webview/dist`）。`caniuse-lite`（CC-BY-4.0）
是 browserslist 的构建期数据，不进入产物。

## 5. 运行时外部依赖（未捆绑分发）

- **ffmpeg / ffprobe**：默认构建（`bun run build`）不捆绑，运行时由用户系统的 PATH 或
  `$MTOOLS_FFMPEG_DIR` 提供。ffmpeg 自身按 LGPL-2.1-or-later 或 GPL 发布，取决于所用
  构建；本应用通过命令行子进程调用它，属独立程序。
  若启用 `-tags with_ffmpeg` 捆绑 ffmpeg，则发行包必须随附 ffmpeg 自身的许可文本与源码
  获取方式（建议只捆绑 `--disable-gpl --disable-nonfree` 的 LGPL 构建）。
- **Windows WebView2 Runtime / macOS WKWebView**：操作系统或 OEM 组件，由
  Microsoft / Apple 分发，本项目不再分发它们。

## 6. `-tags with_webp` 构建变体（默认构建不含）

启用 `-tags with_webp` 时，`webp/` 模块会静态链接以下组件以获得 WebP 编解码能力：

| 组件                        | 版本    | 许可         | 版权声明                              |
| --------------------------- | ------- | ------------ | ------------------------------------- |
| `github.com/chai2010/webp`  | v1.4.0  | BSD 3-Clause | Copyright (c) 2014 chaishushan@gmail.com |
| libwebp（chai2010/webp 内嵌 C 源码） | 1.4.0 | BSD 3-Clause | Copyright (c) Google Inc.       |

两者许可全文见下方 BSD 3-Clause（chai2010/webp 的 "Neither the name..." 条款指其作者；
libwebp 源码与 `PATENTS` 随该模块一同分发，源码获取方式：
<https://github.com/chai2010/webp> 与 <https://developers.google.com/speed/webp>）。
默认构建不含 WebP 编解码，应用表现为：WebP 目标格式不可选、WebP 源文件提示需
`with_webp` 构建。发布带此 tag 的变体时，本节即为其声明保留。

## 许可全文

### MIT License

```text
MIT License

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

各包的版权行见上文第 1、2、3 节的表格，逐条对应各自的 MIT 声明。

### Apache License 2.0

```text
                                 Apache License
                           Version 2.0, January 2004
                        http://www.apache.org/licenses/

   TERMS AND CONDITIONS FOR USE, REPRODUCTION, AND DISTRIBUTION

   1. Definitions.

      "License" shall mean the terms and conditions for use, reproduction,
      and distribution as defined by Sections 1 through 9 of this document.

      "Licensor" shall mean the copyright owner or entity authorized by
      the copyright owner that is granting the License.

      "Legal Entity" shall mean the union of the acting entity and all
      other entities that control, are controlled by, or are under common
      control with that entity. For the purposes of this definition,
      "control" means (i) the power, direct or indirect, to cause the
      direction or management of such entity, whether by contract or
      otherwise, or (ii) ownership of fifty percent (50%) or more of the
      outstanding shares, or (iii) beneficial ownership of such entity.

      "You" (or "Your") shall mean an individual or Legal Entity
      exercising permissions granted by this License.

      "Source" form shall mean the preferred form for making modifications,
      including but not limited to software source code, documentation
      source, and configuration files.

      "Object" form shall mean any form resulting from mechanical
      transformation or translation of a Source form, including but
      not limited to compiled object code, generated documentation,
      and conversions to other media types.

      "Work" shall mean the work of authorship, whether in Source or
      Object form, made available under the License, as indicated by a
      copyright notice that is included in or attached to the work
      (an example is provided in the Appendix below).

      "Derivative Works" shall mean any work, whether in Source or Object
      form, that is based on (or derived from) the Work and for which the
      editorial revisions, annotations, elaborations, or other modifications
      represent, as a whole, an original work of authorship. For the purposes
      of this License, Derivative Works shall not include works that remain
      separable from, or merely link (or bind by name) to the interfaces of,
      the Work and Derivative Works thereof.

      "Contribution" shall mean any work of authorship, including
      the original version of the Work and any modifications or additions
      to that Work or Derivative Works thereof, that is intentionally
      submitted to Licensor for inclusion in the Work by the copyright owner
      or by an individual or Legal Entity authorized to submit on behalf of
      the copyright owner. For the purposes of this definition, "submitted"
      means any form of electronic, verbal, or written communication sent
      to the Licensor or its representatives, including but not limited to
      communication on electronic mailing lists, source code control systems,
      and issue tracking systems that are managed by, or on behalf of, the
      Licensor for the purpose of discussing and improving the Work, but
      excluding communication that is conspicuously marked or otherwise
      designated in writing by the copyright owner as "Not a Contribution."

      "Contributor" shall mean Licensor and any individual or Legal Entity
      on behalf of whom a Contribution has been received by Licensor and
      subsequently incorporated within the Work.

   2. Grant of Copyright License. Subject to the terms and conditions of
      this License, each Contributor hereby grants to You a perpetual,
      worldwide, non-exclusive, no-charge, royalty-free, irrevocable
      copyright license to reproduce, prepare Derivative Works of,
      publicly display, publicly perform, sublicense, and distribute the
      Work and such Derivative Works in Source or Object form.

   3. Grant of Patent License. Subject to the terms and conditions of
      this License, each Contributor hereby grants to You a perpetual,
      worldwide, non-exclusive, no-charge, royalty-free, irrevocable
      (except as stated in this section) patent license to make, have made,
      use, offer to sell, sell, import, and otherwise transfer the Work,
      where such license applies only to those patent claims licensable
      by such Contributor that are necessarily infringed by their
      Contribution(s) alone or by combination of their Contribution(s)
      with the Work to which such Contribution(s) was submitted. If You
      institute patent litigation against any entity (including a
      cross-claim or counterclaim in a lawsuit) alleging that the Work
      or a Contribution incorporated within the Work constitutes direct
      or contributory patent infringement, then any patent licenses
      granted to You under this License for that Work shall terminate
      as of the date such litigation is filed.

   4. Redistribution. You may reproduce and distribute copies of the
      Work or Derivative Works thereof in any medium, with or without
      modifications, and in Source or Object form, provided that You
      meet the following conditions:

      (a) You must give any other recipients of the Work or
          Derivative Works a copy of this License; and

      (b) You must cause any modified files to carry prominent notices
          stating that You changed the files; and

      (c) You must retain, in the Source form of any Derivative Works
          that You distribute, all copyright, patent, trademark, and
          attribution notices from the Source form of the Work,
          excluding those notices that do not pertain to any part of
          the Derivative Works; and

      (d) If the Work includes a "NOTICE" text file as part of its
          distribution, then any Derivative Works that You distribute must
          include a readable copy of the attribution notices contained
          within such NOTICE file, excluding those notices that do not
          pertain to any part of the Derivative Works, in at least one
          of the following places: within a NOTICE text file distributed
          as part of the Derivative Works; within the Source form or
          documentation, if provided along with the Derivative Works; or,
          within a display generated by the Derivative Works, if and
          wherever such third-party notices normally appear. The contents
          of the NOTICE file are for informational purposes only and
          do not modify the License. You may add Your own attribution
          notices within Derivative Works that You distribute, alongside
          or as an addendum to the NOTICE text from the Work, provided
          that such additional attribution notices cannot be construed
          as modifying the License.

      You may add Your own copyright statement to Your modifications and
      may provide additional or different license terms and conditions
      for use, reproduction, or distribution of Your modifications, or
      for any such Derivative Works as a whole, provided Your use,
      reproduction, and distribution of the Work otherwise complies with
      the conditions stated in this License.

   5. Submission of Contributions. Unless You explicitly state otherwise,
      any Contribution intentionally submitted for inclusion in the Work
      by You to the Licensor shall be under the terms and conditions of
      this License, without any additional terms or conditions.
      Notwithstanding the above, nothing herein shall supersede or modify
      the terms of any separate license agreement you may have executed
      with Licensor regarding such Contributions.

   6. Trademarks. This License does not grant permission to use the trade
      names, trademarks, service marks, or product names of the Licensor,
      except as required for reasonable and customary use in describing the
      origin of the Work and reproducing the content of the NOTICE file.

   7. Disclaimer of Warranty. Unless required by applicable law or
      agreed to in writing, Licensor provides the Work (and each
      Contributor provides its Contributions) on an "AS IS" BASIS,
      WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
      implied, including, without limitation, any warranties or conditions
      of TITLE, NON-INFRINGEMENT, MERCHANTABILITY, or FITNESS FOR A
      PARTICULAR PURPOSE. You are solely responsible for determining the
      appropriateness of using or redistributing the Work and assume any
      risks associated with Your exercise of permissions under this License.

   8. Limitation of Liability. In no event and under no legal theory,
      whether in tort (including negligence), contract, or otherwise,
      unless required by applicable law (such as deliberate and grossly
      negligent acts) or agreed to in writing, shall any Contributor be
      liable to You for damages, including any direct, indirect, special,
      incidental, or consequential damages of any character arising as a
      result of this License or out of the use or inability to use the
      Work (including but not limited to damages for loss of goodwill,
      work stoppage, computer failure or malfunction, or any and all
      other commercial damages or losses), even if such Contributor
      has been advised of the possibility of such damages.

   9. Accepting Warranty or Additional Liability. While redistributing
      the Work or Derivative Works thereof, You may choose to offer,
      and charge a fee for, acceptance of support, warranty, indemnity,
      or other liability obligations and/or rights consistent with this
      License. However, in accepting such obligations, You may act only
      on Your own behalf and on Your sole responsibility, not on behalf
      of any other Contributor, and only if You agree to indemnify,
      defend, and hold each Contributor harmless for any liability
      incurred by, or claims asserted against, such Contributor by reason
      of your accepting any such warranty or additional liability.

   END OF TERMS AND CONDITIONS
```

许可证原文亦可从 <http://www.apache.org/licenses/LICENSE-2.0> 获取。

### BSD 3-Clause License

```text
Copyright 2009 The Go Authors.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google LLC nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

Google 另附专利授权声明（Go 项目 `PATENTS` 文件），随 `golang.org/x/*` 一同提供。

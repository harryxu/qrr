# qrr：交互式快捷命令工具实现计划

> 用途：将当前讨论交接给其他 AI agent，继续完善设计或开始实现。
> Status: Initial CLI implementation completed. See README.md for usage and docs/schema.md for the implemented schema; unresolved design items remain below.
> 日期：2026-10-03

## 1. 需求与目标

开发一个正式命名为 **qrr（Quick Recipe Runner）** 的终端快捷命令工具，用于收集常用命令，并通过交互输入简化运行。

典型使用方式：

```bash
qrr myytdl
```

程序依次询问 YouTube 视频地址、分辨率、字幕语言，然后运行下载命令。目标行为是下载视频并嵌入可用字幕；请求的某种字幕不存在时跳过，不阻止视频下载。

核心要求：

- 使用 Go 实现，分发一个可执行文件，无需 Python/Node 运行时。
- 优先支持 Linux、macOS。
- 随时添加、修改快捷命令；大部分新增命令只需添加 YAML，无需修改源码或重新编译。
- 配置格式易读、可验证，方便用户和 AI agent 编写。
- 同时支持交互使用和通过 CLI 参数直接调用。
- 子进程保留正常的进度输出和终端交互能力。
- 产品形态为交互式 CLI：通过逐步提示完成输入、单选、多选和确认，不开发全屏 TUI。

“无语言运行时依赖”指 qrr 自身；快捷命令调用的 `yt-dlp`、`ffmpeg`、`docker` 等仍须另行安装。

## 2. 已确定的方向与建议边界

| 项目 | 方向 | 状态 |
| --- | --- | --- |
| 项目名称 | qrr（Quick Recipe Runner） | 用户已确认 |
| 实现语言 | Go | 用户明确要求 |
| 快捷命令定义 | 声明式 YAML | 当前推荐方案 |
| 执行格式 | argv 数组，直接启动子进程 | 当前推荐方案 |
| 交互方式 | 交互式 CLI，使用输入、单选、多选等逐步提示 | 用户已确认 |
| 全屏 TUI | 不纳入开发范围 | 用户明确不需要 |
| 复杂业务逻辑 | 少量内置 handler；外部可执行扩展后续评估 | 尚未定稿 |

第一版优先做好配置、CLI 交互、执行和验证。避免同时构建复杂工作流语言和插件市场。

## 3. 技术栈

| 职责 | 建议实现 |
| --- | --- |
| CLI 命令、帮助、补全 | [Cobra](https://github.com/spf13/cobra) |
| 输入、单选、多选、确认 | [Charm Huh](https://github.com/charmbracelet/huh) |
| 终端样式 | [Lip Gloss](https://github.com/charmbracelet/lipgloss)，按需要引入 |
| YAML 解析 | YAML 库；讨论中的候选为 [go.yaml.in/yaml/v3](https://pkg.go.dev/go.yaml.in/yaml/v3) |
| 参数模板 | Go 标准库 [text/template](https://pkg.go.dev/text/template) |
| 外部命令执行 | Go 标准库 [os/exec](https://pkg.go.dev/os/exec) |

Dependencies are pinned in go.mod and go.sum. The current implementation requires Go 1.26 or newer. Huh uses Bubble Tea internally for inline prompts; qrr does not provide a full-screen interface.

## 4. 核心架构

执行流程：解析 CLI → 加载命令配置 → 合并参数 → 校验 → 交互补齐 → 渲染 argv → 执行子进程。

模块职责：

- **CLI**：管理命令、动态快捷命令、参数解析和帮助输出。
- **Loader / Schema**：发现 YAML，检查格式、版本、命名和参数类型。
- **Prompt**：对未通过 CLI 指定的参数展示交互界面。
- **Template**：逐个渲染 argv 元素；缺失引用立即报错。
- **Runner**：启动进程，连接标准输入输出，处理取消和退出码。
- **Handler**：为极少数需要业务逻辑的命令提供扩展入口。

Implemented project layout:

```text
qrr/
├── cmd/qrr/main.go
├── internal/
│   ├── cli/
│   ├── recipe/
│   ├── prompt/
│   └── runner/
├── examples/commands/
├── docs/schema.md
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── .gitignore
```

建议配置路径：`$XDG_CONFIG_HOME/qrr`，未设置时使用 `~/.config/qrr`；这是为 Linux/macOS 统一使用提出的约定，需要明确记录。增加 `--config-dir` 便于测试和自定义。

```text
~/.config/qrr/
├── config.yaml
└── commands/
    ├── myytdl.yaml
    ├── ssh-home.yaml
    └── git-clone.yaml
```

首版只加载明确的命令目录，不递归发现任意脚本。复杂命令目录布局可在扩展协议确定后加入。

## 5. CLI 接口

| 命令 | 行为 |
| --- | --- |
| `qrr` | 在当前终端中列出快捷命令，支持直接输入关键词过滤、上下箭头选择，回车后逐步询问参数 |
| `qrr <name>` | 运行指定快捷命令 |
| `qrr run <name>` | 显式运行，作为稳定调用入口 |
| `qrr list` | 列出名称和说明 |
| `qrr show <name>` | 显示参数和配置 |
| `qrr add <name>` | 创建最小配置模板，可调用编辑器 |
| `qrr edit <name>` | 用 `$EDITOR` 打开配置 |
| `qrr remove <name>` | 确认后删除配置；非交互模式需明确确认参数 |
| `qrr validate [name]` | 校验一个或全部配置 |

### 无参数启动：命令选择器

在交互式终端中直接输入 `qrr` 并回车，进入命令选择器：

- 加载配置目录中的有效自定义命令，显示名称和说明；首版按名称排序，不额外维护使用频率或收藏记录。
- 搜索输入默认激活，直接输入关键词即可实时过滤，无需先按 `/`；匹配命令名称和说明，忽略英文大小写。
- 使用 `↑` / `↓` 在过滤结果中移动，列表超过可见高度时滚动。
- 回车确认当前命令，然后进入与 `qrr <name>` 相同的参数提示和执行流程。
- Backspace 编辑关键词，清空后恢复完整列表；无匹配结果时提示 `No matching commands`，不能确认执行。
- Ctrl+C 取消并退出，恢复终端状态；命令执行结束后返回 Shell，首版不自动重新打开选择器。
- 无已配置命令时提示 `No commands configured. Use qrr add <name> to create one.`。
- 非 TTY 或指定 `--non-interactive` 时不打开选择器；未指定命令则显示用法并返回参数错误。

交互在当前终端中占用有限行数，不启用全屏界面。优先使用 [Huh Select](https://github.com/charmbracelet/huh/blob/main/field_select.go) 的列表过滤能力，将过滤默认激活；开发时验证锁定版本的键位和确认行为是否符合上述约定，必要时做局部适配。

参数调用示例：

```bash
qrr myytdl
qrr myytdl --url 'https://www.youtube.com/watch?v=xxx' --resolution 1440
qrr myytdl --url 'https://www.youtube.com/watch?v=xxx' --resolution 1440 --subtitles zh-Hans,ja,en --non-interactive
qrr myytdl --dry-run
```

建议规则：

- CLI 显式值优先于配置默认值。
- 交互模式：缺少的参数继续询问，配置默认值作为预选项。
- `--non-interactive`：未指定的参数采用默认值；缺少必填值则失败，不能等待输入。
- 非 TTY 环境不自动启动交互。
- `--dry-run` 完成参数解析和渲染，但不运行目标命令；内置 handler 也不得偷偷联网或启动准备进程。
- 管理命令名称和全局 flag 为保留项；配置冲突时明确报错。
- 空多选是有效的显式输入，应与“未提供参数”区分；具体 CLI 表达方式在实现前确定。
- 提供命令级帮助，例如 `qrr myytdl --help`，显示 YAML 定义的参数。

Cobra 的动态命令和动态 flag 注册需要作为首个技术验证点，确保支持上述调用方式。

## 6. YAML schema 草案

schema 是项目最重要的扩展接口。建议从第一版加入 `version: 1`，未知字段、重复键、重复参数名和无效类型均报错。

以下为**声明式命令示例**，展示配置结构，不代表已实现完整字幕容错：

```yaml
version: 1
name: myytdl
description: 下载 YouTube 视频并嵌入字幕
type: command

params:
  - name: url
    type: input
    prompt: 视频地址
    required: true

  - name: resolution
    type: select
    prompt: 最高分辨率
    default: "1440"
    options:
      - { label: "4K / 2160p", value: "2160" }
      - { label: "1440p", value: "1440" }
      - { label: "1080p", value: "1080" }
      - { label: "720p", value: "720" }

  - name: subtitles
    type: multiselect
    prompt: 字幕语言
    default: [zh-Hans, ja, en]
    options:
      - { label: 简体中文, value: zh-Hans }
      - { label: 繁體中文, value: zh-Hant }
      - { label: 日本語, value: ja }
      - { label: English, value: en }

command:
  - yt-dlp
  - -f
  - 'bestvideo[height<={{ .resolution }}]+bestaudio/best[height<={{ .resolution }}]'

optional_args:
  - when: subtitles
    args:
      - --write-subs
      - --write-auto-subs
      - --sub-langs
      - '{{ join .subtitles "," }}'
      - --embed-subs

args_tail:
  - --
  - '{{ .url }}'
```

`optional_args`、`args_tail` 是为了明确条件参数及最终 argv 顺序提出的草案，需在实现前定稿：基础 `command` → 按顺序追加满足条件的 `optional_args` → `args_tail`。

最低参数类型：`input`、`select`、`multiselect`、`confirm`。字符串型选项的值和默认值保持字符串，布尔型参数保存为 bool，多选保存为字符串数组。

建议模板函数只暴露少量确定性操作，如 `join`。`when` 第一版只判断对应参数是否非空/为真，不执行任意表达式。模板使用 `missingkey=error`，禁止模板读取文件或执行命令。

配置可选声明所需外部程序，以便执行前提示缺少依赖；依赖检查与纯配置校验分开。

## 7. 子进程与错误行为

使用 argv 数组执行：

```go
cmd := exec.Command(args[0], args[1:]...)
cmd.Stdin = os.Stdin
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
err := cmd.Run()
```

- 每个模板渲染结果始终对应一个 argv 元素，不再次拆词，也不将多选内容自动展开为多个参数。
- 不默认经由 `sh -c` 执行。管道、重定向和复杂脚本应留到显式扩展协议设计时处理。
- 命令参数中的空格、引号、`$()` 等按普通文本传递。
- argv 能避免 shell 注入，但不能阻止目标程序把用户输入当作选项；支持的命令应在用户位置参数前使用 `--`。
- dry-run 展示可读命令，同时可提供结构化 argv 输出便于调试；展示文本不作为执行依据。
- 默认保留子进程退出码；配置/参数错误使用明确的 qrr 错误码。
- Ctrl+C 正确取消提示或子进程，并恢复终端状态。
- 不默认忽略非零退出码。若以后增加 `allowed_exit_codes`，它只用于已知语义的命令，不能作为通用容错方案。

## 8. YouTube 下载示例与字幕容错

业务目标：选择最高分辨率，优先嵌入中文、日语、英语的可用字幕。自动生成和自动翻译字幕在来源可用时支持。

需要区分：

1. **语言不存在**：跳过该语言，继续下载视频。
2. **字幕存在但下载失败**：例如 429、超时、认证失败；单独提示原因，不伪装成“语言不存在”。
3. **视频下载失败**：明确失败，不因容错配置返回成功。

先获取字幕列表再过滤只能处理第 1 类，不能保证第 2 类一定成功。忽略退出码也不能恢复已经被中断的下载流程。

建议分阶段落地：

- 通用核心完成后，先核验当前 yt-dlp 对不存在语言的真实行为；如果本身能够跳过，不额外查询列表。
- 如果要求“字幕失败也必须保留成功的视频”，考虑分离视频下载、字幕下载、最终嵌入三个阶段。嵌入失败保留原视频，汇总字幕失败信息。
- 无字幕时不传字幕参数；分辨率参数表示“最高不超过所选值”，不是强制源视频必须具有该分辨率。
- 多字幕封装优先评估 MKV；最终容器、ffmpeg 嵌入方式及输出文件管理需验证后确定。
- cookies、等待、重试可作为命令参数扩展，但不保证能绕过限流，也不默认保存登录数据。

上述流程属于具体命令逻辑，qrr 核心不应耦合 YouTube API。若纯声明配置无法合理表达，可为这个示例加入少量内置 handler：

```yaml
version: 1
name: myytdl
description: 下载 YouTube 视频并尽力嵌入所选字幕
type: handler
handler: builtin:ytdlp
# params 沿用上方示例
```

`type: command` 与 `type: handler` 的执行字段互斥。handler 是候选设计，不是新增任何快捷命令的必经路径。

## 9. 扩展策略

分三层处理复杂度：

1. **纯声明命令**：参数、默认值、argv 模板，覆盖主要场景。
2. **有限条件参数**：按参数值追加可选 argv，控制规模。
3. **少量复杂扩展**：内置 Go handler；后续可评估外部可执行程序协议，使复杂扩展也不必重新编译核心。

暂不使用 Go 动态 plugin 机制。内置 handler 修改需要重新编译，应保持数量少。外部脚本可能重新引入运行时依赖；外部 Go 可执行程序可保持无语言运行时需求，但分发不再只有一个文件。

## 10. 校验和 AI agent 协作规范

`qrr validate` 至少检查：

- YAML 解析、schema 版本、未知字段、重复键。
- 命令名称与文件名约定、重复名称、保留名称冲突。
- 参数类型、默认值类型、选项值和必填规则。
- 空 command、模板语法、无效参数引用、条件参数引用。
- handler 名称及执行字段互斥约束。

纯校验不应执行命令或联网。可单独增加 `--check-deps` 检查 PATH 中的外部程序。

建议仓库 `AGENTS.md` 包含以下规范：

```markdown
# 添加 qrr 快捷命令

- 优先新增 YAML，不修改核心。
- 遵循 docs/schema.md 和已有 examples。
- command 使用 argv 数组，输入不可经由 shell 拼接执行。
- 参数明确声明类型、提示、默认值和必填规则。
- 新命令不得覆盖已有名称或 CLI 保留名称。
- 新增配置后运行 qrr validate，并通过 dry-run 检查实际 argv。
- 只有无法合理用现有 schema 表达的逻辑才讨论 handler 或扩展协议。
- 不将密码、cookies、token 写入共享示例配置。
```

## 11. 开发顺序与验收

### 阶段一：最小执行链路

实现加载一个 YAML、动态 CLI 参数、Huh 提示、模板渲染和子进程执行。先用输出参数的本地测试程序验证空格、Unicode、特殊字符和参数边界。

验收：无参数 `qrr` 能显示命令选择器，支持直接输入关键词过滤、上下箭头选择和回车确认；`qrr <name>` 能交互运行；CLI 可补齐部分参数；完整参数配合非交互模式无需提示。

### 阶段二：命令管理与可靠性

加入 list/show/add/edit/remove、validate、dry-run、帮助、配置目录选择、TTY 判断、取消和退出码传播。

验收：新增 YAML 后无需编译即可发现并运行；无效配置定位到文件和字段；错误配置不导致整个工具崩溃。

### 阶段三：YouTube 示例

验证 yt-dlp 当前参数，完成分辨率和字幕选择；根据实际行为决定是否需要 handler 和分阶段下载。

验收：无所选语言时仍下载视频；字幕网络失败与不存在语言明确区分；视频失败返回失败；字幕嵌入可被播放器识别。网络相关验收采用人工测试，不依赖固定视频永久可用。

### 阶段四：发布与体验完善

构建 Linux/macOS 的常用架构二进制，完善安装说明、Shell Tab 补全和示例配置，并验证命令选择器在不同终端尺寸和较多命令时的滚动与过滤体验。

测试重点：参数合并、模板缺失引用、argv 边界、条件参数排序、配置校验、退出码和取消。通用测试不调用真实 YouTube、不使用真实登录 cookies。

## 12. 继续讨论时需决定的事项

- YAML 最终字段名，以及 `optional_args` / `args_tail` 是否采用。
- 确认型参数、显式空多选和非交互默认值的 CLI 表达。
- 是否需要每次执行确认，或仅由具体配置声明；不要默认所有命令都多一步确认。
- `$EDITOR` 包含多个参数时的解析，以及未设置编辑器时的回退。
- 首版 myytdl 是否直接采用普通配置，或增加 handler。
- 视频容器、输出目录、文件命名和字幕失败后的状态报告。
- 是否引入外部可执行扩展协议，以及配置仓库的 Git 同步方式。
- 库版本、最低 Go 版本和安装方式。

## 13. 可直接交给 AI agent 的说明

请基于本文继续讨论或实现 qrr。用户明确要求使用 Go，使 qrr 本身无需 Python/Node 运行时；新增普通快捷命令应尽量只添加 YAML。产品形态已确定为交互式 CLI，不开发全屏 TUI；交互采用当前终端中的逐步提示。

首先确认 schema 和 CLI 参数规则，然后按阶段一至三逐步实现。重点验证动态命令解析、argv 安全传递、非交互模式和错误行为。不要将所有快捷命令写死为 Go 函数，也不要提前构建复杂插件系统。

myytdl 是首个实际示例：输入 URL、选择最高分辨率和字幕语言，然后下载并嵌入可用字幕。必须区分语言不存在、字幕下载失败和视频下载失败；不能仅靠忽略退出码宣称实现了容错。

Implementation now includes YAML command recipes, an inline searchable selector, parameter prompts, management commands, validation, dry-run, and dynamic shell completion. Unit tests and terminal smoke tests cover the core CLI. The YouTube example remains a declarative recipe; live subtitle failure handling and handler extensions are not implemented or verified. See README.md and docs/schema.md for current behavior.

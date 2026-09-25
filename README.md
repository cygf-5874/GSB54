# ignorepath

一个 ignore 规则匹配引擎（Go，仅标准库）。

给它一组 ignore 模式和一条相对路径，它回答「这条路径该不该被忽略」。
语义对齐常见的 `.gitignore` 风格规则，但**只做字符串匹配**：
不读文件系统，也不读任何 ignore 文件本身。

```go
m, err := ignorepath.NewMatcher([]string{
    "*.log",
    "!important.log",
    "build/",
})
if err != nil {
    return err
}
ignored, err := m.Match("build/out/app.js", false)
```

## 怎么跑

```bash
go test ./...               # 既有用例（10 个测试函数）
go run ./ignorecheck        # 验收场景（固定件，10 个）
go run ./ignorecheck -list
go run ./ignorecheck -only negate
```

## API

| 函数 | 作用 |
| --- | --- |
| `NewMatcher(patterns []string) (*Matcher, error)` | 把模式列表编译成一个匹配器 |
| `(*Matcher).Match(relPath string, isDir bool) (bool, error)` | 判断路径是否被忽略 |
| `(*Matcher).Patterns() []string` | 返回生效的模式文本（原样、按输入顺序） |
| `(*Matcher).IsEmpty() bool` | 是否没有任何生效的模式 |
| `CheckPath(relPath string) error` | 校验一条相对路径 |
| `MatchPath(patterns []string, relPath string, isDir bool) (bool, error)` | 一次性编译 + 匹配 |

## 对外契约

以下 9 条同时成立，一条都不能少。

### 1. 模式语法

`NewMatcher` 的每个元素是一行模式，逐行按下面的规则解析（`\` 是转义符）：

- **注释**：以 `#` 开头的行是注释，整行忽略。要匹配字面 `#`，写成 `\#`。
- **行尾空白**：行尾的空格与制表符被裁剪。被反斜杠转义的空白保留 —— `\ ` 是字面空格。
- **空行**：裁剪之后为空的行忽略。
- **反选**：前导 `!` 表示反选（否定）。要匹配字面 `!`，写成 `\!`。
- **目录限定**：模式尾部是 `/` 时，它只匹配目录。
- **锚定**：模式前导是 `/` 时，它锚定到仓库根。
- **通配**：`*` 匹配任意长（可以为空）的一段、不跨 `/`；`?` 匹配单个非 `/` 字符；
  `[a-z]` / `[!a-z]` 是字符类（`!` 或 `^` 表示取反）；`\x` 表示字面字符 `x`
  （可用来转义 `#`、`!`、` `、`*`、`?`、`[`、`\` 等）。
- **`**`**：只有当一个模式**整段**就是 `**` 时它才特殊 —— 它匹配零个或多个路径段。
  其它位置的连续 `*` 按普通 `*` 处理。

模式去掉 `!`、尾部 `/`、前导 `/` 之后按 `/` 切段，路径也按 `/` 切段：

- **锚定模式**（前导有 `/`，或者去掉尾部 `/` 之后中间含 `/`）：模式的段序列必须与路径的
  **全部**段整段匹配。
- **非锚定模式**（不含 `/`，只有一段）：只要与路径的**最后一段**匹配即可，
  于是它天然能在任意目录层级命中。

### 2. `NewMatcher` 与非法模式

`NewMatcher(patterns []string) (*Matcher, error)` 编译模式列表。
任一模式语法非法（例如字符类没有闭合的 `]`）→ 返回 `(nil, ErrBadPattern)`。
没有非法模式时总返回 `(非 nil, nil)`，即使模式列表为空。

### 3. `Match` 与路径校验

`Match(relPath string, isDir bool) (bool, error)` 判断一条路径是否被忽略。

`relPath` 必须是 `/` 分隔的相对路径。下面这些 → `ErrBadPath`（返回 `false, err`）：

- 空字符串；
- 以 `/` 开头的绝对路径；
- 含 `..` 段的路径。

`CheckPath(relPath string) error` 用**同一套判据**校验路径并返回同样错误。

### 4. 最后匹配胜出

一条路径可能同时命中多条模式。以**最后**命中的那条为准：

- 最后命中的是反选模式 → 该路径**不被**忽略（`false`）；
- 最后命中的是普通模式 → 该路径**被**忽略（`true`）。

一条模式都没命中 → 不被忽略（`false`）。
注意判序按模式的**输入顺序**，所以把两条模式交换位置可能改变结果。

### 5. 父目录传播

若一条路径的某个祖先**目录**被忽略（并且该目录自身的忽略判定没有被后续模式反选掉），
那么这条路径**一律**被忽略 —— 无论有没有模式命中这条路径本身，第 4 条对它都不再适用。

### 6. 空模式列表

没有任何生效模式时，任何路径都不被忽略（`false`）。

### 7. 只看路径字符串

匹配只看传进来的字符串：不读文件系统，不读 `.gitignore` 文件本身，
`Match` 与 `CheckPath` 不产生任何 I/O。

### 8. 大小写与反斜杠

匹配大小写敏感。反斜杠只在转义位置有特殊含义（见第 1 条）。

### 9. 无状态、可复现

同一个 `*Matcher` 可以被重复使用。相同输入重复调用结果一致，
不依赖时间、随机源或任何外部状态。

## 验收

`ignorecheck/` 是固定验收程序，**不要修改**。10 个场景分四组：

| 组 | 场景 | 覆盖 |
| --- | --- | --- |
| `syntax` | `SYN1_doublestar_across_dirs` | `**` 跨目录（契约 1） |
| `syntax` | `SYN2_character_class_and_question` | 字符类与 `?`（契约 1） |
| `syntax` | `SYN3_escapes_and_trailing_space` | 转义、注释、行尾空白（契约 1） |
| `negate` | `NEG1_negation_basic` | 反选（契约 4） |
| `negate` | `NEG2_last_match_wins` | 最后匹配胜出与顺序敏感（契约 4） |
| `negate` | `NEG3_dir_only` | 目录限定（契约 1） |
| `ancestor` | `ANC1_parent_dir_ignored` | 父目录被忽略则子树全忽略（契约 5） |
| `ancestor` | `ANC2_negation_cannot_reinclude` | 父目录被排除时反选救不回来（契约 4、5） |
| `robust` | `ROB1_bad_path` | `ErrBadPath`（契约 3） |
| `robust` | `ROB2_bad_pattern` | `ErrBadPattern`（契约 2） |

`-only syntax|negate|ancestor|robust` 可以只跑一组（`--only` 等价）。
每个场景在独立子进程里执行，挂死不会遮蔽其余场景；子进程有 25 秒看门狗，
超时会转储全部 goroutine 栈并以退出码 3 结束。

## 目录

```
.
├── go.mod                 module ignorepath
├── errors.go              ErrNotImplemented / ErrBadPattern / ErrBadPath
├── ignorepath.go          Matcher 与六个待实现的函数
├── ignorepath_test.go     既有用例（10 个测试函数）
├── ignorecheck/main.go    固定验收程序（勿改）
└── README.md
```

> 现状：`ignorepath.go` 里的 6 个函数全是空壳 —— `Patterns` 返回 `nil`、
> `IsEmpty` 返回 `false`，其余四个直接返回 `ErrNotImplemented`。
> 既有用例当前全部失败（10 个测试函数全红，但能编译）。

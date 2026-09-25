// Package ignorepath 是一个 ignore 规则匹配引擎。
//
// 它拿一组 ignore 模式和一条相对路径，判断这条路径该不该被忽略。
// 语义对齐常见的 ".gitignore" 风格规则，但**只做字符串匹配**：
// 不读文件系统，也不读任何 ignore 文件本身。
//
// 详细语义见 README.md 的「对外契约」一节（9 条）。
//
// 当前 ignorepath.go 里 6 个函数都是空壳，需要把它们实现出来。
package ignorepath

// Matcher 是一组编译好的 ignore 模式。
//
// 同一个 *Matcher 可以被重复使用；Match 不修改它（无状态）。
type Matcher struct {
}

// NewMatcher 把模式列表编译成一个 Matcher。
//
// 每个元素是一行模式（注释行、空行会被丢掉，规则见 README「对外契约」第 1 条）。
// 任一模式语法非法时返回 (nil, ErrBadPattern)。
func NewMatcher(patterns []string) (*Matcher, error) {
	return nil, ErrNotImplemented
}

// Match 判断 relPath 是否被忽略。
//
// relPath 必须是以 "/" 分隔的相对路径：空字符串、以 "/" 开头的绝对路径、
// 含 ".." 段的路径都返回 ErrBadPath。isDir 表示这条路径是不是目录。
//
// 出错时返回 (false, err)。
func (m *Matcher) Match(relPath string, isDir bool) (bool, error) {
	return false, ErrNotImplemented
}

// Patterns 按输入顺序返回所有生效模式的文本（注释行与空行已丢弃；
// 每行已按第 1 条裁剪过行尾空白）。"!"、尾部 "/"、前导 "/"、转义都保持原样。
func (m *Matcher) Patterns() []string {
	return nil
}

// IsEmpty 报告这个 Matcher 有没有任何生效的模式。
func (m *Matcher) IsEmpty() bool {
	return false
}

// CheckPath 校验一个相对路径，判据与 Match 相同：
// 空字符串、以 "/" 开头的绝对路径、含 ".." 段的路径都返回 ErrBadPath，
// 其余返回 nil。
func CheckPath(relPath string) error {
	return ErrNotImplemented
}

// MatchPath 是 NewMatcher 加 Match 的一次性写法：
// 先用 patterns 编译，再判断 relPath 是否被忽略。
// 出错时返回 (false, err)。
func MatchPath(patterns []string, relPath string, isDir bool) (bool, error) {
	return false, ErrNotImplemented
}

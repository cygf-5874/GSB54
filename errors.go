package ignorepath

import "errors"

var (
	// ErrNotImplemented 表示该函数还是空壳。
	ErrNotImplemented = errors.New("ignorepath: not implemented")

	// ErrBadPattern 表示某个模式语法非法（比如没有闭合的字符类）。
	// 出现它时 NewMatcher 返回 (nil, ErrBadPattern)。
	ErrBadPattern = errors.New("ignorepath: bad pattern")

	// ErrBadPath 表示传入的路径不是合法的、以 "/" 分隔的相对路径。
	ErrBadPath = errors.New("ignorepath: bad path")
)

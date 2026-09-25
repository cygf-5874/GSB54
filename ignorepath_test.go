package ignorepath_test

import (
	"errors"
	"testing"

	"ignorepath"
)

func mustMatcher(t *testing.T, patterns []string) *ignorepath.Matcher {
	t.Helper()
	m, err := ignorepath.NewMatcher(patterns)
	if err != nil {
		t.Fatalf("NewMatcher(%q) 出错: %v", patterns, err)
	}
	return m
}

func mustMatch(t *testing.T, m *ignorepath.Matcher, path string, isDir, want bool) {
	t.Helper()
	got, err := m.Match(path, isDir)
	if err != nil {
		t.Fatalf("Match(%q, isDir=%v) 出错: %v", path, isDir, err)
	}
	if got != want {
		t.Fatalf("Match(%q, isDir=%v) = %v，期望 %v", path, isDir, got, want)
	}
}

func TestEmptyPatternListMatchesNothing(t *testing.T) {
	m := mustMatcher(t, nil)
	if !m.IsEmpty() {
		t.Fatalf("空模式列表的 IsEmpty() = false")
	}
	if got := m.Patterns(); len(got) != 0 {
		t.Fatalf("空模式列表的 Patterns() = %q", got)
	}
	for _, p := range []string{"a", "a/b", "a/b/c.txt"} {
		mustMatch(t, m, p, false, false)
	}

	// 只有注释行与空行时，等价于空模式列表
	m2 := mustMatcher(t, []string{"# 注释", "   ", ""})
	if !m2.IsEmpty() {
		t.Fatalf("只含注释/空行时 IsEmpty() = false")
	}
	mustMatch(t, m2, "a/b/c.txt", false, false)
}

func TestNonAnchoredGlobMatchesAnyLevel(t *testing.T) {
	m := mustMatcher(t, []string{"*.log"})

	mustMatch(t, m, "a.log", false, true)
	mustMatch(t, m, "dir/a.log", false, true)
	mustMatch(t, m, "dir/sub/a.log", false, true)
	mustMatch(t, m, "a.log.bak", false, false)
	mustMatch(t, m, "dir/a.txt", false, false)

	if got := m.Patterns(); len(got) != 1 || got[0] != "*.log" {
		t.Fatalf("Patterns() = %q，期望 [\"*.log\"]", got)
	}
}

func TestDoublestarAcrossDirs(t *testing.T) {
	m := mustMatcher(t, []string{"a/**/b.txt", "**/deep.log"})

	mustMatch(t, m, "a/b.txt", false, true)
	mustMatch(t, m, "a/x/b.txt", false, true)
	mustMatch(t, m, "a/x/y/b.txt", false, true)
	mustMatch(t, m, "a/x/c.txt", false, false)

	mustMatch(t, m, "deep.log", false, true)
	mustMatch(t, m, "p/q/deep.log", false, true)
	mustMatch(t, m, "p/deep.log.bak", false, false)
}

func TestCharacterClassAndQuestion(t *testing.T) {
	m := mustMatcher(t, []string{"file[0-9].txt", "data[!0-9].bin", "q?.log"})

	mustMatch(t, m, "file7.txt", false, true)
	mustMatch(t, m, "dir/file3.txt", false, true)
	mustMatch(t, m, "filex.txt", false, false)
	mustMatch(t, m, "file12.txt", false, false)

	mustMatch(t, m, "dataA.bin", false, true)
	mustMatch(t, m, "data5.bin", false, false)

	mustMatch(t, m, "q1.log", false, true)
	mustMatch(t, m, "qq.log", false, true)
	mustMatch(t, m, "q.log", false, false)
	mustMatch(t, m, "q/1.log", false, false)
}

func TestEscapesCommentAndTrailingSpace(t *testing.T) {
	patterns := []string{
		"\\#notes.txt", // 字面 "#"，不是注释
		"\\!bang.txt",  // 字面 "!"，不是反选
		"trailing\\ ",  // 被转义的行尾空格要保留
		"a\\*b.txt",    // 字面 "*"
		"trim.log ",    // 未转义的行尾空格被裁剪
		"# 真注释",
	}
	m := mustMatcher(t, patterns)

	mustMatch(t, m, "#notes.txt", false, true)
	mustMatch(t, m, "!bang.txt", false, true)
	mustMatch(t, m, "bang.txt", false, false)
	mustMatch(t, m, "trailing ", false, true)
	mustMatch(t, m, "trailing", false, false)
	mustMatch(t, m, "a*b.txt", false, true)
	mustMatch(t, m, "axb.txt", false, false)
	mustMatch(t, m, "trim.log", false, true)

	want := []string{"\\#notes.txt", "\\!bang.txt", "trailing\\ ", "a\\*b.txt", "trim.log"}
	got := m.Patterns()
	if len(got) != len(want) {
		t.Fatalf("Patterns() = %q，期望 %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Patterns()[%d] = %q，期望 %q", i, got[i], want[i])
		}
	}
}

func TestNegationBasic(t *testing.T) {
	m := mustMatcher(t, []string{"*.log", "!keep.log"})

	mustMatch(t, m, "a.log", false, true)
	mustMatch(t, m, "keep.log", false, false)
	mustMatch(t, m, "dir/keep.log", false, false)
	mustMatch(t, m, "dir/b.log", false, true)
	mustMatch(t, m, "keep.log.bak", false, false)
}

func TestLastMatchWinsOrderSensitive(t *testing.T) {
	m1 := mustMatcher(t, []string{"*.tmp", "!*.tmp", "important.tmp"})
	mustMatch(t, m1, "a.tmp", false, false)
	mustMatch(t, m1, "sub/a.tmp", false, false)
	mustMatch(t, m1, "important.tmp", false, true)

	// 交换顺序之后同一路径的结果会变
	m2 := mustMatcher(t, []string{"!*.tmp", "*.tmp"})
	mustMatch(t, m2, "a.tmp", false, true)
	mustMatch(t, m2, "sub/a.tmp", false, true)
}

func TestDirectoryOnly(t *testing.T) {
	m := mustMatcher(t, []string{"cache/"})

	mustMatch(t, m, "cache", true, true)
	mustMatch(t, m, "cache", false, false)
	mustMatch(t, m, "sub/cache", true, true)
	mustMatch(t, m, "cache.bak", false, false)
}

func TestParentDirPropagation(t *testing.T) {
	m := mustMatcher(t, []string{"build/"})
	mustMatch(t, m, "build", true, true)
	mustMatch(t, m, "build", false, false)
	mustMatch(t, m, "build/out.js", false, true)
	mustMatch(t, m, "build/a/b/c.txt", false, true)
	mustMatch(t, m, "builder/out.js", false, false)
	mustMatch(t, m, "src/build/x", false, true)

	// 父目录被排除时，反选救不回来
	blocked := mustMatcher(t, []string{"build/", "!build/keep.txt"})
	mustMatch(t, blocked, "build/keep.txt", false, true)
	mustMatch(t, blocked, "build/other.txt", false, true)

	// 对照：build 目录本身没被排除时，反选是有效的
	open := mustMatcher(t, []string{"build/*", "!build/keep.txt"})
	mustMatch(t, open, "build/keep.txt", false, false)
	mustMatch(t, open, "build/other.txt", false, true)
}

func TestBadPathAndBadPattern(t *testing.T) {
	m := mustMatcher(t, []string{"*.log"})

	for _, bad := range []string{"", "/abs.log", "a/../b.log", "../a.log"} {
		if _, err := m.Match(bad, false); !errors.Is(err, ignorepath.ErrBadPath) {
			t.Fatalf("Match(%q) 返回 %v，期望 ErrBadPath", bad, err)
		}
		if err := ignorepath.CheckPath(bad); !errors.Is(err, ignorepath.ErrBadPath) {
			t.Fatalf("CheckPath(%q) 返回 %v，期望 ErrBadPath", bad, err)
		}
	}

	if err := ignorepath.CheckPath("a/b.log"); err != nil {
		t.Fatalf("CheckPath(\"a/b.log\") 返回 %v，期望 nil", err)
	}
	if _, err := m.Match("a/b.log", false); err != nil {
		t.Fatalf("Match(\"a/b.log\") 返回 %v，期望 nil", err)
	}
	if _, err := ignorepath.MatchPath([]string{"*.log"}, "src/a.log", false); err != nil {
		t.Fatalf("MatchPath 返回 %v，期望 nil", err)
	}

	for _, bad := range [][]string{{"ok", "bad[unclosed"}, {"["}, {"a[b"}} {
		if _, err := ignorepath.NewMatcher(bad); !errors.Is(err, ignorepath.ErrBadPattern) {
			t.Fatalf("NewMatcher(%q) 返回 %v，期望 ErrBadPattern", bad, err)
		}
	}

	// 转义过的 "[" 不算未闭合；正常字符类也不该报错
	if _, err := ignorepath.NewMatcher([]string{"\\[literal"}); err != nil {
		t.Fatalf("NewMatcher 对转义后的 \"[\" 返回 %v，期望 nil", err)
	}
	if _, err := ignorepath.NewMatcher([]string{"fine[0-9].log"}); err != nil {
		t.Fatalf("NewMatcher 对合法字符类返回 %v，期望 nil", err)
	}
}

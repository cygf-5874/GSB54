// Command ignorecheck 是 ignorepath 的固定验收程序。
//
// ⚠️ 不要修改本文件。它是判定「题目有没有做对」的依据；
// 修改它只会让判定失效，不会让实现变对。
//
// 10 个场景分四组：
//
//	syntax   3 个：** 跨目录 / 字符类与 ? / 转义与行尾空白
//	negate   3 个：反选 / 最后匹配胜出 / 目录限定
//	ancestor 2 个：父目录传播 / 父目录被排除时反选无效
//	robust   2 个：ErrBadPath / ErrBadPattern
//
// 用法：
//
//	go run ./ignorecheck                  # 跑全部场景
//	go run ./ignorecheck -only negate     # 只跑一组（--only 等价）
//	go run ./ignorecheck -list            # 列出场景
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime/pprof"
	"strings"
	"time"

	"ignorepath"
)

const (
	childTimeout = 25 * time.Second
	parentGrace  = 10 * time.Second
)

type scenario struct {
	Group string
	Name  string
	Run   func() error
}

var scenarios = []scenario{
	{"syntax", "SYN1_doublestar_across_dirs", synDoublestar},
	{"syntax", "SYN2_character_class_and_question", synCharClass},
	{"syntax", "SYN3_escapes_and_trailing_space", synEscapes},
	{"negate", "NEG1_negation_basic", negBasic},
	{"negate", "NEG2_last_match_wins", negLastWins},
	{"negate", "NEG3_dir_only", negDirOnly},
	{"ancestor", "ANC1_parent_dir_ignored", ancParentIgnored},
	{"ancestor", "ANC2_negation_cannot_reinclude", ancNegationBlocked},
	{"robust", "ROB1_bad_path", robBadPath},
	{"robust", "ROB2_bad_pattern", robBadPattern},
}

var groups = []string{"syntax", "negate", "ancestor", "robust"}

func main() {
	only := flag.String("only", "", "只跑指定组：syntax / negate / ancestor / robust")
	name := flag.String("scenario", "", "内部使用：只跑单个场景")
	list := flag.Bool("list", false, "列出全部场景")
	flag.Parse()

	if *list {
		for _, sc := range scenarios {
			fmt.Printf("%-8s %s\n", sc.Group, sc.Name)
		}
		return
	}
	if *name != "" {
		os.Exit(runChild(*name))
	}
	os.Exit(runParent(*only))
}

func runChild(name string) int {
	var sc *scenario
	for i := range scenarios {
		if scenarios[i].Name == name {
			sc = &scenarios[i]
			break
		}
	}
	if sc == nil {
		fmt.Fprintf(os.Stderr, "未知场景 %q\n", name)
		return 2
	}

	watchdog := time.AfterFunc(childTimeout, func() {
		fmt.Fprintf(os.Stderr, "看门狗：场景 %s 超过 %s 仍未结束，下面是全部 goroutine 栈\n",
			sc.Name, childTimeout)
		_ = pprof.Lookup("goroutine").WriteTo(os.Stderr, 2)
		os.Exit(3)
	})
	defer watchdog.Stop()

	if err := sc.Run(); err != nil {
		fmt.Println(err.Error())
		return 1
	}
	return 0
}

func runParent(only string) int {
	exe, err := os.Executable()
	if err != nil {
		fmt.Println("无法定位自身可执行文件:", err)
		return 1
	}

	runGroups := groups
	if only != "" {
		found := false
		for _, g := range groups {
			if g == only {
				found = true
			}
		}
		if !found {
			fmt.Printf("未知分组 %q（可选：syntax / negate / ancestor / robust）\n", only)
			return 2
		}
		runGroups = []string{only}
	}

	total, passed := 0, 0
	for _, g := range runGroups {
		fmt.Printf("== 组 %s ==\n", g)
		for _, sc := range scenarios {
			if sc.Group != g {
				continue
			}
			total++

			ctx, cancel := context.WithTimeout(context.Background(), childTimeout+parentGrace)
			cmd := exec.CommandContext(ctx, exe, "-scenario", sc.Name)
			cmd.WaitDelay = 5 * time.Second
			out, runErr := cmd.CombinedOutput()
			cancel()

			if runErr == nil {
				passed++
				fmt.Printf("PASS %s/%s\n", sc.Group, sc.Name)
				continue
			}

			text := strings.TrimSpace(string(out))
			lines := strings.Split(text, "\n")
			header := runErr.Error()
			if text != "" {
				header = lines[0]
			}
			fmt.Printf("FAIL %s/%s  %s\n", sc.Group, sc.Name, header)
			for _, line := range tailLines(lines, 8) {
				fmt.Printf("        | %s\n", line)
			}
		}
	}

	fmt.Println()
	fmt.Printf("结果：通过 %d/%d\n", passed, total)
	if passed != total {
		return 1
	}
	return 0
}

func tailLines(lines []string, n int) []string {
	if len(lines) <= 1 {
		return nil
	}
	rest := lines[1:]
	if len(rest) > n {
		rest = append([]string{"…"}, rest[len(rest)-n:]...)
	}
	return rest
}

// ---------------------------------------------------------------- 测试辅助

func newMatcher(patterns []string) (*ignorepath.Matcher, error) {
	return ignorepath.NewMatcher(patterns)
}

type caseT struct {
	path  string
	isDir bool
	want  bool
}

func runCases(m *ignorepath.Matcher, cases []caseT) error {
	for _, c := range cases {
		got, err := m.Match(c.path, c.isDir)
		if err != nil {
			return fmt.Errorf("Match(%q, isDir=%v) 期望=%v 实际=出错 %v",
				c.path, c.isDir, c.want, err)
		}
		if got != c.want {
			return fmt.Errorf("Match(%q, isDir=%v) 期望=%v 实际=%v",
				c.path, c.isDir, c.want, got)
		}
	}
	return nil
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ------------------------------------------------------------------ syntax

func synDoublestar() error {
	m, err := newMatcher([]string{"a/**/b.txt", "**/deep.log"})
	if err != nil {
		return fmt.Errorf("NewMatcher 期望=成功 实际=出错 %v", err)
	}
	return runCases(m, []caseT{
		{"a/b.txt", false, true},
		{"a/x/b.txt", false, true},
		{"a/x/y/b.txt", false, true},
		{"a/x/c.txt", false, false},
		{"deep.log", false, true},
		{"p/q/deep.log", false, true},
		{"p/deep.log.bak", false, false},
	})
}

func synCharClass() error {
	m, err := newMatcher([]string{"file[0-9].txt", "data[!0-9].bin", "q?.log"})
	if err != nil {
		return fmt.Errorf("NewMatcher 期望=成功 实际=出错 %v", err)
	}
	return runCases(m, []caseT{
		{"file7.txt", false, true},
		{"dir/file3.txt", false, true},
		{"filex.txt", false, false},
		{"file12.txt", false, false},
		{"dataA.bin", false, true},
		{"data5.bin", false, false},
		{"q1.log", false, true},
		{"qq.log", false, true},
		{"q.log", false, false},
		{"q/1.log", false, false},
	})
}

func synEscapes() error {
	m, err := newMatcher([]string{
		"\\#notes.txt",
		"\\!bang.txt",
		"a\\*b.txt",
		"trim.log ",
		"# 真注释",
	})
	if err != nil {
		return fmt.Errorf("NewMatcher 期望=成功 实际=出错 %v", err)
	}
	if err := runCases(m, []caseT{
		{"#notes.txt", false, true},
		{"!bang.txt", false, true},
		{"bang.txt", false, false},
		{"a*b.txt", false, true},
		{"axb.txt", false, false},
		{"trim.log", false, true},
	}); err != nil {
		return err
	}

	want := []string{"\\#notes.txt", "\\!bang.txt", "a\\*b.txt", "trim.log"}
	got := m.Patterns()
	if !equalStrings(got, want) {
		return fmt.Errorf("Patterns() 期望=%q 实际=%q", want, got)
	}
	return nil
}

// ------------------------------------------------------------------ negate

func negBasic() error {
	m, err := newMatcher([]string{"*.log", "!keep.log"})
	if err != nil {
		return fmt.Errorf("NewMatcher 期望=成功 实际=出错 %v", err)
	}
	return runCases(m, []caseT{
		{"a.log", false, true},
		{"keep.log", false, false},
		{"dir/keep.log", false, false},
		{"dir/b.log", false, true},
		{"keep.log.bak", false, false},
	})
}

func negLastWins() error {
	m1, err := newMatcher([]string{"*.tmp", "!*.tmp", "important.tmp"})
	if err != nil {
		return fmt.Errorf("NewMatcher 期望=成功 实际=出错 %v", err)
	}
	if err := runCases(m1, []caseT{
		{"a.tmp", false, false},
		{"sub/a.tmp", false, false},
		{"important.tmp", false, true},
	}); err != nil {
		return err
	}

	m2, err := newMatcher([]string{"!*.tmp", "*.tmp"})
	if err != nil {
		return fmt.Errorf("NewMatcher 期望=成功 实际=出错 %v", err)
	}
	return runCases(m2, []caseT{
		{"a.tmp", false, true},
		{"sub/a.tmp", false, true},
	})
}

func negDirOnly() error {
	m, err := newMatcher([]string{"cache/"})
	if err != nil {
		return fmt.Errorf("NewMatcher 期望=成功 实际=出错 %v", err)
	}
	return runCases(m, []caseT{
		{"cache", true, true},
		{"cache", false, false},
		{"sub/cache", true, true},
		{"cache.bak", false, false},
	})
}

// ---------------------------------------------------------------- ancestor

func ancParentIgnored() error {
	m, err := newMatcher([]string{"build/"})
	if err != nil {
		return fmt.Errorf("NewMatcher 期望=成功 实际=出错 %v", err)
	}
	return runCases(m, []caseT{
		{"build", true, true},
		{"build", false, false},
		{"build/out.js", false, true},
		{"build/a/b/c.txt", false, true},
		{"builder/out.js", false, false},
		{"src/build/x", false, true},
	})
}

func ancNegationBlocked() error {
	blocked, err := newMatcher([]string{"build/", "!build/keep.txt"})
	if err != nil {
		return fmt.Errorf("NewMatcher 期望=成功 实际=出错 %v", err)
	}
	if err := runCases(blocked, []caseT{
		{"build/keep.txt", false, true},
		{"build/other.txt", false, true},
	}); err != nil {
		return err
	}

	open, err := newMatcher([]string{"build/*", "!build/keep.txt"})
	if err != nil {
		return fmt.Errorf("NewMatcher 期望=成功 实际=出错 %v", err)
	}
	return runCases(open, []caseT{
		{"build/keep.txt", false, false},
		{"build/other.txt", false, true},
	})
}

// ------------------------------------------------------------------ robust

func robBadPath() error {
	m, err := newMatcher([]string{"*.log"})
	if err != nil {
		return fmt.Errorf("NewMatcher 期望=成功 实际=出错 %v", err)
	}

	for _, bad := range []string{"", "/abs.log", "a/../b.log", "../a.log"} {
		if _, err := m.Match(bad, false); !errors.Is(err, ignorepath.ErrBadPath) {
			return fmt.Errorf("Match(%q) 期望=ErrBadPath 实际=%v", bad, err)
		}
		if err := ignorepath.CheckPath(bad); !errors.Is(err, ignorepath.ErrBadPath) {
			return fmt.Errorf("CheckPath(%q) 期望=ErrBadPath 实际=%v", bad, err)
		}
	}

	if err := ignorepath.CheckPath("a/b.log"); err != nil {
		return fmt.Errorf("CheckPath(\"a/b.log\") 期望=nil 实际=%v", err)
	}
	if _, err := m.Match("a/b.log", false); err != nil {
		return fmt.Errorf("Match(\"a/b.log\") 期望=nil 实际=%v", err)
	}
	return nil
}

func robBadPattern() error {
	for _, bad := range [][]string{{"ok", "bad[unclosed"}, {"["}, {"a[b"}} {
		if _, err := newMatcher(bad); !errors.Is(err, ignorepath.ErrBadPattern) {
			return fmt.Errorf("NewMatcher(%q) 期望=ErrBadPattern 实际=%v", bad, err)
		}
	}

	if _, err := newMatcher([]string{"\\[literal", "fine[0-9].log"}); err != nil {
		return fmt.Errorf("NewMatcher(合法的转义与字符类) 期望=nil 实际=%v", err)
	}

	m, err := newMatcher(nil)
	if err != nil {
		return fmt.Errorf("NewMatcher(nil) 期望=nil 实际=%v", err)
	}
	if !m.IsEmpty() {
		return fmt.Errorf("空模式列表 IsEmpty() 期望=true 实际=false")
	}
	return nil
}

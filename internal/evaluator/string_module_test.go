package evaluator

import (
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func setupStringModule(t *testing.T) *Environment {
	t.Helper()
	return setupSDKModuleEnv(t, "系统.凡", "字符串.凡")
}

func evalStringModule(t *testing.T, src string) object.Object {
	t.Helper()
	env := setupStringModule(t)
	progAST, errs := parser.ParseProgram(`导入 "库/字符串" 作为 文本模块
` + src)
	if len(errs) > 0 {
		t.Fatalf("解析失败：%v", errs)
	}
	var prog *ast.Program = progAST
	res, err := Eval(prog, env)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	return res
}

func TestStringModuleSplitAndJoin(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{`文本模块.拆分("a,b,c", ",")`, "[a, b, c]"},
		{`文本模块.拆分("abc", "")`, "[a, b, c]"},
		{`文本模块.连接(["a", "b", "c"], "-")`, "a-b-c"},
	}
	for _, tc := range cases {
		if got := evalStringModule(t, tc.src).Inspect(); got != tc.want {
			t.Fatalf("%s => %s，期望 %s", tc.src, got, tc.want)
		}
	}
}

func TestStringModuleSearch(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{`文本模块.包含("abc", "b")`, "真"},
		{`文本模块.包含("abc", "d")`, "假"},
		{`文本模块.开头是("abc", "a")`, "真"},
		{`文本模块.结尾是("abc", "c")`, "真"},
		{`文本模块.位置("abc", "b")`, "1"},
		{`文本模块.位置("abc", "d")`, "-1"},
		{`文本模块.截取("abc", 1, 2)`, "b"},
		{`文本模块.截取("abc", 1)`, "bc"},
	}
	for _, tc := range cases {
		if got := evalStringModule(t, tc.src).Inspect(); got != tc.want {
			t.Fatalf("%s => %s，期望 %s", tc.src, got, tc.want)
		}
	}
}

func TestStringModuleTransform(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{`文本模块.替换("abc", "b", "x")`, "axc"},
		{`文本模块.修剪("  abc  ")`, "abc"},
		{`文本模块.转小写("AbC")`, "abc"},
		{`文本模块.转大写("AbC")`, "ABC"},
	}
	for _, tc := range cases {
		if got := evalStringModule(t, tc.src).Inspect(); got != tc.want {
			t.Fatalf("%s => %s，期望 %s", tc.src, got, tc.want)
		}
	}
}

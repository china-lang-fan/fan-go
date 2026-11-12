package evaluator

import (
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func setupDictModule(t *testing.T) *Environment {
	t.Helper()
	return setupSDKModuleEnv(t, "系统.凡", "字典.凡")
}

func evalDictModule(t *testing.T, src string) object.Object {
	t.Helper()
	env := setupDictModule(t)
	progAST, errs := parser.ParseProgram(`导入 "库/字典" 作为 字典模块
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

func TestDictModuleEntriesAndMerge(t *testing.T) {
	src := `变量 d = {"a": 1, "b": 2}
变量 e = 字典模块.条目(d)
字典模块.合并(d, {"c": 3})`
	if got := evalDictModule(t, src).Inspect(); got != "{a: 1, b: 2, c: 3}" {
		t.Fatalf("merge 结果为 %s", got)
	}
}

func TestDictModuleTransform(t *testing.T) {
	src := `函数 加倍(整数 值， 字符串 键) -> 整数
    返回 值 * 2
结束
变量 d = {"a": 1}
字典模块.映射值(d, 加倍)["a"]`
	if got := evalDictModule(t, src).Inspect(); got != "2" {
		t.Fatalf("mapValues 结果为 %s，期望 2", got)
	}
}

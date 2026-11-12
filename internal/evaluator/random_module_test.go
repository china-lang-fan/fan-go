package evaluator

import (
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func setupRandomModule(t *testing.T) *Environment {
	t.Helper()
	return setupSDKModuleEnv(t, "系统.凡", "字符串.凡", "数组.凡", "随机.凡")
}

func evalRandomModule(t *testing.T, src string) object.Object {
	t.Helper()
	env := setupRandomModule(t)
	progAST, errs := parser.ParseProgram(`导入 "库/随机" 作为 随机模块
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

func TestRandomModuleBounds(t *testing.T) {
	src := `变量 值 = 随机模块.随机整数(5)
值 >= 0 且 值 < 5`
	if got := evalRandomModule(t, src).Inspect(); got != "真" {
		t.Fatalf("结果为 %s，期望 真", got)
	}
}

func TestRandomModuleSample(t *testing.T) {
	src := `长度(随机模块.抽样([1, 2, 3], 2))`
	if got := evalRandomModule(t, src).Inspect(); got != "2" {
		t.Fatalf("结果为 %s，期望 2", got)
	}
}

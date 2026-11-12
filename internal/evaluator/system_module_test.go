package evaluator

import (
	"testing"

	"fan/internal/ast"
	"fan/internal/parser"
)

func TestSystemModuleWrapsNativeFunctions(t *testing.T) {
	env := setupSDKModuleEnv(t, "系统.凡")
	src := `导入 "库/系统" 作为 系统
变量 结果 = []
结果 = 追加(结果, 系统.类型名("x"))
结果 = 追加(结果, 系统.截断(1.8))
结果 = 追加(结果, 系统.编码点("A"))
结果 = 追加(结果, 系统.字符(65))
结果 = 追加(结果, 系统.长度([1, 2]))
结果 = 追加(结果, 系统.追加([1], 2))
结果`
	progAST, errs := parser.ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析失败：%v", errs)
	}
	var prog *ast.Program = progAST
	res, err := Eval(prog, env)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != `[string, 1, 65, A, 2, [1, 2]]` {
		t.Fatalf("系统模块结果错误：%s", res.Inspect())
	}
}

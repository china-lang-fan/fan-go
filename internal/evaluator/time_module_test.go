package evaluator

import (
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func setupTimeModule(t *testing.T) *Environment {
	t.Helper()
	return setupSDKModuleEnv(t, "系统.凡", "时间.凡")
}

func evalTimeModule(t *testing.T, src string) object.Object {
	t.Helper()
	env := setupTimeModule(t)
	progAST, errs := parser.ParseProgram(`导入 "库/时间" 作为 时间模块
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

func TestTimeModuleFormatParse(t *testing.T) {
	src := `变量 时间戳, 错误值 = 时间模块.解析日期("2020-01-01")
如果 错误值 != 空 那么
    返回 错误值
结束
时间模块.格式化(时间戳, 时间模块.日期格式)`
	if got := evalTimeModule(t, src).Inspect(); got != "2020-01-01" {
		t.Fatalf("结果为 %s，期望 2020-01-01", got)
	}
}

func TestTimeModuleParts(t *testing.T) {
	src := `变量 时间戳, 错误值 = 时间模块.解析(时间模块.默认格式, "2020-06-15 10:20:30")
变量 分量 = 时间模块.时间分量(时间戳)
分量["year"]`
	if got := evalTimeModule(t, src).Inspect(); got != "2020" {
		t.Fatalf("结果为 %s，期望 2020", got)
	}
}

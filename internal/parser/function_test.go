package parser

import (
	"strings"
	"testing"

	"fan/internal/ast"
)

func TestFunctionDefinitionWithParens(t *testing.T) {
	src := strings.Join([]string{
		`函数 求和（任意 甲，任意 乙）`,
		`    返回 甲 加 乙`,
		`结束`,
	}, "\n")
	prog := mustParse(t, src)
	decl := prog.Statements[0].(*ast.VarDecl)
	fn, ok := decl.Value.(*ast.FunctionLiteral)
	if !ok {
		t.Fatalf("应为函数字面量，实际 %T", decl.Value)
	}
	if len(fn.Params) != 2 || fn.Params[0].Name != "甲" || fn.Params[1].Name != "乙" {
		t.Fatalf("参数解析错误：%v", fn.Params)
	}
	if fn.Name != "求和" {
		t.Fatalf("函数名应为 求和，实际 %s", fn.Name)
	}
}

func TestAnonymousFunction(t *testing.T) {
	src := strings.Join([]string{
		`变量 平方 = 函数（任意 数）`,
		`    返回 数 乘 数`,
		`结束`,
	}, "\n")
	prog := mustParse(t, src)
	decl := prog.Statements[0].(*ast.VarDecl)
	fn, ok := decl.Value.(*ast.FunctionLiteral)
	if !ok {
		t.Fatalf("应为匿名函数，实际 %T", decl.Value)
	}
	if fn.Name != "" {
		t.Fatalf("匿名函数名应为空，实际 %s", fn.Name)
	}
	if len(fn.Params) != 1 || fn.Params[0].Name != "数" {
		t.Fatalf("参数解析错误：%v", fn.Params)
	}
}

func TestReturnNoValue(t *testing.T) {
	prog := mustParse(t, "返回")
	if _, ok := prog.Statements[0].(*ast.ReturnStmt); !ok {
		t.Fatalf("应为 返回 语句，实际 %T", prog.Statements[0])
	}
}

func TestNormalParenCall(t *testing.T) {
	prog := mustParse(t, `打印("你好")`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.CallExpr)
	if id, ok := call.Callee.(*ast.Identifier); !ok || id.Name != "打印" {
		t.Fatalf("调用方错误：%s", call.Callee.String())
	}
	if len(call.Args) != 1 {
		t.Fatalf("应有 1 个参数，实际 %d", len(call.Args))
	}
}

func TestImplicitCallNormal(t *testing.T) {
	prog := mustParse(t, `打印 "你好"`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.CallExpr)
	if id, ok := call.Callee.(*ast.Identifier); !ok || id.Name != "打印" {
		t.Fatalf("调用方错误：%s", call.Callee.String())
	}
	if len(call.Args) != 1 {
		t.Fatalf("应有 1 个参数，实际 %d", len(call.Args))
	}
}

func TestImplicitReceiverCall(t *testing.T) {
	prog := mustParse(t, `名单 追加 "张三"`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.ImplicitReceiverCallExpr)
	if call.Name != "追加" {
		t.Fatalf("方法名应为 追加，实际 %s", call.String())
	}
	if call.Receiver.String() != "名单" || len(call.Args) != 1 {
		t.Fatalf("接收者和参数解析错误，实际 %s", call.String())
	}
}

func TestLiteralImplicitReceiverCall(t *testing.T) {
	prog := mustParse(t, `"a,b,c" 拆分(",")`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.ImplicitReceiverCallExpr)
	if call.Name != "拆分" {
		t.Fatalf("方法名错误：%s", call.String())
	}
	if _, ok := call.Receiver.(*ast.StringLiteral); !ok || len(call.Args) != 1 {
		t.Fatalf("字面量接收者解析错误：%s", call.String())
	}
}

func TestLiteralImplicitReceiverNoArgCall(t *testing.T) {
	prog := mustParse(t, `"abc" 转大写`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.ImplicitReceiverCallExpr)
	if call.Name != "转大写" || len(call.Args) != 0 {
		t.Fatalf("无参接收者方法解析错误：%s", call.String())
	}
}

func TestLiteralImplicitReceiverChainedCall(t *testing.T) {
	prog := mustParse(t, `"a,b,c" 拆分(",") 反转`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.ImplicitReceiverCallExpr)
	if call.Name != "反转" || len(call.Args) != 0 {
		t.Fatalf("链式方法解析错误：%s", call.String())
	}
	if _, ok := call.Receiver.(*ast.ImplicitReceiverCallExpr); !ok {
		t.Fatalf("前一方法调用解析错误：%s", call.String())
	}
}

func TestMemberNoArgCall(t *testing.T) {
	prog := mustParse(t, "名单 的 长度")
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	if _, ok := stmt.Expression.(*ast.MaybeCallExpr); !ok {
		t.Fatalf("应为可能调用表达式，实际 %T", stmt.Expression)
	}
}

func TestMemberMethodCall(t *testing.T) {
	prog := mustParse(t, `名单 的 追加("张三")`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.CallExpr)
	if _, ok := call.Callee.(*ast.MemberExpr); !ok {
		t.Fatalf("应为成员方法调用，实际 %T", stmt.Expression)
	}
}

func TestImplicitCallMultipleArgsWithParens(t *testing.T) {
	prog := mustParse(t, `追加(名单, "张三")`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.CallExpr)
	if call.Callee.(*ast.Identifier).Name != "追加" {
		t.Fatalf("调用方错误")
	}
	if len(call.Args) != 2 {
		t.Fatalf("应有 2 个参数，实际 %d", len(call.Args))
	}
}

func TestImplicitReceiverCallWithMultipleArgs(t *testing.T) {
	prog := mustParse(t, `名单 追加 "张三", 18`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.ImplicitReceiverCallExpr)
	if call.Name != "追加" {
		t.Fatalf("方法名应为 追加，实际 %s", call.String())
	}
	if len(call.Args) != 2 {
		t.Fatalf("应有 2 个参数，实际 %d", len(call.Args))
	}
}

func TestImplicitCallNestedInArg(t *testing.T) {
	prog := mustParse(t, `打印 名单 的 长度`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.CallExpr)
	if call.Callee.(*ast.Identifier).Name != "打印" {
		t.Fatalf("外层应为 打印，实际 %s", call.Callee.String())
	}
	if _, ok := call.Args[0].(*ast.MemberExpr); !ok {
		t.Fatalf("参数应为成员访问，实际 %s", call.Args[0].String())
	}
}

func TestNoImplicitInCondition(t *testing.T) {
	src := strings.Join([]string{
		`变量 甲 = 1`,
		`如果 甲 加 1 等于 2 那么`,
		`    甲 = 2`,
		`结束`,
	}, "\n")
	prog := mustParse(t, src)
	ifStmt := prog.Statements[1].(*ast.IfStmt)
	if _, ok := ifStmt.Branches[0].Condition.(*ast.BinaryExpr); !ok {
		t.Fatalf("条件应为二元表达式，实际 %T", ifStmt.Branches[0].Condition)
	}
}

func TestDefineFunctionOptional(t *testing.T) {
	src := strings.Join([]string{
		`定义 函数 求和(任意 甲, 任意 乙)`,
		`    返回 甲 加 乙`,
		`结束`,
	}, "\n")
	prog := mustParse(t, src)
	decl := prog.Statements[0].(*ast.VarDecl)
	fn := decl.Value.(*ast.FunctionLiteral)
	if fn.Name != "求和" {
		t.Fatalf("函数名应为 求和，实际 %s", fn.Name)
	}
}

func TestTypedParameters(t *testing.T) {
	src := `定义 函数 求和(整数 甲，整数 乙) -> 整数
    返回 甲 加 乙
结束`
	prog := mustParse(t, src)
	fn := prog.Statements[0].(*ast.VarDecl).Value.(*ast.FunctionLiteral)
	if len(fn.Params) != 2 {
		t.Fatalf("应有 2 个参数，实际 %d", len(fn.Params))
	}
	if fn.Params[0].Type != ast.TypeInt || fn.Params[0].Name != "甲" {
		t.Fatalf("首个参数错误：%+v", fn.Params[0])
	}
	if fn.Params[1].Type != ast.TypeInt || fn.Params[1].Name != "乙" {
		t.Fatalf("第二个参数错误：%+v", fn.Params[1])
	}
	if len(fn.ReturnTypes) != 1 || fn.ReturnTypes[0] != ast.TypeInt {
		t.Fatalf("返回类型错误：%+v", fn.ReturnTypes)
	}
}

func TestDefaultAndVariadicParameters(t *testing.T) {
	src := `函数 f(整数 a, 整数 b = 2, 整数... 其余)
结束`
	prog := mustParse(t, src)
	fn := prog.Statements[0].(*ast.VarDecl).Value.(*ast.FunctionLiteral)
	if len(fn.Params) != 3 {
		t.Fatalf("应有 3 个参数，实际 %d", len(fn.Params))
	}
	if fn.Params[1].Default == nil {
		t.Fatalf("第二个参数应有默认值")
	}
	if !fn.Params[2].Variadic {
		t.Fatalf("第三个参数应是可变参数")
	}
}

func TestInvalidParameterOrder(t *testing.T) {
	expectError(t, "函数 f(整数... 其余, 整数 b)\n结束")
	expectError(t, "函数 f(整数 a = 1, 整数 b)\n结束")
	// 旧语序（名字在前、类型在后）直接报错
	expectError(t, "函数 f(a 整数)\n结束")
	// 无类型参数报错：必须显式声明类型
	expectError(t, "函数 f(乙)\n结束")
}

func TestTypedParamsMultiReturn(t *testing.T) {
	src := "函数 商余(整数 a，整数 b) -> (整数，整数)\n    返回 a / b，a % b\n结束"
	prog := mustParse(t, src)
	fn := prog.Statements[0].(*ast.VarDecl).Value.(*ast.FunctionLiteral)
	if len(fn.ReturnTypes) != 2 ||
		fn.ReturnTypes[0] != ast.TypeInt || fn.ReturnTypes[1] != ast.TypeInt {
		t.Fatalf("返回类型解析错误：%+v", fn.ReturnTypes)
	}
	ret := fn.Body.Statements[0].(*ast.ReturnStmt)
	if len(ret.Values) != 2 {
		t.Fatalf("返回值数量应为 2，实际 %d", len(ret.Values))
	}
}

func TestMultiDecl(t *testing.T) {
	src := "变量 a、b = 商余(10, 3)"
	prog := mustParse(t, src)
	decl := prog.Statements[0].(*ast.MultiDecl)
	if len(decl.Names) != 2 || decl.Names[0] != "a" || decl.Names[1] != "b" {
		t.Fatalf("多变量声明解析错误：%+v", decl.Names)
	}
}

func TestMultiAssign(t *testing.T) {
	src := "a、b = 商余(10, 3)"
	prog := mustParse(t, src)
	assign := prog.Statements[0].(*ast.MultiAssign)
	if len(assign.Names) != 2 {
		t.Fatalf("多目标赋值解析错误：%+v", assign.Names)
	}
}

func TestCheckExpression(t *testing.T) {
	src := "变量 x = 检查 f()"
	prog := mustParse(t, src)
	decl := prog.Statements[0].(*ast.VarDecl)
	if _, ok := decl.Value.(*ast.CheckExpr); !ok {
		t.Fatalf("应为检查表达式，实际 %T", decl.Value)
	}
}

func TestDeferAndCrashCall(t *testing.T) {
	src := "稍后 清理(1)\n崩溃了(\"失败\")"
	prog := mustParse(t, src)
	deferStmt, ok := prog.Statements[0].(*ast.DeferStmt)
	if !ok {
		t.Fatalf("应为稍后语句，实际 %T", prog.Statements[0])
	}
	if _, ok := deferStmt.Call.(*ast.CallExpr); !ok {
		t.Fatalf("稍后调用解析错误：%T", deferStmt.Call)
	}
	// 崩溃了 已函数化：第二行是普通表达式语句（函数调用），不再有独立语句节点
	exprStmt, ok := prog.Statements[1].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("崩溃了应为表达式语句，实际 %T", prog.Statements[1])
	}
	if _, ok := exprStmt.Expression.(*ast.CallExpr); !ok {
		t.Fatalf("崩溃了调用解析错误：%T", exprStmt.Expression)
	}
}

func TestReadablePrefixKeywords(t *testing.T) {
	prog := mustParse(t, "变量 x = 从 f(1)")
	decl := prog.Statements[0].(*ast.VarDecl)
	call, ok := decl.Value.(*ast.CallExpr)
	if !ok {
		t.Fatalf("协助关键字应透明解析为后续表达式，实际 %T", decl.Value)
	}
	if ident, ok := call.Callee.(*ast.Identifier); !ok || ident.Name != "f" {
		t.Fatalf("协助关键字后的调用解析错误：%T", call.Callee)
	}
}

func TestThroughExpression(t *testing.T) {
	prog := mustParse(t, "通过 工具 求(数列)的最大值()")
	expr, ok := prog.Statements[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("应为表达式语句，实际 %T", prog.Statements[0])
	}
	through, ok := expr.Expression.(*ast.ThroughExpr)
	if !ok {
		t.Fatalf("应为 通过 表达式，实际 %T", expr.Expression)
	}
	if through.ModuleName != "工具" {
		t.Fatalf("模块名错误：%s", through.ModuleName)
	}
}

func TestErrorType(t *testing.T) {
	src := "函数 f() -> 错误\n    返回 空\n结束"
	prog := mustParse(t, src)
	fn := prog.Statements[0].(*ast.VarDecl).Value.(*ast.FunctionLiteral)
	if len(fn.ReturnTypes) != 1 || fn.ReturnTypes[0] != ast.TypeError {
		t.Fatalf("错误返回类型解析错误：%+v", fn.ReturnTypes)
	}
}

func TestImportBasic(t *testing.T) {
	src := `导入 "mathlib/运算"`
	prog := mustParse(t, src)
	imp, ok := prog.Statements[0].(*ast.ImportStmt)
	if !ok {
		t.Fatalf("应为导入语句，实际 %T", prog.Statements[0])
	}
	if imp.Path != "mathlib/运算" || imp.Name != "运算" {
		t.Fatalf("导入解析错误：%+v", imp)
	}
}

func TestImportAlias(t *testing.T) {
	src := `导入 "mathlib/运算" 作为 m`
	prog := mustParse(t, src)
	imp := prog.Statements[0].(*ast.ImportStmt)
	if imp.Name != "m" {
		t.Fatalf("别名解析错误：%s", imp.Name)
	}
}

func TestImportMissingPath(t *testing.T) {
	expectError(t, `导入`)
}

func TestExportVar(t *testing.T) {
	src := `公开 变量 x = 1`
	prog := mustParse(t, src)
	exp, ok := prog.Statements[0].(*ast.VisibilityStmt)
	if !ok {
		t.Fatalf("应为导出语句，实际 %T", prog.Statements[0])
	}
	if exp.Name != "x" {
		t.Fatalf("导出名错误：%s", exp.Name)
	}
}

func TestExportFunction(t *testing.T) {
	src := "公开 函数 f()\n    返回 1\n结束"
	prog := mustParse(t, src)
	exp := prog.Statements[0].(*ast.VisibilityStmt)
	if exp.Name != "f" {
		t.Fatalf("导出函数名错误：%s", exp.Name)
	}
}

func TestExportClass(t *testing.T) {
	src := "公开 模型 点\n    属性 x\n结束"
	prog := mustParse(t, src)
	exp := prog.Statements[0].(*ast.VisibilityStmt)
	if exp.Name != "点" {
		t.Fatalf("导出模型名错误：%s", exp.Name)
	}
}

// 新语序：类型在前、名字在后，且必须显式声明类型。
func TestTypeFirstParameterOrder(t *testing.T) {
	cases := []struct {
		src  string
		name string
		typ  ast.DeclType
	}{
		{"函数 f(整数 n)\n结束", "n", ast.TypeInt},
		{"函数 f(字符串 名)\n结束", "名", ast.TypeString},
		{"函数 f(任意 x)\n结束", "x", ast.TypeAny},
	}
	for _, c := range cases {
		prog := mustParse(t, c.src)
		fn := prog.Statements[0].(*ast.VarDecl).Value.(*ast.FunctionLiteral)
		if len(fn.Params) != 1 {
			t.Fatalf("%s：应有 1 个参数，实际 %d", c.src, len(fn.Params))
		}
		if fn.Params[0].Name != c.name || fn.Params[0].Type != c.typ {
			t.Fatalf("%s：参数解析错误 %+v", c.src, fn.Params[0])
		}
	}
}

func TestTypeFirstVariadic(t *testing.T) {
	src := "函数 f(整数... 数字们)\n结束"
	prog := mustParse(t, src)
	fn := prog.Statements[0].(*ast.VarDecl).Value.(*ast.FunctionLiteral)
	p := fn.Params[0]
	if !p.Variadic || p.Name != "数字们" || p.Type != ast.TypeInt {
		t.Fatalf("可变参数解析错误：%+v", p)
	}
}

// 任意 是显式动态类型占位，解析为未标注类型（TypeAny），不参与类型检查。
func TestAnyKeywordResolvesToTypeAny(t *testing.T) {
	src := "函数 f(任意 x)\n结束"
	prog := mustParse(t, src)
	fn := prog.Statements[0].(*ast.VarDecl).Value.(*ast.FunctionLiteral)
	if fn.Params[0].Type != ast.TypeAny {
		t.Fatalf("任意 应解析为 TypeAny，实际 %q", fn.Params[0].Type)
	}
}

// 参数位置不允许匿名联合，必须先用 定义 类型 起别名。
func TestAnonymousUnionParamRejected(t *testing.T) {
	expectError(t, "函数 f(整数 或 字符串 x)\n结束")
}

// 方法参数同样使用新语序。
func TestMethodTypeFirstParams(t *testing.T) {
	src := "定义 矩形 的 方法 放大(整数 倍数)\n结束"
	prog := mustParse(t, src)
	method := prog.Statements[0].(*ast.MethodDef)
	if len(method.Function.Params) != 1 ||
		method.Function.Params[0].Name != "倍数" ||
		method.Function.Params[0].Type != ast.TypeInt {
		t.Fatalf("方法参数解析错误：%+v", method.Function.Params)
	}
}

func TestNativeFunctionDecl(t *testing.T) {
	src := `定义 本地("打印") 函数 输出(任意... 内容)
    返回 空
结束`
	prog := mustParse(t, src)
	fn := prog.Statements[0].(*ast.VarDecl).Value.(*ast.FunctionLiteral)
	if fn.Native == nil {
		t.Fatalf("应解析出本地绑定")
	}
	if fn.Native.Spec != "打印" || fn.Native.Name != "打印" || fn.Native.Module != "" {
		t.Fatalf("本地绑定解析错误：%+v", fn.Native)
	}
}

func TestNativeDeclModuleSplit(t *testing.T) {
	src := `定义 本地("正则.匹配") 函数 匹配(字符串 文本) -> 布尔
    返回 真
结束`
	prog := mustParse(t, src)
	fn := prog.Statements[0].(*ast.VarDecl).Value.(*ast.FunctionLiteral)
	if fn.Native == nil || fn.Native.Module != "正则" || fn.Native.Name != "匹配" {
		t.Fatalf("带模块的本地绑定解析错误：%+v", fn.Native)
	}
}

func TestNativeDeclHardRules(t *testing.T) {
	// 本地 后必须有括号
	expectError(t, `定义 本地 函数 f()\n结束`)
	// 括号内不能为空
	expectError(t, `定义 本地() 函数 f()\n结束`)
	// 只能有一个字符串参数
	expectError(t, `定义 本地("a", "b") 函数 f()\n结束`)
	// 括号内不能是变量
	expectError(t, `定义 本地(x) 函数 f()\n结束`)
	// 本地 声明后必须是 函数
	expectError(t, `定义 本地("打印") 模型 甲\n结束`)
}

package evaluator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func testdataDir() string {
	abs, _ := filepath.Abs("testdata")
	return abs
}

func moduleTestEnv() *Environment {
	dir := testdataDir()
	env := NewEnvironment()
	env.BaseDir = dir
	env.SetLoader(NewLoader(dir))
	return env
}

func runModuleSource(t *testing.T, src string) (object.Object, error) {
	t.Helper()
	prog, errs := parser.ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析失败：%v", errs)
	}
	return Eval(prog, moduleTestEnv())
}

var _ ast.Statement

func TestModuleImportFunction(t *testing.T) {
	src := "导入 \"lib/工具\"\n工具.加一(41)"
	res, err := runModuleSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "42" {
		t.Fatalf("模块函数调用错误：%s", res.Inspect())
	}
}

func TestModuleImportVar(t *testing.T) {
	src := "导入 \"lib/工具\"\n工具.名"
	res, err := runModuleSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "库" {
		t.Fatalf("模块变量读取错误：%s", res.Inspect())
	}
}

func TestModuleImportClass(t *testing.T) {
	src := "导入 \"lib/工具\"\n变量 x = 工具.项()\nx.值 = 5\nx.值"
	res, err := runModuleSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "5" {
		t.Fatalf("模块模型实例化错误：%s", res.Inspect())
	}
}

func TestModuleImportPrimitiveMethod(t *testing.T) {
	src := "导入 \"lib/工具\"\n\"x\".库标记()"
	res, err := runModuleSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "x-库" {
		t.Fatalf("模块原生方法调用错误：%s", res.Inspect())
	}
}

func TestModuleUnexportedHidden(t *testing.T) {
	src := "导入 \"lib/工具\"\n工具.私有函数()"
	_, err := runModuleSource(t, src)
	if err == nil {
		t.Fatal("未导出的函数应不可访问")
	}
}

func TestModuleNotFound(t *testing.T) {
	src := "导入 \"lib/不存在\""
	_, err := runModuleSource(t, src)
	if err == nil {
		t.Fatal("找不到模块应报错")
	}
}

func TestModuleCached(t *testing.T) {
	env := moduleTestEnv()
	m1, err := env.Loader.Load(testdataDir(), "lib/工具")
	if err != nil {
		t.Fatalf("加载失败：%v", err)
	}
	m2, err := env.Loader.Load(testdataDir(), "lib/工具")
	if err != nil {
		t.Fatalf("第二次加载失败：%v", err)
	}
	if m1 != m2 {
		t.Fatal("重复导入应返回缓存的同一模块")
	}
}

func TestModuleCircularImport(t *testing.T) {
	src := "导入 \"lib/a\""
	_, err := runModuleSource(t, src)
	if err == nil {
		t.Fatal("循环导入应报错")
	}
}

func TestModuleChineseExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "工具.凡")
	if err := os.WriteFile(path, []byte("公开 变量 名 = \"库\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	loader := NewLoader(dir)
	mod, err := loader.Load(dir, "工具")
	if err != nil {
		t.Fatalf("加载失败：%v", err)
	}
	if mod.Exports["名"].Inspect() != "库" {
		t.Fatalf("模块内容错误：%s", mod.Exports["名"].Inspect())
	}
}

func TestModuleExplicitChineseExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "工具.凡")
	if err := os.WriteFile(path, []byte("公开 变量 名 = \"库\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	loader := NewLoader(dir)
	mod, err := loader.Load(dir, "工具.凡")
	if err != nil {
		t.Fatalf("加载失败：%v", err)
	}
	if mod.Exports["名"].Inspect() != "库" {
		t.Fatalf("模块内容错误：%s", mod.Exports["名"].Inspect())
	}
}

func TestModuleLegacyExtensionStillSupported(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "工具.fan")
	if err := os.WriteFile(path, []byte("公开 变量 名 = \"旧\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	loader := NewLoader(dir)
	mod, err := loader.Load(dir, "工具")
	if err != nil {
		t.Fatalf("加载失败：%v", err)
	}
	if mod.Exports["名"].Inspect() != "旧" {
		t.Fatalf("模块内容错误：%s", mod.Exports["名"].Inspect())
	}
}

func TestModuleResolvesFromConfiguredSdkDir(t *testing.T) {
	base := t.TempDir()
	sdkDir := t.TempDir()
	path := filepath.Join(sdkDir, "基础.凡")
	if err := os.WriteFile(path, []byte("公开 变量 版本 = 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	loader := NewLoader(sdkDir)
	mod, err := loader.Load(base, "基础")
	if err != nil {
		t.Fatalf("应从 SDK 目录加载：%v", err)
	}
	if mod.Exports["版本"].Inspect() != "1" {
		t.Fatalf("SDK 模块内容错误：%s", mod.Exports["版本"].Inspect())
	}
}

func writeTempModule(t *testing.T, name, src string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name+".凡")
	if err := os.WriteFile(path, []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestCrossModuleMixfixFunction(t *testing.T) {
	dir := writeTempModule(t, "数学工具", `
公开 函数 求(数组 数列)的最大值() -> 整数
    变量 结果 = 数列[0]
    遍历 数列 中的 值
        如果 值 > 结果 那么 结果 = 值 结束
    结束
    返回 结果
结束
`)
	src := "导入 \"" + filepath.Join(dir, "数学工具") + "\" 作为 工具\n工具.求([3, 9, 2])的最大值()"
	res, err := runModuleSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Inspect() != "9" {
		t.Fatalf("跨模块分段函数结果错误：%s", res.Inspect())
	}
}

func TestCrossModuleMixfixMustBePublic(t *testing.T) {
	dir := writeTempModule(t, "数学工具", `
函数 求(数组 数列)的最大值() -> 整数
    返回 数列[0]
结束
`)
	src := "导入 \"" + filepath.Join(dir, "数学工具") + "\" 作为 工具\n工具.求([3, 9])的最大值()"
	if _, err := runModuleSource(t, src); err == nil {
		t.Fatal("非公开分段函数不应跨模块调用")
	}
}

func TestThroughModuleNormalFunction(t *testing.T) {
	dir := writeTempModule(t, "数学工具", `
公开 函数 最大值(数组 数列) -> 整数
    变量 结果 = 数列[0]
    遍历 数列 中的 值
        如果 值 > 结果 那么 结果 = 值 结束
    结束
    返回 结果
结束
`)
	src := "导入 \"" + filepath.Join(dir, "数学工具") + "\" 作为 工具\n通过 工具 最大值([5, 1, 8])"
	res, err := runModuleSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Inspect() != "8" {
		t.Fatalf("通过 模块 普通函数结果错误：%s", res.Inspect())
	}
}

func TestThroughModuleMixfixFunction(t *testing.T) {
	dir := writeTempModule(t, "数学工具", `
公开 函数 求(数组 数列)的最大值() -> 整数
    返回 数列[0]
结束
`)
	src := "导入 \"" + filepath.Join(dir, "数学工具") + "\" 作为 工具\n通过 工具 求([4, 1])的最大值()"
	res, err := runModuleSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Inspect() != "4" {
		t.Fatalf("通过 模块 分段函数结果错误：%s", res.Inspect())
	}
}

func TestThroughModuleRejectsPrivate(t *testing.T) {
	dir := writeTempModule(t, "数学工具", `
函数 内部() -> 整数
    返回 1
结束
`)
	src := "导入 \"" + filepath.Join(dir, "数学工具") + "\" 作为 工具\n通过 工具 内部()"
	if _, err := runModuleSource(t, src); err == nil {
		t.Fatal("通过 模块 不应访问非公开函数")
	}
}

func TestImportBareNameAccess(t *testing.T) {
	dir := writeTempModule(t, "运算", `
公开 变量 版本 = "1.0"
公开 函数 加倍(整数 n) -> 整数
    返回 n 乘 2
结束
公开 模型 圆
    属性 半径
结束
`)
	src := "导入 \"" + filepath.Join(dir, "运算") + "\" 作为 运算模块\n" +
		"[加倍(21), 版本, 圆{半径: 3}.半径, 运算模块.加倍(5)]"
	res, err := runModuleSource(t, src)
	if err != nil {
		t.Fatalf("裸名访问失败：%v", err)
	}
	if res.Inspect() != "[42, 1.0, 3, 10]" {
		t.Fatalf("裸名访问结果错误：%s", res.Inspect())
	}
}

func TestImportLocalShadowsMember(t *testing.T) {
	dir := writeTempModule(t, "运算", `
公开 函数 加倍(整数 n) -> 整数
    返回 n 乘 2
结束
`)
	src := "导入 \"" + filepath.Join(dir, "运算") + "\" 作为 运算模块\n" +
		"函数 加倍(整数 n) -> 整数\n    返回 n 加 1\n结束\n" +
		"[加倍(10), 运算模块.加倍(10)]"
	res, err := runModuleSource(t, src)
	if err != nil {
		t.Fatalf("遮蔽访问失败：%v", err)
	}
	if res.Inspect() != "[11, 20]" {
		t.Fatalf("局部遮蔽结果错误：%s", res.Inspect())
	}
}

func TestImportAmbiguousBareNameErrors(t *testing.T) {
	dirA := writeTempModule(t, "甲", `
公开 函数 共同() -> 整数
    返回 1
结束
`)
	dirB := writeTempModule(t, "乙", `
公开 函数 共同() -> 整数
    返回 2
结束
`)
	src := "导入 \"" + filepath.Join(dirA, "甲") + "\" 作为 甲模块\n" +
		"导入 \"" + filepath.Join(dirB, "乙") + "\" 作为 乙模块\n" +
		"共同()"
	if _, err := runModuleSource(t, src); err == nil || !strings.Contains(err.Error(), "冲突") {
		t.Fatalf("歧义裸名应报冲突，实际：%v", err)
	}
	// 限定访问不受歧义影响。
	okSrc := "导入 \"" + filepath.Join(dirA, "甲") + "\" 作为 甲模块\n" +
		"导入 \"" + filepath.Join(dirB, "乙") + "\" 作为 乙模块\n" +
		"[甲模块.共同(), 乙模块.共同()]"
	res, err := runModuleSource(t, okSrc)
	if err != nil {
		t.Fatalf("限定访问不应失败：%v", err)
	}
	if res.Inspect() != "[1, 2]" {
		t.Fatalf("限定访问结果错误：%s", res.Inspect())
	}
}

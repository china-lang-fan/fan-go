package evaluator

import (
	"testing"

	"fan/internal/object"
)

// 本地绑定命中已注册原生函数时，调用走原生实现而非函数体。
func TestNativeBindingHitsNative(t *testing.T) {
	RegisterNativeFunction("g2.翻倍", func(_ *Runtime, args []object.Object) ([]object.Object, error) {
		n := args[0].(*object.Integer)
		return []object.Object{&object.Integer{Value: n.Value * 2}}, nil
	})
	src := `定义 本地("g2.翻倍") 函数 翻倍(整数 n) -> 整数
    返回 n 加 100
结束
翻倍(21)`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if got := res.(*object.Integer).Value; got != 42 {
		t.Fatalf("应走原生实现得 42，实际 %d", got)
	}
}

// 未注册的本地绑定不报错，静默回退执行函数体。
func TestNativeBindingFallbackToBody(t *testing.T) {
	src := `定义 本地("g2.不存在的函数") 函数 加倍(整数 n) -> 整数
    返回 n 加 n
结束
加倍(5)`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("未注册本地绑定应静默回退函数体，却报错：%v", err)
	}
	if got := res.(*object.Integer).Value; got != 10 {
		t.Fatalf("应回退函数体得 10，实际 %d", got)
	}
}

// 带模块前缀的本地绑定按完整 spec 命中扁平 map（如内置 测试.加一）。
func TestNativeBindingModulePrefixHit(t *testing.T) {
	src := `定义 本地("测试.加一") 函数 加一(整数 n) -> 整数
    返回 0
结束
加一(41)`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if got := res.(*object.Integer).Value; got != 42 {
		t.Fatalf("模块前缀绑定应命中原生命中得 42，实际 %d", got)
	}
}

// 无可变参的普通本地绑定命中后正常传参。
func TestNativeBindingNoModulePrefix(t *testing.T) {
	src := `定义 本地("打印") 函数 输出(任意... 内容)
    返回 空
结束
输出("hi")`
	if _, err := runSource(t, src); err != nil {
		t.Fatalf("内置 打印 本地绑定运行错误：%v", err)
	}
}

package evaluator

import (
	"bytes"
	"testing"
)

func TestTagTooManyPositionalArgs(t *testing.T) {
	src := `模型 标记
    属性 值
结束
@标记("一", "二")
定义 函数 测试()
结束`
	_, err := runSource(t, src)
	if err == nil {
		t.Fatal("多个未命名参数应报错")
	}
}

func TestTagMissingValueField(t *testing.T) {
	src := `模型 标记
    属性 名称
结束
@标记("值")
定义 函数 测试()
结束`
	_, err := runSource(t, src)
	if err == nil {
		t.Fatal("没有 值 字段时不应接受未命名参数")
	}
}

func TestUnknownTagSilentlyIgnored(t *testing.T) {
	src := `@不存在的标签
定义 函数 测试()
    返回 7
结束
测试()`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("未注册标签应静默忽略，却报错：%v", err)
	}
	if res.Inspect() != "7" {
		t.Fatalf("应正常执行函数体得 7，实际 %s", res.Inspect())
	}
}

func TestFanTagHandler(t *testing.T) {
	src := `模型 自定义
    属性 值
结束
函数 新实现()
    返回 "来自凡语言"
结束
函数 自定义处理(任意 上下文)
    上下文.替换实现(新实现)
结束
注册标签处理("自定义", 自定义处理)
@自定义("参数")
定义 函数 测试()
    返回 空
结束
测试()`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "来自凡语言" {
		t.Fatalf("返回值错误：%s", res.Inspect())
	}
}

func TestTagInstanceFields(t *testing.T) {
	src := `模型 路由信息
    属性 值
    属性 请求方法
结束
函数 处理路由(任意 上下文)
    变量 结果 = 函数()
        返回 上下文.标签.请求方法 + ":" + 上下文.标签.值
    结束
    上下文.替换实现(结果)
结束
注册标签处理("路由信息", 处理路由)
@路由信息("/用户", 请求方法 = "POST")
定义 函数 创建用户()
    返回 空
结束
创建用户()`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "POST:/用户" {
		t.Fatalf("标签字段不符：%s", res.Inspect())
	}
}

func TestFanTagHandlerReadsTag(t *testing.T) {
	src := `模型 自定义
    属性 值
结束
函数 自定义处理(任意 上下文)
    变量 结果 = 函数()
        返回 上下文.标签.值
    结束
    上下文.替换实现(结果)
结束
注册标签处理("自定义", 自定义处理)
@自定义("标签值")
定义 函数 测试()
    返回 空
结束
测试()`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "标签值" {
		t.Fatalf("返回值错误：%s", res.Inspect())
	}
}

func TestFanMethodHandlerReceivesInstance(t *testing.T) {
	src := `模型 自定义
    属性 值
结束
模型 用户
    属性 名称
结束
函数 新实现(任意 实例， 整数 基数)
    如果 实例.名称 == "小明" 那么
        返回 基数
    结束
    返回 0
结束
函数 处理自定义(任意 上下文)
    上下文.替换实现(新实现)
结束
注册标签处理("自定义", 处理自定义)
@自定义
定义 用户 的 方法 编号(整数 基数)
    返回 空
结束
变量 用户实例 = 用户()
用户实例.名称 = "小明"
用户实例.编号(42)`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "42" {
		t.Fatalf("方法返回值错误：%s", res.Inspect())
	}
}

func TestFanTagHandlerReadsCurrentCallable(t *testing.T) {
	var buf bytes.Buffer
	src := `模型 自定义
    属性 值
结束
函数 自定义处理(任意 上下文)
    变量 原实现 = 上下文.当前实现
    打印(原实现)
结束
注册标签处理("自定义", 自定义处理)
@自定义
定义 函数 测试()
    返回 空
结束
测试()`
	if _, _, err := runSourceWithOutput(t, src, &buf); err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if buf.String() != "<函数 测试>\n" {
		t.Fatalf("当前实现读取错误：%q", buf.String())
	}
}

func TestSecondTagReceivesReplacedCallable(t *testing.T) {
	src := `模型 第一标记
    属性 值
结束
模型 第二标记
    属性 值
结束
函数 第一处理(任意 上下文)
    变量 新实现 = 函数(整数 基数)
        返回 基数 + 10
    结束
    上下文.替换实现(新实现)
结束
函数 第二处理(任意 上下文)
    变量 原实现 = 上下文.当前实现
    变量 新实现 = 函数(整数 基数)
        返回 原实现(基数) + 1
    结束
    上下文.替换实现(新实现)
结束
注册标签处理("第一标记", 第一处理)
注册标签处理("第二标记", 第二处理)
@第一标记
@第二标记
定义 函数 测试(整数 基数) -> 整数
    返回 基数
结束
测试(31)`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "42" {
		t.Fatalf("连续标签返回值错误：%s", res.Inspect())
	}
}

func TestTagCreationDoesNotCallInit(t *testing.T) {
	src := `模型 标记
    属性 值
结束
定义 标记 的 方法 初始化()
    错误("不应调用初始化")
结束
@标记("测试")
定义 函数 测试()
    返回 通行
结束
变量 通行 = 真
测试()`
	mustBool(t, src, true)
}

func TestTagWithoutHandlerHasNoEffect(t *testing.T) {
	src := `模型 标记
结束
@标记
定义 函数 测试()
    返回 42
结束
测试()`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "42" {
		t.Fatalf("返回值错误：%s", res.Inspect())
	}
}

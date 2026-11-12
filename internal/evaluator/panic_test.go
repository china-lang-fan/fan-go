package evaluator

import (
	"strings"
	"testing"

	"fan/internal/object"
)

func TestDeferLIFO(t *testing.T) {
	src := strings.Join([]string{
		`函数 f()`,
		`    稍后 追加记录("第二")`,
		`    稍后 追加记录("第一")`,
		`结束`,
		`变量 记录 = []`,
		`函数 追加记录(字符串 文本)`,
		`    记录 = 追加(记录, 文本)`,
		`结束`,
		`f()`,
		`记录`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != `[第一, 第二]` {
		t.Fatalf("稍后应逆序执行，实际 %s", res.Inspect())
	}
}

func TestCrashRecover(t *testing.T) {
	src := strings.Join([]string{
		`函数 f()`,
		`    稍后 函数()`,
		`        全局结果 = 恢复().消息`,
		`    结束`,
		`    崩溃了("失败")`,
		`结束`,
		`变量 全局结果 = 空`,
		`f()`,
		`全局结果`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != `失败` {
		t.Fatalf("应恢复崩溃错误的消息，实际 %s", res.Inspect())
	}
}

func TestCrashBuildsErrorObject(t *testing.T) {
	src := strings.Join([]string{
		`函数 f()`,
		`    稍后 函数()`,
		`        全局错误 = 恢复()`,
		`    结束`,
		`    崩溃了("磁盘满")`,
		`结束`,
		`变量 全局错误 = 空`,
		`f()`,
		`全局错误.错误码`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != `` {
		t.Fatalf("崩溃错误默认错误码应为空串，实际 %s", res.Inspect())
	}
}

func TestDeferImmediateArguments(t *testing.T) {
	src := strings.Join([]string{
		`变量 结果 = ""`,
		`函数 记录(字符串 文本)`,
		`    结果 = 文本`,
		`结束`,
		`函数 f()`,
		`    变量 文本 = "旧"`,
		`    稍后 记录(文本)`,
		`    文本 = "新"`,
		`结束`,
		`f()`,
		`结果`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != `旧` {
		t.Fatalf("稍后参数应立即求值，实际 %s", res.Inspect())
	}
}

func TestNestedCrashRecover(t *testing.T) {
	src := strings.Join([]string{
		`函数 内层()`,
		`    崩溃了("深层失败")`,
		`结束`,
		`函数 外层()`,
		`    稍后 函数()`,
		`        全局结果 = 恢复().消息`,
		`    结束`,
		`    内层()`,
		`结束`,
		`变量 全局结果 = 空`,
		`外层()`,
		`全局结果`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != `深层失败` {
		t.Fatalf("应恢复嵌套崩溃，实际 %s", res.Inspect())
	}
}

func TestRecoverOnlyInDefer(t *testing.T) {
	src := strings.Join([]string{
		`函数 f()`,
		`    返回 恢复()`,
		`结束`,
		`f()`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Kind() != object.KindNil {
		t.Fatalf("非稍后中的恢复应为空，实际 %s", res.Inspect())
	}
}

func TestUnrecoveredCrash(t *testing.T) {
	src := `崩溃了("致命")`
	if _, err := runSource(t, src); err == nil {
		t.Fatal("未恢复的崩溃应返回错误")
	}
}

func TestCrashArgCount(t *testing.T) {
	if _, err := runSource(t, `崩溃了()`); err == nil {
		t.Fatal("无参崩溃了应报错")
	}
	if _, err := runSource(t, `崩溃了("a", "b")`); err == nil {
		t.Fatal("多参崩溃了应报错")
	}
}

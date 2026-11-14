package parser

import (
	"testing"

	"fan/internal/ast"
)

func TestSwitchValueBranches(t *testing.T) {
	src := `判断 x
    为 1、2
        打印(x)
    其他
        打印(0)
结束`
	prog := mustParse(t, src)
	stmt, ok := prog.Statements[0].(*ast.SwitchStmt)
	if !ok {
		t.Fatalf("应为判断语句，实际 %T", prog.Statements[0])
	}
	if len(stmt.Branches) != 2 {
		t.Fatalf("应有两个分支，实际 %d", len(stmt.Branches))
	}
	if len(stmt.Branches[0].Values) != 2 {
		t.Fatalf("首分支应有两个匹配值")
	}
	if !stmt.Branches[1].Default {
		t.Fatalf("第二分支应是其他分支")
	}
}

func TestSwitchGuardBranches(t *testing.T) {
	src := `判断
    当 x >= 1
        打印(x)
结束`
	prog := mustParse(t, src)
	stmt := prog.Statements[0].(*ast.SwitchStmt)
	if stmt.Subject != nil {
		t.Fatalf("条件判断不应有匹配对象")
	}
	if !stmt.Branches[0].Guard {
		t.Fatalf("首分支应是条件分支")
	}
}

func TestSwitchInvalidBranches(t *testing.T) {
	expectError(t, "判断\n    为 1\n结束")
	expectError(t, "判断 x\n    其他\n        打印(0)\n    为 1\n结束")
	expectError(t, "判断 x\n    为 1")
}

func TestSwitchCaseThenSingleLine(t *testing.T) {
	src := `判断 状态码
    为 200 则 打印("成功")
    为 301、302 时 打印("重定向")
    其他 则 打印("其他")
结束`
	prog := mustParse(t, src)
	stmt := prog.Statements[0].(*ast.SwitchStmt)
	if len(stmt.Branches) != 3 {
		t.Fatalf("应有三个分支，实际 %d", len(stmt.Branches))
	}
	for i, b := range stmt.Branches {
		if len(b.Body.Statements) != 1 {
			t.Fatalf("分支 %d 的则/时单行体应只有一条语句，实际 %d", i, len(b.Body.Statements))
		}
	}
	if len(stmt.Branches[1].Values) != 2 {
		t.Fatalf("时 引导同样应支持多值")
	}
	if !stmt.Branches[2].Default {
		t.Fatalf("第三分支应是其他分支")
	}
}

func TestSwitchCaseThenMultilineBlock(t *testing.T) {
	src := `判断 甲
    为 1 则
        打印("一")
        打印("二")
    其他
        打印("其他")
结束`
	prog := mustParse(t, src)
	stmt := prog.Statements[0].(*ast.SwitchStmt)
	if len(stmt.Branches[0].Body.Statements) != 2 {
		t.Fatalf("则 后换行应支持多行块，实际 %d 条", len(stmt.Branches[0].Body.Statements))
	}
}

func TestSwitchGuardCaseThen(t *testing.T) {
	src := `判断
    当 甲 > 5 则 打印("大")
    其他 则 打印("小")
结束`
	prog := mustParse(t, src)
	stmt := prog.Statements[0].(*ast.SwitchStmt)
	if !stmt.Branches[0].Guard {
		t.Fatalf("首分支应是条件守卫分支")
	}
	if len(stmt.Branches[0].Body.Statements) != 1 {
		t.Fatalf("则 单行体应只有一条语句")
	}
}

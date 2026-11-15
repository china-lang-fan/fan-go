package object

import (
	"fmt"
	"sync/atomic"
)

// HandleKind 标识句柄指向的资源种类，仅用于分类展示与 FFI 模块分派校验。
type HandleKind string

const (
	HandleKindLibrary HandleKind = "库"
	HandleKindSymbol  HandleKind = "符号"
	HandleKindMemory  HandleKind = "内存"
)

// 句柄 id 的进程内单调来源。句柄可能被复制（浅拷贝共享同一登记项），
// id 在构造时一次性分配，复制体保留相同 id，与 FFI 注册表里的同一项对应。
var handleSeq atomic.Uint64

func nextHandleID() uint64 {
	return handleSeq.Add(1)
}

// Handle 是不透明句柄值：承载动态库、函数符号、原生内存等宿主资源。
//
// 它在 object 层只保存种类标记与唯一 id，不持有任何裸指针——真正的 OS
// 资源由 FFI 内建按 id 登记管理，从而让对象层不依赖平台/第三方库。
// 句柄不参与指针算术，也不与整数互转，只能通过「动态库」模块的内建操作。
type Handle struct {
	KindTag HandleKind
	ID      uint64
}

// NewHandle 构造指定种类的新句柄，自动分配唯一 id。
func NewHandle(kind HandleKind) *Handle {
	return &Handle{KindTag: kind, ID: nextHandleID()}
}

func (h *Handle) Kind() Kind { return KindHandle }

// Inspect 不暴露可参与运算的地址，只给种类与 id，便于排查又无法被误用为指针。
func (h *Handle) Inspect() string {
	return fmt.Sprintf("句柄(%s#%d)", h.KindTag, h.ID)
}

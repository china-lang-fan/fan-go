package evaluator

// 动态库 FFI 内建：让凡程序加载并调用外部 C 动态库。
//
// 本文件是 Go 端 FFI 的 unsafe/平台边界集中地，对应 Rust 端
// fan-native/src/ext/ffi.rs。凡语言只见不透明句柄（object.Handle，仅含种类与
// 唯一 id），真正的宿主资源登记在本文件的进程内注册表里，按 id 索引。
//
// 实现依赖 purego（纯 Go，CGO_ENABLED=0 即可在 Linux/macOS/Windows 加载动态库），
// 不引入 cgo。设计约束与 Rust 端一致：句柄不透明、不做指针算术、不与整数互转、
// 原生内存需显式释放；一期仅支持标量、C 字符串、裸内存与不透明句柄。

import (
	"fmt"
	"math"
	"sync"
	"unsafe"

	"fan/internal/object"

	"github.com/ebitengine/purego"
)

const ffiModule = "ffi"

// cType 是受支持的 C 参数/返回类型白名单。
type cType int

const (
	ctInt cType = iota
	ctFloat
	ctByte
	ctPtr
	ctVoid
)

func parseCType(name string) (cType, error) {
	switch name {
	case "整数", "int", "long":
		return ctInt, nil
	case "小数", "double", "float":
		return ctFloat, nil
	case "字节", "byte":
		return ctByte, nil
	case "句柄", "指针", "ptr", "handle":
		return ctPtr, nil
	case "空", "void":
		return ctVoid, nil
	default:
		return ctInt, fmt.Errorf("不支持的 FFI 类型：%s", name)
	}
}

// symEntry 缓存函数符号地址与类型签名。
type symEntry struct {
	addr   uintptr
	ret    cType
	params []cType
}

// memEntry 是一块原生内存缓冲。
type memEntry struct {
	// 用 unsafe.Pointer 而非 uintptr 承载地址：Pointer 会被 GC 视为有效引用，
	// 且 vet 认可 Pointer 的直接解引用；仅在发起系统调用的边界转为 uintptr。
	ptr     unsafe.Pointer
	len     int
	cstring bool
	// Go 后备数组：alloc/cstring 用 make 分配，由它保活防止 GC 回收；
	// free 时只需从注册表摘除，Go GC 自动回收，绝不跨分配器释放。
	backing []byte
}

// ffiRegistry 登记已加载库、函数符号与原生内存。库句柄用 uintptr 持有，
// purego 的 Dlopen 返回的句柄在 Dlclose 前有效；这里显式保活直到关闭。
type ffiRegistry struct {
	mu   sync.Mutex
	libs map[uint64]uintptr
	syms map[uint64]*symEntry
	mems map[uint64]*memEntry
}

var ffiReg = &ffiRegistry{
	libs: map[uint64]uintptr{},
	syms: map[uint64]*symEntry{},
	mems: map[uint64]*memEntry{},
}

func asHandle(o object.Object) (*object.Handle, error) {
	h, ok := o.(*object.Handle)
	if !ok {
		return nil, fmt.Errorf("参数必须是句柄，实际为 %s", o.Kind())
	}
	return h, nil
}

// registerMem 登记一块原生内存并返回内存句柄。
func (r *ffiRegistry) registerMem(p unsafe.Pointer, n int, cstring bool) *object.Handle {
	r.mu.Lock()
	defer r.mu.Unlock()
	h := object.NewHandle(object.HandleKindMemory)
	r.mems[h.ID] = &memEntry{ptr: p, len: n, cstring: cstring}
	return h
}

// registerMemGo 登记一块由 Go 切片支撑的缓冲，保留切片引用以防 GC 回收。
func (r *ffiRegistry) registerMemGo(backing []byte, p unsafe.Pointer, n int) *object.Handle {
	r.mu.Lock()
	defer r.mu.Unlock()
	h := object.NewHandle(object.HandleKindMemory)
	r.mems[h.ID] = &memEntry{ptr: p, len: n, backing: backing}
	return h
}

func (r *ffiRegistry) memPtr(id uint64) (unsafe.Pointer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.mems[id]
	if !ok {
		return nil, fmt.Errorf("内存句柄无效或已释放")
	}
	return e.ptr, nil
}

// handleToPointer 返回句柄对应的 unsafe.Pointer（用于内存读写场景，全程保持
// Pointer 类型以通过 vet 保活检查）。
func handleToPointer(o object.Object) (unsafe.Pointer, error) {
	if o == object.Null {
		return nil, nil
	}
	h, ok := o.(*object.Handle)
	if !ok {
		return nil, fmt.Errorf("指针参数必须是句柄或空，实际为 %s", o.Kind())
	}
	if h.KindTag != object.HandleKindMemory {
		return nil, fmt.Errorf("只有内存句柄能作为指针参数传递")
	}
	return ffiReg.memPtr(h.ID)
}

// handleToPtr 把句柄/空转为 C 指针参数（系统调用边界用的 uintptr）。
func handleToPtr(o object.Object) (uintptr, error) {
	p, err := handleToPointer(o)
	if err != nil {
		return 0, err
	}
	return uintptr(p), nil
}

func ffiOk(v object.Object) []object.Object {
	return []object.Object{v, object.Null}
}

func ffiFail(msg string) []object.Object {
	return []object.Object{object.Null, object.NewError(msg)}
}

// pointerFromCall 把系统调用返回的 uintptr 地址转为 unsafe.Pointer。
//
// 这是 FFI 边界上唯一的 uintptr→Pointer 转换：结果立即写入注册表（memEntry.ptr
// 以 unsafe.Pointer 持有）从而被 GC 视为有效引用，不存在 uintptr 临时值跨越
// 函数调用而失效的窗口。封装为独立函数以便集中审计这一处边界转换。
func pointerFromCall(addr uintptr) unsafe.Pointer {
	return unsafe.Pointer(addr) //nolint:govet // FFI 返回地址，立即由注册表保活
}

// marshalArg 把凡值按声明 C 类型封送为 uintptr 调用载体。
func marshalArg(o object.Object, ty cType) (uintptr, error) {
	o = unwrapNamed(o)
	switch ty {
	case ctInt:
		i, ok := o.(*object.Integer)
		if !ok {
			return 0, fmt.Errorf("整数参数实际为 %s", o.Kind())
		}
		return uintptr(i.Value), nil
	case ctByte:
		switch v := o.(type) {
		case *object.Byte:
			return uintptr(v.Value), nil
		case *object.Integer:
			if v.Value < 0 || v.Value > 255 {
				return 0, fmt.Errorf("字节超出范围：%d", v.Value)
			}
			return uintptr(v.Value), nil
		default:
			return 0, fmt.Errorf("字节参数实际为 %s", o.Kind())
		}
	case ctFloat:
		f, ok := o.(*object.Float)
		if !ok {
			// 整数自动提升为小数。
			if i, ok := o.(*object.Integer); ok {
				return uintptr(math.Float64bits(float64(i.Value))), nil
			}
			return 0, fmt.Errorf("小数参数实际为 %s", o.Kind())
		}
		return uintptr(math.Float64bits(f.Value)), nil
	case ctPtr:
		return handleToPtr(o)
	default:
		return 0, fmt.Errorf("void 不能作为参数类型")
	}
}

// invoke 按符号签名封送参数并用 SyscallN 单次调用，再按返回类型解包。
func invoke(entry *symEntry, args []object.Object) (object.Object, error) {
	if len(args) != len(entry.params) {
		return nil, fmt.Errorf("参数数量不符：函数需要 %d 个，实际 %d 个",
			len(entry.params), len(args))
	}
	if len(args) > 6 {
		return nil, fmt.Errorf("FFI 一期最多支持 6 个参数，实际 %d 个", len(args))
	}
	uints := make([]uintptr, len(args))
	for i, a := range args {
		v, err := marshalArg(a, entry.params[i])
		if err != nil {
			return nil, err
		}
		uints[i] = v
	}
	// purego 按平台 C ABI 发起调用：整数/指针走整数寄存器，浮点位模式经
	// Float64bits 传入后由 SyscallN 在浮点寄存器还原。
	r1, r2, _ := purego.SyscallN(entry.addr, uints...)
	switch entry.ret {
	case ctVoid:
		return object.Null, nil
	case ctFloat:
		return &object.Float{Value: math.Float64frombits(uint64(r1))}, nil
	case ctByte:
		return &object.Byte{Value: byte(r1)}, nil
	case ctPtr:
		if r1 == 0 {
			return object.Null, nil
		}
		// 系统调用返回的地址立即登记进注册表并由其保活；转换封装在
		// pointerFromCall 内，是 FFI 边界上不可避免的 uintptr→Pointer 转换。
		return ffiReg.registerMem(pointerFromCall(r1), 0, false), nil
	default: // ctInt
		_ = r2
		return &object.Integer{Value: int64(r1)}, nil
	}
}

// --- 内建实现 ---------------------------------------------------------------

func nativeFFIOpen(_ *Runtime, args []object.Object) ([]object.Object, error) {
	path, ok := args[0].(*object.String)
	if !ok {
		return ffiFail("ffi.open 需要字符串路径参数"), nil
	}
	handle, err := purego.Dlopen(path.Value, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return ffiFail("加载动态库失败：" + err.Error()), nil
	}
	ffiReg.mu.Lock()
	h := object.NewHandle(object.HandleKindLibrary)
	ffiReg.libs[h.ID] = handle
	ffiReg.mu.Unlock()
	return ffiOk(h), nil
}

func nativeFFISymbol(_ *Runtime, args []object.Object) ([]object.Object, error) {
	if len(args) < 3 {
		return ffiFail("ffi.symbol 至少需要 库、符号名、返回类型 三个参数"), nil
	}
	libHandle, err := asHandle(args[0])
	if err != nil {
		return ffiFail(err.Error()), nil
	}
	if libHandle.KindTag != object.HandleKindLibrary {
		return ffiFail("ffi.symbol 第一个参数必须是库句柄"), nil
	}
	name, ok := args[1].(*object.String)
	if !ok {
		return ffiFail("符号名必须是字符串"), nil
	}
	retName, ok := args[2].(*object.String)
	if !ok {
		return ffiFail("返回类型必须是字符串"), nil
	}
	ret, err := parseCType(retName.Value)
	if err != nil {
		return ffiFail(err.Error()), nil
	}
	params := make([]cType, 0, len(args)-3)
	for _, a := range args[3:] {
		tn, ok := a.(*object.String)
		if !ok {
			return ffiFail("参数类型必须是字符串"), nil
		}
		t, err := parseCType(tn.Value)
		if err != nil {
			return ffiFail(err.Error()), nil
		}
		if t == ctVoid {
			return ffiFail("void 不能作为参数类型"), nil
		}
		params = append(params, t)
	}
	ffiReg.mu.Lock()
	libAddr, exists := ffiReg.libs[libHandle.ID]
	ffiReg.mu.Unlock()
	if !exists {
		return ffiFail("库句柄无效或已关闭"), nil
	}
	addr, err := purego.Dlsym(libAddr, name.Value)
	if err != nil {
		return ffiFail("找不到符号 " + name.Value + "：" + err.Error()), nil
	}
	h := object.NewHandle(object.HandleKindSymbol)
	ffiReg.mu.Lock()
	ffiReg.syms[h.ID] = &symEntry{addr: addr, ret: ret, params: params}
	ffiReg.mu.Unlock()
	return ffiOk(h), nil
}

func nativeFFICall(_ *Runtime, args []object.Object) ([]object.Object, error) {
	if len(args) < 1 {
		return ffiFail("ffi.call 需要函数句柄参数"), nil
	}
	symHandle, err := asHandle(args[0])
	if err != nil {
		return ffiFail(err.Error()), nil
	}
	if symHandle.KindTag != object.HandleKindSymbol {
		return ffiFail("ffi.call 第一个参数必须是函数句柄"), nil
	}
	ffiReg.mu.Lock()
	entry, ok := ffiReg.syms[symHandle.ID]
	ffiReg.mu.Unlock()
	if !ok {
		return ffiFail("函数句柄无效或已释放"), nil
	}
	result, err := invoke(entry, args[1:])
	if err != nil {
		return ffiFail(err.Error()), nil
	}
	return ffiOk(result), nil
}

func nativeFFIAlloc(_ *Runtime, args []object.Object) ([]object.Object, error) {
	n, ok := args[0].(*object.Integer)
	if !ok {
		return ffiFail("ffi.alloc 需要整数字节数"), nil
	}
	if n.Value <= 0 {
		return ffiFail("分配字节数必须为正数"), nil
	}
	// 用 make 取得由 Go 运行时管理的后备数组，再取其首地址；free 时不释放
	// Go 内存，仅从注册表摘除（Go 的 GC 负责回收）。这避免了跨分配器 free。
	buf := make([]byte, n.Value)
	h := ffiReg.registerMemGo(buf, unsafe.Pointer(&buf[0]), int(n.Value))
	return ffiOk(h), nil
}

func nativeFFIFree(_ *Runtime, args []object.Object) ([]object.Object, error) {
	h, err := asHandle(args[0])
	if err != nil {
		return ffiFail(err.Error()), nil
	}
	if h.KindTag != object.HandleKindMemory {
		return ffiFail("ffi.free 只能释放内存句柄"), nil
	}
	ffiReg.mu.Lock()
	_, ok := ffiReg.mems[h.ID]
	if ok {
		delete(ffiReg.mems, h.ID)
	}
	ffiReg.mu.Unlock()
	if !ok {
		return ffiFail("内存句柄无效或已释放"), nil
	}
	return []object.Object{object.Null}, nil
}

func nativeFFIRead(_ *Runtime, args []object.Object) ([]object.Object, error) {
	h, err := asHandle(args[0])
	if err != nil {
		return ffiFail(err.Error()), nil
	}
	tn, ok := args[1].(*object.String)
	if !ok {
		return ffiFail("类型必须是字符串"), nil
	}
	ty, err := parseCType(tn.Value)
	if err != nil {
		return ffiFail(err.Error()), nil
	}
	if ty == ctVoid {
		return ffiFail("void 不能读取"), nil
	}
	ptr, err := ffiReg.memPtr(h.ID)
	if err != nil {
		return ffiFail(err.Error()), nil
	}
	var v object.Object
	switch ty {
	case ctInt:
		v = &object.Integer{Value: *(*int64)(ptr)}
	case ctFloat:
		v = &object.Float{Value: *(*float64)(ptr)}
	case ctByte:
		v = &object.Byte{Value: *(*byte)(ptr)}
	case ctPtr:
		p := *(*unsafe.Pointer)(ptr)
		if p == nil {
			v = object.Null
		} else {
			v = ffiReg.registerMem(p, 0, false)
		}
	}
	return ffiOk(v), nil
}

func nativeFFIWrite(_ *Runtime, args []object.Object) ([]object.Object, error) {
	h, err := asHandle(args[0])
	if err != nil {
		return ffiFail(err.Error()), nil
	}
	tn, ok := args[1].(*object.String)
	if !ok {
		return ffiFail("类型必须是字符串"), nil
	}
	ty, err := parseCType(tn.Value)
	if err != nil {
		return ffiFail(err.Error()), nil
	}
	ptr, err := ffiReg.memPtr(h.ID)
	if err != nil {
		return ffiFail(err.Error()), nil
	}
	val := unwrapNamed(args[2])
	switch ty {
	case ctInt:
		i, ok := val.(*object.Integer)
		if !ok {
			return ffiFail("整数写入值实际为 " + string(val.Kind())), nil
		}
		*(*int64)(ptr) = i.Value
	case ctFloat:
		f, ok := val.(*object.Float)
		if !ok {
			return ffiFail("小数写入值实际为 " + string(val.Kind())), nil
		}
		*(*float64)(ptr) = f.Value
	case ctByte:
		switch v := val.(type) {
		case *object.Byte:
			*(*byte)(ptr) = v.Value
		case *object.Integer:
			if v.Value < 0 || v.Value > 255 {
				return ffiFail(fmt.Sprintf("字节写入值超出范围：%d", v.Value)), nil
			}
			*(*byte)(ptr) = byte(v.Value)
		default:
			return ffiFail("字节写入值实际为 " + string(val.Kind())), nil
		}
	case ctPtr:
		p, err := handleToPointer(val)
		if err != nil {
			return ffiFail(err.Error()), nil
		}
		*(*unsafe.Pointer)(ptr) = p
	default:
		return ffiFail("void 不能写入"), nil
	}
	return []object.Object{object.Null}, nil
}

func nativeFFICString(_ *Runtime, args []object.Object) ([]object.Object, error) {
	text, ok := args[0].(*object.String)
	if !ok {
		return ffiFail("ffi.cstring 需要字符串参数"), nil
	}
	// 多分配一个字节放结尾 NUL；后备切片由 Go GC 管理，free 仅摘除登记。
	buf := append([]byte(text.Value), 0)
	h := ffiReg.registerMemGo(buf, unsafe.Pointer(&buf[0]), len(buf))
	return ffiOk(h), nil
}

func nativeFFIToCString(_ *Runtime, args []object.Object) ([]object.Object, error) {
	h, err := asHandle(args[0])
	if err != nil {
		return ffiFail(err.Error()), nil
	}
	ptr, err := ffiReg.memPtr(h.ID)
	if err != nil {
		return ffiFail(err.Error()), nil
	}
	// 默认按 NUL 结尾读取（strlen 语义）。
	s := goStringFromC(ptr)
	return ffiOk(&object.String{Value: s}), nil
}

// goStringFromC 从 NUL 结尾的原生缓冲读取 Go 字符串。
func goStringFromC(ptr unsafe.Pointer) string {
	if ptr == nil {
		return ""
	}
	// 先扫描 NUL 长度，再用 unsafe.Slice 一次性取出字节，全程不做 uintptr 运算。
	n := 0
	for *(*byte)(unsafe.Add(ptr, n)) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(ptr), n))
}

func init() {
	RegisterNativeFunction("ffi.open", nativeFFIOpen)
	RegisterNativeFunction("ffi.symbol", nativeFFISymbol)
	RegisterNativeFunction("ffi.call", nativeFFICall)
	RegisterNativeFunction("ffi.alloc", nativeFFIAlloc)
	RegisterNativeFunction("ffi.free", nativeFFIFree)
	RegisterNativeFunction("ffi.read", nativeFFIRead)
	RegisterNativeFunction("ffi.write", nativeFFIWrite)
	RegisterNativeFunction("ffi.cstring", nativeFFICString)
	RegisterNativeFunction("ffi.toCString", nativeFFIToCString)
}

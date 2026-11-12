package evaluator

import (
	"fmt"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/tagsys"
)

type NativeFunction func(rt *Runtime, args []object.Object) ([]object.Object, error)

type nativeCallable struct {
	name string
	rt   *Runtime
	fn   NativeFunction
}

func (c *nativeCallable) Kind() object.Kind { return "函数" }
func (c *nativeCallable) Inspect() string {
	return fmt.Sprintf("<函数 %s>", c.name)
}

func (c *nativeCallable) Call(args []tagsys.Object) ([]tagsys.Object, error) {
	callArgs := make([]object.Object, len(args))
	for i, arg := range args {
		value, ok := arg.(object.Object)
		if !ok || arg == nil {
			value = object.Null
		}
		callArgs[i] = value
	}
	values, err := c.fn(c.rt, callArgs)
	if err != nil {
		return nil, err
	}
	out := make([]tagsys.Object, len(values))
	for i, value := range values {
		out[i] = value
	}
	return out, nil
}

// withRuntime 返回绑定到指定运行实例的原生函数副本。
func (c *nativeCallable) withRuntime(rt *Runtime) *nativeCallable {
	return &nativeCallable{name: c.name, rt: rt, fn: c.fn}
}

type classCallable struct {
	cls *Class
}

func (c *classCallable) Kind() object.Kind { return "函数" }
func (c *classCallable) Inspect() string {
	return fmt.Sprintf("<函数 %s>", c.cls.Name)
}

func (c *classCallable) Call(args []tagsys.Object) ([]tagsys.Object, error) {
	callArgs := make([]object.Object, len(args))
	for i, arg := range args {
		callArgs[i] = arg.(object.Object)
	}
	inst, err := c.cls.instantiate(nil, ast.Position{}, callArgs)
	if err != nil {
		return nil, err
	}
	return []tagsys.Object{inst}, nil
}

type variadicUnwrapCallable struct {
	callable tagsys.Callable
}

func (c *variadicUnwrapCallable) Kind() object.Kind { return "函数" }
func (c *variadicUnwrapCallable) Inspect() string {
	return "<函数 可变参数>"
}

func (c *variadicUnwrapCallable) Call(args []tagsys.Object) ([]tagsys.Object, error) {
	callArgs := make([]tagsys.Object, len(args))
	copy(callArgs, args)
	if len(callArgs) > 0 {
		if elements, ok := callArgs[len(callArgs)-1].(*object.Array); ok {
			callArgs = callArgs[:len(callArgs)-1]
			for _, value := range elements.Elements {
				callArgs = append(callArgs, value)
			}
		}
	}
	return c.callable.Call(callArgs)
}

type fanCallable struct {
	fn *Function
}

func (c *fanCallable) Kind() object.Kind { return "函数" }
func (c *fanCallable) Inspect() string {
	return fmt.Sprintf("<函数 %s>", c.fn.Name)
}

func (c *fanCallable) Call(args []tagsys.Object) ([]tagsys.Object, error) {
	return c.fn.Call(args)
}

type runtimeTagContext struct {
	rt       *Runtime
	tag      object.Object
	target   tagsys.Target
	callable tagsys.Callable
}

func (c *runtimeTagContext) Tag() tagsys.Object { return c.tag }
func (c *runtimeTagContext) Target() tagsys.Target {
	return c.target
}
func (c *runtimeTagContext) Callable() tagsys.Callable {
	return c.callable
}
func (c *runtimeTagContext) ReplaceCallable(callable tagsys.Callable) {
	switch target := c.target.(type) {
	case *Function:
		if target.Owner == nil {
			target.Impl = callable
			return
		}
		target.MethodImpl = callable
	case *Class:
		target.Impl = callable
	}
}

func (c *runtimeTagContext) ReplaceFunction(callable tagsys.Callable) {
	if fn, ok := c.target.(*Function); ok && fn.Owner == nil {
		fn.Impl = callable
	}
}

func (c *runtimeTagContext) ReplaceMethod(callable tagsys.Callable) {
	fn, ok := c.target.(*Function)
	if !ok || fn.Owner == nil {
		return
	}
	fn.MethodImpl = callable
}

func (c *runtimeTagContext) member(name string) (object.Object, error) {
	switch name {
	case "标签", "Tag":
		return c.tag, nil
	case "目标":
		return c.target.(object.Object), nil
	case "当前实现":
		return c.callable.(object.Object), nil
	case "替换实现":
		return &tagContextReplaceFn{ctx: c}, nil
	}
	return nil, fmt.Errorf("标签上下文没有成员 %s", name)
}

type tagContextReplaceFn struct {
	ctx *runtimeTagContext
}

func (f *tagContextReplaceFn) Kind() object.Kind { return "函数" }
func (f *tagContextReplaceFn) Inspect() string {
	return "<函数 替换实现>"
}

func (f *tagContextReplaceFn) Call(args []tagsys.Object) ([]tagsys.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("替换实现 需要 1 个参数")
	}
	var callable tagsys.Callable
	switch value := args[0].(type) {
	case tagsys.Callable:
		callable = value
	case *Function:
		callable = value
	default:
		return nil, fmt.Errorf("替换实现 的参数必须是可调用对象")
	}
	f.ctx.ReplaceCallable(callable)
	return nil, nil
}

func (c *runtimeTagContext) Kind() object.Kind { return "标签上下文" }
func (c *runtimeTagContext) Inspect() string {
	return "<标签上下文>"
}

func (c *runtimeTagContext) 标签() object.Object { return c.tag }
func (c *runtimeTagContext) 目标() tagsys.Target {
	return c.target
}
func (c *runtimeTagContext) 当前实现() tagsys.Callable {
	return c.callable
}
func (c *runtimeTagContext) 替换实现(callable tagsys.Callable) {
	c.ReplaceCallable(callable)
}

// 进程级只读模板：各 *_native.go 的 init() 在启动时注册一次，
// NewRuntime 时复制成每个运行实例私有的可变副本。
var (
	builtinClassTemplate   = map[string][]ast.FieldDecl{}
	nativeFunctionTemplate = map[string]NativeFunction{}
	goTagHandlerTemplate   = map[string]tagsys.Handler{}
)

func cloneBuiltinClasses() map[string]*Class {
	out := make(map[string]*Class, len(builtinClassTemplate))
	for name, fields := range builtinClassTemplate {
		out[name] = &Class{
			Name:    name,
			Fields:  append([]ast.FieldDecl(nil), fields...),
			Methods: map[string]*Function{},
		}
	}
	return out
}

func cloneNativeFunctions(rt *Runtime) map[string]tagsys.Callable {
	out := make(map[string]tagsys.Callable, len(nativeFunctionTemplate))
	for name, fn := range nativeFunctionTemplate {
		out[name] = &nativeCallable{name: name, rt: rt, fn: fn}
	}
	return out
}

func cloneGoTagHandlers() map[string]tagsys.Handler {
	out := make(map[string]tagsys.Handler, len(goTagHandlerTemplate))
	for name, handler := range goTagHandlerTemplate {
		out[name] = handler
	}
	return out
}

// RegisterBuiltinClass 在标签模型模板中登记一个标签模型。
func RegisterBuiltinClass(name string, fields []ast.FieldDecl) *Class {
	builtinClassTemplate[name] = append([]ast.FieldDecl(nil), fields...)
	return &Class{
		Name:    name,
		Fields:  append([]ast.FieldDecl(nil), fields...),
		Methods: map[string]*Function{},
	}
}

// RegisterNativeFunction 在原生函数模板中登记一个内建实现。
func RegisterNativeFunction(name string, fn NativeFunction) {
	nativeFunctionTemplate[name] = fn
}

// RegisterTagHandler 在 Go 标签处理器模板中登记一个处理器。
func RegisterTagHandler(name string, handler tagsys.Handler) {
	goTagHandlerTemplate[name] = handler
}

func init() {
	RegisterBuiltinClass("测试", nil)
	RegisterBuiltinClass("test", nil)
	RegisterNativeFunction("print", nativePrint)
	RegisterNativeFunction("打印", nativePrint)
	// 标签工具模块（fan-sdk/库/标签工具.凡）的注册入口，与 Rust 版
	// fan-ext-tag / 默认回落暴露的原生名保持一致。
	RegisterNativeFunction("tag.tagRegisterHandler", func(rt *Runtime, args []object.Object) ([]object.Object, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("注册标签处理 需要 2 个参数")
		}
		tagName, ok := args[0].(*object.String)
		if !ok {
			return nil, fmt.Errorf("标签名必须是字符串")
		}
		fn, ok := args[1].(*Function)
		if !ok {
			return nil, fmt.Errorf("标签处理器必须是函数")
		}
		rt.registerFanTagHandler(tagName.Value, fn)
		return []object.Object{object.Null}, nil
	})
	RegisterNativeFunction("测试.加一", func(_ *Runtime, args []object.Object) ([]object.Object, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("测试.加一 需要 1 个参数")
		}
		n, ok := args[0].(*object.Integer)
		if !ok {
			return nil, fmt.Errorf("测试.加一 参数必须是整数")
		}
		return []object.Object{&object.Integer{Value: n.Value + 1}}, nil
	})
}

func nativePrint(rt *Runtime, args []object.Object) ([]object.Object, error) {
	if rt == nil {
		rt = NewRuntime()
	}
	values := args
	if len(args) == 1 {
		if elements, ok := args[0].(*object.Array); ok {
			values = elements.Elements
		}
	}
	writeObjects(rt.Stdout(), values)
	return []object.Object{object.Null}, nil
}

func createTagInstance(rt *Runtime, cls *Class, tagNode *ast.TagExpr, env *Environment) (*Instance, error) {
	if len(tagNode.Positional) > 1 {
		return nil, fmt.Errorf("标签未命名参数最多一个")
	}
	inst := newZeroInstance(cls)
	if len(tagNode.Positional) == 1 {
		value, err := Eval(tagNode.Positional[0], env)
		if err != nil {
			return nil, err
		}
		if value == nil {
			value = object.Null
		}
		if r := inst.resolveField("值"); !r.found {
			return nil, fmt.Errorf("标签模型 %s 没有字段 值", cls.Name)
		}
		if err := inst.setFieldValue("值", value); err != nil {
			return nil, err
		}
	}
	for _, arg := range tagNode.Named {
		value, err := Eval(arg.Value, env)
		if err != nil {
			return nil, err
		}
		if value == nil {
			value = object.Null
		}
		if r := inst.resolveField(arg.Name); !r.found {
			return nil, fmt.Errorf("标签模型 %s 没有字段 %s", cls.Name, arg.Name)
		}
		if err := inst.setFieldValue(arg.Name, value); err != nil {
			return nil, err
		}
	}
	return inst, nil
}

func processTags(rt *Runtime, target tagsys.Target, tagExprs []*ast.TagExpr, env *Environment) error {
	for _, tagNode := range tagExprs {
		cls, ok := rt.resolveClass(tagNode.TypeName, env)
		if !ok {
			// 未注册的标签静默忽略：不影响声明求值。
			continue
		}
		instance, err := createTagInstance(rt, cls, tagNode, env)
		if err != nil {
			return &EvalError{Pos: tagNode.Position, Reason: err.Error()}
		}
		attachTag(target, instance)
		callable := initialCallable(target)
		ctx := &runtimeTagContext{rt: rt, tag: instance, target: target, callable: callable}
		if handler, ok := rt.goTagHandlers[tagNode.TypeName]; ok {
			if err := handler.Handle(ctx); err != nil {
				return &EvalError{Pos: tagNode.Position, Reason: err.Error()}
			}
			continue
		}
		if handler, ok := rt.fanTagHandlers[tagNode.TypeName]; ok {
			if _, err := handler.Call([]tagsys.Object{ctx}); err != nil {
				return &EvalError{Pos: tagNode.Position, Reason: err.Error()}
			}
		}
	}
	return nil
}

func attachTag(target tagsys.Target, tag *Instance) {
	switch value := target.(type) {
	case *Function:
		value.TagInstances = append(value.TagInstances, tag)
	case *Class:
		value.TagInstances = append(value.TagInstances, tag)
	}
}

func initialCallable(target tagsys.Target) tagsys.Callable {
	switch value := target.(type) {
	case *Function:
		if value.Impl != nil {
			return value.Impl
		}
		if value.MethodImpl != nil {
			return value.MethodImpl
		}
		return &fanCallable{fn: value}
	case *Class:
		if value.Impl != nil {
			return value.Impl
		}
		return &classCallable{cls: value}
	}
	return nil
}

func callTagsysCallable(callable tagsys.Callable, args []object.Object, pos ast.Position) (object.Object, error) {
	callArgs := make([]tagsys.Object, len(args))
	for i, arg := range args {
		callArgs[i] = unwrapNamed(arg)
	}
	results, err := callable.Call(callArgs)
	if err != nil {
		return nil, err
	}
	if len(results) == 1 {
		if result, ok := results[0].(object.Object); ok {
			return result, nil
		}
	}
	return object.Null, nil
}

func callDynamicMember(rt *Runtime, obj object.Object, member *ast.MemberExpr, args []object.Object, pos ast.Position) (object.Object, error) {
	field, err := evalFieldAccess(member, nil, obj)
	if err != nil {
		return nil, err
	}
	switch callable := field.(type) {
	case *Function:
		return applyFunction(callable, args, pos)
	case tagsys.Callable:
		return callTagsysCallable(callable, args, pos)
	}
	return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("%s 不是可调用对象", member.Name)}
}

func processFunctionTags(rt *Runtime, fn *Function, tags []*ast.TagExpr, env *Environment) error {
	return processTags(rt, fn, tags, env)
}

func processClassTags(rt *Runtime, cls *Class, tags []*ast.TagExpr, env *Environment) error {
	return processTags(rt, cls, tags, env)
}

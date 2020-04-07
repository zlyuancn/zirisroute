/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/4/3
   Description :
-------------------------------------------------
*/

package zirisroute

import (
    "errors"
    "fmt"
    "reflect"
    "strings"

    "github.com/kataras/iris/v12"
)

const (
    // 控制器名后缀
    ControllerSuffix = "Controller"
    // 默认请求方法
    DefaultRequestMethod = "Get"
    // 在上下文中保存路径参数的字段
    PathParamsField = "_path_params"
)

var requestMethods = [...]string{"Get", "Post", "Delete", "Put", "Patch", "Head"}
var typeOfContext = reflect.TypeOf((*Context)(nil)).Elem()

type Handler func(ctx Context, arg *ReqArg)

type methodType struct {
    name string
    fn   reflect.Value
}

type controller struct {
    name       string
    pathName   string
    parentPath string
    typ        reflect.Type
    methods    map[string]*methodType
    handlers   []Handler
}

func newController(party iris.Party, a interface{}, handlers ...Handler) *controller {
    typ := reflect.TypeOf(a)
    if typ.Kind() != reflect.Ptr {
        panic("必须传入一个指针或接口")
    }

    m := new(controller)
    m.typ = typ.Elem()

    name := m.typ.Name()
    m.name = name
    if strings.HasSuffix(name, ControllerSuffix) {
        name = name[:len(name)-len(ControllerSuffix)]
    }
    if name == "" {
        panic("控制器没有名称")
    }
    m.pathName = snakeString(name)

    path := party.GetRelPath()
    if strings.HasSuffix(path, "/") {
        path = path[:len(path)-1]
    }
    m.parentPath = path

    m.methods = suitableMethods(typ)
    m.handlers = append(m.handlers, handlers...)
    return m
}

// 根据请求方法和原始路径参数搜索尝试执行方法, 返回执行方法和处理后的路径参数
func (m *controller) searchExecMethod(reqMethod, rawPathParams string) (string, string) {
    controlMethod, pathParams := rawPathParams, ""
    // 尝试分离控制器方法
    if k := strings.Index(rawPathParams, "/"); k != -1 {
        controlMethod, pathParams = rawPathParams[:k], rawPathParams[k+1:]
    }

    if method, ok := m.methods[fmt.Sprintf("%s/%s", reqMethod, controlMethod)]; ok {
        return method.name, pathParams
    }

    // 对空方法进行搜索
    if controlMethod != "" {
        if method, ok := m.methods[fmt.Sprintf("%s/", reqMethod)]; ok {
            return method.name, rawPathParams
        }
    }
    return controlMethod, pathParams
}

func (m *controller) handler(ctx Context, ctx_val reflect.Value, reqArg *ReqArg) {
    for _, handler := range m.handlers {
        handler(ctx, reqArg)
        if reqArg.IsStop() {
            return
        }
    }

    ctx.Before(reqArg)
    if reqArg.IsStop() {
        return
    }

    reqMethod, controlMethod := reqArg.ReqMethod(), reqArg.ControlMethod()
    method, ok := m.methods[makeMethodKey(controlMethod, reqMethod)]
    if !ok || method.name != controlMethod {
        if controlMethod == "" {
            ctx.SetResult(errors.New(fmt.Sprintf("未定义的控制器: [%s] <%s/%s>", reqMethod, m.parentPath, m.name)))
        } else {
            ctx.SetResult(errors.New(fmt.Sprintf("未定义的控制器: [%s] <%s/%s.%s>", reqMethod, m.parentPath, m.name, controlMethod)))
        }
        return
    }

    returnValues := method.fn.Call([]reflect.Value{reflect.New(m.typ), ctx_val})
    if len(returnValues) == 1 {
        ctx.SetResult(returnValues[0].Interface())
    } else {
        ctx.SetResult(nil)
    }
}

type Route struct {
    parent      *Route
    ctx_typ     reflect.Type
    party       iris.Party
    handlers    []Handler
    controllers map[string]*controller
}

// 创建一个路由
func NewRoute(party iris.Party, ctx Context) *Route {
    typ := reflect.TypeOf(ctx)
    if typ.Kind() != reflect.Ptr {
        panic("必须传入一个指针或接口")
    }

    return &Route{
        ctx_typ:     typ.Elem(),
        party:       party,
        controllers: make(map[string]*controller),
    }
}

// 使用中间件
func (m *Route) Use(handlers ...Handler) {
    m.handlers = append(m.handlers, handlers...)
}

// 创建一个子路由
func (m *Route) Party(path string, handlers ...Handler) *Route {
    return &Route{
        parent:      m,
        ctx_typ:     m.ctx_typ,
        party:       m.party.Party(path),
        handlers:    handlers,
        controllers: make(map[string]*controller),
    }
}

// 注册控制器
func (m *Route) Registry(controller interface{}, handlers ...Handler) {
    c := newController(m.party, controller, handlers...)
    m.controllers[c.pathName] = c

    handler := m.process(c)
    m.party.CreateRoutes(nil, fmt.Sprintf("/%s", c.pathName), handler)
    m.party.CreateRoutes(nil, fmt.Sprintf("/%s/{%s:path}", c.pathName, PathParamsField), handler)
}

func (m *Route) handle(ctx Context, reqArg *ReqArg) bool {
    if m.parent != nil {
        if !m.parent.handle(ctx, reqArg) {
            return false
        }
    }

    for _, handler := range m.handlers {
        handler(ctx, reqArg)
        if reqArg.IsStop() {
            return false
        }
    }
    return true
}

func (m *Route) process(c *controller) iris.Handler {
    name := c.name
    return func(irisctx iris.Context) {
        reqMethod := irisctx.Method()
        if len(reqMethod) > 0 {
            reqMethod = strings.ToUpper(reqMethod[:1]) + strings.ToLower(reqMethod[1:])
        }

        rawParams := irisctx.Params().Get(PathParamsField)
        rawParams = strings.Trim(rawParams, "/")
        controlMethod, pathParams := c.searchExecMethod(reqMethod, rawParams)

        reqArg := &ReqArg{
            reqMethod:     reqMethod,
            controlName:   name,
            controlMethod: controlMethod,
            pathParams:    pathParams,
        }

        ctx_val := reflect.New(m.ctx_typ)
        ctx := ctx_val.Interface().(Context)
        ctx.Init(irisctx, reqArg)
        if reqArg.IsStop() {
            return
        }

        if !m.handle(ctx, reqArg) {
            return
        }

        c.handler(ctx, ctx_val, reqArg)
    }
}

// 转为蛇形字符串
func snakeString(s string) string {
    if s == "" {
        return ""
    }

    data := make([]byte, 0, len(s)*2)
    j := false
    num := len(s)
    for i := 0; i < num; i++ {
        d := s[i]
        if i > 0 && d >= 'A' && d <= 'Z' && j {
            data = append(data, '_')
        }
        if d != '_' {
            j = true
        }
        data = append(data, d)
    }
    return strings.ToLower(string(data[:]))
}

// 适配用于注册的方法
func suitableMethods(typ reflect.Type) map[string]*methodType {
    methods := make(map[string]*methodType, 0)
    for i := 0; i < typ.NumMethod(); i++ {
        method := typ.Method(i)
        mtype := method.Type

        // 未导出的方法过滤掉
        if method.PkgPath != "" {
            continue
        }

        // 包括自己本身和接收参数数量
        if mtype.NumIn() != 2 {
            continue
        }

        // 第一个参数必须是给定的Context
        ctxType := mtype.In(1)
        if !ctxType.Implements(typeOfContext) {
            continue
        }

        // 方法最多只能有一个输出
        if mtype.NumOut() > 1 {
            continue
        }

        name := method.Name
        key := makeMethodKey(name, DefaultRequestMethod)
        methods[key] = &methodType{name: name, fn: method.Func}
    }
    return methods
}

// 根据控制器方法构建key
func makeMethodKey(controlMethod string, defaultReqMethod string) string {
    for _, s := range requestMethods {
        if strings.HasPrefix(controlMethod, s) {
            return fmt.Sprintf("%s/%s", s, snakeString(controlMethod[len(s):]))
        }
    }
    return fmt.Sprintf("%s/%s", defaultReqMethod, snakeString(controlMethod))
}

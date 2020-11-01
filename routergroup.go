package gig

import (
	"net/http"
	"path"
	"reflect"
	"strings"
)

// 定义路由分组
type RouterGroup struct {
	prefix      string
	middlewares []HandlerFunc // 支持中间件
	parent      *RouterGroup  // 父分组，支持嵌套
	engine      *Engine       // 分组持有引擎实例
}

// 创建一个新的Group
func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		parent: group,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

// 添加中间件到group
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

// 新增路由
func (group *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	group.engine.router.addRoute(method, pattern, handler)

	if IsDebugging() {
		ConsolePrint("Route  %5s - %s", method, pattern)
	}
}

// GET路由
func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodGet, pattern, handler)
}

// POST路由
func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodPost, pattern, handler)
}

// DELETE路由
func (group *RouterGroup) DELETE(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodDelete, pattern, handler)
}

// PATCH路由
func (group *RouterGroup) PATCH(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodPatch, pattern, handler)
}

// PUT路由
func (group *RouterGroup) PUT(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodPut, pattern, handler)
}

// OPTIONS路由
func (group *RouterGroup) OPTIONS(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodOptions, pattern, handler)
}

// HEAD路由
func (group *RouterGroup) HEAD(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodHead, pattern, handler)
}

// ANY路由
func (group *RouterGroup) ANY(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodPost, pattern, handler)
	group.addRoute(http.MethodGet, pattern, handler)
	group.addRoute(http.MethodDelete, pattern, handler)
	group.addRoute(http.MethodPatch, pattern, handler)
	group.addRoute(http.MethodPut, pattern, handler)
	group.addRoute(http.MethodOptions, pattern, handler)
	group.addRoute(http.MethodHead, pattern, handler)
}

// Router 新增控制器路由，这里的路由需要对应一个控制器和方法，控制器继承 gig.Controller
// 示例：router.Router("get", "/index", &IndexController{}, "Index")
//      router.Router("post", "/login", &IndexController{}, "Login")
func (group *RouterGroup) Router(pattern string, c ControllerInterface, mappingMethod string) {
	semis := strings.Split(mappingMethod, ":")
	if len(semis) != 2 {
		panic("router funcname is error")
	}
	method := strings.ToUpper(semis[0])
	funcName := semis[1]

	if !HTTPMETHODS[method] {
		panic("router method is not allowed")
	}

	// 通过反射，确定控制器和方法
	reflectVal := reflect.ValueOf(c)
	t := reflect.Indirect(reflectVal).Type()
	vc := reflect.New(t)
	execController, ok := vc.Interface().(ControllerInterface)
	if !ok {
		panic("controller is not ControllerInterface")
	}
	runFunc := vc.MethodByName(funcName);
	if !runFunc.IsValid() {
		panic(mappingMethod + " is an invalid method mapping")
	}

	// 把Controller方法转化为 HandlerFunc
	group.addRoute(method, pattern, func(ctx *Context) {
		execController.Init(ctx, t.Name(), funcName, execController)
		execController.Prepare()
		runFunc.Call(nil)
	})
}

// 静态文件
// use :
//     router.Static("/static", "/var/www")
func (group *RouterGroup) Static(relativePath, root string) {
	group.StaticFS(relativePath, http.Dir(root))
}

func (group *RouterGroup) StaticFS(relativePath string, fs http.FileSystem) {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("URL parameters can not be used when serving a static folder")
	}
	handler := group.createStaticHandler(relativePath, fs)
	urlPattern := path.Join(relativePath, "/*filepath")

	// 注册 GET / HEAD handler
	group.GET(urlPattern, handler)
	group.HEAD(urlPattern, handler)
}

// 创建静态文件handler
func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(group.prefix, relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))

	return func(ctx *Context) {
		file := ctx.Param("filepath")

		// 检查文件是否存在且有权限访问
		f, err := fs.Open(file);
		if err != nil {
			ctx.Status(http.StatusNotFound)
			return
		}
		f.Close()

		fileServer.ServeHTTP(ctx.Writer, ctx.Request)
	}
}

package gig

import (
	"net/http"
	"path"
	"regexp"
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
		debugPrint("Route  %5s - %s", method, pattern)
	}
}

// Handle
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
func (group *RouterGroup) Handle(httpMethod, relativePath string, handler HandlerFunc) {
	if matches, err := regexp.MatchString("^[A-Z]+$", httpMethod); !matches || err != nil {
		panic("http method " + httpMethod + " is not valid")
	}
	group.addRoute(httpMethod, relativePath, handler)
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
		f, err := fs.Open(file)
		if err != nil {
			ctx.Status(http.StatusNotFound)
			return
		}
		f.Close()

		fileServer.ServeHTTP(ctx.Writer, ctx.Request)
	}
}

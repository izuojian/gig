package gig

import (
	"net/http"
	"strings"
)

// 默认最大Multipart内存占用
const defaultMultipartMemory = 32 << 20 // 32 MB

// 定义HandlerFunc, 提供给框架用户，用来定义路由映射的处理方法
type HandlerFunc func(*Context)

// 定义Engine，实现ServeHTPP接口
type Engine struct {
	*RouterGroup                // 继承了分组的所有属性和方法
	router       *router        // 路由
	groups       []*RouterGroup // 全部分组

	// If enabled, the router checks if another method is allowed for the
	// current route, if the current request can not be routed.
	// If this is the case, the request is answered with 'Method Not Allowed'
	// and HTTP status code 405.
	// If no other Method is allowed, the request is delegated to the NotFound
	// handler.
	HandleMethodNotAllowed bool
	ForwardedByClientIP    bool

	// #726 #755 If enabled, it will thrust some headers starting with
	// 'X-AppEngine...' for better integration with that PaaS.
	AppEngine bool

	// Value of 'maxMemory' param that is given to http.Request's ParseMultipartForm
	// method call.
	MaxMultipartMemory int64
}

// 创建一个新的引擎
func New() *Engine {
	engine := &Engine{
		router:              newRouter(),
		ForwardedByClientIP: true,
		AppEngine:           false,
		MaxMultipartMemory:  defaultMultipartMemory,
	}
	engine.RouterGroup = &RouterGroup{
		engine: engine,
	}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

// 默认引擎，使用了日志和错误恢复中间件
func Default() *Engine {
	engine := New()
	engine.Use(Logger(), Recovery())
	return engine
}

// AddFuncMap
func (engine *Engine) AddFuncMap(key string, fn interface{}) error {
	return AddFuncMap(key, fn)
}

// 加载全部模板文件
func (engine *Engine) Templates(viewPath string) {
	if err := LoadTemplates(viewPath); err != nil {
		panic("Load Templates error:" + err.Error())
	}
}

// 运行http server
func (engine *Engine) Run(addr string) (err error) {
	debugPrint("Running in \"%s\" mode", Mode())
	debugPrint("HTTP server running on%s %s %s", ConsoleFrontColorCyan, addr, ConsoleFrontColorReset)
	return http.ListenAndServe(addr, engine)
}

// 实现ServeHTTP
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	middlewares := make([]HandlerFunc, 0)
	for _, group := range engine.groups {
		// 把所有分组包括嵌套分组的中间件都保存下来，后面依次执行
		// 如 req.URL.Path：/v1/v2/v3/login，则需要执行分组 v1 v2 v3的所有中间件
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}

	ctx := newContext(w, req)
	ctx.handlers = middlewares
	ctx.engine = engine

	engine.router.handle(ctx)
}

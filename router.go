package gig

import (
	"net/http"
	"strings"
)

// roots key eg, roots['GET'] roots['POST']
// handlers key eg, handlers['GET-/p/:lang/doc'], handlers['POST-/p/book']
type router struct {
	// 使用 roots 来存储每种请求方式的 Trie 树根节点
	roots map[string]*node
	// 使用 handlers 存储每种请求方式的 HandlerFunc
	handlers map[string]HandlerFunc
}

// 支持的Methods
var (
	// HTTPMETHOD list the supported http methods.
	HTTPMETHODS = map[string]bool{
		"GET":     true,
		"POST":    true,
		"PUT":     true,
		"DELETE":  true,
		"PATCH":   true,
		"OPTIONS": true,
		"HEAD":    true,
	}
)

// router type
const (
	routerTypeGig = iota
	routerTypeRESTFul
	routerTypeHandler
)

func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

// 新增路由
func (r *router) addRoute(method, pattern string, handler HandlerFunc) {
	parts := _parsePattern(pattern)

	key := method + "-" + pattern
	// 根节点不存在则新建
	if _, ok := r.roots[method]; !ok {
		r.roots[method] = &node{}
	}
	// 新增节点
	r.roots[method].insert(pattern, parts, 0)
	r.handlers[key] = handler
}

// 获取路由
func (r *router) getRoute(method, urlPath string) (*node, map[string]string) {
	searchParts := _parsePattern(urlPath)
	params := make(map[string]string)
	root, ok := r.roots[method]
	if !ok {
		return nil, nil
	}

	n := root.search(searchParts, 0)
	if n != nil {
		parts := _parsePattern(n.pattern)
		for index, part := range parts {
			if part[0] == ':' {
				// 兼容 :id.html 这样的路由
				if paramPointIndex := strings.Index(part, "."); paramPointIndex != -1 {
					if valuePointIndex := strings.Index(searchParts[index], "."); valuePointIndex != -1 {
						params[part[1:paramPointIndex]] = searchParts[index][:valuePointIndex]
					} else {
						params[part[1:paramPointIndex]] = searchParts[index]
					}
				} else {
					params[part[1:]] = searchParts[index]
				}
			}
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return n, params
	}

	return nil, nil
}

// 获取全部路由
func (r *router) getRoutes(method string) []*node {
	root, ok := r.roots[method]
	if !ok {
		return nil
	}

	nodes := make([]*node, 0)
	root.travel(&nodes)
	return nodes
}

// 执行
func (r *router) handle(c *Context) {
	httpMethod := c.Request.Method
	rPath := c.Request.URL.Path
	routerNode, params := r.getRoute(httpMethod, rPath)
	if routerNode != nil {
		c.Params = params

		key := httpMethod + "-" + routerNode.pattern

		// 把匹配的HandleFunc添加到中间的handlers中
		// 由 Next() 统一执行
		c.handlers = append(c.handlers, r.handlers[key])
	} else {
		c.handlers = append(c.handlers, func(ctx *Context) {
			c.String(http.StatusNotFound, "404 NOT FOUND: %s \n", rPath)
		})
	}
	c.Next()
}

// 解析路由pattern，获取所有路由段segment
// 注意：pattern 只允许存在一个通配符*
func _parsePattern(pattern string) []string {
	patternSlice := strings.Split(pattern, "/")

	parts := make([]string, 0)
	for _, item := range patternSlice {
		if item != "" {
			parts = append(parts, item)
			// 匹配到 * 就退出
			if item[0] == '*' {
				break
			}
		}
	}
	return parts
}

package gig

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/izuojian/gig/binding"
	"github.com/izuojian/gig/internal/bytesconv"
	"io"
	"io/ioutil"
	"math"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type H map[string]interface{}

// handlerFunc map 最大容量
const abortIndex int8 = math.MaxInt8 / 2

const (
	noWritten     = -1
	defaultStatus = http.StatusOK
)

type Context struct {
	// HttpWriter
	Writer http.ResponseWriter
	// HttpRequest
	Request *http.Request

	// 持有engine
	engine *Engine

	// 路由参数
	Params map[string]string

	// 响应状态码
	StatusCode int

	// 中间件方法
	handlers []HandlerFunc

	// 记录执行到第几个HandlerFunc,包括中间件
	index int8

	// Errors is a list of errors attached to all the handlers/middlewares who used this context.
	Errors errorMsgs

	// URL Query参数缓存
	queryCache url.Values

	// Form 参数缓存，比如post、put请求
	postFormCache url.Values

	// 全部参数缓存，包括post和query参数
	formCache url.Values

	// 读写锁，保护keys字典
	mu sync.RWMutex

	// 给每个请求的Context存储元数据
	Keys map[string]interface{}

	// SameSite allows a server to define a cookie attribute making it impossible for
	// the browser to send this cookie along with cross-site requests.
	sameSite http.SameSite
}

/************************************/
/********** CONTEXT 操作 ************/
/************************************/

// newContext 构造方法
func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer:        w,
		Request:       req,
		StatusCode:    defaultStatus,
		Params:        nil,
		handlers:      nil,
		index:         -1,
		Keys:          nil,
		Errors:        nil,
		queryCache:    nil,
		postFormCache: nil,
		formCache:     nil,
	}
}

/************************************/
/*********** 流程控制 ***********/
/************************************/

// Next 执行中间件HandleFunc和匹配到的路由HandleFunc
func (c *Context) Next() {
	c.index++
	for c.index < int8(len(c.handlers)) {
		// 执行HandleFunc
		c.handlers[c.index](c)
		c.index++
	}
}

// AbortWithStatus calls `Abort()` and writes the headers with the specified status code.
// For example, a failed attempt to authenticate a request could use: context.AbortWithStatus(401).
func (c *Context) AbortWithStatus(code int) {
	c.Status(code)
	//c.Writer.WriteHeaderNow()
	c.Abort()
}

func (c *Context) WriteHeaderNow() {
	//if !w.Written() {
	//	w.size = 0
	//	c.Writer.WriteHeader(w.status)
	//}
}

// IsAborted returns true if the current context was aborted.
func (c *Context) IsAborted() bool {
	return c.index >= abortIndex
}

// Abort prevents pending handlers from being called. Note that this will not stop the current handler.
// Let's say you have an authorization middleware that validates that the current request is authorized.
// If the authorization fails (ex: the password does not match), call Abort to ensure the remaining handlers
// for this request are not called.
func (c *Context) Abort() {
	c.index = abortIndex
}

/************************************/
/**************** Cookie ************/
/************************************/

// SetSameSite with cookie
func (c *Context) SetSameSite(samesite http.SameSite) {
	c.sameSite = samesite
}

// SetCookie adds a Set-Cookie header to the ResponseWriter's headers.
// The provided cookie must have a valid Name. Invalid cookies may be
// silently dropped.
func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	if path == "" {
		path = "/"
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    url.QueryEscape(value),
		MaxAge:   maxAge,
		Path:     path,
		Domain:   domain,
		SameSite: c.sameSite,
		Secure:   secure,
		HttpOnly: httpOnly,
	})
}

// Cookie returns the named cookie provided in the request or
// ErrNoCookie if not found. And return the named cookie is unescaped.
// If multiple cookies match the given name, only one cookie will
// be returned.
func (c *Context) Cookie(name string) (string, error) {
	cookie, err := c.Request.Cookie(name)
	if err != nil {
		return "", err
	}
	val, _ := url.QueryUnescape(cookie.Value)
	return val, nil
}

/************************************/
/************  元数据管理  ***********/
/************************************/

// Set 设置元数据
func (c *Context) Set(key string, value interface{}) {
	c.mu.Lock()
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys[key] = value
	c.mu.Unlock()
}

// Get 获取元数据
func (c *Context) Get(key string) (value interface{}, exists bool) {
	c.mu.Lock()
	value, exists = c.Keys[key]
	c.mu.Unlock()
	return
}

// GetString 获取String元数据
func (c *Context) GetString(key string) (s string) {
	if val, ok := c.Get(key); ok && val != nil {
		s, _ = val.(string)
	}
	return
}

// GetInt 获取Int元数据
func (c *Context) GetInt(key string) (i int) {
	if val, ok := c.Get(key); ok && val != nil {
		i, _ = val.(int)
	}
	return
}

// GetInt64 获取Int64元数据
func (c *Context) GetInt64(key string) (i64 int64) {
	if val, ok := c.Get(key); ok && val != nil {
		i64, _ = val.(int64)
	}
	return
}

/************************************/
/************ INPUT 数据  ************/
/************************************/

// Param 获取路由参数
// 比如： /a/:name  Param("name")
func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

// Query 获取单个Query参数
func (c *Context) Query(key string) string {
	value, _ := c.GetQuery(key)
	return value
}

// DefaultQuery 获取Query参数，如果获取失败，返回默认值
func (c *Context) DefaultQuery(key, defaultValue string) string {
	if value, ok := c.GetQuery(key); ok {
		return value
	}
	return defaultValue
}

// QueryArray 获取参数切片
func (c *Context) QueryArray(key string) []string {
	values, _ := c.GetQueryArray(key)
	return values
}

// GetQuery 获取单个Query参数，获取不到会返回false
func (c *Context) GetQuery(key string) (string, bool) {
	if values, ok := c.GetQueryArray(key); ok {
		return values[0], true
	}
	return "", false
}

// GetQueryArray 获取参数key对应的参数Slice
func (c *Context) GetQueryArray(key string) ([]string, bool) {
	c.initQueryCache()
	if values, ok := c.queryCache[key]; ok && len(values) > 0 {
		return values, true
	}
	return []string{}, false
}

// initQueryCache 初始化Query参数缓存
func (c *Context) initQueryCache() {
	if c.queryCache == nil {
		c.queryCache = c.Request.URL.Query()
	}
}

// PostForm 解析Post方式提交的Form表单参数
func (c *Context) PostForm(key string) string {
	value, _ := c.GetPostForm(key)
	return value
}

// DefaultPostForm 解析Post方式提交的Form表单参数, 失败时返回默认值
func (c *Context) DefaultPostForm(key, defaultValue string) string {
	if value, ok := c.GetPostForm(key); ok {
		return value
	}
	return defaultValue
}

// GetPostForm 获取PostForm参数
func (c *Context) GetPostForm(key string) (string, bool) {
	if values, ok := c.GetPostFormArray(key); ok {
		return values[0], true
	}
	return "", false
}

// PostFormArray 获取PostForm参数slice
func (c *Context) PostFormArray(key string) []string {
	values, _ := c.GetPostFormArray(key)
	return values
}

// GetPostFormArray 获取PostForm参数slice
func (c *Context) GetPostFormArray(key string) ([]string, bool) {
	c.initPostFormCache()
	if values, ok := c.postFormCache[key]; ok && len(values) > 0 {
		return values, true
	}
	return []string{}, false
}

// initPostFormCache 获取PostForm参数缓存
func (c *Context) initPostFormCache() {
	if c.postFormCache == nil {
		c.postFormCache = make(url.Values)
		req := c.Request
		if err := req.ParseMultipartForm(c.engine.MaxMultipartMemory); err != nil {
			if err != http.ErrNotMultipart {
				debugPrint("error on parse multipart form array: %v", err)
			}
		}
		c.postFormCache = req.PostForm
	}
}

// FormFile 获取上传的第一个文件
func (c *Context) FormFile(name string) (*multipart.FileHeader, error) {
	if c.Request.MultipartForm == nil {
		if err := c.Request.ParseMultipartForm(c.engine.MaxMultipartMemory); err != nil {
			return nil, err
		}
	}
	f, fh, err := c.Request.FormFile(name)
	if err != nil {
		return nil, err
	}
	f.Close()
	return fh, err
}

// MultipartForm is the parsed multipart form, including file uploads.
func (c *Context) MultipartForm() (*multipart.Form, error) {
	err := c.Request.ParseMultipartForm(c.engine.MaxMultipartMemory)
	return c.Request.MultipartForm, err
}

// ShouldBindJSON 通过JSON绑定引擎绑定请求体参数
func (c *Context) ShouldBindJSON(obj interface{}) error {
	return c.ShouldBindWith(obj, binding.JSON)
}

// ShouldBindWith 通过绑定引擎绑定请求体参数
func (c *Context) ShouldBindWith(obj interface{}, b binding.Binding) error {
	return b.Bind(c.Request, obj)
}

// 获取单个请求头
func (c *Context) requestHeader(key string) string {
	return c.Request.Header.Get(key)
}

// IsAjax 判断是否Ajax请求
func (c *Context) IsAjax() bool {
	return c.requestHeader("X-Requested-With") == "XMLHttpRequest"
}

// IsUpload 判断是否上传文件
func (c *Context) IsUpload() bool {
	return strings.Contains(c.requestHeader("Content-Type"), "multipart/form-data")
}

// RequestBody 获取RequestBody
func (c *Context) RequestBody() []byte {
	if c.Request.Body == nil {
		return []byte{}
	}

	var requestbody []byte
	safe := &io.LimitedReader{R: c.Request.Body, N: defaultMultipartMemory}
	if c.requestHeader("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(safe)
		if err != nil {
			return nil
		}
		requestbody, _ = ioutil.ReadAll(reader)
	} else {
		requestbody, _ = ioutil.ReadAll(safe)
	}

	_ = c.Request.Body.Close()
	bf := bytes.NewBuffer(requestbody)
	c.Request.Body = http.MaxBytesReader(c.Writer, ioutil.NopCloser(bf), defaultMultipartMemory)
	return requestbody
}

// ClientIP 获取客户端IP
func (c *Context) ClientIP() string {
	if c.engine.ForwardedByClientIP {
		clientIP := c.requestHeader("X-Forwarded-For")
		clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
		if clientIP == "" {
			clientIP = strings.TrimSpace(c.requestHeader("X-Real-Ip"))
		}
		if clientIP != "" {
			return clientIP
		}
	}

	if c.engine.AppEngine {
		if addr := c.requestHeader("X-Appengine-Remote-Addr"); addr != "" {
			return addr
		}
	}

	if ip, _, err := net.SplitHostPort(strings.TrimSpace(c.Request.RemoteAddr)); err == nil {
		return ip
	}

	return ""
}

/************************************/
/******** 响应数据渲染 ********/
/************************************/

// Redirect 跳转
func (c *Context) Redirect(status int, localurl string) {
	http.Redirect(c.Writer, c.Request, localurl, status)
}

// Status 设置HTTP响应状态码
func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

// Header 设置HTTP响应头信息
// 如果 value == "" 则会删除当前header
func (c *Context) Header(key, value string) {
	if value == "" {
		c.Writer.Header().Del(key)
		return
	}
	c.Writer.Header().Set(key, value)
}

// Fail 响应失败
func (c *Context) Fail(code int, err string) {
	c.index = int8(len(c.handlers))
	c.JSON(code, H{"message": err})
}

// Data 响应数据
func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	_, err := c.Writer.Write(data)
	if err != nil {
		fmt.Printf("c.Writer.Write error: %v\n", err)
	}
}

// String 响应String格式数据
func (c *Context) String(code int, format string, values ...interface{}) {
	c.Header("Content-Type", "text/plain")
	c.Status(code)
	_, err := c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
	if err != nil {
		fmt.Printf("c.Writer.Write error: %v\n", err)
	}
}

// JSON 响应JSON格式数据
func (c *Context) JSON(code int, obj interface{}) {
	c.Header("Content-Type", "application/json")
	c.Status(code)

	ret, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	// 从gin借鉴
	var buffer bytes.Buffer
	for _, r := range bytesconv.BytesToString(ret) {
		cvt := string(r)
		if r >= 128 {
			cvt = fmt.Sprintf("\\u%04x", int64(r))
		}
		buffer.WriteString(cvt)
	}

	_, err = c.Writer.Write(buffer.Bytes())
	panic(err)
}

// HTML 响应HTML格式数据
// 类似Beego使用的方法
func (c *Context) HTML(code int, name string, data interface{}) {
	c.Header("Content-Type", "text/html")
	c.Status(code)
	if err := ExecuteTemplate(c.Writer, name, data); err != nil {
		c.Fail(http.StatusInternalServerError, err.Error())
	}
}

// 响应HTML格式数据
// 类似 Gin 中使用的方法
/*
func (c *Context) HTML2(code int, name string, data interface{}) {
	c.Header("Content-Type", "text/html")
	c.Status(code)
	if err := c.engine.htmlTemplates.ExecuteTemplate(c.Writer, name, data); err != nil {
		c.Fail(http.StatusInternalServerError, err.Error())
	}
}*/

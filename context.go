package gig

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/satori/go.uuid"
	"io"
	"io/ioutil"
	"math"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type H map[string]interface{}

// handlerFunc map 最大容量
const abortIndex int8 = math.MaxInt8 / 2

const (
	noWritten     = -1
	defaultStatus = http.StatusOK
)

type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request

	// 请求信息
	Path   string
	Method string
	// 路由参数
	Params map[string]string

	// 响应状态码
	StatusCode int

	// 中间件方法
	handlers []HandlerFunc

	// 记录执行到第几个HandlerFunc,包括中间件
	index int8

	// 持有engine
	engine *Engine

	// Errors is a list of errors attached to all the handlers/middlewares who used this context.
	Errors errorMsgs

	// URL Query参数缓存
	queryCache url.Values

	// Form 参数缓存，比如post、put请求
	postFormCache url.Values

	// 全部参数缓存，包括post和query参数
	formCache url.Values

	// SameSite allows a server to define a cookie attribute making it impossible for
	// the browser to send this cookie along with cross-site requests.
	sameSite http.SameSite

	//
	RequestId string
}

// 构造方法
func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer:     w,
		Request:    req,
		Path:       req.URL.Path,
		Method:     req.Method,
		index:      -1,
		StatusCode: defaultStatus,
	}
}

/************************************/
/*********** 流程控制 ***********/
/************************************/

// 执行中间件HandleFunc和匹配到的路由HandleFunc
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
/********   Cookie  ********/
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
/************ INPUT 数据  ************/
/************************************/
// 请求方式
func (c *Context) MethodIs(method string) bool {
	return c.Method == method
}

// 获取单个请求头
func (c *Context) requestHeader(key string) string {
	return c.Request.Header.Get(key)
}

// GET
func (c *Context) IsGet() bool {
	return c.MethodIs("GET")
}

// POST
func (c *Context) IsPost() bool {
	return c.MethodIs("POST")
}

// HEAD
func (c *Context) IsHead() bool {
	return c.MethodIs("HEAD")
}

// OPTIONS
func (c *Context) IsOptions() bool {
	return c.MethodIs("OPTIONS")
}

// PUT
func (c *Context) IsPut() bool {
	return c.MethodIs("PUT")
}

// DELETE
func (c *Context) IsDelete() bool {
	return c.MethodIs("DELETE")
}

// PATCH
func (c *Context) IsPatch() bool {
	return c.MethodIs("PATCH")
}

// AJAX
func (c *Context) IsAjax() bool {
	return c.requestHeader("X-Requested-With") == "XMLHttpRequest"
}

// IsUpload returns boolean of whether file uploads in this request or not..
func (c *Context) IsUpload() bool {
	return strings.Contains(c.requestHeader("Content-Type"), "multipart/form-data")
}

// 获取路由参数
// 比如： /a/:name  Param("name")
func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

// 获取单个Query参数
func (c *Context) Query(key string) string {
	value, _ := c.GetQuery(key)
	return value
}

// 获取Query参数，如果获取失败，返回默认值
func (c *Context) DefaultQuery(key, defaultValue string) string {
	if value, ok := c.GetQuery(key); ok {
		return value
	}
	return defaultValue
}

// 获取参数切片
func (c *Context) QueryArray(key string) []string {
	values, _ := c.GetQueryArray(key)
	return values
}

// 获取单个Query参数
func (c *Context) GetQuery(key string) (string, bool) {
	if values, ok := c.GetQueryArray(key); ok {
		return values[0], true
	}
	return "", false
}

// 获取参数key对应的参数Slice
func (c *Context) GetQueryArray(key string) ([]string, bool) {
	c.getQueryCache()
	if values, ok := c.queryCache[key]; ok && len(values) > 0 {
		return values, true
	}
	return []string{}, false
}

// 获取Query参数缓存
func (c *Context) getQueryCache() {
	if c.queryCache == nil {
		c.queryCache = c.Request.URL.Query()
	}
}

// 解析PostForm参数
func (c *Context) PostForm(key string) string {
	value, _ := c.GetPostForm(key)
	return value
}

// 获取PostForm参数, 失败时返回默认值
func (c *Context) DefaultPostForm(key, defaultValue string) string {
	if value, ok := c.GetPostForm(key); ok {
		return value
	}
	return defaultValue
}

// 获取PostForm参数slice
func (c *Context) PostFormArray(key string) []string {
	values, _ := c.GetPostFormArray(key)
	return values
}

// 获取PostForm参数
func (c *Context) GetPostForm(key string) (string, bool) {
	if values, ok := c.GetPostFormArray(key); ok {
		return values[0], true
	}
	return "", false
}

// 获取PostForm参数slice
func (c *Context) GetPostFormArray(key string) ([]string, bool) {
	c.getPostFormCache()
	if values, ok := c.postFormCache[key]; ok && len(values) > 0 {
		return values, true
	}
	return []string{}, false
}

// 获取PostForm参数缓存
func (c *Context) getPostFormCache() {
	if c.postFormCache == nil {
		c.postFormCache = make(url.Values)
		req := c.Request
		if err := req.ParseMultipartForm(c.engine.MaxMultipartMemory); err != nil {
			if err != http.ErrNotMultipart {
				ConsolePrintError("error on parse multipart form array: %v", err)
				ErrorLogger.Errorf("error on parse multipart form array: %v", err)
			}
		}
		c.postFormCache = req.PostForm
	}
}

// 解析Form参数
func (c *Context) Form(key string) string {
	value, _ := c.GetForm(key)
	return value
}

// 获取Form参数, 失败时返回默认值
func (c *Context) DefaultForm(key, defaultValue string) string {
	if value, ok := c.GetForm(key); ok {
		return value
	}
	return defaultValue
}

// 获取Form参数slice
func (c *Context) FormArray(key string) []string {
	values, _ := c.GetFormArray(key)
	return values
}

// 获取Form参数
func (c *Context) GetForm(key string) (string, bool) {
	if values, ok := c.GetFormArray(key); ok {
		return values[0], true
	}
	return "", false
}

// 获取Form参数slice
func (c *Context) GetFormArray(key string) ([]string, bool) {
	c.getFormCache()
	if values, ok := c.formCache[key]; ok && len(values) > 0 {
		return values, true
	}
	return []string{}, false
}

// 获取Form参数缓存
func (c *Context) getFormCache() {
	if c.formCache == nil {
		c.formCache = make(url.Values)
		req := c.Request
		if err := req.ParseMultipartForm(c.engine.MaxMultipartMemory); err != nil {
			if err != http.ErrNotMultipart {
				ErrorLogger.Errorf("error on parse multipart form array: %v", err)
			}
		}
		c.formCache = req.Form
	}
}

// 获取 RequestBody
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

// 获取上传的第一个文件
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

// 全部文件
func (c *Context) FormFiles() (map[string][]*multipart.FileHeader, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, err
	}
	return form.File, nil
}

// 获取客户端IP
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

// Status 设置 HTTP 响应状态码
func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

// Header 设置 HTTP 响应头信息
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
	c.Header("Content-Type", "test/plain")
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

	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
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

// GenerateRequestId 每个请求生成一个RequestID
func (c *Context) GenerateRequestId() string {
	return uuid.NewV4().String()
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

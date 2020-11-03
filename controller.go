package gig

import (
	"errors"
	"mime/multipart"
	"strconv"
)

type Controller struct {
	Ctx *Context

	// route controller info
	controllerName string
	actionName     string
	methodMapping  map[string]func() //method:routertree
	AppController  interface{}
}

var (
	ErrUserExit = errors.New("user stop request manually")
)

// ControllerInterface is an interface to uniform all controller handler.
type ControllerInterface interface {
	Init(ctx *Context, controllerName, actionName string, appController interface{})
	Prepare()
	HandlerFunc(fn string) bool
}

// Init generates default values of controller operations.
func (c *Controller) Init(ctx *Context, controllerName, actionName string, appController interface{}) {
	c.Ctx = ctx
	c.controllerName = controllerName
	c.actionName = actionName
	c.AppController = appController
	c.methodMapping = make(map[string]func())
}

// Prepare runs after Init before request function execution.
func (c *Controller) Prepare() {}

// HandlerFunc call function with the name
func (c *Controller) HandlerFunc(fnname string) bool {
	if v, ok := c.methodMapping[fnname]; ok {
		v()
		return true
	}
	return false
}

/************************************/
/******** Cookie  ********/
/************************************/
func (c *Controller) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	c.Ctx.SetCookie(name, value, maxAge, path, domain, secure, httpOnly)
}

func (c *Controller) Cookie(name string) (string, error) {
	return c.Ctx.Cookie(name)
}

/************************************/
/************ INPUT 数据  ************/
/************************************/
func (c *Controller) IsGet() bool {
	return c.Ctx.IsGet()
}

func (c *Controller) IsPost() bool {
	return c.Ctx.IsPost()
}

func (c *Controller) IsHead() bool {
	return c.Ctx.IsHead()
}

func (c *Controller) IsOptions() bool {
	return c.Ctx.IsOptions()
}

func (c *Controller) IsPut() bool {
	return c.Ctx.IsPut()
}

func (c *Controller) IsDelete() bool {
	return c.Ctx.IsDelete()
}

func (c *Controller) IsPatch() bool {
	return c.Ctx.IsPatch()
}

func (c *Controller) IsAjax() bool {
	return c.Ctx.IsAjax()
}

func (c *Controller) IsUpload() bool {
	return c.Ctx.IsUpload()
}

func (c *Controller) PathParam(key string) string {
	return c.Ctx.Param(key)
}

func (c *Controller) PathParamInt(key string) int {
	v := c.Ctx.Param(key)

	i, err := strconv.Atoi(v)
	if err == nil {
		return i
	}
	return 0
}

func (c *Controller) PathParamInt64(key string) int64 {
	v := c.Ctx.Param(key)
	i64, err := strconv.ParseInt(v, 10, 64)
	if err == nil {
		return i64
	}
	return 0
}

func (c *Controller) Query(key string) string {
	return c.Ctx.Query(key)
}

func (c *Controller) DefaultQuery(key, defaultValue string) string {
	return c.Ctx.DefaultQuery(key, defaultValue)
}

func (c *Controller) QueryArray(key string) []string {
	return c.Ctx.QueryArray(key)
}

func (c *Controller) PostForm(key string) string {
	return c.Ctx.PostForm(key)
}

func (c *Controller) DefaultPostForm(key, defaultValue string) string {
	return c.Ctx.DefaultPostForm(key, defaultValue)
}

func (c *Controller) PostFormArray(key string) []string {
	return c.Ctx.PostFormArray(key)
}

// 请求主体信息
func (c *Controller) RequestBody() []byte {
	return c.Ctx.RequestBody()
}

func (c *Controller) RequestHeader(key string) string {
	return c.Ctx.requestHeader(key)
}

// Form参数
// 获取参数 返回各种类型
// 示例：
//      name := c.Param("name")
//      name := c.Param("name", "hello")
func (c *Controller) Param(key string, defaultValue ...string) string {
	if v := c.Ctx.Form(key); v != "" {
		return v
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// 获取Int型参数
// 示例：
//      age := c.ParamInt("age")
//      age := c.ParamInt("age",10)
func (c *Controller) ParamInt(key string, defaultValue ...int) int {
	if v := c.Ctx.Form(key); v != "" {
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

func (c *Controller) ParamInt32(key string, defaultValue ...int32) int32 {
	if v := c.Ctx.Form(key); v != "" {
		i64, err := strconv.ParseInt(v, 10, 32)
		if err == nil {
			return int32(i64)
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

func (c *Controller) ParamInt64(key string, defaultValue ...int64) int64 {
	if v := c.Ctx.Form(key); v != "" {
		i64, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return i64
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

func (c *Controller) ParamFloat32(key string, defaultValue ...float32) float32 {
	if v := c.Ctx.Form(key); v != "" {
		f64, err := strconv.ParseFloat(v, 32)
		if err == nil {
			return float32(f64)
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

func (c *Controller) ParamFloat64(key string, defaultValue ...float64) float64 {
	if v := c.Ctx.Form(key); v != "" {
		f64, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return f64
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

// 获取上传的第一个文件
func (c *Controller) ParamFile(key string) *multipart.FileHeader {
	f, err := c.Ctx.FormFile(key)
	if err != nil {
		return nil
	}
	return f
}

// 获取全部上传文件
func (c *Controller) ParamFiles() map[string][]*multipart.FileHeader {
	allFiles, err := c.Ctx.FormFiles()
	if err != nil {
		return nil
	}
	return allFiles
}

/************************************/
/******** 响应数据渲染 ********/
/************************************/
func (c *Controller) Redirect(url string, code int) {
	c.Ctx.Redirect(code, url)
}

// 在 Controller 层调用 Context 的响应渲染在
func (c *Controller) JSON(code int, obj interface{}) {
	c.Ctx.JSON(code, obj)
}

func (c *Controller) HTML(code int, name string, data interface{}) {
	c.Ctx.HTML(code, name, data)
}

/************************************/
/******** Helper Functions ********/
/************************************/
func (c *Controller) Exit() {
	panic(ErrUserExit)
}
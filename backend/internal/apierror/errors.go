package apierror

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 错误码分类：
// 1xxx 认证/参数类
// 2xxx 资源/状态类
// 3xxx 业务逻辑/内部类
const (
	CodeUnauthorized  = 1001 // 未认证、登录失败、token 失效
	CodeForbidden     = 1002 // 无权限
	CodeInvalidParam  = 1003 // 参数错误（格式、类型、缺失）
	CodeInvalidBody   = 1004 // 请求体解析失败

	CodeNotFound     = 2001 // 资源不存在
	CodeInvalidState = 2002 // 资源状态异常（已禁用等）
	CodeConflict     = 2003 // 资源冲突（已存在）

	CodeServiceError  = 3001 // 服务未配置
	CodeRoutingError  = 3002 // 无可用供应商密钥
	CodeInternalError = 3003 // 内部错误（DB 等）
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Write 以 HTTP 200 返回业务错误，避免 4xx/5xx 污染业务层。
func Write(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{Code: code, Message: message})
}

// Abort 在中间件中使用，中断请求并返回业务错误码。
func Abort(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(http.StatusOK, Response{Code: code, Message: message})
}

package logger

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

var output = log.New(os.Stdout, "", 0)

// Infof 输出普通运行日志，用绿色标识正常流程。
func Infof(format string, args ...interface{}) {
	printLevel("INFO", colorGreen, format, args...)
}

// Warnf 输出警告日志，用黄色标识需要关注但不中断服务的问题。
func Warnf(format string, args ...interface{}) {
	printLevel("WARN", colorYellow, format, args...)
}

// Errorf 输出错误日志，用红色标识失败或异常信息。
func Errorf(format string, args ...interface{}) {
	printLevel("ERROR", colorRed, format, args...)
}

// Fatalf 输出致命错误日志后退出进程，保持启动失败时立即中断。
func Fatalf(format string, args ...interface{}) {
	Errorf(format, args...)
	os.Exit(1)
}

// HTTPLogger 输出接口请求日志，状态码按结果分色便于快速定位异常请求。
func HTTPLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		latency := time.Since(start)
		method := c.Request.Method
		path := c.Request.URL.RequestURI()
		statusText := colorize(fmt.Sprintf("%3d", status), statusColor(status))
		methodText := colorize(method, methodColor(method))

		output.Printf(
			"%s %s %s %s %s %s",
			time.Now().Format("2006-01-02 15:04:05"),
			colorize("HTTP", colorCyan),
			statusText,
			methodText,
			path,
			colorize(latency.String(), colorGray),
		)

		if len(c.Errors) > 0 {
			Errorf("接口处理错误: %s", c.Errors.String())
		}
	}
}

func printLevel(level string, levelColor string, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	output.Printf(
		"%s %s %s",
		time.Now().Format("2006-01-02 15:04:05"),
		colorize(level, levelColor),
		message,
	)
}

func colorize(text string, color string) string {
	if os.Getenv("NO_COLOR") != "" {
		return text
	}
	return color + text + colorReset
}

func statusColor(status int) string {
	switch {
	case status >= 500:
		return colorRed
	case status >= 400:
		return colorYellow
	case status >= 300:
		return colorBlue
	default:
		return colorGreen
	}
}

func methodColor(method string) string {
	switch method {
	case "GET":
		return colorBlue
	case "POST":
		return colorGreen
	case "PUT", "PATCH":
		return colorYellow
	case "DELETE":
		return colorRed
	default:
		return colorCyan
	}
}

package legal

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

type pageData struct {
	Title   string
	Content string
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Register(router *gin.Engine) {
	router.GET("/user-agreement", h.userAgreement)
	router.HEAD("/user-agreement", h.userAgreementHead)
	router.GET("/legal/user-agreement.html", h.userAgreement)
	router.HEAD("/legal/user-agreement.html", h.userAgreementHead)
	router.GET("/privacy-policy", h.privacyPolicy)
	router.HEAD("/privacy-policy", h.privacyPolicyHead)
	router.GET("/legal/privacy-policy.html", h.privacyPolicy)
	router.HEAD("/legal/privacy-policy.html", h.privacyPolicyHead)
}

func (h *Handler) userAgreement(c *gin.Context) {
	renderAgreement(c, "用户协议", UserAgreement)
}

func (h *Handler) privacyPolicy(c *gin.Context) {
	renderAgreement(c, "隐私政策", PrivacyPolicy)
}

func (h *Handler) userAgreementHead(c *gin.Context) {
	writeAgreementHead(c)
}

func (h *Handler) privacyPolicyHead(c *gin.Context) {
	writeAgreementHead(c)
}

func writeAgreementHead(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
}

func renderAgreement(c *gin.Context, title string, content string) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := agreementTemplate.Execute(c.Writer, pageData{
		Title:   title,
		Content: content,
	}); err != nil {
		c.String(http.StatusInternalServerError, "页面渲染失败")
	}
}

var agreementTemplate = template.Must(template.New("agreement").Parse(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
  <title>{{.Title}}</title>
  <style>
    :root {
      color-scheme: light;
      --text: #111827;
      --muted: #4b5563;
      --bg: #f8faef;
      --card: rgba(255, 255, 255, 0.92);
      --brand: #b8df35;
    }
    * {
      box-sizing: border-box;
    }
    body {
      margin: 0;
      min-height: 100vh;
      color: var(--text);
      background:
        radial-gradient(circle at 16% 0%, rgba(184, 223, 53, 0.36), transparent 34%),
        radial-gradient(circle at 88% 12%, rgba(232, 247, 175, 0.62), transparent 30%),
        linear-gradient(180deg, var(--bg) 0%, #ffffff 42%);
      font-family: -apple-system, BlinkMacSystemFont, "HarmonyOS Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif;
      line-height: 1.72;
    }
    main {
      width: min(100%, 760px);
      margin: 0 auto;
      padding: max(24px, env(safe-area-inset-top)) 18px max(32px, env(safe-area-inset-bottom));
    }
    .card {
      padding: 26px 22px 30px;
      border: 1px solid rgba(255, 255, 255, 0.76);
      border-radius: 28px;
      background: var(--card);
      box-shadow: 0 22px 60px rgba(79, 95, 22, 0.12);
      backdrop-filter: blur(18px);
    }
    h1 {
      margin: 0 0 18px;
      font-size: 26px;
      line-height: 1.24;
      letter-spacing: -0.02em;
    }
    .accent {
      width: 46px;
      height: 5px;
      margin-bottom: 18px;
      border-radius: 999px;
      background: var(--brand);
    }
    .content {
      white-space: pre-wrap;
      word-break: break-word;
      color: var(--muted);
      font-size: 15px;
    }
    @media (min-width: 640px) {
      main {
        padding-top: 42px;
      }
      .card {
        padding: 34px 34px 40px;
      }
      h1 {
        font-size: 30px;
      }
      .content {
        font-size: 16px;
      }
    }
  </style>
</head>
<body>
  <main>
    <section class="card">
      <div class="accent"></div>
      <h1>{{.Title}}</h1>
      <article class="content">{{.Content}}</article>
    </section>
  </main>
</body>
</html>`))

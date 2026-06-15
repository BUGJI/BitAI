package respond

import (
	stdhttp "net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(stdhttp.StatusOK, Response{Code: 0, Message: "成功", Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(stdhttp.StatusCreated, Response{Code: 0, Message: "创建成功", Data: data})
}

func Fail(c *gin.Context, status int, message string) {
	c.JSON(status, Response{Code: status, Message: LocalizeMessage(message)})
}

func LocalizeMessage(message string) string {
	value := strings.TrimSpace(message)
	if value == "" {
		return "请求失败"
	}
	if translated := localizeValidationError(value); translated != "" {
		return translated
	}
	exact := map[string]string{
		"missing access token":           "缺少访问令牌",
		"invalid access token":           "访问令牌无效",
		"forbidden":                      "无权访问",
		"user not found":                 "用户不存在",
		"invalid id":                     "编号无效",
		"no updates":                     "没有可更新的内容",
		"group not found":                "分组不存在",
		"upstream account not found":     "上游账号不存在",
		"missing api key":                "缺少接口密钥",
		"invalid api key":                "接口密钥无效",
		"invalid request body":           "请求体格式错误",
		"expires_at must be RFC3339":     "过期时间必须使用 RFC3339 格式",
		"amount_micros must be positive": "金额必须大于 0",
		"invalid credentials":            "账号或密码错误",
		"token invalid":                  "令牌无效或已过期",
		"api key not found":              "调用密钥不存在",
		"api key inactive":               "调用密钥未启用",
		"api key quota exceeded":         "调用密钥额度已用尽",
		"api key expired":                "调用密钥已过期",
		"no available upstream account":  "没有可用的上游账号",
		"insufficient balance":           "余额不足",
		"rate limited":                   "请求过于频繁，请稍后再试",
		"redeem code invalid":            "兑换码无效",
		"redeem code already used":       "兑换码已使用",
		"record not found":               "记录不存在",
	}
	if translated, ok := exact[value]; ok {
		return translated
	}
	lower := strings.ToLower(value)
	switch {
	case value == "EOF":
		return "请求体不能为空"
	case strings.Contains(lower, "invalid character"):
		return "请求体格式错误"
	case strings.Contains(lower, "unique constraint failed: users.email"):
		return "邮箱已存在"
	case strings.Contains(lower, "unique constraint failed: group_accounts"):
		return "绑定关系已存在"
	case strings.Contains(lower, "unique constraint failed"):
		return "数据已存在"
	case strings.Contains(lower, "constraint failed"):
		return "数据约束冲突"
	case strings.Contains(lower, "record not found"):
		return "记录不存在"
	case strings.Contains(lower, "message authentication failed"):
		return "上游凭据解密失败"
	}
	return value
}

var validationPattern = regexp.MustCompile(`Key: '([^']+)' Error:Field validation for '([^']+)' failed on the '([^']+)' tag`)

func localizeValidationError(value string) string {
	matches := validationPattern.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return ""
	}
	messages := make([]string, 0, len(matches))
	for _, match := range matches {
		field := fieldLabel(match[2])
		switch match[3] {
		case "required":
			messages = append(messages, field+"不能为空")
		case "email":
			messages = append(messages, field+"格式不正确")
		case "min":
			messages = append(messages, field+"长度不足")
		case "max":
			messages = append(messages, field+"超出允许长度")
		case "oneof":
			messages = append(messages, field+"取值不受支持")
		default:
			messages = append(messages, field+"校验失败")
		}
	}
	return strings.Join(messages, "；")
}

func fieldLabel(field string) string {
	labels := map[string]string{
		"Email":                "邮箱",
		"Password":             "密码",
		"DisplayName":          "显示名称",
		"CaptchaToken":         "图形验证码凭证",
		"CaptchaCode":          "图形验证码",
		"EmailToken":           "邮箱验证码凭证",
		"EmailCode":            "邮箱验证码",
		"RefreshToken":         "刷新令牌",
		"Name":                 "名称",
		"GroupID":              "分组编号",
		"QuotaLimitMicros":     "额度上限",
		"ExpiresAt":            "过期时间",
		"AmountMicros":         "金额",
		"Provider":             "处理方式",
		"Code":                 "兑换码",
		"Role":                 "角色",
		"Status":               "状态",
		"BalanceMicros":        "余额",
		"TotalRechargedMicros": "累计充值金额",
		"ConcurrencyLimit":     "并发限制",
		"RPMLimit":             "每分钟请求数",
		"Platform":             "平台",
		"Mode":                 "计费模式",
		"AuthType":             "鉴权方式",
		"Credentials":          "凭据",
		"BaseURL":              "基础地址",
		"ProxyURL":             "代理地址",
		"Priority":             "优先级",
		"Weight":               "权重",
		"RateMultiplierPPM":    "费率倍率",
		"Schedulable":          "可调度状态",
		"QuotaJSON":            "额度配置",
		"MetadataJSON":         "元数据配置",
		"UpstreamAccountID":    "上游账号编号",
		"Enabled":              "启用状态",
		"Key":                  "键名",
		"Value":                "值",
		"IsPublic":             "公开状态",
		"MaxUses":              "最大使用次数",
	}
	if label, ok := labels[field]; ok {
		return label
	}
	return "参数 " + field
}

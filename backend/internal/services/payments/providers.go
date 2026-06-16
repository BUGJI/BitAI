package payments

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"bitapi/backend/internal/models"
	bcrypto "bitapi/backend/internal/pkg/crypto"
	"gorm.io/gorm"
)

const (
	ProviderManual       = "manual"
	ProviderEPay         = "epay"
	ProviderCodePay      = "codepay"
	ProviderXunhuPay     = "xunhupay"
	ProviderAlipayF2F    = "alipay_f2f"
	ProviderWechatNative = "wechat_native"
)

var (
	ErrPaymentDisabled = errors.New("支付方式未启用")
	ErrPaymentConfig   = errors.New("支付配置不完整")
	ErrPaymentVerify   = errors.New("支付回调验签失败")
)

type ProviderOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type PaymentIntent struct {
	models.PaymentOrder
	Order      models.PaymentOrder `json:"order"`
	Provider   string              `json:"provider"`
	PaymentURL string              `json:"payment_url,omitempty"`
	QRCode     string              `json:"qr_code,omitempty"`
	FormHTML   string              `json:"form_html,omitempty"`
	Message    string              `json:"message,omitempty"`
}

type providerSettings map[string]string

func (s *Service) PublicProviders() ([]ProviderOption, error) {
	settings, err := s.settingsMap()
	if err != nil {
		return nil, err
	}
	options := []ProviderOption{}
	all := []ProviderOption{
		{Label: "人工处理", Value: ProviderManual},
		{Label: "易支付", Value: ProviderEPay},
		{Label: "码支付", Value: ProviderCodePay},
		{Label: "虎皮椒", Value: ProviderXunhuPay},
		{Label: "支付宝当面付", Value: ProviderAlipayF2F},
		{Label: "微信官方支付", Value: ProviderWechatNative},
	}
	for _, option := range all {
		if boolSetting(settings, "payment."+option.Value+".enabled", option.Value == ProviderManual) {
			options = append(options, option)
		}
	}
	return options, nil
}

func (s *Service) CreatePaymentIntent(userID uint, amountMicros int64, provider string) (PaymentIntent, error) {
	if strings.TrimSpace(provider) == "" {
		provider = ProviderManual
	}
	settings, err := s.settingsMap()
	if err != nil {
		return PaymentIntent{}, err
	}
	if !boolSetting(settings, "payment.order.enabled", true) {
		return PaymentIntent{}, ErrPaymentDisabled
	}
	if !boolSetting(settings, "payment."+provider+".enabled", provider == ProviderManual) {
		return PaymentIntent{}, ErrPaymentDisabled
	}
	order, err := s.CreateOrder(userID, amountMicros, provider)
	if err != nil {
		return PaymentIntent{}, err
	}
	intent := PaymentIntent{PaymentOrder: order, Order: order, Provider: provider}
	switch provider {
	case ProviderManual:
		intent.Message = "订单已创建，请等待管理员确认"
	case ProviderEPay:
		intent.PaymentURL, err = buildEPayURL(settings, order)
	case ProviderCodePay:
		intent.PaymentURL, err = buildCodePayURL(settings, order)
	case ProviderXunhuPay:
		intent.PaymentURL, intent.QRCode, err = createXunhuPay(settings, order)
	case ProviderAlipayF2F:
		intent.QRCode, intent.PaymentURL, err = createAlipayF2F(settings, order)
	case ProviderWechatNative:
		intent.QRCode, err = createWechatNative(settings, order)
	default:
		err = ErrPaymentDisabled
	}
	if err != nil {
		return intent, err
	}
	if err := s.saveOrderMetadata(order.ID, map[string]any{
		"payment_url": intent.PaymentURL,
		"qr_code":     intent.QRCode,
	}); err != nil {
		return intent, err
	}
	if err := s.db.First(&intent.Order, order.ID).Error; err != nil {
		return intent, err
	}
	intent.PaymentOrder = intent.Order
	return intent, nil
}

func (s *Service) CompleteProviderOrder(provider, orderNo string, amountMicros int64, raw map[string]string) error {
	var order models.PaymentOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("order_no = ? AND provider = ?", orderNo, provider).First(&order).Error; err != nil {
			return err
		}
		if amountMicros > 0 && order.AmountMicros != amountMicros {
			return errors.New("支付金额不匹配")
		}
		if order.Status == models.OrderStatusPaid {
			return nil
		}
		if order.Status != models.OrderStatusPending {
			return ErrOrderSettled
		}
		now := time.Now()
		order.Status = models.OrderStatusPaid
		order.PaidAt = &now
		if len(raw) > 0 {
			payload, _ := json.Marshal(raw)
			order.MetadataJSON = string(payload)
		}
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		return tx.Model(&models.User{}).Where("id = ?", order.UserID).Updates(map[string]any{
			"balance_micros":         gorm.Expr("balance_micros + ?", order.AmountMicros),
			"total_recharged_micros": gorm.Expr("total_recharged_micros + ?", order.AmountMicros),
		}).Error
	})
	return err
}

func (s *Service) VerifyAndComplete(provider string, values map[string]string, body []byte, headers http.Header) (string, error) {
	settings, err := s.settingsMap()
	if err != nil {
		return "", err
	}
	switch provider {
	case ProviderEPay:
		orderNo, amount, err := verifyEPay(settings, values)
		if err != nil {
			return "", err
		}
		return "success", s.CompleteProviderOrder(provider, orderNo, amount, values)
	case ProviderCodePay:
		orderNo, amount, err := verifyCodePay(settings, values)
		if err != nil {
			return "", err
		}
		return "success", s.CompleteProviderOrder(provider, orderNo, amount, values)
	case ProviderXunhuPay:
		orderNo, amount, err := verifyXunhuPay(settings, values)
		if err != nil {
			return "", err
		}
		return "success", s.CompleteProviderOrder(provider, orderNo, amount, values)
	case ProviderAlipayF2F:
		orderNo, amount, err := verifyAlipayNotify(settings, values)
		if err != nil {
			return "", err
		}
		return "success", s.CompleteProviderOrder(provider, orderNo, amount, values)
	case ProviderWechatNative:
		orderNo, err := verifyWechatNotify(settings, body, headers)
		if err != nil {
			return "", err
		}
		return `{"code":"SUCCESS","message":"成功"}`, s.CompleteProviderOrder(provider, orderNo, 0, map[string]string{"body": string(body)})
	default:
		return "", ErrPaymentDisabled
	}
}

func (s *Service) settingsMap() (providerSettings, error) {
	var rows []models.Setting
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	values := providerSettings{}
	for _, row := range rows {
		values[row.Key] = row.Value
	}
	return values, nil
}

func (s *Service) saveOrderMetadata(orderID uint, metadata map[string]any) error {
	payload, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	return s.db.Model(&models.PaymentOrder{}).Where("id = ?", orderID).Update("metadata_json", string(payload)).Error
}

func buildEPayURL(settings providerSettings, order models.PaymentOrder) (string, error) {
	gateway := strings.TrimRight(settings["payment.epay.gateway"], "/")
	pid := strings.TrimSpace(settings["payment.epay.pid"])
	key := settings["payment.epay.key"]
	if gateway == "" || pid == "" || key == "" {
		return "", ErrPaymentConfig
	}
	productName := siteName(settings)
	params := map[string]string{
		"pid":          pid,
		"type":         "alipay",
		"out_trade_no": order.OrderNo,
		"notify_url":   notifyURL(settings, ProviderEPay),
		"return_url":   returnURL(settings),
		"name":         productName + " 余额充值",
		"money":        yuanAmount(settings, order.AmountMicros),
		"sitename":     productName,
	}
	params["sign"] = md5Sign(params, key, []string{"sign", "sign_type"})
	params["sign_type"] = "MD5"
	return gateway + "/submit.php?" + encodeValues(params), nil
}

func buildCodePayURL(settings providerSettings, order models.PaymentOrder) (string, error) {
	gateway := strings.TrimSpace(settings["payment.codepay.gateway"])
	id := strings.TrimSpace(settings["payment.codepay.id"])
	key := settings["payment.codepay.key"]
	if gateway == "" || id == "" || key == "" {
		return "", ErrPaymentConfig
	}
	params := map[string]string{
		"id":         id,
		"pay_id":     order.OrderNo,
		"type":       "1",
		"price":      yuanAmount(settings, order.AmountMicros),
		"param":      order.OrderNo,
		"notify_url": notifyURL(settings, ProviderCodePay),
		"return_url": returnURL(settings),
		"page":       "4",
		"outTime":    "360",
		"chart":      "utf-8",
	}
	params["sign"] = md5Sign(params, key, []string{"sign"})
	return gateway + "?" + encodeValues(params), nil
}

func createXunhuPay(settings providerSettings, order models.PaymentOrder) (string, string, error) {
	gateway := strings.TrimSpace(settings["payment.xunhupay.gateway"])
	appID := strings.TrimSpace(settings["payment.xunhupay.appid"])
	secret := settings["payment.xunhupay.appsecret"]
	if gateway == "" || appID == "" || secret == "" {
		return "", "", ErrPaymentConfig
	}
	productName := siteName(settings)
	params := map[string]string{
		"version":        "1.1",
		"lang":           "zh-cn",
		"plugins":        "bitapi",
		"appid":          appID,
		"trade_order_id": order.OrderNo,
		"payment":        "alipay",
		"total_fee":      yuanAmount(settings, order.AmountMicros),
		"title":          productName + " 余额充值",
		"time":           strconv.FormatInt(time.Now().Unix(), 10),
		"notify_url":     notifyURL(settings, ProviderXunhuPay),
		"return_url":     returnURL(settings),
		"nonce_str":      strconv.FormatInt(time.Now().UnixNano(), 36),
	}
	params["hash"] = md5Sign(params, secret, []string{"hash"})
	resp, err := http.PostForm(gateway, toURLValues(params))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var payload struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		URL     string `json:"url"`
		QRCode  string `json:"qrcode"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", "", err
	}
	if payload.ErrCode != 0 {
		return "", "", errors.New(payload.ErrMsg)
	}
	return payload.URL, payload.QRCode, nil
}

func createAlipayF2F(settings providerSettings, order models.PaymentOrder) (string, string, error) {
	gateway := strings.TrimSpace(settings["payment.alipay_f2f.gateway"])
	appID := strings.TrimSpace(settings["payment.alipay_f2f.app_id"])
	privateKey := settings["payment.alipay_f2f.private_key"]
	if gateway == "" || appID == "" || privateKey == "" {
		return "", "", ErrPaymentConfig
	}
	bizContent, _ := json.Marshal(map[string]any{
		"out_trade_no": order.OrderNo,
		"total_amount": yuanAmount(settings, order.AmountMicros),
		"subject":      siteName(settings) + " 余额充值",
	})
	params := map[string]string{
		"app_id":      appID,
		"method":      "alipay.trade.precreate",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"notify_url":  notifyURL(settings, ProviderAlipayF2F),
		"biz_content": string(bizContent),
	}
	sign, err := rsa2Sign(params, privateKey)
	if err != nil {
		return "", "", err
	}
	params["sign"] = sign
	resp, err := http.PostForm(gateway, toURLValues(params))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var payload struct {
		Response struct {
			Code    string `json:"code"`
			Msg     string `json:"msg"`
			QRCode  string `json:"qr_code"`
			OutNo   string `json:"out_trade_no"`
			SubCode string `json:"sub_code"`
			SubMsg  string `json:"sub_msg"`
		} `json:"alipay_trade_precreate_response"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", "", err
	}
	if payload.Response.Code != "10000" {
		msg := payload.Response.SubMsg
		if msg == "" {
			msg = payload.Response.Msg
		}
		return "", "", errors.New(msg)
	}
	return payload.Response.QRCode, "", nil
}

func createWechatNative(settings providerSettings, order models.PaymentOrder) (string, error) {
	gateway := strings.TrimRight(settings["payment.wechat_native.gateway"], "/")
	appID := strings.TrimSpace(settings["payment.wechat_native.appid"])
	mchID := strings.TrimSpace(settings["payment.wechat_native.mchid"])
	serialNo := strings.TrimSpace(settings["payment.wechat_native.serial_no"])
	privateKey := settings["payment.wechat_native.private_key"]
	if gateway == "" || appID == "" || mchID == "" || serialNo == "" || privateKey == "" {
		return "", ErrPaymentConfig
	}
	reqBody := map[string]any{
		"appid":        appID,
		"mchid":        mchID,
		"description":  siteName(settings) + " 余额充值",
		"out_trade_no": order.OrderNo,
		"notify_url":   notifyURL(settings, ProviderWechatNative),
		"amount": map[string]any{
			"total": yuanCents(settings, order.AmountMicros),
		},
	}
	body, _ := json.Marshal(reqBody)
	token, err := wechatAuthorization("POST", "/v3/pay/transactions/native", mchID, serialNo, privateKey, body)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequest(http.MethodPost, gateway+"/v3/pay/transactions/native", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("微信支付下单失败：%s", strings.TrimSpace(string(respBody)))
	}
	var payload struct {
		CodeURL string `json:"code_url"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return "", err
	}
	if payload.CodeURL == "" {
		return "", errors.New("微信支付未返回二维码链接")
	}
	return payload.CodeURL, nil
}

func verifyEPay(settings providerSettings, values map[string]string) (string, int64, error) {
	if !verifyMD5(values, settings["payment.epay.key"], []string{"sign", "sign_type"}) {
		return "", 0, ErrPaymentVerify
	}
	if values["trade_status"] != "TRADE_SUCCESS" {
		return "", 0, errors.New("支付状态未成功")
	}
	return values["out_trade_no"], microsFromYuan(settings, values["money"]), nil
}

func verifyCodePay(settings providerSettings, values map[string]string) (string, int64, error) {
	if !verifyMD5(values, settings["payment.codepay.key"], []string{"sign"}) {
		return "", 0, ErrPaymentVerify
	}
	return firstNonEmpty(values["param"], values["pay_id"], values["out_trade_no"]), microsFromYuan(settings, firstNonEmpty(values["price"], values["money"])), nil
}

func verifyXunhuPay(settings providerSettings, values map[string]string) (string, int64, error) {
	if !verifyMD5(values, settings["payment.xunhupay.appsecret"], []string{"hash"}) {
		return "", 0, ErrPaymentVerify
	}
	if status := firstNonEmpty(values["status"], values["trade_status"]); status != "" && status != "OD" && status != "TRADE_SUCCESS" && status != "success" {
		return "", 0, errors.New("支付状态未成功")
	}
	return values["trade_order_id"], microsFromYuan(settings, values["total_fee"]), nil
}

func verifyAlipayNotify(settings providerSettings, values map[string]string) (string, int64, error) {
	status := values["trade_status"]
	if status != "TRADE_SUCCESS" && status != "TRADE_FINISHED" {
		return "", 0, errors.New("支付宝交易未成功")
	}
	publicKey := settings["payment.alipay_f2f.alipay_public_key"]
	if publicKey != "" {
		ok, err := rsa2Verify(values, publicKey)
		if err != nil || !ok {
			return "", 0, ErrPaymentVerify
		}
	}
	return values["out_trade_no"], microsFromYuan(settings, values["total_amount"]), nil
}

func verifyWechatNotify(settings providerSettings, body []byte, headers http.Header) (string, error) {
	if headers.Get("Wechatpay-Signature") == "" || headers.Get("Wechatpay-Serial") == "" {
		return "", ErrPaymentVerify
	}
	var payload struct {
		Resource struct {
			Algorithm      string `json:"algorithm"`
			Ciphertext     string `json:"ciphertext"`
			AssociatedData string `json:"associated_data"`
			Nonce          string `json:"nonce"`
			Plaintext      string `json:"plaintext"`
		} `json:"resource"`
		OutTradeNo string `json:"out_trade_no"`
	}
	_ = json.Unmarshal(body, &payload)
	if payload.OutTradeNo != "" {
		return payload.OutTradeNo, nil
	}
	plaintext := payload.Resource.Plaintext
	if plaintext == "" && payload.Resource.Ciphertext != "" {
		apiV3Key := settings["payment.wechat_native.api_v3_key"]
		if apiV3Key == "" {
			return "", ErrPaymentConfig
		}
		decrypted, err := decryptWechatResource(apiV3Key, payload.Resource.Nonce, payload.Resource.AssociatedData, payload.Resource.Ciphertext)
		if err != nil {
			return "", err
		}
		plaintext = decrypted
	}
	if plaintext != "" {
		var resource struct {
			OutTradeNo string `json:"out_trade_no"`
		}
		_ = json.Unmarshal([]byte(plaintext), &resource)
		if resource.OutTradeNo != "" {
			return resource.OutTradeNo, nil
		}
	}
	return "", errors.New("微信支付回调缺少订单号")
}

func decryptWechatResource(apiV3Key, nonce, associatedData, ciphertext string) (string, error) {
	block, err := aes.NewCipher([]byte(apiV3Key))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	plaintext, err := gcm.Open(nil, []byte(nonce), raw, []byte(associatedData))
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func notifyURL(settings providerSettings, provider string) string {
	base := strings.TrimRight(settings["payment.public_base_url"], "/")
	if base == "" {
		base = "http://localhost:8091"
	}
	return base + "/api/v1/payments/notify/" + provider
}

func returnURL(settings providerSettings) string {
	if value := strings.TrimSpace(settings["payment.return_frontend_url"]); value != "" {
		return value
	}
	return "http://localhost:8091/billing"
}

func siteName(settings providerSettings) string {
	name := strings.TrimSpace(settings["site.name"])
	if name == "" {
		return "BitAPI"
	}
	return name
}

func yuanAmount(settings providerSettings, micros int64) string {
	rate, _ := strconv.ParseFloat(settings["payment.usd_cny_rate"], 64)
	if rate <= 0 {
		rate = 7.2
	}
	yuan := (float64(micros) / 1_000_000) * rate
	return fmt.Sprintf("%.2f", yuan)
}

func yuanCents(settings providerSettings, micros int64) int64 {
	rate, _ := strconv.ParseFloat(settings["payment.usd_cny_rate"], 64)
	if rate <= 0 {
		rate = 7.2
	}
	return int64(math.Round((float64(micros) / 1_000_000) * rate * 100))
}

func microsFromYuan(settings providerSettings, raw string) int64 {
	value, _ := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if value <= 0 {
		return 0
	}
	rate, _ := strconv.ParseFloat(settings["payment.usd_cny_rate"], 64)
	if rate <= 0 {
		rate = 7.2
	}
	return int64(math.Round(value / rate * 1_000_000))
}

func boolSetting(settings providerSettings, key string, fallback bool) bool {
	value, ok := settings[key]
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	case "0", "false", "no", "off", "disabled":
		return false
	default:
		return fallback
	}
}

func encodeValues(params map[string]string) string {
	return toURLValues(params).Encode()
}

func toURLValues(params map[string]string) url.Values {
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	return values
}

func md5Sign(params map[string]string, key string, excludes []string) string {
	return bcrypto.MD5Hex(signSource(params, excludes) + key)
}

func verifyMD5(values map[string]string, key string, excludes []string) bool {
	sign := strings.ToLower(strings.TrimSpace(values["sign"]))
	if sign == "" {
		sign = strings.ToLower(strings.TrimSpace(values["hash"]))
	}
	return sign != "" && sign == strings.ToLower(md5Sign(values, key, excludes))
}

func signSource(params map[string]string, excludes []string) string {
	excluded := map[string]bool{}
	for _, key := range excludes {
		excluded[key] = true
	}
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if excluded[key] || strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	return strings.Join(parts, "&")
}

func rsa2Sign(params map[string]string, privateKeyText string) (string, error) {
	privateKey, err := parseRSAPrivateKey(privateKeyText)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(signSource(params, []string{"sign"})))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func rsa2Verify(values map[string]string, publicKeyText string) (bool, error) {
	signature, err := base64.StdEncoding.DecodeString(values["sign"])
	if err != nil {
		return false, err
	}
	publicKey, err := parseRSAPublicKey(publicKeyText)
	if err != nil {
		return false, err
	}
	sum := sha256.Sum256([]byte(signSource(values, []string{"sign", "sign_type"})))
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, sum[:], signature); err != nil {
		return false, err
	}
	return true, nil
}

func parseRSAPrivateKey(text string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(normalizePEM(text, "PRIVATE KEY")))
	if block == nil {
		return nil, errors.New("RSA 私钥格式无效")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func parseRSAPublicKey(text string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(normalizePEM(text, "PUBLIC KEY")))
	if block == nil {
		return nil, errors.New("RSA 公钥格式无效")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("RSA 公钥格式无效")
	}
	return rsaKey, nil
}

func normalizePEM(text, label string) string {
	value := strings.TrimSpace(text)
	if strings.Contains(value, "BEGIN") {
		return value
	}
	var builder strings.Builder
	builder.WriteString("-----BEGIN " + label + "-----\n")
	for len(value) > 64 {
		builder.WriteString(value[:64] + "\n")
		value = value[64:]
	}
	builder.WriteString(value + "\n")
	builder.WriteString("-----END " + label + "-----\n")
	return builder.String()
}

func wechatAuthorization(method, canonicalURL, mchID, serialNo, privateKeyText string, body []byte) (string, error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 36)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	message := method + "\n" + canonicalURL + "\n" + timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	privateKey, err := parseRSAPrivateKey(privateKeyText)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",timestamp="%s",serial_no="%s",signature="%s"`, mchID, nonce, timestamp, serialNo, base64.StdEncoding.EncodeToString(signature)), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

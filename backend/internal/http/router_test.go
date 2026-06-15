package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"bitapi/backend/internal/config"
	"bitapi/backend/internal/db"
	"bitapi/backend/internal/models"
	bcrypto "bitapi/backend/internal/pkg/crypto"
	"gorm.io/gorm"
)

func TestRouterAuthAdminAndGatewayModels(t *testing.T) {
	cfg := config.Config{
		AppName:            "BitAPI Test",
		Env:                "test",
		HTTPAddr:           ":0",
		DatabaseDSN:        "file::memory:?cache=shared",
		JWTSecret:          "test-secret",
		AccessTokenTTL:     time.Hour,
		RefreshTokenTTL:    24 * time.Hour,
		CORSOrigins:        []string{"*"},
		BootstrapEmail:     "admin@example.test",
		BootstrapPassword:  "bitapi-admin",
		BootstrapName:      "Test Admin",
		DefaultUserBalance: 1_000_000,
	}
	conn, err := db.Open(cfg)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(conn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Seed(conn, cfg); err != nil {
		t.Fatalf("seed: %v", err)
	}
	router := NewRouter(conn, cfg)

	token := login(t, router, conn, cfg.BootstrapEmail, cfg.BootstrapPassword)
	adminGet(t, router, token, "/api/v1/admin/stats")
	upsertSetting(t, router, token)
	createEncryptedUpstream(t, router, conn, token)
	exerciseBilling(t, router, conn, token)
	apiKey := createAPIKey(t, router, token)
	models := gatewayGet(t, router, apiKey, "/v1/models")
	if models.Code != 0 {
		t.Fatalf("models response should be raw openai compatible, got code envelope: %+v", models)
	}
}

func exerciseBilling(t *testing.T, handler http.Handler, conn *gorm.DB, token string) {
	t.Helper()
	orderBody := postJSON(t, handler, token, "/api/v1/user/orders", map[string]any{
		"amount_micros": 500000,
		"provider":      "manual",
	})
	var order models.PaymentOrder
	if err := json.Unmarshal(orderBody.Data, &order); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	if order.Status != models.OrderStatusPending {
		t.Fatalf("new order status = %s", order.Status)
	}
	rejectOrderBody := postJSON(t, handler, token, "/api/v1/user/orders", map[string]any{
		"amount_micros": 300000,
		"provider":      "manual",
	})
	var rejectOrder models.PaymentOrder
	if err := json.Unmarshal(rejectOrderBody.Data, &rejectOrder); err != nil {
		t.Fatalf("decode reject order: %v", err)
	}
	rejected := postJSON(t, handler, token, "/api/v1/admin/orders/"+strconv.Itoa(int(rejectOrder.ID))+"/reject", nil)
	if rejected.Code != 0 {
		t.Fatalf("reject order failed: %+v", rejected)
	}
	if err := conn.First(&rejectOrder, rejectOrder.ID).Error; err != nil {
		t.Fatalf("read rejected order: %v", err)
	}
	if rejectOrder.Status != models.OrderStatusRejected {
		t.Fatalf("rejected order status = %s", rejectOrder.Status)
	}
	paid := postJSON(t, handler, token, "/api/v1/admin/orders/"+strconv.Itoa(int(order.ID))+"/mark-paid", nil)
	if paid.Code != 0 {
		t.Fatalf("mark paid failed: %+v", paid)
	}
	codeBody := postJSON(t, handler, token, "/api/v1/admin/redeem-codes", map[string]any{
		"amount_micros": 250000,
		"max_uses":      1,
	})
	var codeData struct {
		Code string `json:"code"`
		Item struct {
			Code       string `json:"code"`
			CodePrefix string `json:"code_prefix"`
		} `json:"item"`
	}
	if err := json.Unmarshal(codeBody.Data, &codeData); err != nil {
		t.Fatalf("decode redeem code: %v", err)
	}
	if codeData.Code == "" {
		t.Fatal("missing redeem code")
	}
	if len(codeData.Code) != 16 || strings.Contains(codeData.Code, "_") || codeData.Code != strings.ToUpper(codeData.Code) {
		t.Fatalf("redeem code format invalid: %q", codeData.Code)
	}
	if codeData.Item.Code != codeData.Code || codeData.Item.CodePrefix != codeData.Code {
		t.Fatalf("redeem code should be visible in admin data, got code=%q prefix=%q created=%q", codeData.Item.Code, codeData.Item.CodePrefix, codeData.Code)
	}
	redeemBody := postJSON(t, handler, token, "/api/v1/user/redeem", map[string]any{"code": codeData.Code})
	if redeemBody.Code != 0 {
		t.Fatalf("redeem failed: %+v", redeemBody)
	}
	disabledCodeBody := postJSON(t, handler, token, "/api/v1/admin/redeem-codes", map[string]any{
		"amount_micros": 100000,
		"max_uses":      1,
	})
	var disabledCode struct {
		Code string `json:"code"`
		Item struct {
			ID uint `json:"id"`
		} `json:"item"`
	}
	if err := json.Unmarshal(disabledCodeBody.Data, &disabledCode); err != nil {
		t.Fatalf("decode disabled redeem code: %v", err)
	}
	disabled := postJSON(t, handler, token, "/api/v1/admin/redeem-codes/"+strconv.Itoa(int(disabledCode.Item.ID))+"/disable", nil)
	if disabled.Code != 0 {
		t.Fatalf("disable redeem code failed: %+v", disabled)
	}
	enabled := postJSON(t, handler, token, "/api/v1/admin/redeem-codes/"+strconv.Itoa(int(disabledCode.Item.ID))+"/enable", nil)
	if enabled.Code != 0 {
		t.Fatalf("enable redeem code failed: %+v", enabled)
	}
	deletedCodeBody := postJSON(t, handler, token, "/api/v1/admin/redeem-codes", map[string]any{
		"amount_micros": 100000,
		"max_uses":      1,
	})
	var deletedCode struct {
		Item struct {
			ID uint `json:"id"`
		} `json:"item"`
	}
	if err := json.Unmarshal(deletedCodeBody.Data, &deletedCode); err != nil {
		t.Fatalf("decode deleted redeem code: %v", err)
	}
	deleteRedeemCode(t, handler, token, "/api/v1/admin/redeem-codes/"+strconv.Itoa(int(deletedCode.Item.ID)))
	var user models.User
	if err := conn.Where("email = ?", "admin@example.test").First(&user).Error; err != nil {
		t.Fatalf("read user: %v", err)
	}
	if user.BalanceMicros < 1_750_000 {
		t.Fatalf("balance not recharged, got %d", user.BalanceMicros)
	}
}

func deleteRedeemCode(t *testing.T, handler http.Handler, token, path string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("DELETE %s: status %d body %s", path, rec.Code, rec.Body.String())
	}
}

type envelope struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

func login(t *testing.T, handler http.Handler, conn *gorm.DB, email, password string) string {
	t.Helper()
	captcha := createChallenge(t, conn, models.ChallengeTypeCaptcha, "", "1234")
	body := postJSON(t, handler, "", "/api/v1/auth/login", map[string]any{
		"email":         email,
		"password":      password,
		"captcha_token": captcha,
		"captcha_code":  "1234",
	})
	if body.Code != 0 {
		t.Fatalf("login failed: %+v", body)
	}
	var data struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body.Data, &data); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if data.AccessToken == "" {
		t.Fatal("missing access token")
	}
	return data.AccessToken
}

func createChallenge(t *testing.T, conn *gorm.DB, challengeType, target, code string) string {
	t.Helper()
	token := "test-" + challengeType + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	item := models.VerificationChallenge{
		Type:      challengeType,
		Token:     token,
		Target:    target,
		CodeHash:  bcrypto.SHA256Hex(strings.ToLower(code)),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	if err := conn.Create(&item).Error; err != nil {
		t.Fatalf("create challenge: %v", err)
	}
	return token
}

func adminGet(t *testing.T, handler http.Handler, token, path string) envelope {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d body %s", path, rec.Code, rec.Body.String())
	}
	var body envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return body
}

func upsertSetting(t *testing.T, handler http.Handler, token string) {
	t.Helper()
	body := postJSON(t, handler, token, "/api/v1/admin/settings", map[string]any{
		"key":       "test.setting",
		"value":     "enabled",
		"is_public": true,
	})
	if body.Code != 0 {
		t.Fatalf("setting failed: %+v", body)
	}
}

func createAPIKey(t *testing.T, handler http.Handler, token string) string {
	t.Helper()
	body := postJSON(t, handler, token, "/api/v1/user/api-keys", map[string]any{
		"name":                 "test key",
		"quota_limit_micros":   1000000,
		"rate_limit_1d_micros": 0,
	})
	if body.Code != 0 {
		t.Fatalf("create key failed: %+v", body)
	}
	var data struct {
		Key    string `json:"key"`
		APIKey struct {
			Key string `json:"key"`
		} `json:"api_key"`
	}
	if err := json.Unmarshal(body.Data, &data); err != nil {
		t.Fatalf("decode key: %v", err)
	}
	if data.Key == "" {
		t.Fatal("missing api key")
	}
	if data.APIKey.Key != data.Key {
		t.Fatalf("api key should be available for later copy, got %q want %q", data.APIKey.Key, data.Key)
	}
	return data.Key
}

func createEncryptedUpstream(t *testing.T, handler http.Handler, conn *gorm.DB, token string) {
	t.Helper()
	body := postJSON(t, handler, token, "/api/v1/admin/upstream-accounts", map[string]any{
		"name":        "encrypted test",
		"credentials": "sk-test-secret",
		"schedulable": false,
		"status":      "disabled",
	})
	if body.Code != 0 {
		t.Fatalf("create upstream failed: %+v", body)
	}
	var data struct {
		ID          uint   `json:"id"`
		Credential  string `json:"credentials"`
		Schedulable bool   `json:"schedulable"`
	}
	if err := json.Unmarshal(body.Data, &data); err != nil {
		t.Fatalf("decode upstream: %v", err)
	}
	if data.Credential != "********" {
		t.Fatalf("credential should be masked, got %q", data.Credential)
	}
	if data.Schedulable {
		t.Fatal("schedulable=false should be preserved")
	}
	var account models.UpstreamAccount
	if err := conn.First(&account, data.ID).Error; err != nil {
		t.Fatalf("read account: %v", err)
	}
	if !strings.HasPrefix(account.CredentialsEnc, "enc:") {
		t.Fatalf("credential should be encrypted, got %q", account.CredentialsEnc)
	}
	if account.Schedulable {
		t.Fatal("database schedulable=false should be preserved")
	}
}

func postJSON(t *testing.T, handler http.Handler, token, path string, payload map[string]any) envelope {
	t.Helper()
	raw, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("POST %s: status %d body %s", path, rec.Code, rec.Body.String())
	}
	var body envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return body
}

func gatewayGet(t *testing.T, handler http.Handler, apiKey, path string) envelope {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d body %s", path, rec.Code, rec.Body.String())
	}
	var body envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return body
}

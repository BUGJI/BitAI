package verification

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/big"
	mrand "math/rand"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"bitapi/backend/internal/models"
	bcrypto "bitapi/backend/internal/pkg/crypto"
	"github.com/google/uuid"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"gorm.io/gorm"
)

var (
	ErrChallengeInvalid = errors.New("验证码无效或已过期")
	ErrSMTPNotReady     = errors.New("SMTP 未启用或配置不完整")
)

type Service struct {
	db *gorm.DB
}

type Captcha struct {
	Token     string    `json:"captcha_token"`
	Image     string    `json:"captcha_image"`
	ExpiresAt time.Time `json:"expires_at"`
}

type EmailCodeInput struct {
	Email        string
	CaptchaToken string
	CaptchaCode  string
}

type SMTPConfig struct {
	Enabled    bool
	Host       string
	Port       int
	Username   string
	Password   string
	FromEmail  string
	FromName   string
	SiteName   string
	Encryption string
}

func New(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreateCaptcha() (Captcha, error) {
	code, err := randomCaptchaCode(5)
	if err != nil {
		return Captcha{}, err
	}
	token := uuid.NewString()
	expiresAt := time.Now().Add(5 * time.Minute)
	item := models.VerificationChallenge{
		Type:      models.ChallengeTypeCaptcha,
		Token:     token,
		CodeHash:  bcrypto.SHA256Hex(strings.ToLower(code)),
		ExpiresAt: expiresAt,
	}
	if err := s.db.Create(&item).Error; err != nil {
		return Captcha{}, err
	}
	imageData, err := renderCaptcha(code)
	if err != nil {
		return Captcha{}, err
	}
	return Captcha{Token: token, Image: imageData, ExpiresAt: expiresAt}, nil
}

func (s *Service) VerifyCaptcha(token, code string) error {
	return s.consumeChallenge(models.ChallengeTypeCaptcha, token, "", code)
}

func (s *Service) SendEmailCode(input EmailCodeInput) (string, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" {
		return "", errors.New("邮箱不能为空")
	}
	if err := s.VerifyCaptcha(input.CaptchaToken, input.CaptchaCode); err != nil {
		return "", err
	}
	cfg, err := s.SMTPConfig()
	if err != nil {
		return "", err
	}
	if !cfg.Ready() {
		return "", ErrSMTPNotReady
	}
	code, err := randomDigits(6)
	if err != nil {
		return "", err
	}
	token := uuid.NewString()
	expiresAt := time.Now().Add(10 * time.Minute)
	item := models.VerificationChallenge{
		Type:      models.ChallengeTypeEmail,
		Token:     token,
		Target:    email,
		CodeHash:  bcrypto.SHA256Hex(code),
		ExpiresAt: expiresAt,
	}
	if err := s.db.Create(&item).Error; err != nil {
		return "", err
	}
	if err := sendMail(cfg, email, cfg.SiteName+" 注册邮箱验证码", fmt.Sprintf("您的 %s 注册邮箱验证码是：%s\n\n验证码 10 分钟内有效。", cfg.SiteName, code)); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Service) VerifyEmailCode(email, token, code string) error {
	return s.consumeChallenge(models.ChallengeTypeEmail, token, strings.ToLower(strings.TrimSpace(email)), code)
}

func (s *Service) SMTPConfig() (SMTPConfig, error) {
	values, err := s.settingsMap()
	if err != nil {
		return SMTPConfig{}, err
	}
	port, _ := strconv.Atoi(strings.TrimSpace(values["smtp.port"]))
	if port == 0 {
		port = 587
	}
	return SMTPConfig{
		Enabled:    parseBool(values["smtp.enabled"]),
		Host:       strings.TrimSpace(values["smtp.host"]),
		Port:       port,
		Username:   strings.TrimSpace(values["smtp.username"]),
		Password:   values["smtp.password"],
		FromEmail:  strings.TrimSpace(values["smtp.from_email"]),
		FromName:   strings.TrimSpace(values["smtp.from_name"]),
		SiteName:   siteName(values),
		Encryption: strings.ToLower(strings.TrimSpace(values["smtp.encryption"])),
	}, nil
}

func (s *Service) consumeChallenge(challengeType, token, target, code string) error {
	token = strings.TrimSpace(token)
	code = strings.ToLower(strings.TrimSpace(code))
	if token == "" || code == "" {
		return ErrChallengeInvalid
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var item models.VerificationChallenge
		query := tx.Where("type = ? AND token = ?", challengeType, token)
		if target != "" {
			query = query.Where("target = ?", target)
		}
		if err := query.First(&item).Error; err != nil {
			return ErrChallengeInvalid
		}
		if item.UsedAt != nil || time.Now().After(item.ExpiresAt) || item.CodeHash != bcrypto.SHA256Hex(code) {
			return ErrChallengeInvalid
		}
		now := time.Now()
		item.UsedAt = &now
		return tx.Save(&item).Error
	})
}

func (s *Service) settingsMap() (map[string]string, error) {
	var rows []models.Setting
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	values := map[string]string{}
	for _, row := range rows {
		values[row.Key] = row.Value
	}
	return values, nil
}

func (cfg SMTPConfig) Ready() bool {
	return cfg.Enabled && cfg.Host != "" && cfg.Port > 0 && cfg.FromEmail != ""
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}

func siteName(values map[string]string) string {
	name := strings.TrimSpace(values["site.name"])
	if name == "" {
		return "BitAPI"
	}
	return name
}

func sendMail(cfg SMTPConfig, to, subject, body string) error {
	fromName := cfg.FromName
	if fromName == "" {
		fromName = cfg.SiteName
	}
	if fromName == "" {
		fromName = "BitAPI"
	}
	fromHeader := fmt.Sprintf("%s <%s>", fromName, cfg.FromEmail)
	message := strings.Join([]string{
		"From: " + fromHeader,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	auth := smtp.Auth(nil)
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}
	if cfg.Encryption == "ssl" || cfg.Encryption == "tls" {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return err
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return err
		}
		defer client.Close()
		return sendWithClient(client, auth, cfg.FromEmail, []string{to}, []byte(message))
	}
	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Close()
	if cfg.Encryption == "" || cfg.Encryption == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
				return err
			}
		}
	}
	return sendWithClient(client, auth, cfg.FromEmail, []string{to}, []byte(message))
}

func sendWithClient(client *smtp.Client, auth smtp.Auth, from string, to []string, message []byte) error {
	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return err
			}
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func randomDigits(length int) (string, error) {
	var builder strings.Builder
	builder.Grow(length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		builder.WriteByte(byte('0' + n.Int64()))
	}
	return builder.String(), nil
}

func renderCaptcha(code string) (string, error) {
	const width = 180
	const height = 60
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	seed, err := randomInt64()
	if err != nil {
		return "", err
	}
	rng := mrand.New(mrand.NewSource(seed))
	paintCaptchaBackground(img, rng)
	for i := 0; i < 5; i++ {
		drawCurve(img, rng, randomSoftColor(rng, 85))
	}
	face := captchaFontFace(36)
	defer face.Close()
	palette := []color.RGBA{
		{R: 31, G: 92, B: 171, A: 255},
		{R: 202, G: 67, B: 74, A: 255},
		{R: 42, G: 133, B: 93, A: 255},
		{R: 143, G: 79, B: 180, A: 255},
		{R: 212, G: 111, B: 32, A: 255},
	}
	for i, ch := range code {
		glyph := renderGlyph(face, string(ch), palette[rng.Intn(len(palette))])
		centerX := 25 + i*32 + rng.Intn(9) - 4
		centerY := 32 + rng.Intn(11) - 5
		angle := (rng.Float64() - 0.5) * 0.58
		drawRotated(img, glyph, centerX, centerY, angle)
	}
	for i := 0; i < 4; i++ {
		drawLine(img, rng.Intn(width), rng.Intn(height), rng.Intn(width), rng.Intn(height), randomSoftColor(rng, 110))
	}
	addNoise(img, rng, 520)
	addStrikeDots(img, rng, 70)
	img = waveDistort(img, rng)
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

func randomCaptchaCode(length int) (string, error) {
	const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	var builder strings.Builder
	builder.Grow(length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		builder.WriteByte(alphabet[n.Int64()])
	}
	return builder.String(), nil
}

func randomInt64() (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}

func captchaFontFace(size float64) font.Face {
	for _, path := range fontCandidates() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		parsed, err := opentype.Parse(data)
		if err != nil {
			continue
		}
		face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: size, DPI: 96, Hinting: font.HintingFull})
		if err == nil {
			return face
		}
	}
	return basicfont.Face7x13
}

func fontCandidates() []string {
	windir := os.Getenv("WINDIR")
	if windir == "" {
		windir = `C:\Windows`
	}
	return []string{
		windir + `\Fonts\arialbd.ttf`,
		windir + `\Fonts\arial.ttf`,
		windir + `\Fonts\segoeuib.ttf`,
		windir + `\Fonts\segoeui.ttf`,
		windir + `\Fonts\calibrib.ttf`,
		windir + `\Fonts\calibri.ttf`,
		windir + `\Fonts\simhei.ttf`,
		"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf",
		"/System/Library/Fonts/Supplemental/Arial Bold.ttf",
	}
}

func paintCaptchaBackground(img *image.RGBA, rng *mrand.Rand) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			noise := uint8(rng.Intn(10))
			c := color.RGBA{R: 244 - noise, G: 248 - noise, B: 252, A: 255}
			if (x+y)%17 == 0 {
				c = color.RGBA{R: 232, G: 239, B: 250, A: 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
}

func randomSoftColor(rng *mrand.Rand, alpha uint8) color.RGBA {
	return color.RGBA{
		R: uint8(80 + rng.Intn(150)),
		G: uint8(80 + rng.Intn(145)),
		B: uint8(80 + rng.Intn(145)),
		A: alpha,
	}
}

func renderGlyph(face font.Face, text string, fg color.RGBA) *image.RGBA {
	glyph := image.NewRGBA(image.Rect(0, 0, 50, 54))
	draw.Draw(glyph, glyph.Bounds(), image.Transparent, image.Point{}, draw.Src)
	d := font.Drawer{
		Dst:  glyph,
		Src:  image.NewUniform(fg),
		Face: face,
		Dot:  fixed.P(6, 39),
	}
	d.DrawString(text)
	return glyph
}

func drawRotated(dst *image.RGBA, src *image.RGBA, centerX, centerY int, angle float64) {
	sw := src.Bounds().Dx()
	sh := src.Bounds().Dy()
	cosA := math.Cos(angle)
	sinA := math.Sin(angle)
	for dy := -sh / 2; dy < sh/2; dy++ {
		for dx := -sw / 2; dx < sw/2; dx++ {
			sx := int(cosA*float64(dx)+sinA*float64(dy)) + sw/2
			sy := int(-sinA*float64(dx)+cosA*float64(dy)) + sh/2
			if sx < 0 || sx >= sw || sy < 0 || sy >= sh {
				continue
			}
			c := color.RGBAModel.Convert(src.At(sx, sy)).(color.RGBA)
			if c.A == 0 {
				continue
			}
			blendPixel(dst, centerX+dx, centerY+dy, c)
		}
	}
}

func drawCurve(img *image.RGBA, rng *mrand.Rand, c color.RGBA) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	x0 := rng.Intn(w / 3)
	y0 := rng.Intn(h)
	x1 := w/3 + rng.Intn(w/3)
	y1 := rng.Intn(h)
	x2 := w*2/3 + rng.Intn(w/3)
	y2 := rng.Intn(h)
	lastX, lastY := x0, y0
	for i := 1; i <= 80; i++ {
		t := float64(i) / 80
		x := int(math.Pow(1-t, 2)*float64(x0) + 2*(1-t)*t*float64(x1) + t*t*float64(x2))
		y := int(math.Pow(1-t, 2)*float64(y0) + 2*(1-t)*t*float64(y1) + t*t*float64(y2))
		drawLine(img, lastX, lastY, x, y, c)
		lastX, lastY = x, y
	}
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := abs(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -abs(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		blendPixel(img, x0, y0, c)
		blendPixel(img, x0+1, y0, color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A / 2})
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func addNoise(img *image.RGBA, rng *mrand.Rand, count int) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	for i := 0; i < count; i++ {
		x := rng.Intn(w)
		y := rng.Intn(h)
		blendPixel(img, x, y, randomSoftColor(rng, uint8(45+rng.Intn(80))))
	}
}

func addStrikeDots(img *image.RGBA, rng *mrand.Rand, count int) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	for i := 0; i < count; i++ {
		x := rng.Intn(w)
		y := rng.Intn(h)
		c := randomSoftColor(rng, 95)
		blendPixel(img, x, y, c)
		blendPixel(img, x+1, y, c)
	}
}

func waveDistort(src *image.RGBA, rng *mrand.Rand) *image.RGBA {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.RGBA{R: 245, G: 248, B: 252, A: 255}}, image.Point{}, draw.Src)
	phase := rng.Float64() * math.Pi * 2
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		shift := int(math.Sin(float64(y)/8+phase) * 2)
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			sx := x + shift
			if sx < bounds.Min.X || sx >= bounds.Max.X {
				continue
			}
			dst.Set(x, y, src.At(sx, y))
		}
	}
	return dst
}

func blendPixel(img *image.RGBA, x, y int, src color.RGBA) {
	if !image.Pt(x, y).In(img.Bounds()) || src.A == 0 {
		return
	}
	offset := img.PixOffset(x, y)
	dstR := uint32(img.Pix[offset])
	dstG := uint32(img.Pix[offset+1])
	dstB := uint32(img.Pix[offset+2])
	dstA := uint32(img.Pix[offset+3])
	srcA := uint32(src.A)
	invA := 255 - srcA
	img.Pix[offset] = uint8((uint32(src.R)*srcA + dstR*invA) / 255)
	img.Pix[offset+1] = uint8((uint32(src.G)*srcA + dstG*invA) / 255)
	img.Pix[offset+2] = uint8((uint32(src.B)*srcA + dstB*invA) / 255)
	img.Pix[offset+3] = uint8(srcA + dstA*invA/255)
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

package adminauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Claims struct {
	Sub         int64  `json:"sub"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Exp         int64  `json:"exp"`
}

type Service struct {
	secret []byte
}

func New(secret string) *Service {
	return &Service{secret: []byte(strings.TrimSpace(secret))}
}

func (s *Service) IsConfigured() bool {
	return s != nil && len(s.secret) > 0
}

func (s *Service) IssueToken(userID int64, username string, displayName string, ttl time.Duration) (string, error) {
	if len(s.secret) == 0 {
		return "", errors.New("admin auth secret is empty")
	}

	claims := Claims{
		Sub:         userID,
		Username:    strings.TrimSpace(username),
		DisplayName: strings.TrimSpace(displayName),
		Exp:         time.Now().Add(ttl).Unix(),
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := s.sign(encodedPayload)
	return encodedPayload + "." + signature, nil
}

func (s *Service) ParseToken(token string) (Claims, error) {
	if len(s.secret) == 0 {
		return Claims{}, errors.New("admin auth secret is empty")
	}

	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 {
		return Claims{}, errors.New("invalid token format")
	}

	expectedSignature := s.sign(parts[0])
	if !hmac.Equal([]byte(expectedSignature), []byte(parts[1])) {
		return Claims{}, errors.New("invalid token signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, errors.New("invalid token payload")
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, errors.New("invalid token claims")
	}
	if claims.Sub <= 0 || strings.TrimSpace(claims.Username) == "" {
		return Claims{}, errors.New("invalid token subject")
	}
	if claims.Exp <= time.Now().Unix() {
		return Claims{}, errors.New("token expired")
	}

	return claims, nil
}

func (s *Service) sign(value string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func ClaimsFromContext(c *gin.Context) (Claims, bool) {
	value, ok := c.Get("admin_claims")
	if !ok {
		return Claims{}, false
	}
	claims, ok := value.(Claims)
	if !ok {
		return Claims{}, false
	}
	return claims, true
}

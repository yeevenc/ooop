package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ooop-admin-api/internal/config"
)

const (
	TokenTypeAccess = "access"
)

var (
	ErrInvalidToken = errors.New("无效的登录凭证")
	ErrExpiredToken = errors.New("登录凭证已过期")
)

type Token struct {
	AccessToken          string `json:"access_token"`
	AccessTokenExpiresIn int64  `json:"access_token_expires_in"`
}

type Claims struct {
	UserID    int64
	TokenID   string
	TokenType string
	ExpiresAt time.Time
	IssuedAt  time.Time
	Issuer    string
}

type TokenManager struct {
	secret         []byte
	issuer         string
	accessTokenTTL time.Duration
}

func NewTokenManager(cfg config.JWTConfig) *TokenManager {
	return &TokenManager{
		secret:         []byte(cfg.Secret),
		issuer:         cfg.Issuer,
		accessTokenTTL: cfg.AccessTokenTTL,
	}
}

func (m *TokenManager) NewToken(userID int64) (Token, error) {
	now := time.Now()

	accessToken, err := m.signJWT(Claims{
		UserID:    userID,
		TokenID:   randomTokenID(),
		TokenType: TokenTypeAccess,
		IssuedAt:  now,
		ExpiresAt: now.Add(m.accessTokenTTL),
		Issuer:    m.issuer,
	})
	if err != nil {
		return Token{}, err
	}

	return Token{
		AccessToken:          accessToken,
		AccessTokenExpiresIn: int64(m.accessTokenTTL.Seconds()),
	}, nil
}

func (m *TokenManager) Parse(token string, expectedType string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSignature := m.sign(signingInput)
	if !hmac.Equal([]byte(expectedSignature), []byte(parts[2])) {
		return Claims{}, ErrInvalidToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return Claims{}, ErrInvalidToken
	}

	claims, err := claimsFromPayload(payload)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	if claims.TokenType != expectedType || claims.Issuer != m.issuer {
		return Claims{}, ErrInvalidToken
	}
	if time.Now().After(claims.ExpiresAt) {
		return Claims{}, ErrExpiredToken
	}
	return claims, nil
}

func (m *TokenManager) signJWT(claims Claims) (string, error) {
	headerBytes, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}

	payloadBytes, err := json.Marshal(map[string]interface{}{
		"sub": strconv.FormatInt(claims.UserID, 10),
		"jti": claims.TokenID,
		"typ": claims.TokenType,
		"iss": claims.Issuer,
		"iat": claims.IssuedAt.Unix(),
		"exp": claims.ExpiresAt.Unix(),
	})
	if err != nil {
		return "", err
	}

	header := base64.RawURLEncoding.EncodeToString(headerBytes)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signingInput := header + "." + payload
	return signingInput + "." + m.sign(signingInput), nil
}

func (m *TokenManager) sign(signingInput string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(signingInput))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func claimsFromPayload(payload map[string]interface{}) (Claims, error) {
	sub, ok := payload["sub"].(string)
	if !ok {
		return Claims{}, ErrInvalidToken
	}
	userID, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	tokenID, _ := payload["jti"].(string)
	tokenType, _ := payload["typ"].(string)
	issuer, _ := payload["iss"].(string)
	issuedAt, err := numberTime(payload["iat"])
	if err != nil {
		return Claims{}, err
	}
	expiresAt, err := numberTime(payload["exp"])
	if err != nil {
		return Claims{}, err
	}

	return Claims{
		UserID:    userID,
		TokenID:   tokenID,
		TokenType: tokenType,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
		Issuer:    issuer,
	}, nil
}

func numberTime(value interface{}) (time.Time, error) {
	number, ok := value.(float64)
	if !ok {
		return time.Time{}, ErrInvalidToken
	}
	return time.Unix(int64(number), 0), nil
}

func randomTokenID() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}

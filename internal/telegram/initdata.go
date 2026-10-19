package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var ErrInitData = errors.New("invalid Telegram initData")

type User struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

// Validate checks the raw query string, never initDataUnsafe. The HMAC includes
// every received field except hash, including signature when Telegram sends it.
// https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
func Validate(raw, botToken string, now time.Time, maxAge time.Duration) (User, error) {
	var user User
	if botToken == "" || raw == "" || len(raw) > 16384 || maxAge <= 0 {
		return user, ErrInitData
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return user, ErrInitData
	}
	fields := make([]string, 0, len(values))
	for key, entries := range values {
		if len(entries) != 1 || strings.ContainsAny(key, "\r\n=") {
			return user, ErrInitData
		}
		if key != "hash" {
			fields = append(fields, key+"="+entries[0])
		}
	}
	sort.Strings(fields)
	received, err := hex.DecodeString(values.Get("hash"))
	if err != nil || len(received) != sha256.Size {
		return user, ErrInitData
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	check := hmac.New(sha256.New, secret.Sum(nil))
	check.Write([]byte(strings.Join(fields, "\n")))
	if !hmac.Equal(received, check.Sum(nil)) {
		return user, ErrInitData
	}
	seconds, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	authTime := time.Unix(seconds, 0)
	if err != nil || seconds <= 0 || authTime.After(now.Add(30*time.Second)) || now.Sub(authTime) > maxAge {
		return user, ErrInitData
	}
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil || user.ID <= 0 || strings.TrimSpace(user.FirstName) == "" {
		return User{}, ErrInitData
	}
	return user, nil
}

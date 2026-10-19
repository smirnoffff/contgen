package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const sessionCookie = "contgen_session"
const sessionLifetime = 24 * time.Hour

type sessions struct {
	key    []byte
	secure bool
}

func (s sessions) sign(payload string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s sessions) issue(w http.ResponseWriter, userID int64, now time.Time) {
	expires := now.Add(sessionLifetime)
	payload := strconv.FormatInt(userID, 10) + "." + strconv.FormatInt(expires.Unix(), 10)
	sameSite := http.SameSiteLaxMode
	if s.secure {
		// Telegram Web can embed Mini Apps in an iframe.
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: payload + "." + s.sign(payload), Path: "/",
		HttpOnly: true, Secure: s.secure, SameSite: sameSite,
		MaxAge: int(sessionLifetime.Seconds()), Expires: expires,
	})
}

func (s sessions) userID(r *http.Request, now time.Time) (int64, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return 0, false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 3 {
		return 0, false
	}
	received, err := base64.RawURLEncoding.DecodeString(parts[2])
	expected, _ := base64.RawURLEncoding.DecodeString(s.sign(parts[0] + "." + parts[1]))
	if err != nil || !hmac.Equal(received, expected) {
		return 0, false
	}
	uid, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || uid <= 0 {
		return 0, false
	}
	expires, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || !now.Before(time.Unix(expires, 0)) {
		return 0, false
	}
	return uid, true
}

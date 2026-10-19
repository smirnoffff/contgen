package telegram

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

// Golden vector calculated independently with Python hashlib/hmac. Includes
// signature to catch accidentally treating bot-token HMAC like Ed25519 checks.
const fixture = "auth_date=1720000000&query_id=AAE-test&signature=fixture-signature&user=%7B%22id%22%3A123456789%2C%22first_name%22%3A%22%D0%90%D0%BD%D0%BD%D0%B0%22%2C%22username%22%3A%22anna%22%2C%22language_code%22%3A%22ru%22%7D&hash=5786bbcbceed5253ec00f7404585a92e94c24458752f476deb66536163a8e3c5"

func TestValidateTelegramData(t *testing.T) {
	now := time.Unix(1720000000, 0)
	user, err := Validate(fixture, "123456:fixture_token", now, time.Hour)
	if err != nil || user.ID != 123456789 || user.FirstName != "Анна" || user.Username != "anna" || user.LanguageCode != "ru" {
		t.Fatalf("valid vector: %+v, %v", user, err)
	}
	values, _ := url.ParseQuery(fixture)
	values.Set("user", `{"id":999,"first_name":"attacker"}`)
	cases := []struct {
		name, raw, token string
		now              time.Time
	}{
		{"tampered user", values.Encode(), "123456:fixture_token", now},
		{"wrong bot", fixture, "different token", now},
		{"expired", fixture, "123456:fixture_token", now.Add(time.Hour + time.Second)},
		{"future", fixture, "123456:fixture_token", now.Add(-time.Minute)},
		{"duplicate user", fixture + "&user=duplicate", "123456:fixture_token", now},
		{"duplicate hash", fixture + "&hash=duplicate", "123456:fixture_token", now},
		{"broken URL", fixture + "&bad=%ZZ", "123456:fixture_token", now},
		{"missing hash", strings.Split(fixture, "&hash=")[0], "123456:fixture_token", now},
		{"empty token", fixture, "", now},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Validate(tc.raw, tc.token, tc.now, time.Hour); err == nil {
				t.Fatal("invalid initData accepted")
			}
		})
	}
}

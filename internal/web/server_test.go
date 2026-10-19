package web

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"contgen/internal/storage/dbgen"
	"github.com/jackc/pgx/v5"
)

type fakeStore struct {
	writes  int
	profile dbgen.FindOrCreateUserParams
}

func (f *fakeStore) FindOrCreateUser(_ context.Context, p dbgen.FindOrCreateUserParams) (dbgen.User, error) {
	f.writes++
	f.profile = p
	return dbgen.User{ID: 1, TgID: p.TgID, FirstName: p.FirstName, Language: p.Language, Username: p.Username}, nil
}
func (f *fakeStore) GetUserByID(_ context.Context, id int64) (dbgen.User, error) {
	if id != 1 {
		return dbgen.User{}, pgx.ErrNoRows
	}
	return dbgen.User{ID: 1, FirstName: "Анна"}, nil
}
func (f *fakeStore) GetAvailableDatasets(_ context.Context, id *int64) ([]dbgen.Dataset, error) {
	return []dbgen.Dataset{{ID: 1, Title: "Языки"}, {ID: 2, UserID: id, Title: "<script>alert(1)</script>"}}, nil
}
func (f *fakeStore) GetDatasetForUser(_ context.Context, p dbgen.GetDatasetForUserParams) (dbgen.Dataset, error) {
	if p.ID == 1 {
		return dbgen.Dataset{ID: 1, Title: "Языки"}, nil
	}
	if p.ID == 2 && p.ViewerID == 1 {
		return dbgen.Dataset{ID: 2, UserID: &p.ViewerID, Title: "<script>alert(1)</script>"}, nil
	}
	return dbgen.Dataset{}, pgx.ErrNoRows
}
func (f *fakeStore) GetDataPointsByDatasetID(_ context.Context, id int64) ([]dbgen.DataPoint, error) {
	return []dbgen.DataPoint{{EntityName: "Go", Value: 10, RecordedAt: 2020}, {EntityName: "Go", Value: 20, RecordedAt: 2024}, {EntityName: "Python", Value: 40, RecordedAt: 2024}}, nil
}

const testKey = "a-session-secret-at-least-32-bytes-long"
const testBot = "123456:test-token"

func testHandler(t *testing.T, dev bool) (http.Handler, *fakeStore) {
	t.Helper()
	store := &fakeStore{}
	h, err := New(store, Options{PublicURL: "https://studio.example", BotToken: testBot, SessionSecret: testKey})
	if dev {
		h, err = New(store, Options{PublicURL: "http://localhost:8080", DevLogin: true, SessionSecret: testKey})
	}
	if err != nil {
		t.Fatal(err)
	}
	return h, store
}
func request(h http.Handler, method, path, body string, cookie *http.Cookie, hx bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "https://studio.example"+path, strings.NewReader(body))
	if cookie != nil {
		r.AddCookie(cookie)
	}
	if hx {
		r.Header.Set("HX-Request", "true")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func signedData(t *testing.T, user string, date time.Time) string {
	t.Helper()
	fields := url.Values{"auth_date": {strconv.FormatInt(date.Unix(), 10)}, "query_id": {"test"}, "user": {user}}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+"="+fields.Get(k))
	}
	key := hmac.New(sha256.New, []byte("WebAppData"))
	key.Write([]byte(testBot))
	mac := hmac.New(sha256.New, key.Sum(nil))
	mac.Write([]byte(strings.Join(lines, "\n")))
	fields.Set("hash", hex.EncodeToString(mac.Sum(nil)))
	return fields.Encode()
}
func loginBody(data string) string {
	b, _ := json.Marshal(map[string]string{"init_data": data})
	return string(b)
}

func TestTelegramLoginAndStudio(t *testing.T) {
	h, store := testHandler(t, false)
	w := request(h, "GET", "/", "", nil, false)
	if w.Code != 200 || strings.Contains(w.Body.String(), "dev-login") {
		t.Fatalf("public shell: %d", w.Code)
	}
	if w := request(h, "GET", "/studio", "", nil, true); w.Code != 401 {
		t.Fatalf("unauthorized studio: %d", w.Code)
	}
	if w := request(h, "GET", "/studio?dataset_id=1", "", nil, false); w.Code != 200 || !strings.Contains(w.Body.String(), "login-title") {
		t.Fatalf("bookmarked studio must allow signing in: %d", w.Code)
	}
	data := signedData(t, `{"id":123456789,"first_name":"Анна","username":"anna","language_code":"en"}`, time.Now())
	w = request(h, "POST", "/auth/telegram", loginBody(data), nil, false)
	if w.Code != 204 || store.writes != 1 || store.profile.TgID != 123456789 || store.profile.Language != dbgen.AppLanguageEN {
		t.Fatalf("login: %d %+v", w.Code, store.profile)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteNoneMode {
		t.Fatalf("session cookie: %+v", cookies)
	}
	cookie := cookies[0]
	w = request(h, "GET", "/studio?dataset_id=2&theme=VINTAGE_PRINT", "", cookie, true)
	html := w.Body.String()
	if w.Code != 200 || strings.Contains(html, "<!doctype") || strings.Contains(html, "<script>alert") || !strings.Contains(html, "&lt;script&gt;") || !strings.Contains(html, `data-theme="VINTAGE_PRINT"`) || !strings.Contains(html, "Python") {
		t.Fatalf("fragment: %d %s", w.Code, html)
	}
	w = request(h, "GET", "/studio?dataset_id=1&theme=CORPORATE_LIGHT", "", cookie, false)
	if w.Code != 200 || !strings.Contains(strings.ToLower(w.Body.String()), "<!doctype html>") {
		t.Fatalf("full page: %d", w.Code)
	}
	for _, tc := range []struct {
		path   string
		status int
	}{{"/studio?dataset_id=3", 404}, {"/studio?dataset_id=-1", 400}, {"/studio?dataset_id=bad", 400}, {"/studio?theme=unknown", 400}} {
		if w := request(h, "GET", tc.path, "", cookie, true); w.Code != tc.status {
			t.Fatalf("%s: %d", tc.path, w.Code)
		}
	}
	cookie.Value = "999" + cookie.Value[1:]
	if w := request(h, "GET", "/studio", "", cookie, true); w.Code != 401 {
		t.Fatalf("forged session: %d", w.Code)
	}
}

func TestBadLoginNeverWrites(t *testing.T) {
	h, store := testHandler(t, false)
	for _, body := range []string{`{"init_data":"forged"}`, loginBody(signedData(t, `{"id":1,"first_name":"Test"}`, time.Now().Add(-2*time.Hour))), loginBody(signedData(t, `{"id":0,"first_name":"Test"}`, time.Now())), `{"init_data":"x","user_id":1}`, `{} {}`, `{"init_data":"` + strings.Repeat("x", 25000) + `"}`} {
		w := request(h, "POST", "/auth/telegram", body, nil, false)
		if w.Code != 401 && w.Code != 400 {
			t.Fatalf("invalid login: %d", w.Code)
		}
	}
	if store.writes != 0 {
		t.Fatal("invalid credentials changed DB")
	}
	if w := request(h, "POST", "/auth/dev", "{}", nil, false); w.Code != 404 {
		t.Fatalf("production dev route: %d", w.Code)
	}
	r := httptest.NewRequest("POST", "https://studio.example/auth/telegram", strings.NewReader("{}"))
	r.Header.Set("Origin", "https://evil.example")
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatalf("cross origin login: %d", w.Code)
	}
}

func TestSessionExpiryAndDevConfig(t *testing.T) {
	s := sessions{key: []byte(testKey)}
	now := time.Unix(1720000000, 0)
	w := httptest.NewRecorder()
	s.issue(w, 1, now)
	r := httptest.NewRequest("GET", "http://localhost/", nil)
	r.AddCookie(w.Result().Cookies()[0])
	if id, ok := s.userID(r, now.Add(time.Hour)); !ok || id != 1 {
		t.Fatal("valid session rejected")
	}
	if _, ok := s.userID(r, now.Add(sessionLifetime)); ok {
		t.Fatal("expired session accepted")
	}
	for _, opts := range []Options{{PublicURL: "https://example.com", DevLogin: true}, {PublicURL: "http://example.com", BotToken: testBot, SessionSecret: testKey}, {PublicURL: "https://example.com", BotToken: testBot, SessionSecret: "short"}, {PublicURL: "https://example.com", SessionSecret: testKey}, {PublicURL: "https://example.com", BotToken: testBot, SessionSecret: testKey, AdsEnabled: true}} {
		if _, err := New(&fakeStore{}, opts); err == nil {
			t.Fatalf("bad configuration accepted: %+v", opts)
		}
	}
	h, store := testHandler(t, true)
	w = request(h, "POST", "/auth/dev", "{}", nil, false)
	if w.Code != 204 || store.profile.TgID != -1 || w.Result().Cookies()[0].Secure {
		t.Fatalf("local login: %d", w.Code)
	}
	if w := request(h, "GET", "/static/app.css", "", nil, false); w.Code != 200 || !strings.Contains(w.Header().Get("Content-Type"), "text/css") {
		t.Fatalf("assets: %d", w.Code)
	}
}

func TestLanguageSwitchUsesSafeRedirect(t *testing.T) {
	h, _ := testHandler(t, false)
	r := httptest.NewRequest(http.MethodGet, "https://studio.example/language?lang=en", nil)
	r.Header.Set("Referer", "https://studio.example/studio?dataset_id=1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/studio?dataset_id=1" {
		t.Fatalf("language redirect: %d %q", w.Code, w.Header().Get("Location"))
	}
	cookie := w.Result().Cookies()
	if len(cookie) != 1 || cookie[0].Name != "contgen_lang" || cookie[0].Value != "en" || !cookie[0].Secure {
		t.Fatalf("language cookie: %+v", cookie)
	}
	bad := httptest.NewRequest(http.MethodGet, "https://studio.example/language?lang=ru", nil)
	bad.Header.Set("Referer", "https://evil.example/phish")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, bad)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/" {
		t.Fatalf("unsafe language redirect: %d %q", w.Code, w.Header().Get("Location"))
	}
	for _, tc := range []struct {
		query string
		want  string
	}{
		{query: "uk-UA", want: "uk"},
		{query: "not-a-locale", want: "ru"},
	} {
		r := httptest.NewRequest(http.MethodGet, "https://studio.example/language?lang="+url.QueryEscape(tc.query), nil)
		r.Header.Set("Referer", "https://studio.example/studio")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		cookies := w.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Value != tc.want || w.Header().Get("Location") != "/studio" {
			t.Fatalf("locale %q: cookie=%+v location=%q", tc.query, cookies, w.Header().Get("Location"))
		}
	}
}

func TestLastFramePreview(t *testing.T) {
	s := &server{store: &fakeStore{}}
	r := httptest.NewRequest("GET", "https://studio.example/studio", nil)
	d, status, err := s.studioData(r, dbgen.User{ID: 1})
	if err != nil || status != 200 || d.FirstYear != 2020 || d.LastYear != 2024 || d.PointCount != 3 || d.EntityCount != 2 || len(d.Bars) != 2 || d.Bars[0].Name != "Python" || d.Bars[0].Width != "100.00%" || d.Bars[1].Width != "50.00%" {
		t.Fatalf("preview: %+v, %v", d, err)
	}
	if fmt.Sprint(d.Theme) != "CYBER_HUD" {
		t.Fatal("default theme changed")
	}
}

func TestDevAdsEndpointIsDisabledOutsideLocalMode(t *testing.T) {
	h, _ := testHandler(t, false)
	w := request(h, http.MethodPost, "/cart/ads/complete", "order_id=1", nil, false)
	if w.Code != http.StatusNotFound {
		t.Fatalf("production accepted dev ads endpoint: %d", w.Code)
	}
}

func TestAdsGramConfigurationAndCSP(t *testing.T) {
	h, err := New(&fakeStore{}, Options{
		PublicURL:      "https://studio.example",
		BotToken:       testBot,
		SessionSecret:  testKey,
		AdsEnabled:     true,
		AdsGramBlockID: "123456",
		AdsGramDebug:   true,
	})
	if err != nil {
		t.Fatalf("valid AdsGram configuration rejected: %v", err)
	}
	w := request(h, http.MethodGet, "/", "", nil, false)
	csp := w.Header().Get("Content-Security-Policy")
	for _, origin := range []string{"https://sad.adsgram.ai", "https://image.adsgram.ai", "https://images.adsgram.ai", "media-src 'self' blob: https:", "connect-src 'self' https://api.adsgram.ai https://tma.adsgram.ai", "frame-src https://*.adsgram.ai", "worker-src 'self' blob:"} {
		if !strings.Contains(csp, origin) {
			t.Fatalf("CSP missing %q: %s", origin, csp)
		}
	}
	if !strings.Contains(w.Body.String(), `data-adsgram-debug="true"`) {
		t.Fatal("AdsGram debug flag was not passed to the page")
	}
	if _, err := New(&fakeStore{}, Options{PublicURL: "https://studio.example", BotToken: testBot, SessionSecret: testKey, AdsEnabled: true, AdsGramBlockID: "abc"}); err == nil {
		t.Fatal("non-numeric AdsGram block ID accepted")
	}
	h, err = New(&fakeStore{}, Options{PublicURL: "https://studio.example", BotToken: testBot, SessionSecret: testKey, AdsGramBlockID: "123456"})
	if err != nil {
		t.Fatalf("AdsGram-disabled configuration rejected: %v", err)
	}
	w = request(h, http.MethodGet, "/", "", nil, false)
	if strings.Contains(w.Body.String(), "data-adsgram-block-id=\"123456\"") || strings.Contains(w.Body.String(), "sad.adsgram.ai/js/sad.min.js") {
		t.Fatal("AdsGram block leaked into a disabled page")
	}
}

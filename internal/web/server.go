package web

import (
	"bytes"
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"contgen/internal/storage"
	"contgen/internal/storage/dbgen"
	"contgen/internal/telegram"
	"contgen/internal/web/views"

	"github.com/a-h/templ"
	"github.com/jackc/pgx/v5"
)

//go:embed static/*
var assets embed.FS

type Store interface {
	FindOrCreateUser(context.Context, dbgen.FindOrCreateUserParams) (dbgen.User, error)
	GetUserByID(context.Context, int64) (dbgen.User, error)
	GetAvailableDatasets(context.Context, *int64) ([]dbgen.Dataset, error)
	GetDatasetForUser(context.Context, dbgen.GetDatasetForUserParams) (dbgen.Dataset, error)
	GetDataPointsByDatasetID(context.Context, int64) ([]dbgen.DataPoint, error)
}

type Options struct {
	BotToken         string
	SessionSecret    string
	PublicURL        string
	DevLogin         bool
	DataDir          string
	AdminTelegramIDs []int64
	Telegram         *telegram.Client
	AdsEnabled       bool
	AdsGramBlockID   string
	AdsGramDebug     bool
}

type adsSession struct {
	UserID         int64
	OrderID        int64
	PaymentID      int64
	PaymentPayload string
	Total          int
	Completed      int
	Finalizing     bool
	Finalized      bool
	ExpiresAt      time.Time
}

type server struct {
	store    Store
	opts     Options
	sessions sessions
	db       *storage.Storage
	adsMu    sync.Mutex
	ads      map[string]*adsSession
}

func IsLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func New(store Store, opts Options) (http.Handler, error) {
	u, err := url.Parse(opts.PublicURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("APP_URL must be an HTTP(S) origin")
	}
	if u.Scheme != "https" && !IsLoopback(u.Hostname()) {
		return nil, errors.New("APP_URL must use HTTPS outside localhost")
	}
	if opts.DevLogin && !IsLoopback(u.Hostname()) {
		return nil, errors.New("DEV_LOGIN is only available on localhost")
	}
	if !opts.DevLogin && opts.BotToken == "" {
		return nil, errors.New("TELEGRAM_BOT_TOKEN is required")
	}
	if opts.AdsEnabled && strings.TrimSpace(opts.AdsGramBlockID) == "" {
		return nil, errors.New("ADS_ENABLED requires ADSGRAM_BLOCK_ID")
	}
	if opts.AdsEnabled && !numericBlockID(opts.AdsGramBlockID) {
		return nil, errors.New("ADSGRAM_BLOCK_ID must contain only digits")
	}
	key := []byte(opts.SessionSecret)
	if opts.DevLogin && len(key) == 0 {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
	}
	if len(key) < 32 {
		return nil, errors.New("SESSION_SECRET must contain at least 32 bytes")
	}
	s := &server{store: store, opts: opts, sessions: sessions{key: key, secure: u.Scheme == "https"}, ads: make(map[string]*adsSession)}
	s.db, _ = store.(*storage.Storage)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.home)
	mux.HandleFunc("GET /studio", s.studio)
	mux.HandleFunc("POST /auth/telegram", s.login)
	s.registerWorkflow(mux)
	if opts.DevLogin {
		mux.HandleFunc("POST /auth/dev", s.devLogin)
	}
	static, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	protection := http.NewCrossOriginProtection()
	if err := protection.AddTrustedOrigin(u.Scheme + "://" + u.Host); err != nil {
		return nil, err
	}
	return headers(protection.Handler(mux)), nil
}

func numericBlockID(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	for _, r := range raw {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func newAdsToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		// AdsGram renders media in the Mini App and may use a blob worker or
		// an adsgram-owned frame depending on the campaign format. Keep the
		// allowlist explicit while retaining HTTPS media support for campaign
		// CDNs returned by the provider.
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' https://telegram.org https://sad.adsgram.ai; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://image.adsgram.ai https://images.adsgram.ai; media-src 'self' blob: https:; connect-src 'self' https://api.adsgram.ai https://tma.adsgram.ai; frame-src https://*.adsgram.ai; worker-src 'self' blob:; font-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'self' https://web.telegram.org https://*.telegram.org")
		if !strings.HasPrefix(r.URL.Path, "/static/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

// AdsGram is a production-only integration. Do not expose its block ID or
// debug flag to local/demo pages, even if an operator accidentally leaves the
// values in a shared .env file.
func (s *server) adsGramBlockID() string {
	if s.opts.DevLogin || !s.opts.AdsEnabled {
		return ""
	}
	return s.opts.AdsGramBlockID
}

func (s *server) adsGramDebug() bool {
	return !s.opts.DevLogin && s.opts.AdsEnabled && s.opts.AdsGramDebug
}

func render(w http.ResponseWriter, r *http.Request, component templ.Component) {
	var buf bytes.Buffer
	if err := component.Render(r.Context(), &buf); err != nil {
		slog.Error("render HTML", "err", err)
		http.Error(w, "Не удалось открыть студию", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func (s *server) currentUser(r *http.Request) (dbgen.User, error) {
	id, ok := s.sessions.userID(r, time.Now())
	if !ok {
		return dbgen.User{}, pgx.ErrNoRows
	}
	return s.store.GetUserByID(r.Context(), id)
}

func (s *server) home(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if errors.Is(err, pgx.ErrNoRows) {
		render(w, r, views.Page(s.opts.DevLogin, false, s.languageForRequest(r, "ru"), s.adsGramBlockID(), s.adsGramDebug(), views.Welcome(s.opts.DevLogin, s.languageForRequest(r, "ru"))))
		return
	}
	if err != nil {
		s.unavailable(w, err)
		return
	}
	data, status, err := s.studioData(r, user)
	if err != nil {
		s.respondError(w, status, err)
		return
	}
	render(w, r, views.Page(s.opts.DevLogin, true, data.Language, s.adsGramBlockID(), s.adsGramDebug(), views.Studio(data)))
}

func (s *server) studio(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if errors.Is(err, pgx.ErrNoRows) {
		if r.Header.Get("HX-Request") == "true" {
			http.Error(w, "Нужен вход через Telegram", http.StatusUnauthorized)
		} else {
			render(w, r, views.Page(s.opts.DevLogin, false, s.languageForRequest(r, "ru"), s.adsGramBlockID(), s.adsGramDebug(), views.Welcome(s.opts.DevLogin, s.languageForRequest(r, "ru"))))
		}
		return
	}
	if err != nil {
		s.unavailable(w, err)
		return
	}
	data, status, err := s.studioData(r, user)
	if err != nil {
		s.respondError(w, status, err)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		render(w, r, views.Studio(data))
	} else {
		render(w, r, views.Page(s.opts.DevLogin, true, data.Language, s.adsGramBlockID(), s.adsGramDebug(), views.Studio(data)))
	}
}

func (s *server) unavailable(w http.ResponseWriter, err error) {
	slog.Error("Mini App storage", "err", err)
	http.Error(w, "Студия временно недоступна. Попробуй ещё раз.", http.StatusServiceUnavailable)
}

func (s *server) respondError(w http.ResponseWriter, status int, err error) {
	if status == http.StatusServiceUnavailable {
		s.unavailable(w, err)
		return
	}
	http.Error(w, err.Error(), status)
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	if s.opts.BotToken == "" {
		http.Error(w, "Вход через Telegram не настроен", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		InitData string `json:"init_data"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 20*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		http.Error(w, "Некорректные данные входа", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "Некорректные данные входа", http.StatusBadRequest)
		return
	}
	user, err := telegram.Validate(body.InitData, s.opts.BotToken, time.Now(), time.Hour)
	if err != nil {
		http.Error(w, "Открой Mini App заново через Telegram", http.StatusUnauthorized)
		return
	}
	displayLanguage := views.TelegramLanguage(user.LanguageCode)
	// Keep the database enum stable (it is deliberately only EN/RU), while
	// retaining a supported Telegram locale in the presentation cookie.
	lang := dbgen.AppLanguageRU
	if displayLanguage == string(views.LanguageEN) {
		lang = dbgen.AppLanguageEN
	}
	var username *string
	if user.Username != "" {
		username = &user.Username
	}
	stored, err := s.store.FindOrCreateUser(r.Context(), dbgen.FindOrCreateUserParams{TgID: user.ID, Username: username, FirstName: user.FirstName, Language: lang})
	if err != nil {
		s.unavailable(w, err)
		return
	}
	if s.db != nil {
		for _, id := range s.opts.AdminTelegramIDs {
			if id == user.ID {
				if _, err := s.db.GrantAdminByTelegramID(r.Context(), id); err != nil {
					s.unavailable(w, err)
					return
				}
				break
			}
		}
	}
	s.sessions.issue(w, stored.ID, time.Now())
	if displayLanguage != string(views.LanguageRU) && displayLanguage != string(views.LanguageEN) {
		if _, cookieErr := r.Cookie("contgen_lang"); cookieErr != nil {
			http.SetCookie(w, &http.Cookie{Name: "contgen_lang", Value: displayLanguage, Path: "/", MaxAge: 180 * 24 * 60 * 60, Secure: s.sessions.secure, HttpOnly: false, SameSite: http.SameSiteLaxMode})
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) devLogin(w http.ResponseWriter, r *http.Request) {
	// This route is registered only for localhost development. It never accepts
	// client-supplied identities and uses an ID Telegram cannot issue.
	user, err := s.store.FindOrCreateUser(r.Context(), dbgen.FindOrCreateUserParams{TgID: -1, FirstName: "Локальная студия", Language: dbgen.AppLanguageRU})
	if err != nil {
		s.unavailable(w, err)
		return
	}
	if s.db != nil {
		if _, err := s.db.GrantAdminByTelegramID(r.Context(), -1); err != nil {
			s.unavailable(w, err)
			return
		}
	}
	s.sessions.issue(w, user.ID, time.Now())
	w.WriteHeader(http.StatusNoContent)
}

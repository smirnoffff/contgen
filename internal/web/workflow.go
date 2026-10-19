package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"contgen/internal/storage/dbgen"
	"contgen/internal/web/views"
	"github.com/a-h/templ"
	"github.com/jackc/pgx/v5"
)

func (s *server) registerWorkflow(m *http.ServeMux) {
	m.HandleFunc("GET /language", s.setLanguage)
	m.HandleFunc("GET /library", s.library)
	m.HandleFunc("GET /datasets/example.csv", s.exampleCSV)
	m.HandleFunc("POST /datasets/upload", s.uploadDataset)
	m.HandleFunc("POST /datasets/{id}/delete", s.deleteDataset)
	m.HandleFunc("POST /music/upload", s.uploadMusic)
	m.HandleFunc("GET /music/{id}/audio", s.audio)
	m.HandleFunc("POST /music/{id}/delete", s.archiveMusic)
	m.HandleFunc("GET /cart", s.cart)
	m.HandleFunc("POST /cart/items", s.addCartItem)
	m.HandleFunc("POST /cart/items/{id}/delete", s.removeCartItem)
	m.HandleFunc("POST /cart/checkout", s.checkout)
	m.HandleFunc("POST /cart/ads/start", s.startAds)
	m.HandleFunc("GET /cart/ads/status", s.adsStatus)
	m.HandleFunc("POST /cart/ads/complete", s.completeAds)
	m.HandleFunc("POST /cart/ads/cancel", s.cancelAds)
	m.HandleFunc("GET /status", s.orders)
	// Keep the old URL as a compatibility redirect for bookmarks and old clients.
	m.HandleFunc("GET /orders", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/status", http.StatusSeeOther) })
	m.HandleFunc("GET /jobs/{id}", s.jobCard)
	m.HandleFunc("GET /jobs/{id}/video", s.video)
	m.HandleFunc("POST /jobs/{id}/retry", s.retryJob)
	m.HandleFunc("POST /jobs/{id}/telegram", s.sendVideo)
	m.HandleFunc("GET /admin", s.admin)
	m.HandleFunc("GET /admin/jobs/{id}", s.adminJobCard)
	m.HandleFunc("GET /admin/jobs/{id}/video", s.adminVideo)
	m.HandleFunc("POST /admin/jobs/{id}/retry", s.adminRetryJob)
	m.HandleFunc("POST /admin/datasets/upload", s.uploadDataset)
	m.HandleFunc("POST /admin/music/upload", s.uploadMusic)
	m.HandleFunc("POST /admin/music/{id}/toggle", s.adminMusicToggle)
	m.HandleFunc("POST /admin/datasets/{id}/publish", s.adminDatasetPublish)
	m.HandleFunc("POST /admin/music/{id}/publish", s.adminMusicPublish)
	m.HandleFunc("POST /admin/users/{id}/role", s.adminUserRole)
	m.HandleFunc("POST /admin/promos", s.adminCreatePromo)
	m.HandleFunc("POST /admin/promos/{id}/toggle", s.adminPromoToggle)
}

func (s *server) requireUser(w http.ResponseWriter, r *http.Request, admin bool) (dbgen.User, bool) {
	u, err := s.currentUser(r)
	if errors.Is(err, pgx.ErrNoRows) {
		if r.Method == http.MethodGet && r.Header.Get("HX-Request") != "true" {
			render(w, r, views.Page(s.opts.DevLogin, false, s.languageForRequest(r, "ru"), s.adsGramBlockID(), s.adsGramDebug(), views.Welcome(s.opts.DevLogin, s.languageForRequest(r, "ru"))))
		} else {
			http.Error(w, "Нужен вход через Telegram", http.StatusUnauthorized)
		}
		return u, false
	}
	if err != nil {
		s.unavailable(w, err)
		return u, false
	}
	if admin && u.Role != dbgen.UserRoleADMIN {
		http.Error(w, "Этот раздел доступен администратору", http.StatusForbidden)
		return u, false
	}
	if s.db == nil {
		http.Error(w, "Сервис временно недоступен", http.StatusServiceUnavailable)
		return u, false
	}
	return u, true
}

func (s *server) pageData(r *http.Request, u dbgen.User) (views.PageData, error) {
	language := s.languageForRequest(r, string(u.Language))
	d := views.PageData{UserName: u.FirstName, Language: language, IsAdmin: u.Role == dbgen.UserRoleADMIN,
		AdsEnabled: s.opts.AdsEnabled || s.opts.DevLogin,
		Notice:     shortMessage(r.URL.Query().Get("notice")), Error: shortMessage(r.URL.Query().Get("error"))}
	if s.db != nil {
		items, err := s.db.ListCartItems(r.Context(), u.ID)
		if err != nil {
			return d, err
		}
		d.CartCount = len(items)
	}
	return d, nil
}

func (s *server) languageForRequest(r *http.Request, fallback string) string {
	if cookie, err := r.Cookie("contgen_lang"); err == nil {
		return views.NormalizeLanguage(cookie.Value)
	}
	return views.NormalizeLanguage(fallback)
}

// setLanguage stores the display preference in a signed-free, low-risk cookie:
// it only changes presentation and never controls authorization or payments.
// The next location is taken from the same-origin Referer/HTMX URL to keep the
// language toggle usable from every Mini App screen.
func (s *server) setLanguage(w http.ResponseWriter, r *http.Request) {
	language := views.NormalizeLanguage(r.URL.Query().Get("lang"))
	secure := s.sessions.secure
	http.SetCookie(w, &http.Cookie{Name: "contgen_lang", Value: language, Path: "/", MaxAge: 180 * 24 * 60 * 60, Secure: secure, HttpOnly: false, SameSite: http.SameSiteLaxMode})
	next := r.Header.Get("HX-Current-URL")
	if next == "" {
		next = r.Referer()
	}
	if nextURL, err := url.Parse(next); err == nil && nextURL.Path != "" && nextURL.Path[0] == '/' && !strings.HasPrefix(nextURL.Path, "//") && (nextURL.Host == "" || nextURL.Host == r.Host) {
		target := nextURL.Path
		if nextURL.RawQuery != "" {
			target += "?" + nextURL.RawQuery
		}
		if nextURL.Fragment != "" {
			target += "#" + nextURL.Fragment
		}
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", target)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func shortMessage(s string) string { return string([]rune(s)[:min(len([]rune(s)), 300)]) }

func (s *server) page(w http.ResponseWriter, r *http.Request, c templ.Component) {
	if r.Header.Get("HX-Request") == "true" {
		render(w, r, c)
	} else {
		render(w, r, views.Page(s.opts.DevLogin, true, s.languageForRequest(r, "ru"), s.adsGramBlockID(), s.adsGramDebug(), c))
	}
}

func redirectMessage(w http.ResponseWriter, r *http.Request, path, notice, problem string) {
	u, _ := url.Parse(path)
	q := u.Query()
	if notice != "" {
		q.Set("notice", notice)
	}
	if problem != "" {
		q.Set("error", problem)
	}
	u.RawQuery = q.Encode()
	if r.Header.Get("HX-Request") == "true" {
		location, _ := json.Marshal(map[string]string{"path": u.String(), "target": "#studio", "swap": "outerHTML"})
		w.Header().Set("HX-Location", string(location))
		w.WriteHeader(http.StatusNoContent)
	} else {
		http.Redirect(w, r, u.String(), http.StatusSeeOther)
	}
}

func idParam(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("Некорректный идентификатор")
	}
	return id, nil
}

func parseForm(w http.ResponseWriter, r *http.Request) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Некорректная форма", http.StatusBadRequest)
		return false
	}
	return true
}

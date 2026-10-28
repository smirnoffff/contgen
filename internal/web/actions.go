package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"contgen/internal/service"
	"contgen/internal/storage"
	"contgen/internal/storage/dbgen"
	"contgen/internal/web/views"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *server) cart(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	language := s.languageForRequest(r, string(u.Language))
	data, err := s.cartData(r.Context(), u, language)
	if err != nil {
		s.unavailable(w, err)
		return
	}
	data.PageData, err = s.pageData(r, u)
	if err != nil {
		s.unavailable(w, err)
		return
	}
	s.page(w, r, views.Cart(data))
}

func (s *server) cartData(ctx context.Context, u dbgen.User, language string) (views.CartData, error) {
	items, err := s.db.ListCartItems(ctx, u.ID)
	if err != nil {
		return views.CartData{}, err
	}
	data := views.CartData{CheckoutEnabled: len(items) > 0, AdVideoCount: len(items), AdTotal: len(items) * 2}
	if len(items) > 0 {
		data.OrderID = items[0].RenderJob.OrderID
	}
	for _, item := range items {
		data.Items = append(data.Items, views.CartItem{ID: item.RenderJob.ID, Title: views.DatasetTitle(item.DatasetTitle, language), Theme: themeLabel(item.RenderJob.Theme), Music: musicLabel(item.MusicTitle, language), Price: fmt.Sprintf("%d ⭐", storage.RenderPrice)})
	}
	data.Subtotal = fmt.Sprintf("%d ⭐", len(items)*storage.RenderPrice)
	data.Total = data.Subtotal
	return data, nil
}

func (s *server) addCartItem(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	if !parseForm(w, r) {
		return
	}
	datasetID, err := strconv.ParseInt(r.FormValue("dataset_id"), 10, 64)
	if err != nil {
		redirectMessage(w, r, "/studio", "", "Выбери датасет")
		return
	}
	theme := dbgen.RenderTheme(r.FormValue("theme"))
	if !theme.Valid() {
		redirectMessage(w, r, "/studio", "", "Выбери тему")
		return
	}
	var musicID *int64
	if raw := r.FormValue("music_id"); raw != "" && raw != "0" {
		v, e := strconv.ParseInt(raw, 10, 64)
		if e != nil || v <= 0 {
			redirectMessage(w, r, "/studio", "", "Выбери корректную музыку")
			return
		}
		musicID = &v
	}
	if _, err = s.db.AddCartItem(r.Context(), u.ID, datasetID, musicID, theme); err != nil {
		s.operationError(w, r, "/studio", err)
		return
	}
	redirectMessage(w, r, "/cart", "Добавлено в корзину", "")
}

func (s *server) removeCartItem(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err = s.db.RemoveCartItem(r.Context(), u.ID, id); err != nil {
		s.operationError(w, r, "/cart", err)
		return
	}
	redirectMessage(w, r, "/cart", "Видео убрано из корзины", "")
}

func (s *server) checkout(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	if !parseForm(w, r) {
		return
	}
	orderID, err := strconv.ParseInt(r.FormValue("order_id"), 10, 64)
	if err != nil {
		redirectMessage(w, r, "/cart", "", "Корзина изменилась. Открой её заново.")
		return
	}
	method := dbgen.PaymentMethod(strings.ToUpper(strings.TrimSpace(r.FormValue("payment_method"))))
	if method == "" {
		method = dbgen.PaymentMethodSTARS
	}
	if method != dbgen.PaymentMethodSTARS && method != dbgen.PaymentMethodADS {
		redirectMessage(w, r, "/cart", "", "Выбери способ оплаты")
		return
	}
	if method == dbgen.PaymentMethodSTARS {
		// An abandoned ad attempt must not block the explicit Stars fallback,
		// including after the process has been restarted.
		s.cancelAdsForOrder(r.Context(), u.ID, orderID)
	}
	payment, err := s.db.PreparePaymentWithMethod(r.Context(), u.ID, orderID, method, r.FormValue("code"))
	if err != nil {
		s.operationError(w, r, "/cart", err)
		return
	}
	if method == dbgen.PaymentMethodSTARS && payment.Method == dbgen.PaymentMethodADS {
		_ = s.db.CancelPayment(r.Context(), payment.ID)
		payment, err = s.db.PreparePaymentWithMethod(r.Context(), u.ID, orderID, method, r.FormValue("code"))
		if err != nil {
			s.operationError(w, r, "/cart", err)
			return
		}
	}
	if payment.Method == dbgen.PaymentMethodPROMO {
		if payment.ExternalID == nil || s.db.CompletePayment(r.Context(), *payment.ExternalID, "promo-"+*payment.ExternalID) != nil {
			s.operationError(w, r, "/cart", storage.ErrPaymentInvalid)
			return
		}
		redirectMessage(w, r, "/status", "Заказ оформлен по промокоду 100%. Видео в очереди.", "")
		return
	}
	if payment.Method == dbgen.PaymentMethodADS {
		_ = s.db.CancelPayment(r.Context(), payment.ID)
		redirectMessage(w, r, "/cart", "", "После подключения рекламного просмотра здесь откроются две короткие рекламы.")
		return
	}
	if s.opts.Telegram == nil || u.TgID <= 0 {
		_ = s.db.CancelPayment(r.Context(), payment.ID)
		redirectMessage(w, r, "/cart", "", "Для оплаты Stars открой приложение через Telegram.")
		return
	}
	if payment.ExternalID == nil {
		s.operationError(w, r, "/cart", storage.ErrPaymentInvalid)
		return
	}
	amount, valid := numericStars(payment.Amount)
	if !valid || amount <= 0 {
		s.operationError(w, r, "/cart", storage.ErrPaymentInvalid)
		return
	}
	link, err := s.opts.Telegram.CreateInvoiceLink(r.Context(), "CONTGEN · видео", "Рендер вертикального видео", *payment.ExternalID, amount)
	if err != nil {
		_ = s.db.CancelPayment(r.Context(), payment.ID)
		s.operationError(w, r, "/cart", err)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", link)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, link, http.StatusSeeOther)
}

// completeDevAds exists only for the local UI test mode. It exercises the
// same storage transition as a verified ad provider without trusting a
// browser request in a real Telegram deployment.
func (s *server) completeDevAds(w http.ResponseWriter, r *http.Request) {
	if !s.opts.DevLogin {
		http.NotFound(w, r)
		return
	}
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	if !parseForm(w, r) {
		return
	}
	orderID, err := strconv.ParseInt(r.FormValue("order_id"), 10, 64)
	if err != nil || orderID <= 0 {
		http.Error(w, "Некорректный заказ", http.StatusBadRequest)
		return
	}
	payment, err := s.db.PreparePaymentWithMethod(r.Context(), u.ID, orderID, dbgen.PaymentMethodADS, "")
	if err != nil {
		http.Error(w, "Не удалось подготовить тестовый платёж", http.StatusBadRequest)
		return
	}
	if payment.ExternalID == nil {
		http.Error(w, "Не удалось подготовить тестовый платёж", http.StatusBadRequest)
		return
	}
	if err := s.db.CompletePayment(r.Context(), *payment.ExternalID, "dev-ad-"+*payment.ExternalID); err != nil {
		http.Error(w, "Не удалось завершить тестовый платёж", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// startAds reserves the normal ADS payment and issues a short-lived server
// session. Each cart video requires two client-side AdsGram reward callbacks;
// the session prevents a browser from reusing a payment for another order or
// replaying a completed step.
func (s *server) startAds(w http.ResponseWriter, r *http.Request) {
	if !s.opts.AdsEnabled || s.opts.DevLogin || strings.TrimSpace(s.opts.AdsGramBlockID) == "" {
		http.NotFound(w, r)
		return
	}
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	if u.TgID <= 0 {
		http.Error(w, "Рекламная оплата доступна только внутри Telegram", http.StatusForbidden)
		return
	}
	if !parseForm(w, r) {
		return
	}
	if s.db == nil {
		http.Error(w, "Хранилище оплаты недоступно", http.StatusServiceUnavailable)
		return
	}
	s.cleanupAds(r.Context())
	orderID, err := strconv.ParseInt(r.FormValue("order_id"), 10, 64)
	if err != nil || orderID <= 0 {
		http.Error(w, "Некорректный заказ", http.StatusBadRequest)
		return
	}
	itemCount, err := s.db.CountOrderItems(r.Context(), orderID)
	if err != nil || itemCount <= 0 {
		http.Error(w, "В заказе нет видео", http.StatusConflict)
		return
	}
	totalAds := int(itemCount) * 2
	payment, err := s.db.PreparePaymentWithMethod(r.Context(), u.ID, orderID, dbgen.PaymentMethodADS, r.FormValue("code"))
	if err != nil {
		s.operationError(w, r, "/cart", err)
		return
	}
	if payment.Method != dbgen.PaymentMethodADS && payment.Method != dbgen.PaymentMethodPROMO {
		_ = s.db.CancelPayment(r.Context(), payment.ID)
		payment, err = s.db.PreparePaymentWithMethod(r.Context(), u.ID, orderID, dbgen.PaymentMethodADS, r.FormValue("code"))
		if err != nil {
			s.operationError(w, r, "/cart", err)
			return
		}
	}
	if payment.Method != dbgen.PaymentMethodADS || payment.ExternalID == nil {
		if payment.Method == dbgen.PaymentMethodPROMO && payment.ExternalID != nil {
			if err := s.db.CompletePayment(r.Context(), *payment.ExternalID, "promo-"+*payment.ExternalID); err != nil {
				_ = s.db.CancelPayment(r.Context(), payment.ID)
				http.Error(w, "Не удалось применить промокод", http.StatusBadGateway)
				return
			}
			writeAdsJSON(w, http.StatusOK, map[string]any{"completed": totalAds, "total": totalAds, "videos": int(itemCount), "done": true, "promo": true})
			return
		}
		_ = s.db.CancelPayment(r.Context(), payment.ID)
		http.Error(w, "Рекламная оплата недоступна для этой корзины", http.StatusConflict)
		return
	}
	token, err := newAdsToken()
	if err != nil {
		s.unavailable(w, err)
		return
	}
	s.adsMu.Lock()
	now := time.Now()
	for token, session := range s.ads {
		if session.UserID == u.ID && session.OrderID == orderID && !now.After(session.ExpiresAt) && !session.Finalized {
			total := session.Total
			if total <= 0 {
				total = totalAds
				session.Total = total
			}
			completed := session.Completed
			s.adsMu.Unlock()
			writeAdsJSON(w, http.StatusOK, map[string]any{"token": token, "completed": completed, "total": total, "videos": total / 2})
			return
		}
	}
	s.ads[token] = &adsSession{UserID: u.ID, OrderID: orderID, PaymentID: payment.ID, PaymentPayload: *payment.ExternalID, Total: totalAds, ExpiresAt: now.Add(20 * time.Minute)}
	s.adsMu.Unlock()
	writeAdsJSON(w, http.StatusOK, map[string]any{"token": token, "completed": 0, "total": totalAds, "videos": int(itemCount)})
}

func (s *server) adsStatus(w http.ResponseWriter, r *http.Request) {
	if !s.opts.AdsEnabled || s.opts.DevLogin {
		http.NotFound(w, r)
		return
	}
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	if u.TgID <= 0 {
		http.Error(w, "Рекламная оплата доступна только внутри Telegram", http.StatusForbidden)
		return
	}
	s.cleanupAds(r.Context())
	token := strings.TrimSpace(r.URL.Query().Get("session"))
	if token == "" {
		http.Error(w, "Рекламная сессия не указана", http.StatusBadRequest)
		return
	}
	s.adsMu.Lock()
	session, exists := s.ads[token]
	if !exists || session.UserID != u.ID || time.Now().After(session.ExpiresAt) {
		s.adsMu.Unlock()
		http.Error(w, "Сессия рекламы истекла", http.StatusGone)
		return
	}
	completed, done, finalizing, total := session.Completed, session.Finalized, session.Finalizing, session.Total
	if total <= 0 {
		total = 2
		session.Total = total
	}
	s.adsMu.Unlock()
	writeAdsJSON(w, http.StatusOK, map[string]any{"completed": completed, "done": done, "finalizing": finalizing, "total": total, "videos": (total + 1) / 2})
}

func (s *server) completeAds(w http.ResponseWriter, r *http.Request) {
	if s.opts.DevLogin && r.FormValue("session") == "" {
		s.completeDevAds(w, r)
		return
	}
	if !s.opts.AdsEnabled || s.opts.DevLogin {
		http.NotFound(w, r)
		return
	}
	u, ok := s.requireUser(w, r, false)
	if !ok || !parseForm(w, r) {
		return
	}
	if u.TgID <= 0 {
		http.Error(w, "Рекламная оплата доступна только внутри Telegram", http.StatusForbidden)
		return
	}
	if s.db == nil {
		http.Error(w, "Хранилище оплаты недоступно", http.StatusServiceUnavailable)
		return
	}
	token := strings.TrimSpace(r.FormValue("session"))
	step, parseErr := strconv.Atoi(r.FormValue("step"))
	if token == "" || parseErr != nil || step <= 0 {
		http.Error(w, "Некорректный просмотр рекламы", http.StatusBadRequest)
		return
	}
	s.adsMu.Lock()
	session, exists := s.ads[token]
	if !exists || time.Now().After(session.ExpiresAt) || session.UserID != u.ID {
		s.adsMu.Unlock()
		http.Error(w, "Сессия рекламы истекла", http.StatusGone)
		return
	}
	total := session.Total
	if total <= 0 {
		total = 2
		session.Total = total
	}
	if step > total {
		s.adsMu.Unlock()
		http.Error(w, "Лишний просмотр рекламы", http.StatusBadRequest)
		return
	}
	if session.Finalized {
		s.adsMu.Unlock()
		writeAdsJSON(w, http.StatusOK, map[string]any{"completed": total, "done": true, "total": total})
		return
	}
	if step == total && session.Finalizing {
		s.adsMu.Unlock()
		http.Error(w, "Подтверждаем предыдущий просмотр", http.StatusConflict)
		return
	}
	if step <= session.Completed {
		completed := session.Completed
		s.adsMu.Unlock()
		writeAdsJSON(w, http.StatusOK, map[string]any{"completed": completed, "done": completed >= total, "total": total})
		return
	}
	if step != session.Completed+1 && !(step == total && session.Completed == total) {
		s.adsMu.Unlock()
		http.Error(w, "Просмотры должны идти по порядку", http.StatusConflict)
		return
	}
	payload := session.PaymentPayload
	if step < total {
		session.Completed = step
		s.adsMu.Unlock()
		writeAdsJSON(w, http.StatusOK, map[string]any{"completed": step, "done": false, "total": total})
		return
	}
	session.Completed = total
	session.Finalizing = true
	s.adsMu.Unlock()
	if err := s.db.CompletePayment(r.Context(), payload, "adsgram-"+token); err != nil {
		s.adsMu.Lock()
		if current, exists := s.ads[token]; exists {
			current.Completed = total - 1
			current.Finalizing = false
		}
		s.adsMu.Unlock()
		http.Error(w, "Не удалось подтвердить рекламную оплату", http.StatusBadGateway)
		return
	}
	s.adsMu.Lock()
	if current, exists := s.ads[token]; exists {
		current.Finalizing = false
		current.Finalized = true
		current.ExpiresAt = time.Now().Add(10 * time.Minute)
	}
	s.adsMu.Unlock()
	writeAdsJSON(w, http.StatusOK, map[string]any{"completed": total, "done": true, "total": total})
}

func (s *server) cancelAds(w http.ResponseWriter, r *http.Request) {
	if !s.opts.AdsEnabled || s.opts.DevLogin {
		http.NotFound(w, r)
		return
	}
	u, ok := s.requireUser(w, r, false)
	if !ok || !parseForm(w, r) {
		return
	}
	if u.TgID <= 0 {
		http.Error(w, "Рекламная оплата доступна только внутри Telegram", http.StatusForbidden)
		return
	}
	if s.db == nil {
		http.Error(w, "Хранилище оплаты недоступно", http.StatusServiceUnavailable)
		return
	}
	token := strings.TrimSpace(r.FormValue("session"))
	s.adsMu.Lock()
	session, exists := s.ads[token]
	if exists && session.UserID == u.ID {
		delete(s.ads, token)
	}
	s.adsMu.Unlock()
	if exists && session.UserID == u.ID && !session.Finalized {
		_ = s.db.CancelPayment(r.Context(), session.PaymentID)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) cleanupAds(ctx context.Context) {
	now := time.Now()
	var paymentIDs []int64
	s.adsMu.Lock()
	for token, session := range s.ads {
		if !now.After(session.ExpiresAt) || session.Finalizing {
			continue
		}
		if !session.Finalized {
			paymentIDs = append(paymentIDs, session.PaymentID)
		}
		delete(s.ads, token)
	}
	s.adsMu.Unlock()
	if s.db != nil {
		for _, paymentID := range paymentIDs {
			_ = s.db.CancelPayment(ctx, paymentID)
		}
	}
}

func (s *server) cancelAdsForOrder(ctx context.Context, userID, orderID int64) {
	var paymentIDs []int64
	s.adsMu.Lock()
	for token, session := range s.ads {
		if session.UserID != userID || session.OrderID != orderID || session.Finalized {
			continue
		}
		paymentIDs = append(paymentIDs, session.PaymentID)
		delete(s.ads, token)
	}
	s.adsMu.Unlock()
	if s.db != nil {
		for _, paymentID := range paymentIDs {
			_ = s.db.CancelPayment(ctx, paymentID)
		}
	}
}

func writeAdsJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func numericStars(n pgtype.Numeric) (int64, bool) {
	if !n.Valid {
		return 0, false
	}
	v, err := n.Float64Value()
	if err != nil || !v.Valid || v.Float64 <= 0 || v.Float64 != math.Trunc(v.Float64) {
		return 0, false
	}
	return int64(v.Float64), true
}

func (s *server) orders(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	page, err := s.pageData(r, u)
	if err != nil {
		s.unavailable(w, err)
		return
	}
	d := views.OrdersData{PageData: page}
	language := s.languageForRequest(r, string(u.Language))
	orders, err := s.db.ListUserOrders(r.Context(), u.ID)
	if err != nil {
		s.unavailable(w, err)
		return
	}
	jobs, err := s.db.ListUserJobs(r.Context(), u.ID)
	if err != nil {
		s.unavailable(w, err)
		return
	}
	byOrder := map[int64][]views.Job{}
	for _, row := range jobs {
		byOrder[row.RenderJob.OrderID] = append(byOrder[row.RenderJob.OrderID], s.viewJob(row.RenderJob, views.DatasetTitle(row.DatasetTitle, language), row.MusicTitle, u, language))
	}
	for _, row := range orders {
		d.Orders = append(d.Orders, views.Order{ID: row.Order.ID, DisplayNumber: row.OrderNumber, Status: string(row.Order.Status), CreatedAt: row.Order.CreatedAt.Format("02.01.2006 15:04"), Total: numericLabel(row.Order.TotalAmount) + " ⭐", Payment: paymentLabel(row.Order.PaymentMethod), Jobs: byOrder[row.Order.ID]})
	}
	for _, j := range jobs {
		if j.RenderJob.Status == dbgen.JobStatusPENDING || j.RenderJob.Status == dbgen.JobStatusPROCESSING {
			d.HasActive = true
			break
		}
	}
	s.page(w, r, views.Orders(d))
}

func paymentLabel(p *dbgen.PaymentMethod) string {
	if p == nil {
		return ""
	}
	return string(*p)
}

func (s *server) viewJob(job dbgen.RenderJob, title string, music *string, u dbgen.User, language string) views.Job {
	v := views.Job{ID: job.ID, Title: title, Theme: themeLabel(job.Theme), Music: musicLabel(music, language), Status: string(job.Status), CreatedAt: job.CreatedAt.Format("02.01.2006 15:04"), CanSendTelegram: u.TgID > 0 && s.opts.Telegram != nil && job.TelegramFileID == nil, TelegramSent: job.TelegramFileID != nil}
	if job.ErrorMessage != nil {
		v.Error = *job.ErrorMessage
	}
	if job.Status == dbgen.JobStatusCOMPLETED {
		prefix := "/jobs"
		if u.Role == dbgen.UserRoleADMIN {
			prefix = "/admin/jobs"
		}
		v.VideoURL = prefix + "/" + strconv.FormatInt(job.ID, 10) + "/video"
		v.DownloadURL = v.VideoURL + "?download=1"
	}
	return v
}

func (s *server) jobCard(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	row, err := s.db.GetOwnedJob(r.Context(), dbgen.GetOwnedJobParams{ID: id, UserID: u.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.unavailable(w, err)
		return
	}
	language := s.languageForRequest(r, string(u.Language))
	render(w, r, views.JobState(s.viewJob(row.RenderJob, views.DatasetTitle(row.DatasetTitle, language), row.MusicTitle, u, language), false, language))
}

func (s *server) video(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	row, err := s.db.GetOwnedJob(r.Context(), dbgen.GetOwnedJobParams{ID: id, UserID: u.ID})
	if err != nil || row.OrderStatus != dbgen.OrderStatusPAID || row.RenderJob.Status != dbgen.JobStatusCOMPLETED {
		http.NotFound(w, r)
		return
	}
	s.serveFile(w, r, fmt.Sprintf("renders/job-%d.mp4", id), r.URL.Query().Get("download") == "1")
}

func (s *server) retryJob(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err = s.db.RetryJob(r.Context(), u.ID, id); err != nil {
		s.operationError(w, r, "/status", err)
		return
	}
	redirectMessage(w, r, "/status", "Видео снова поставлено в очередь", "")
}

func (s *server) sendVideo(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	row, err := s.db.GetOwnedJob(r.Context(), dbgen.GetOwnedJobParams{ID: id, UserID: u.ID})
	if err != nil || row.OrderStatus != dbgen.OrderStatusPAID || row.RenderJob.Status != dbgen.JobStatusCOMPLETED || u.TgID <= 0 || s.opts.Telegram == nil {
		redirectMessage(w, r, "/status", "", "Видео пока недоступно для отправки")
		return
	}
	path, err := service.SafePath(s.opts.DataDir, fmt.Sprintf("renders/job-%d.mp4", id))
	if err != nil {
		redirectMessage(w, r, "/status", "", "Файл видео недоступен")
		return
	}
	fileInfo, statErr := os.Stat(path)
	if statErr != nil || !fileInfo.Mode().IsRegular() {
		redirectMessage(w, r, "/status", "", "Файл видео недоступен")
		return
	}
	fileID, err := s.opts.Telegram.SendVideo(r.Context(), u.TgID, path, "")
	if err != nil {
		redirectMessage(w, r, "/status", "", "Telegram не принял видео. Попробуй ещё раз.")
		return
	}
	if _, err := s.db.SetJobTelegramFile(r.Context(), dbgen.SetJobTelegramFileParams{ID: id, TelegramFileID: &fileID}); err != nil {
		s.unavailable(w, err)
		return
	}
	redirectMessage(w, r, "/status", "Видео отправлено в Telegram", "")
}

func (s *server) adminVideo(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireUser(w, r, true); !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	row, err := s.db.GetAdminJob(r.Context(), id)
	if err != nil || row.RenderJob.Status != dbgen.JobStatusCOMPLETED {
		http.NotFound(w, r)
		return
	}
	s.serveFile(w, r, fmt.Sprintf("renders/job-%d.mp4", id), r.URL.Query().Get("download") == "1")
}

func (s *server) adminJobCard(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, true)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	row, err := s.db.GetAdminJob(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	language := s.languageForRequest(r, string(u.Language))
	j := s.viewJob(row.RenderJob, views.DatasetTitle(row.DatasetTitle, language), row.MusicTitle, u, language)
	j.CanSendTelegram = false
	render(w, r, views.JobState(j, true, s.languageForRequest(r, string(u.Language))))
}
func (s *server) adminMusicToggle(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, true)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	v, err := s.db.GetMusicTrackByID(r.Context(), id)
	if err != nil {
		s.operationError(w, r, "/admin?tab=library", err)
		return
	}
	if _, err = s.db.AdminSetMusicActive(r.Context(), dbgen.AdminSetMusicActiveParams{ID: id, IsActive: !v.IsActive}); err != nil {
		s.operationError(w, r, "/admin?tab=library", err)
		return
	}
	redirectMessage(w, r, "/admin?tab=library", views.Translate(s.languageForRequest(r, string(u.Language)), "admin.notice.music.updated"), "")
}
func (s *server) adminDatasetPublish(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, true)
	if !ok {
		return
	}
	id, e := idParam(r)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	if changed, err := s.db.AdminPublishDataset(r.Context(), id); err != nil || changed != 1 {
		if err == nil {
			err = storage.ErrNotFound
		}
		e = err
		s.operationError(w, r, "/admin?tab=library", e)
		return
	}
	redirectMessage(w, r, "/admin?tab=library", views.Translate(s.languageForRequest(r, string(u.Language)), "admin.notice.dataset.published"), "")
}
func (s *server) adminMusicPublish(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, true)
	if !ok {
		return
	}
	id, e := idParam(r)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	if changed, err := s.db.AdminPublishMusic(r.Context(), id); err != nil || changed != 1 {
		if err == nil {
			err = storage.ErrNotFound
		}
		e = err
		s.operationError(w, r, "/admin?tab=library", e)
		return
	}
	redirectMessage(w, r, "/admin?tab=library", views.Translate(s.languageForRequest(r, string(u.Language)), "admin.notice.music.published"), "")
}
func (s *server) adminRetryJob(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, true)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	changed, err := s.db.AdminRetryJob(r.Context(), id)
	if err != nil {
		s.operationError(w, r, "/admin?tab=orders", err)
		return
	}
	if changed != 1 {
		redirectMessage(w, r, "/admin?tab=orders", "", views.Translate(s.languageForRequest(r, string(u.Language)), "admin.retry.unavailable"))
		return
	}
	redirectMessage(w, r, "/admin?tab=orders", views.Translate(s.languageForRequest(r, string(u.Language)), "admin.notice.job.queued"), "")
}
func (s *server) adminUserRole(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, true)
	if !ok {
		return
	}
	if !parseForm(w, r) {
		return
	}
	id, e := idParam(r)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	role := dbgen.UserRole(r.FormValue("role"))
	if role != dbgen.UserRoleUSER && role != dbgen.UserRoleADMIN {
		redirectMessage(w, r, "/admin?tab=users", "", views.Translate(s.languageForRequest(r, string(u.Language)), "admin.error.role"))
		return
	}
	if id == u.ID && role != dbgen.UserRoleADMIN {
		redirectMessage(w, r, "/admin?tab=users", "", views.Translate(s.languageForRequest(r, string(u.Language)), "admin.error.self.demote"))
		return
	}
	if _, e = s.db.AdminSetUserRole(r.Context(), dbgen.AdminSetUserRoleParams{ID: id, Role: role}); e != nil {
		s.operationError(w, r, "/admin?tab=users", e)
		return
	}
	redirectMessage(w, r, "/admin?tab=users", views.Translate(s.languageForRequest(r, string(u.Language)), "admin.notice.role.updated"), "")
}
func (s *server) adminCreatePromo(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, true)
	if !ok {
		return
	}
	if !parseForm(w, r) {
		return
	}
	code := strings.ToUpper(strings.TrimSpace(r.FormValue("code")))
	discount, e := strconv.Atoi(r.FormValue("discount_percent"))
	if e != nil || code == "" || len(code) > 50 || discount < 1 || discount > 100 {
		redirectMessage(w, r, "/admin?tab=promos", "", views.Translate(s.languageForRequest(r, string(u.Language)), "admin.error.promo"))
		return
	}
	var maxUses *int
	if raw := r.FormValue("max_uses"); raw != "" {
		v, er := strconv.Atoi(raw)
		if er != nil || v < 1 {
			redirectMessage(w, r, "/admin?tab=promos", "", views.Translate(s.languageForRequest(r, string(u.Language)), "admin.error.limit"))
			return
		}
		maxUses = &v
	}
	var until *time.Time
	if raw := r.FormValue("valid_until"); raw != "" {
		day, er := time.Parse("2006-01-02", raw)
		if er != nil {
			redirectMessage(w, r, "/admin?tab=promos", "", views.Translate(s.languageForRequest(r, string(u.Language)), "admin.error.date"))
			return
		}
		end := day.AddDate(0, 0, 1)
		until = &end
	}
	if _, e = s.db.CreatePromocode(r.Context(), dbgen.CreatePromocodeParams{Code: code, DiscountPercent: discount, MaxUses: maxUses, ValidUntil: until}); e != nil {
		s.operationError(w, r, "/admin?tab=promos", e)
		return
	}
	redirectMessage(w, r, "/admin?tab=promos", views.Translate(s.languageForRequest(r, string(u.Language)), "admin.notice.promo.created"), "")
}
func (s *server) adminPromoToggle(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, true)
	if !ok {
		return
	}
	id, e := idParam(r)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	p, e := s.db.GetPromocodeByID(r.Context(), id)
	if e != nil {
		s.operationError(w, r, "/admin?tab=promos", e)
		return
	}
	if _, e = s.db.AdminSetPromocodeActive(r.Context(), dbgen.AdminSetPromocodeActiveParams{ID: id, IsActive: !p.IsActive}); e != nil {
		s.operationError(w, r, "/admin?tab=promos", e)
		return
	}
	redirectMessage(w, r, "/admin?tab=promos", views.Translate(s.languageForRequest(r, string(u.Language)), "admin.notice.promo.updated"), "")
}

func (s *server) operationError(w http.ResponseWriter, r *http.Request, path string, err error) {
	msg := "Не удалось выполнить действие. Попробуй ещё раз."
	switch {
	case errors.Is(err, storage.ErrNotFound), errors.Is(err, pgx.ErrNoRows):
		msg = "Объект не найден или недоступен"
	case errors.Is(err, storage.ErrEmptyCart):
		msg = "Сначала добавь видео в корзину"
	case errors.Is(err, storage.ErrUnavailableResource):
		msg = "Один из материалов больше недоступен. Убери его из корзины."
	case errors.Is(err, storage.ErrInvalid):
		msg = "Проверь выбранные данные и параметры заказа"
	case errors.Is(err, storage.ErrCartFull):
		msg = "В корзине уже 10 видео"
	case errors.Is(err, storage.ErrPromoUnavailable):
		msg = "Промокод недействителен, исчерпан или уже использован"
	case errors.Is(err, storage.ErrPaymentPending):
		msg = "Для этой корзины уже создан счёт. Заверши оплату в Telegram."
	case errors.Is(err, storage.ErrPaymentInvalid):
		msg = "Заказ больше нельзя оплатить"
	default:
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) && pgerr.Code == "23505" {
			msg = "Такая запись уже существует"
		}
	}
	redirectMessage(w, r, path, "", msg)
}
func numericLabel(n pgtype.Numeric) string {
	if !n.Valid {
		return "0"
	}
	v, err := n.Float64Value()
	if err != nil || !v.Valid {
		return "0"
	}
	return strconv.FormatFloat(v.Float64, 'f', -1, 64)
}
func themeLabel(t dbgen.RenderTheme) string {
	switch t {
	case dbgen.RenderThemeDARKGLASS:
		return "Dark Glass"
	case dbgen.RenderThemeCORPORATELIGHT:
		return "Corporate Light"
	case dbgen.RenderThemeCYBERHUD:
		return "Cyber HUD"
	case dbgen.RenderThemeVINTAGEPRINT:
		return "Vintage Print"
	}
	return string(t)
}
func musicLabel(title *string, language string) string {
	if title == nil {
		return views.Translate(language, "studio.no.music")
	}
	return *title
}
func durationLabel(seconds *int) string {
	if seconds == nil {
		return "Аудиотрек"
	}
	return fmt.Sprintf("%d:%02d", *seconds/60, *seconds%60)
}
func isAdminPath(r *http.Request) bool { return strings.HasPrefix(r.URL.Path, "/admin/") }

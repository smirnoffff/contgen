package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"contgen/internal/storage/dbgen"
	"contgen/internal/web/views"
)

const adminOrderPageSize = 20

func parseTab(r *http.Request) string {
	switch tab := r.URL.Query().Get("tab"); tab {
	case "library", "users", "orders", "promos":
		return tab
	}
	return "overview"
}

func (s *server) admin(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, true)
	if !ok {
		return
	}
	page, err := s.pageData(r, u)
	if err != nil {
		s.unavailable(w, err)
		return
	}
	d := views.AdminData{PageData: page, Tab: parseTab(r), IncludeDemo: r.URL.Query().Get("demo") == "1", UpdatedAt: time.Now().Format("15:04"), CurrentPage: 1}
	summary, err := s.db.AdminSummary(r.Context(), d.IncludeDemo)
	if err != nil {
		s.unavailable(w, err)
		return
	}
	d.UserCount, d.OrderCount, d.JobCount = summary.Users, summary.Orders, summary.CompletedJobs
	d.QueuedCount, d.ProcessingCount, d.FailedCount = summary.PendingJobs, summary.ProcessingJobs, summary.FailedJobs
	d.PendingCount = summary.PendingJobs + summary.ProcessingJobs
	d.CartCountTotal, d.DeliveredCount = summary.Carts, summary.DeliveredJobs
	d.AdOrders, d.PromoOrders = summary.AdOrders, summary.PromoOrders
	revenue, err := s.db.AdminRevenue(r.Context(), d.IncludeDemo)
	if err != nil {
		s.unavailable(w, err)
		return
	}
	d.Revenue, d.StarsPayments = numericLabel(revenue.StarsReceived)+" ⭐", revenue.StarsPayments
	switch d.Tab {
	case "users":
		rows, err := s.db.AdminListUsers(r.Context(), d.IncludeDemo)
		if err != nil {
			s.unavailable(w, err)
			return
		}
		for _, user := range rows {
			username := ""
			if user.Username != nil {
				username = "@" + *user.Username
			}
			d.Users = append(d.Users, views.AdminUser{ID: user.ID, Name: user.FirstName, Username: username, TelegramID: strconv.FormatInt(user.TgID, 10), Role: string(user.Role), JoinedAt: user.CreatedAt.Format("02.01.2006"), IsSelf: user.ID == u.ID})
		}
	case "library":
		datasets, err := s.db.AdminListDatasets(r.Context(), d.IncludeDemo)
		if err != nil {
			s.unavailable(w, err)
			return
		}
		for _, row := range datasets {
			owner := views.Translate(d.Language, "library.common")
			if row.OwnerName != nil {
				owner = *row.OwnerName
			}
			d.Datasets = append(d.Datasets, views.DatasetItem{ID: row.Dataset.ID, Title: row.Dataset.Title, Active: true, System: row.Dataset.UserID == nil, CanPublish: row.Dataset.UserID != nil, Detail: fmt.Sprintf(views.Translate(d.Language, "studio.points.short"), row.PointCount) + " · " + owner})
		}
		music, err := s.db.AdminListMusicTracks(r.Context(), d.IncludeDemo)
		if err != nil {
			s.unavailable(w, err)
			return
		}
		for _, row := range music {
			d.Music = append(d.Music, views.MusicItem{ID: row.MusicTrack.ID, Title: row.MusicTrack.Title, Active: row.MusicTrack.IsActive, System: row.MusicTrack.UserID == nil, CanPublish: row.MusicTrack.UserID != nil, Detail: durationLabel(row.MusicTrack.DurationSec), URL: fmt.Sprintf("/music/%d/audio", row.MusicTrack.ID)})
		}
	case "promos":
		rows, err := s.db.AdminListPromocodes(r.Context())
		if err != nil {
			s.unavailable(w, err)
			return
		}
		for _, promo := range rows {
			expiry := views.Translate(d.Language, "admin.promo.no.expiry")
			if promo.ValidUntil != nil {
				expiry = fmt.Sprintf(views.Translate(d.Language, "admin.promo.until"), promo.ValidUntil.Format("02.01.2006"))
			}
			usage := strconv.Itoa(promo.UsedCount)
			if promo.MaxUses != nil {
				usage += fmt.Sprintf(" / %d", *promo.MaxUses)
			}
			d.Promos = append(d.Promos, views.PromoItem{ID: promo.ID, Code: promo.Code, Discount: fmt.Sprintf("−%d%%", promo.DiscountPercent), Usage: usage, Expires: expiry, Active: promo.IsActive, State: promoState(promo, time.Now())})
		}
	case "orders":
		d.FilterStatus = r.URL.Query().Get("status")
		switch d.FilterStatus {
		case "ALL", "CART", "CANCELLED", "PAID":
		default:
			d.FilterStatus = "PAID"
		}
		status := d.FilterStatus
		if status == "ALL" {
			status = ""
		}
		d.Search = strings.TrimSpace(r.URL.Query().Get("q"))
		if len([]rune(d.Search)) > 100 {
			d.Search = string([]rune(d.Search)[:100])
		}
		d.TotalOrders, err = s.db.AdminCountOrders(r.Context(), dbgen.AdminCountOrdersParams{IncludeDemo: d.IncludeDemo, OrderStatus: status, Search: d.Search})
		if err != nil {
			s.unavailable(w, err)
			return
		}
		if n, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && n > 0 {
			d.CurrentPage = n
		}
		lastPage := max(1, int((d.TotalOrders+adminOrderPageSize-1)/adminOrderPageSize))
		d.CurrentPage = min(d.CurrentPage, lastPage)
		d.HasPrevious, d.HasNext = d.CurrentPage > 1, d.CurrentPage < lastPage
		rows, err := s.db.AdminListOrders(r.Context(), dbgen.AdminListOrdersParams{IncludeDemo: d.IncludeDemo, OrderStatus: status, Search: d.Search, PageLimit: adminOrderPageSize, PageOffset: (d.CurrentPage - 1) * adminOrderPageSize})
		if err != nil {
			s.unavailable(w, err)
			return
		}
		ids := make([]int64, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.Order.ID)
		}
		jobs, err := s.db.AdminListJobs(r.Context(), ids)
		if err != nil {
			s.unavailable(w, err)
			return
		}
		byOrder := make(map[int64][]views.Job)
		for _, row := range jobs {
			job := s.viewJob(row.RenderJob, views.DatasetTitle(row.DatasetTitle, d.Language), row.MusicTitle, u, d.Language)
			job.CanSendTelegram = false
			byOrder[row.RenderJob.OrderID] = append(byOrder[row.RenderJob.OrderID], job)
		}
		for _, row := range rows {
			username := ""
			if row.Username != nil {
				username = "@" + *row.Username
			}
			method := ""
			if row.Order.PaymentMethod != nil {
				method = string(*row.Order.PaymentMethod)
			}
			d.Orders = append(d.Orders, views.AdminOrder{Order: views.Order{ID: row.Order.ID, DisplayNumber: row.OrderNumber, Status: string(row.Order.Status), CreatedAt: row.Order.CreatedAt.Format("02.01.2006 15:04"), Total: numericLabel(row.StarsReceived) + " ⭐", Payment: method, Jobs: byOrder[row.Order.ID]}, OwnerName: row.FirstName, OwnerUsername: username, TelegramID: strconv.FormatInt(row.TgID, 10), LocalDemo: row.TgID <= 0, ItemCount: row.ItemCount})
		}
	}
	s.page(w, r, views.Admin(d))
}

func promoState(promo dbgen.Promocode, now time.Time) string {
	if !promo.IsActive {
		return "disabled"
	}
	if promo.ValidUntil != nil && !promo.ValidUntil.After(now) {
		return "expired"
	}
	if promo.MaxUses != nil && promo.UsedCount >= *promo.MaxUses {
		return "exhausted"
	}
	return "active"
}

package views

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"testing"
)

func TestAdminAdOrderNeverDisplaysNominalStarsAsPayment(t *testing.T) {
	order := AdminOrder{Order: Order{ID: 123, DisplayNumber: 4, Status: "PAID", Payment: "ADS", Total: "500 ⭐"}, OwnerName: "Alice", OwnerUsername: "@alice", TelegramID: "42", ItemCount: 2}
	var out bytes.Buffer
	if err := AdminOrderCard(order, "ru").Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	if strings.Contains(html, "500") || !strings.Contains(html, "0 Stars") || !strings.Contains(html, "@alice") || !strings.Contains(html, "ID 123") {
		t.Fatalf("incorrect order display: %s", html)
	}
}

func TestAdminPaginationKeepsSearchAndScope(t *testing.T) {
	d := AdminData{Tab: "orders", IncludeDemo: true, Search: "@alice & bob", FilterStatus: "PAID", CurrentPage: 3}
	u, err := url.Parse(d.CurrentURL())
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("demo") != "1" || q.Get("q") != d.Search || q.Get("page") != "3" || q.Get("status") != "PAID" {
		t.Fatalf("lost filter state: %s", u)
	}
	u, err = url.Parse(d.DemoURL())
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("demo") != "" || u.Query().Get("q") != d.Search {
		t.Fatalf("invalid demo toggle: %s", u)
	}
}

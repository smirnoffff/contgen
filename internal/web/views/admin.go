package views

import (
	"fmt"
	"net/url"
	"strconv"
)

func (d AdminData) URL(tab string) string {
	q := url.Values{"tab": {tab}}
	if d.IncludeDemo {
		q.Set("demo", "1")
	}
	return "/admin?" + q.Encode()
}

func (d AdminData) OrdersURL(status string, page int) string {
	q := url.Values{"tab": {"orders"}, "status": {status}}
	if d.IncludeDemo {
		q.Set("demo", "1")
	}
	if d.Search != "" {
		q.Set("q", d.Search)
	}
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	return "/admin?" + q.Encode()
}

func (d AdminData) CurrentURL() string {
	if d.Tab == "orders" {
		return d.OrdersURL(d.FilterStatus, d.CurrentPage)
	}
	return d.URL(d.Tab)
}

func (d AdminData) DemoURL() string {
	d.IncludeDemo = !d.IncludeDemo
	return d.CurrentURL()
}

func adminOrderNumber(order AdminOrder, language string) string {
	if order.Status == "CART" {
		return tr(language, "order.cart")
	}
	return fmt.Sprintf(tr(language, "status.order"), order.DisplayNumber)
}

func adminOrderPayment(order AdminOrder, language string) string {
	if order.Status != "PAID" {
		return tr(language, "admin.payment.unpaid")
	}
	switch order.Payment {
	case "ADS":
		return tr(language, "admin.payment.ads")
	case "PROMO":
		return tr(language, "admin.payment.promo")
	case "STARS":
		return order.Total
	default:
		return tr(language, "admin.payment.unknown")
	}
}

func adminOrderFilters(language string) []adminTabOption {
	return []adminTabOption{{"PAID", tr(language, "admin.filter.placed")}, {"CART", tr(language, "admin.filter.carts")}, {"CANCELLED", tr(language, "order.cancelled")}, {"ALL", tr(language, "admin.filter.all")}}
}

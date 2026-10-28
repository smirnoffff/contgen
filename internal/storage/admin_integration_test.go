//go:build integration

package storage_test

import (
	"fmt"
	"testing"

	"contgen/internal/storage"
	"contgen/internal/storage/dbgen"
)

func adminOrderFixture(t *testing.T, s *storage.Storage, userID int64, status, method string, total int) int64 {
	t.Helper()
	var id int64
	err := s.Pool.QueryRow(t.Context(), `INSERT INTO orders (user_id,status,payment_method,total_amount) VALUES ($1,$2::order_status,NULLIF($3,'')::payment_method,$4) RETURNING id`, userID, status, method, total).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func adminPaymentFixture(t *testing.T, s *storage.Storage, orderID int64, method, status string, total int) {
	t.Helper()
	_, err := s.Pool.Exec(t.Context(), `INSERT INTO payments (order_id,method,status,amount,external_id) VALUES ($1,$2::payment_method,$3::payment_status,$4,$5)`, orderID, method, status, total, fmt.Sprintf("charge-%d-%s-%d", orderID, status, total))
	if err != nil {
		t.Fatal(err)
	}
}

func TestAdminAccountingSeparatesStarsAdsAndLocalFixtures(t *testing.T) {
	s := testStorage(t)
	ctx := t.Context()
	real, local := createUser(t, s, 1001), createUser(t, s, -1)
	ads := adminOrderFixture(t, s, real.ID, "PAID", "ADS", 200)
	adminPaymentFixture(t, s, ads, "ADS", "SUCCEEDED", 200)
	stars := adminOrderFixture(t, s, real.ID, "PAID", "STARS", 999)
	adminPaymentFixture(t, s, stars, "STARS", "SUCCEEDED", 45)
	adminPaymentFixture(t, s, stars, "STARS", "FAILED", 80)
	adminPaymentFixture(t, s, stars, "STARS", "PENDING", 60)
	adminPaymentFixture(t, s, stars, "STARS", "REFUNDED", 70)
	_, err := s.Pool.Exec(ctx, "INSERT INTO payments(order_id,method,status,amount) VALUES ($1,'STARS','SUCCEEDED',88)", stars)
	if err != nil {
		t.Fatal(err)
	}
	promo := adminOrderFixture(t, s, real.ID, "PAID", "PROMO", 0)
	adminPaymentFixture(t, s, promo, "PROMO", "SUCCEEDED", 0)
	cart := adminOrderFixture(t, s, real.ID, "CART", "", 0)
	adminPaymentFixture(t, s, cart, "STARS", "SUCCEEDED", 77)
	localAds := adminOrderFixture(t, s, local.ID, "PAID", "ADS", 50)
	adminPaymentFixture(t, s, localAds, "ADS", "SUCCEEDED", 50)
	localStars := adminOrderFixture(t, s, local.ID, "PAID", "STARS", 25)
	adminPaymentFixture(t, s, localStars, "STARS", "SUCCEEDED", 25)
	dataset, err := s.CreateDatasetWithPoints(ctx, nil, "Admin test", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		order  int64
		status string
		sent   bool
	}{
		{ads, "COMPLETED", true}, {ads, "COMPLETED", false}, {stars, "PENDING", false},
		{stars, "PROCESSING", false}, {promo, "FAILED", false}, {cart, "PENDING", false}, {localAds, "COMPLETED", false},
	} {
		_, err = s.Pool.Exec(ctx, `INSERT INTO render_jobs (order_id,dataset_id,status,completed_at,error_message,telegram_file_id) VALUES ($1,$2,$3::job_status,CASE WHEN $3='COMPLETED' THEN now() END,CASE WHEN $3='FAILED' THEN 'test failure' END,CASE WHEN $4 THEN 'telegram-file' END)`, fixture.order, dataset.ID, fixture.status, fixture.sent)
		if err != nil {
			t.Fatal(err)
		}
	}
	revenue, err := s.AdminRevenue(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	amount, _ := revenue.StarsReceived.Float64Value()
	if amount.Float64 != 45 || revenue.StarsPayments != 1 {
		t.Fatalf("real revenue: %+v, amount=%v", revenue, amount.Float64)
	}
	revenue, err = s.AdminRevenue(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	amount, _ = revenue.StarsReceived.Float64Value()
	if amount.Float64 != 70 || revenue.StarsPayments != 2 {
		t.Fatalf("test-inclusive revenue: %+v, amount=%v", revenue, amount.Float64)
	}
	summary, err := s.AdminSummary(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Users != 1 || summary.Orders != 3 || summary.Carts != 1 || summary.CompletedJobs != 2 || summary.PendingJobs != 1 || summary.ProcessingJobs != 1 || summary.FailedJobs != 1 || summary.DeliveredJobs != 1 || summary.AdOrders != 1 || summary.PromoOrders != 1 {
		t.Fatalf("real summary: %+v", summary)
	}
	summary, err = s.AdminSummary(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Users != 2 || summary.Orders != 5 || summary.CompletedJobs != 3 || summary.AdOrders != 2 {
		t.Fatalf("all summary: %+v", summary)
	}
	rows, err := s.AdminListOrders(ctx, dbgen.AdminListOrdersParams{OrderStatus: "PAID", PageLimit: 20})
	if err != nil || len(rows) != 3 {
		t.Fatalf("orders: %d, %v", len(rows), err)
	}
	for _, row := range rows {
		got, _ := row.StarsReceived.Float64Value()
		want := float64(0)
		if row.Order.ID == stars {
			want = 45
		}
		if got.Float64 != want {
			t.Errorf("order %d revenue = %v, want %v", row.Order.ID, got.Float64, want)
		}
	}
}

func TestAdminOrdersPaginationSearchAndCompleteJobLists(t *testing.T) {
	s := testStorage(t)
	ctx := t.Context()
	real, local := createUser(t, s, 1002), createUser(t, s, -1)
	_, err := s.Pool.Exec(ctx, "UPDATE users SET username='demo_owner' WHERE id=$1", real.ID)
	if err != nil {
		t.Fatal(err)
	}
	var last int64
	for range 23 {
		last = adminOrderFixture(t, s, real.ID, "PAID", "ADS", 50)
	}
	adminOrderFixture(t, s, local.ID, "PAID", "ADS", 50)
	adminOrderFixture(t, s, real.ID, "CART", "", 0)
	count, err := s.AdminCountOrders(ctx, dbgen.AdminCountOrdersParams{OrderStatus: "PAID", Search: "@DEMO_OWNER"})
	if err != nil || count != 23 {
		t.Fatalf("search count: %d, %v", count, err)
	}
	page, err := s.AdminListOrders(ctx, dbgen.AdminListOrdersParams{OrderStatus: "PAID", Search: "@demo_owner", PageLimit: 20, PageOffset: 20})
	if err != nil || len(page) != 3 {
		t.Fatalf("page: %d, %v", len(page), err)
	}
	for _, row := range page {
		if row.TgID <= 0 || row.Order.Status != "PAID" {
			t.Errorf("wrong scope: %+v", row)
		}
	}
	dataset, err := s.CreateDatasetWithPoints(ctx, nil, "Admin test", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO render_jobs(order_id,dataset_id) SELECT $1,$2 FROM generate_series(1,105)`, last, dataset.ID)
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := s.AdminListJobs(ctx, []int64{last})
	if err != nil || len(jobs) != 105 {
		t.Fatalf("page's complete job list: %d, %v", len(jobs), err)
	}
	empty, err := s.AdminListJobs(ctx, []int64{})
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty job list: %d, %v", len(empty), err)
	}
}

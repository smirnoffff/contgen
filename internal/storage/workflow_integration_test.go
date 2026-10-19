//go:build integration

package storage_test

import (
	"context"
	"errors"
	"testing"

	"contgen/internal/model"
	"contgen/internal/storage"
	"contgen/internal/storage/dbgen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestCartResourceOwnershipAndRemoval(t *testing.T) {
	s := testStorage(t)
	ctx := context.Background()
	user := createUser(t, s, 101)
	other := createUser(t, s, 202)
	points := []model.Point{{EntityName: "Go", Value: 10, RecordedAt: 2024}}
	own, err := s.CreateDatasetWithPoints(ctx, &user.ID, "Мой набор", points)
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := s.CreateDatasetWithPoints(ctx, &other.ID, "Чужой набор", points)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.AddCartItem(ctx, user.ID, own.ID, nil, dbgen.RenderThemeCYBERHUD)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != dbgen.JobStatusPENDING {
		t.Fatalf("new cart job: %s", job.Status)
	}
	if _, err := s.AddCartItem(ctx, user.ID, foreign.ID, nil, dbgen.RenderThemeCYBERHUD); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("foreign dataset accepted: %v", err)
	}
	items, err := s.ListCartItems(ctx, user.ID)
	if err != nil || len(items) != 1 || items[0].RenderJob.ID != job.ID {
		t.Fatalf("cart: %+v, %v", items, err)
	}
	if err := s.RemoveCartItem(ctx, user.ID, job.ID); err != nil {
		t.Fatal(err)
	}
	items, err = s.ListCartItems(ctx, user.ID)
	if err != nil || len(items) != 0 {
		t.Fatalf("removed cart item remains: %+v, %v", items, err)
	}
	if err := s.RemoveCartItem(ctx, user.ID, job.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("second removal: %v", err)
	}
}

func TestWorkerQueueTransitions(t *testing.T) {
	s := testStorage(t)
	ctx := context.Background()
	user := createUser(t, s, 303)
	dataset, err := s.CreateDatasetWithPoints(ctx, &user.ID, "Очередь", []model.Point{{EntityName: "Go", Value: 1, RecordedAt: 2024}})
	if err != nil {
		t.Fatal(err)
	}
	cart, err := s.GetOrCreateCart(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.CreateRenderJob(ctx, dbgen.CreateRenderJobParams{OrderID: cart.ID, DatasetID: dataset.ID, Theme: dbgen.RenderThemeDARKGLASS})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkOrderPaid(ctx, cart.ID, dbgen.PaymentMethodSTARS, numeric(t, "1.00")); err != nil {
		t.Fatal(err)
	}
	lease, err := s.AcquireWorker(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	claimed, err := lease.ClaimNextRenderJob(ctx)
	if err != nil || claimed.ID != job.ID || claimed.Status != dbgen.JobStatusPROCESSING {
		t.Fatalf("claim: %+v, %v", claimed, err)
	}
	if n, err := lease.FinishRenderJob(ctx, job.ID); err != nil || n != 1 {
		t.Fatalf("finish: %d, %v", n, err)
	}
	loaded, err := s.GetJobByID(ctx, job.ID)
	if err != nil || loaded.Status != dbgen.JobStatusCOMPLETED || loaded.CompletedAt == nil {
		t.Fatalf("completed: %+v, %v", loaded, err)
	}
	if _, err := lease.ClaimNextRenderJob(ctx); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("claimed completed job: %v", err)
	}
}

func numeric(t *testing.T, value string) pgtype.Numeric {
	t.Helper()
	var n pgtype.Numeric
	if err := n.Scan(value); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestPaymentPreparationAndCompletion(t *testing.T) {
	s := testStorage(t)
	ctx := context.Background()
	user := createUser(t, s, 404)
	dataset, err := s.CreateDatasetWithPoints(ctx, &user.ID, "Оплата", []model.Point{{EntityName: "Go", Value: 2, RecordedAt: 2024}})
	if err != nil {
		t.Fatal(err)
	}
	cart, err := s.GetOrCreateCart(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateRenderJob(ctx, dbgen.CreateRenderJobParams{OrderID: cart.ID, DatasetID: dataset.ID, Theme: dbgen.RenderThemeDARKGLASS}); err != nil {
		t.Fatal(err)
	}
	payment, err := s.PreparePayment(ctx, user.ID, cart.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if payment.ExternalID == nil || payment.Method != dbgen.PaymentMethodSTARS || payment.Status != dbgen.PaymentStatusPENDING {
		t.Fatalf("prepared payment: %+v", payment)
	}
	if !s.ValidatePayment(ctx, *payment.ExternalID, 50) || s.ValidatePayment(ctx, *payment.ExternalID, 51) {
		t.Fatalf("payment amount validation failed")
	}
	if again, err := s.PreparePayment(ctx, user.ID, cart.ID, ""); err != nil || again.ID != payment.ID {
		t.Fatalf("pending payment was not reused: %+v, %v", again, err)
	}
	if err := s.CompletePayment(ctx, *payment.ExternalID, "charge-1"); err != nil {
		t.Fatal(err)
	}
	if err := s.CompletePayment(ctx, *payment.ExternalID, "charge-1"); err != nil {
		t.Fatalf("idempotent completion: %v", err)
	}
	order, err := s.GetOrderByID(ctx, cart.ID)
	if err != nil || order.Status != dbgen.OrderStatusPAID || order.PaymentMethod == nil || *order.PaymentMethod != dbgen.PaymentMethodSTARS {
		t.Fatalf("paid order: %+v, %v", order, err)
	}
	var charge string
	if err := s.Pool.QueryRow(ctx, "SELECT external_id FROM payments WHERE id = $1", payment.ID).Scan(&charge); err != nil || charge != "charge-1" {
		t.Fatalf("charge id was not saved: %q, %v", charge, err)
	}
}

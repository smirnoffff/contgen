//go:build integration

package storage_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"contgen/internal/model"
	"contgen/internal/storage"
	"contgen/internal/storage/dbgen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Each test owns a schema. Neither fixtures nor cleanup touch existing tables.
func emptyTestStorage(t *testing.T) *storage.Storage {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL is required for integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("contgen_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		admin.Close(context.Background())
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		defer admin.Close(cleanupCtx)
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE"); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatal("TEST_DATABASE_URL must be a PostgreSQL URI")
	}
	params := u.Query()
	params.Set("search_path", schema)
	u.RawQuery = params.Encode()
	s, err := storage.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

func testStorage(t *testing.T) *storage.Storage {
	t.Helper()
	s := emptyTestStorage(t)
	ctx := t.Context()
	migrations, err := filepath.Glob("../../migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) == 0 {
		t.Fatal("no schema migrations found")
	}
	for _, path := range migrations {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.Pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("migration %s: %v", path, err)
		}
	}
	return s
}

func createUser(t *testing.T, s *storage.Storage, tgID int64) dbgen.User {
	t.Helper()
	user, err := s.FindOrCreateUser(t.Context(), dbgen.FindOrCreateUserParams{TgID: tgID, FirstName: "Тест", Language: dbgen.AppLanguageRU})
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func amount(t *testing.T, text string) pgtype.Numeric {
	t.Helper()
	var n pgtype.Numeric
	if err := n.Scan(text); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestUserUpsert(t *testing.T) {
	s := testStorage(t)
	u := createUser(t, s, 42)
	loaded, err := s.GetUserByID(t.Context(), u.ID)
	if err != nil || loaded.TgID != 42 {
		t.Fatalf("user lookup: %+v, %v", loaded, err)
	}
	if u.Username != nil || u.Role != dbgen.UserRoleUSER || u.CreatedAt.IsZero() {
		t.Fatalf("unexpected user: %+v", u)
	}
	name := "new_username"
	updated, err := s.FindOrCreateUser(t.Context(), dbgen.FindOrCreateUserParams{TgID: u.TgID, Username: &name, FirstName: "Новое имя", Language: dbgen.AppLanguageEN})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != u.ID || updated.Username == nil || *updated.Username != name || updated.FirstName != "Новое имя" || updated.Language != dbgen.AppLanguageRU {
		t.Fatalf("upsert changed identity or language: %+v", updated)
	}
	updated, err = s.FindOrCreateUser(t.Context(), dbgen.FindOrCreateUserParams{TgID: u.TgID, FirstName: "Без username", Language: dbgen.AppLanguageRU})
	if err != nil || updated.Username != nil {
		t.Fatalf("clear username: %+v, %v", updated, err)
	}
}

func TestDatasetsAndCopy(t *testing.T) {
	s := testStorage(t)
	ctx := t.Context()
	emptyList, err := s.GetAvailableDatasets(ctx, nil)
	if err != nil || emptyList != nil {
		t.Fatalf("empty list: %v, %v", emptyList, err)
	}
	u := createUser(t, s, 1)
	other := createUser(t, s, 2)
	points := make([]model.Point, 1000)
	for i := range points {
		points[i] = model.Point{EntityName: fmt.Sprintf("Ряд %d", i), Value: float64(i) / 10, RecordedAt: 2020 + i%4}
	}
	system, err := s.CreateDatasetWithPoints(ctx, nil, "Системный", points)
	if err != nil {
		t.Fatal(err)
	}
	if !system.IsSystem() || system.CreatedAt.IsZero() {
		t.Fatalf("system dataset: %+v", system)
	}
	rows, err := s.GetDataPointsByDatasetID(ctx, system.ID)
	if err != nil || len(rows) != len(points) {
		t.Fatalf("COPY: %d rows, %v", len(rows), err)
	}
	want := make(map[string]model.Point, len(points))
	for _, p := range points {
		want[p.EntityName] = p
	}
	for i, row := range rows {
		p, ok := want[row.EntityName]
		if !ok || row.ID == 0 || row.DatasetID != system.ID || row.Value != p.Value || row.RecordedAt != p.RecordedAt {
			t.Fatalf("COPY row: %+v", row)
		}
		delete(want, row.EntityName)
		if i > 0 && rows[i-1].RecordedAt > row.RecordedAt {
			t.Fatal("points are not ordered by year")
		}
	}
	private, err := s.CreateDatasetWithPoints(ctx, &u.ID, "Личный", []model.Point{{EntityName: "Дробь", Value: 1.235, RecordedAt: 2024}, {EntityName: "Минус", Value: -12.5, RecordedAt: 2023}})
	if err != nil || private.IsSystem() {
		t.Fatalf("private dataset: %+v, %v", private, err)
	}
	rounded, err := s.GetDataPointsByDatasetID(ctx, private.ID)
	if err != nil || len(rounded) != 2 || rounded[0].Value != -12.5 || rounded[1].Value != 1.24 {
		t.Fatalf("NUMERIC rounding: %+v, %v", rounded, err)
	}
	foreign, err := s.CreateDatasetWithPoints(ctx, &other.ID, "Чужой", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetDatasetForUser(ctx, dbgen.GetDatasetForUserParams{ID: foreign.ID, ViewerID: u.ID}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("foreign dataset accessible: %v", err)
	}
	if found, err := s.GetDatasetForUser(ctx, dbgen.GetDatasetForUserParams{ID: private.ID, ViewerID: u.ID}); err != nil || found.ID != private.ID {
		t.Fatalf("own dataset lookup: %+v, %v", found, err)
	}
	if found, err := s.GetDatasetForUser(ctx, dbgen.GetDatasetForUserParams{ID: system.ID, ViewerID: u.ID}); err != nil || found.ID != system.ID {
		t.Fatalf("public dataset lookup: %+v, %v", found, err)
	}
	mine, err := s.GetAvailableDatasets(ctx, &u.ID)
	if err != nil || len(mine) != 2 || mine[0].ID != private.ID || mine[1].ID != system.ID {
		t.Fatalf("visibility/order: %+v, %v", mine, err)
	}
	public, err := s.GetAvailableDatasets(ctx, nil)
	if err != nil || len(public) != 1 || public[0].ID != system.ID {
		t.Fatalf("system visibility: %+v, %v", public, err)
	}
	empty, err := s.CreateDatasetWithPoints(ctx, nil, "Пустой", nil)
	if err != nil {
		t.Fatal(err)
	}
	missing, err := s.GetDataPointsByDatasetID(ctx, empty.ID)
	if err != nil || missing != nil {
		t.Fatalf("empty points: %+v, %v", missing, err)
	}
	bad := []model.Point{{EntityName: "Нормальная", Value: 1, RecordedAt: 2024}, {EntityName: strings.Repeat("x", 251), Value: 2, RecordedAt: 2024}}
	if _, err := s.CreateDatasetWithPoints(ctx, nil, "Откат", bad); err == nil {
		t.Fatal("invalid COPY succeeded")
	}
	var count int
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM datasets WHERE title = 'Откат'").Scan(&count); err != nil || count != 0 {
		t.Fatalf("dataset was not rolled back: %d, %v", count, err)
	}
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM data_points").Scan(&count); err != nil || count != 1002 {
		t.Fatalf("partial COPY persisted: %d, %v", count, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.CreateDatasetWithPoints(cancelled, nil, "Отмена", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}

func TestMusicVisibilityAndNulls(t *testing.T) {
	s := testStorage(t)
	ctx := t.Context()
	u := createUser(t, s, 1)
	other := createUser(t, s, 2)
	public, err := s.CreateMusicTrack(ctx, dbgen.CreateMusicTrackParams{Title: "Общий", FilePath: "public.mp3"})
	if err != nil || !public.IsSystem() || public.DurationSec != nil {
		t.Fatalf("public music: %+v, %v", public, err)
	}
	duration := 120
	private, err := s.CreateMusicTrack(ctx, dbgen.CreateMusicTrackParams{UserID: &u.ID, Title: "Мой", FilePath: "private.mp3", DurationSec: &duration})
	if err != nil || private.IsSystem() {
		t.Fatalf("private music: %+v, %v", private, err)
	}
	if _, err := s.CreateMusicTrack(ctx, dbgen.CreateMusicTrackParams{UserID: &other.ID, Title: "Чужой", FilePath: "other.mp3"}); err != nil {
		t.Fatal(err)
	}
	inactive, err := s.CreateMusicTrack(ctx, dbgen.CreateMusicTrackParams{Title: "Неактивный", FilePath: "inactive.mp3"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Pool.Exec(ctx, "UPDATE music_tracks SET is_active = false WHERE id = $1", inactive.ID); err != nil {
		t.Fatal(err)
	}
	mine, err := s.GetAvailableMusicTracks(ctx, &u.ID)
	if err != nil || len(mine) != 2 || mine[0].ID != private.ID || mine[1].ID != public.ID {
		t.Fatalf("music visibility/order: %+v, %v", mine, err)
	}
	loaded, err := s.GetMusicTrackByID(ctx, private.ID)
	if err != nil || loaded.DurationSec == nil || *loaded.DurationSec != duration {
		t.Fatalf("duration: %+v, %v", loaded, err)
	}
	publicList, err := s.GetAvailableMusicTracks(ctx, nil)
	if err != nil || len(publicList) != 1 || publicList[0].ID != public.ID {
		t.Fatalf("system music: %+v, %v", publicList, err)
	}
	if _, err := s.GetMusicTrackByID(ctx, -1); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("missing music: %v", err)
	}
}

func TestCartAndPayment(t *testing.T) {
	s := testStorage(t)
	ctx := t.Context()
	u := createUser(t, s, 1)
	cart, err := s.GetOrCreateCart(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cart.PaymentMethod != nil || cart.PromocodeID != nil || cart.DiscountPercentApplied != nil || cart.Status != dbgen.OrderStatusCART {
		t.Fatalf("new cart: %+v", cart)
	}
	again, err := s.GetOrCreateCart(ctx, u.ID)
	if err != nil || again.ID != cart.ID {
		t.Fatalf("cart reuse: %+v, %v", again, err)
	}
	if err := s.MarkOrderPaid(ctx, cart.ID, dbgen.PaymentMethodSTARS, amount(t, "19.99")); err != nil {
		t.Fatal(err)
	}
	var status dbgen.OrderStatus
	var total pgtype.Numeric
	var method dbgen.PaymentMethod
	var updated time.Time
	if err := s.Pool.QueryRow(ctx, "SELECT status, total_amount, payment_method, updated_at FROM orders WHERE id = $1", cart.ID).Scan(&status, &total, &method, &updated); err != nil {
		t.Fatal(err)
	}
	value, err := total.Value()
	if err != nil || value != "19.99" || status != dbgen.OrderStatusPAID || method != dbgen.PaymentMethodSTARS || !updated.After(cart.UpdatedAt) {
		t.Fatalf("payment: %v %v %v %v, %v", status, method, value, updated, err)
	}
	if err := s.MarkOrderPaid(ctx, cart.ID, dbgen.PaymentMethodADS, amount(t, "0")); err == nil {
		t.Fatal("paid order accepted twice")
	}
	if err := s.MarkOrderPaid(ctx, -1, dbgen.PaymentMethodSTARS, amount(t, "1")); err == nil {
		t.Fatal("missing order accepted")
	}
	fresh, err := s.GetOrCreateCart(ctx, u.ID)
	if err != nil || fresh.ID == cart.ID {
		t.Fatalf("new cart after payment: %+v, %v", fresh, err)
	}
}

func TestPromocodeValidationAndRollback(t *testing.T) {
	s := testStorage(t)
	ctx := t.Context()
	u := createUser(t, s, 1)
	cart, err := s.GetOrCreateCart(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	var promoID int64
	if err := s.Pool.QueryRow(ctx, "INSERT INTO promocodes (code, discount_percent) VALUES ('OK', 25) RETURNING id").Scan(&promoID); err != nil {
		t.Fatal(err)
	}
	promo, err := s.GetValidPromocode(ctx, "OK")
	if err != nil || promo.ValidUntil != nil || promo.MaxUses != nil {
		t.Fatalf("unlimited promo: %+v, %v", promo, err)
	}
	if err := s.ApplyPromocode(ctx, cart.ID, u.ID, promoID, 25); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.GetCart(ctx, u.ID)
	if err != nil || loaded.PromocodeID == nil || *loaded.PromocodeID != promoID || loaded.DiscountPercentApplied == nil || *loaded.DiscountPercentApplied != 25 {
		t.Fatalf("applied promo: %+v, %v", loaded, err)
	}
	if err := s.MarkOrderPaid(ctx, cart.ID, dbgen.PaymentMethodPROMO, amount(t, "0")); err != nil {
		t.Fatal(err)
	}
	fresh, err := s.GetOrCreateCart(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyPromocode(ctx, fresh.ID, u.ID, promoID, 99); err == nil {
		t.Fatal("duplicate usage accepted")
	}
	loaded, err = s.GetCart(ctx, u.ID)
	if err != nil || loaded.PromocodeID != nil || loaded.DiscountPercentApplied != nil {
		t.Fatalf("order was not rolled back: %+v, %v", loaded, err)
	}
	promo, err = s.GetValidPromocode(ctx, "OK")
	if err != nil || promo.UsedCount != 1 {
		t.Fatalf("counter was not rolled back: %+v, %v", promo, err)
	}
	var usages int
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM promocode_usages WHERE promocode_id = $1", promoID).Scan(&usages); err != nil || usages != 1 {
		t.Fatalf("usages: %d, %v", usages, err)
	}
	if _, err := s.Pool.Exec(ctx, `INSERT INTO promocodes (code, discount_percent, valid_until, max_uses, used_count, is_active) VALUES
		('EXPIRED', 10, NOW() - INTERVAL '1 day', NULL, 0, true),
		('INACTIVE', 10, NULL, NULL, 0, false),
		('EXHAUSTED', 10, NULL, 1, 1, true),
		('LIMITED', 10, NOW() + INTERVAL '1 day', 2, 1, true)`); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"EXPIRED", "INACTIVE", "EXHAUSTED", "MISSING"} {
		if _, err := s.GetValidPromocode(ctx, code); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("%s: %v", code, err)
		}
	}
	limited, err := s.GetValidPromocode(ctx, "LIMITED")
	if err != nil || limited.ValidUntil == nil || limited.MaxUses == nil || *limited.MaxUses != 2 {
		t.Fatalf("limited promo: %+v, %v", limited, err)
	}
}

func TestRenderJobLifecycle(t *testing.T) {
	s := testStorage(t)
	ctx := t.Context()
	u := createUser(t, s, 1)
	cart, err := s.GetOrCreateCart(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := s.CreateDatasetWithPoints(ctx, nil, "Данные", nil)
	if err != nil {
		t.Fatal(err)
	}
	track, err := s.CreateMusicTrack(ctx, dbgen.CreateMusicTrackParams{Title: "Музыка", FilePath: "track.mp3"})
	if err != nil {
		t.Fatal(err)
	}
	for i, theme := range dbgen.AllRenderThemeValues() {
		var trackID *int64
		if i%2 == 0 {
			trackID = &track.ID
		}
		job, err := s.CreateRenderJob(ctx, dbgen.CreateRenderJobParams{OrderID: cart.ID, DatasetID: dataset.ID, MusicTrackID: trackID, Theme: theme})
		if err != nil {
			t.Fatal(err)
		}
		if !theme.Valid() || job.Status != dbgen.JobStatusPENDING || job.CompletedAt != nil || job.ErrorMessage != nil || job.TelegramFileID != nil {
			t.Fatalf("pending job: %+v", job)
		}
		if i%2 == 0 {
			if err := s.MarkJobCompleted(ctx, job.ID, "tg_file_id"); err != nil {
				t.Fatal(err)
			}
			loaded, err := s.GetJobByID(ctx, job.ID)
			if err != nil || loaded.Status != dbgen.JobStatusCOMPLETED || loaded.CompletedAt == nil || loaded.TelegramFileID == nil || *loaded.TelegramFileID != "tg_file_id" || loaded.MusicTrackID == nil || *loaded.MusicTrackID != track.ID {
				t.Fatalf("completed job: %+v, %v", loaded, err)
			}
		} else {
			message := "ошибка рендера"
			if err := s.MarkJobFailed(ctx, dbgen.MarkJobFailedParams{ID: job.ID, ErrorMessage: &message}); err != nil {
				t.Fatal(err)
			}
			loaded, err := s.GetJobByID(ctx, job.ID)
			if err != nil || loaded.Status != dbgen.JobStatusFAILED || loaded.ErrorMessage == nil || *loaded.ErrorMessage != message || loaded.MusicTrackID != nil {
				t.Fatalf("failed job: %+v, %v", loaded, err)
			}
		}
	}
	if dbgen.RenderTheme("invalid").Valid() {
		t.Fatal("invalid theme accepted by Valid")
	}
	if _, err := s.GetJobByID(ctx, -1); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("missing job: %v", err)
	}
	if err := s.MarkJobCompleted(ctx, -1, "file"); err == nil {
		t.Fatal("missing job completed")
	}
	message := "failure"
	if err := s.MarkJobFailed(ctx, dbgen.MarkJobFailedParams{ID: -1, ErrorMessage: &message}); err != nil {
		t.Fatalf("exec semantics changed: %v", err)
	}
}

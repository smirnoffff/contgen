// Package worker processes paid render jobs serially. PostgreSQL owns the
// queue; an advisory lock ensures that only one laptop worker claims a job.
package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"contgen/internal/model"
	"contgen/internal/renderer"
	"contgen/internal/storage"
	"contgen/internal/storage/dbgen"
	"contgen/internal/telegram"

	"github.com/jackc/pgx/v5"
)

type Options struct {
	PollInterval time.Duration
	JobTimeout   time.Duration
	OutputDir    string
	DataDir      string
	Telegram     *telegram.Client
}

type lease interface {
	RecoverRenderJobs(context.Context) (int64, error)
	ClaimNextRenderJob(context.Context) (dbgen.RenderJob, error)
	GetDatasetByID(context.Context, int64) (dbgen.Dataset, error)
	GetRenderJobDelivery(context.Context, int64) (dbgen.GetRenderJobDeliveryRow, error)
	GetDataPointsByDatasetID(context.Context, int64) ([]dbgen.DataPoint, error)
	GetMusicTrackByID(context.Context, int64) (dbgen.MusicTrack, error)
	GetJobByID(context.Context, int64) (dbgen.RenderJob, error)
	FinishRenderJob(context.Context, int64) (int64, error)
	FailRenderJob(context.Context, dbgen.FailRenderJobParams) (int64, error)
	RequeueRenderJob(context.Context, int64) (int64, error)
	SetJobTelegramFile(context.Context, dbgen.SetJobTelegramFileParams) (int64, error)
	Close()
}

type renderFunc func(context.Context, string, []model.Point, dbgen.RenderTheme, string) error

type Worker struct {
	acquire func(context.Context) (lease, error)
	options Options
	render  renderFunc
}

func New(db *storage.Storage, options Options) (*Worker, error) {
	if db == nil || db.Pool == nil {
		return nil, fmt.Errorf("для очереди нужен PostgreSQL")
	}
	if options.PollInterval == 0 {
		options.PollInterval = 2 * time.Second
	}
	if options.JobTimeout == 0 {
		options.JobTimeout = 10 * time.Minute
	}
	if options.PollInterval < 0 || options.JobTimeout < 0 {
		return nil, fmt.Errorf("интервалы очереди должны быть положительными")
	}
	if options.OutputDir == "" {
		options.OutputDir = filepath.Join(options.DataDir, "renders")
	}
	if options.OutputDir == "" {
		options.OutputDir = filepath.Join("data", "renders")
	}
	if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("каталог рендера: %w", err)
	}
	return &Worker{acquire: func(ctx context.Context) (lease, error) { return db.AcquireWorker(ctx) }, options: options, render: renderer.RenderRacingBar}, nil
}

// Run reconnects after database errors and exits cleanly on cancellation.
func (w *Worker) Run(ctx context.Context) error {
	for ctx.Err() == nil {
		acquireCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		current, err := w.acquire(acquireCtx)
		cancel()
		if err == nil {
			err = w.runLease(ctx, current)
			current.Close()
		}
		if ctx.Err() != nil {
			break
		}
		if err != nil && !errors.Is(err, storage.ErrWorkerRunning) {
			slog.Error("очередь видео: переподключение", "err", err)
		}
		if !pause(ctx, w.options.PollInterval) {
			break
		}
	}
	return nil
}

func (w *Worker) runLease(ctx context.Context, q lease) error {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	recovered, err := q.RecoverRenderJobs(queryCtx)
	cancel()
	if err != nil {
		return err
	}
	if recovered > 0 {
		slog.Info("восстановлены прерванные задания", "count", recovered)
	}
	for ctx.Err() == nil {
		queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		job, err := q.ClaimNextRenderJob(queryCtx)
		cancel()
		if errors.Is(err, pgx.ErrNoRows) {
			if !pause(ctx, w.options.PollInterval) {
				return nil
			}
			continue
		}
		if err != nil {
			return err
		}
		if err := w.process(ctx, q, job); err != nil {
			return err
		}
	}
	return nil
}

type renderInput struct {
	title  string
	points []model.Point
	audio  string
}

func (w *Worker) loadInput(ctx context.Context, q lease, job dbgen.RenderJob) (renderInput, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	dataset, err := q.GetDatasetByID(queryCtx, job.DatasetID)
	if err != nil {
		return renderInput{}, err
	}
	rows, err := q.GetDataPointsByDatasetID(queryCtx, job.DatasetID)
	if err != nil {
		return renderInput{}, err
	}
	input := renderInput{title: dataset.Title, points: make([]model.Point, len(rows))}
	for i, row := range rows {
		input.points[i] = model.Point{EntityName: row.EntityName, Value: row.Value, RecordedAt: row.RecordedAt}
	}
	if job.MusicTrackID != nil {
		track, err := q.GetMusicTrackByID(queryCtx, *job.MusicTrackID)
		if err != nil {
			return renderInput{}, err
		}
		input.audio = track.FilePath
	}
	return input, nil
}

func (w *Worker) process(ctx context.Context, q lease, job dbgen.RenderJob) error {
	input, err := w.loadInput(ctx, q, job)
	if err != nil {
		if ctx.Err() != nil {
			return requeue(q, job.ID)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return fail(ctx, q, job.ID, "Исходные данные или музыка недоступны.")
		}
		return err
	}
	output := filepath.Join(w.options.OutputDir, fmt.Sprintf("job-%d.mp4", job.ID))
	if err := os.Remove(output); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fail(ctx, q, job.ID, "Не удалось подготовить файл видео.")
	}
	renderCtx, cancel := context.WithTimeout(ctx, w.options.JobTimeout)
	defer cancel()
	if err := w.render(renderCtx, input.title, input.points, job.Theme, output); err != nil {
		if ctx.Err() != nil {
			return requeue(q, job.ID)
		}
		slog.Error("рендер видео не выполнен", "job_id", job.ID, "err", err)
		return fail(ctx, q, job.ID, "Не удалось создать видео. Проверьте данные и повторите задание.")
	}
	if input.audio != "" {
		audioPath, err := safeDataPath(w.options.DataDir, input.audio)
		if err != nil {
			return fail(ctx, q, job.ID, "Музыкальный файл недоступен.")
		}
		if err := renderer.MuxAudio(renderCtx, output, audioPath); err != nil {
			return fail(ctx, q, job.ID, "Не удалось добавить музыку в видео.")
		}
	}
	if ctx.Err() != nil {
		return requeue(q, job.ID)
	}
	queryCtx, queryCancel := context.WithTimeout(ctx, 5*time.Second)
	updated, err := q.FinishRenderJob(queryCtx, job.ID)
	queryCancel()
	if err != nil {
		return err
	}
	if updated != 1 {
		return fmt.Errorf("не удалось завершить задание %d: статус изменился", job.ID)
	}
	if w.options.Telegram != nil {
		w.deliver(ctx, q, job.ID, output)
	}
	slog.Info("видео готово", "job_id", job.ID)
	return nil
}

// deliver sends a completed video to the chat that paid for the order. Delivery
// is deliberately best-effort: a Telegram outage must not turn a successful
// render into a failed job. The web screen exposes a retry button when this
// step fails.
func (w *Worker) deliver(ctx context.Context, q lease, jobID int64, output string) {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	delivery, err := q.GetRenderJobDelivery(queryCtx, jobID)
	cancel()
	if err != nil {
		slog.Error("не удалось получить получателя видео", "job_id", jobID, "err", err)
		return
	}
	if delivery.TgID <= 0 {
		return
	}
	sendCtx, sendCancel := context.WithTimeout(ctx, 2*time.Minute)
	fileID, err := w.options.Telegram.SendVideo(sendCtx, delivery.TgID, output, "")
	sendCancel()
	if err != nil {
		slog.Error("автоматическая отправка видео в Telegram не выполнена", "job_id", jobID, "err", err)
		return
	}
	updateCtx, updateCancel := context.WithTimeout(ctx, 5*time.Second)
	_, err = q.SetJobTelegramFile(updateCtx, dbgen.SetJobTelegramFileParams{ID: jobID, TelegramFileID: &fileID})
	updateCancel()
	if err != nil {
		slog.Error("не удалось сохранить Telegram file_id", "job_id", jobID, "err", err)
		return
	}
	slog.Info("видео отправлено в Telegram", "job_id", jobID, "telegram_id", delivery.TgID)
}

func safeDataPath(root, name string) (string, error) {
	if root == "" {
		root = "data"
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(rootAbs, filepath.FromSlash(name)))
	if err != nil || (candidate != rootAbs && !hasPathPrefix(candidate, rootAbs)) {
		return "", fmt.Errorf("путь выходит за пределы DATA_DIR")
	}
	return candidate, nil
}

func hasPathPrefix(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && len(rel) > 2 && rel[:3] != ".."+string(filepath.Separator)
}

func requeue(q lease, id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := q.RequeueRenderJob(ctx, id)
	return err
}

func fail(ctx context.Context, q lease, id int64, message string) error {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	n, err := q.FailRenderJob(queryCtx, dbgen.FailRenderJobParams{ID: id, ErrorMessage: &message})
	if err == nil && n != 1 {
		return fmt.Errorf("не удалось сохранить ошибку задания %d", id)
	}
	return err
}

func pause(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

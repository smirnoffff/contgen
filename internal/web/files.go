package web

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"contgen/internal/service"
	"contgen/internal/storage/dbgen"
	"contgen/internal/web/views"
)

const exampleCSV = "year,entity,value\n2020,Python,34\n2020,Go,12\n2020,Rust,8\n2022,Python,43\n2022,Go,24\n2022,Rust,17\n2024,Python,58\n2024,Go,41\n2024,Rust,32\n2026,Python,74\n2026,Go,65\n2026,Rust,52\n"

func (s *server) exampleCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="contgen-example.csv"`)
	_, _ = io.WriteString(w, exampleCSV)
}

func (s *server) library(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	page, err := s.pageData(r, u)
	if err != nil {
		s.unavailable(w, err)
		return
	}
	d := views.LibraryData{PageData: page}
	datasets, err := s.db.GetAvailableDatasets(r.Context(), &u.ID)
	if err != nil {
		s.unavailable(w, err)
		return
	}
	for _, v := range datasets {
		d.Datasets = append(d.Datasets, views.DatasetItem{ID: v.ID, Title: views.DatasetTitle(v.Title, d.Language), System: v.IsSystem(), Active: true, CanDelete: v.UserID != nil && *v.UserID == u.ID, Detail: v.CreatedAt.Format("02.01.2006")})
	}
	tracks, err := s.db.GetAvailableMusicTracks(r.Context(), &u.ID)
	if err != nil {
		s.unavailable(w, err)
		return
	}
	for _, v := range tracks {
		d.Music = append(d.Music, views.MusicItem{ID: v.ID, Title: v.Title, System: v.IsSystem(), Active: v.IsActive, CanDelete: v.UserID != nil && *v.UserID == u.ID, Detail: durationLabel(v.DurationSec), URL: fmt.Sprintf("/music/%d/audio", v.ID)})
	}
	s.page(w, r, views.Library(d))
}

func uploadTitle(r *http.Request) (string, error) {
	title := strings.TrimSpace(r.FormValue("title"))
	if !utf8.ValidString(title) || utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > 120 {
		return "", errors.New("Название должно содержать от 1 до 120 символов")
	}
	return title, nil
}

func (s *server) uploadDataset(w http.ResponseWriter, r *http.Request) {
	admin := isAdminPath(r)
	u, ok := s.requireUser(w, r, admin)
	if !ok {
		return
	}
	back := "/library"
	if admin {
		back = "/admin?tab=library"
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if err := r.ParseMultipartForm(256 << 10); err != nil {
		redirectMessage(w, r, back, "", "CSV должен быть не больше 2 МБ")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	title, err := uploadTitle(r)
	if err != nil {
		redirectMessage(w, r, back, "", err.Error())
		return
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		redirectMessage(w, r, back, "", "Выбери CSV-файл")
		return
	}
	defer f.Close()
	points, err := service.ParseCSV(f)
	if err != nil {
		redirectMessage(w, r, back, "", shortMessage(err.Error()))
		return
	}
	var owner *int64
	if !admin {
		owner = &u.ID
	}
	dataset, err := s.db.CreateDatasetWithPoints(r.Context(), owner, title, points)
	if err != nil {
		s.operationError(w, r, back, err)
		return
	}
	if !admin {
		back = "/studio?dataset_id=" + strconv.FormatInt(dataset.ID, 10)
	}
	redirectMessage(w, r, back, "Датасет загружен", "")
}

func (s *server) uploadMusic(w http.ResponseWriter, r *http.Request) {
	admin := isAdminPath(r)
	u, ok := s.requireUser(w, r, admin)
	if !ok {
		return
	}
	back := "/library"
	if admin {
		back = "/admin?tab=library"
	}
	r.Body = http.MaxBytesReader(w, r.Body, 21<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		redirectMessage(w, r, back, "", "Аудиофайл должен быть не больше 20 МБ")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	title, err := uploadTitle(r)
	if err != nil {
		redirectMessage(w, r, back, "", err.Error())
		return
	}
	f, header, err := r.FormFile("file")
	if err != nil {
		redirectMessage(w, r, back, "", "Выбери аудиофайл")
		return
	}
	defer f.Close()
	var owner *int64
	if !admin {
		owner = &u.ID
	}
	audio, err := service.SaveAudio(r.Context(), s.opts.DataDir, header.Filename, f)
	if err != nil {
		redirectMessage(w, r, back, "", err.Error())
		return
	}
	_, err = s.db.CreateMusicTrack(r.Context(), dbgen.CreateMusicTrackParams{UserID: owner, Title: title, FilePath: audio.Path, DurationSec: &audio.DurationSec})
	if err != nil {
		if path, e := service.SafePath(s.opts.DataDir, audio.Path); e == nil {
			_ = os.Remove(path)
		}
		s.operationError(w, r, back, err)
		return
	}
	redirectMessage(w, r, back, "Музыка добавлена", "")
}

func (s *server) deleteDataset(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	n, err := s.db.DeleteOwnedDataset(r.Context(), dbgen.DeleteOwnedDatasetParams{ID: id, UserID: &u.ID})
	if err != nil {
		s.operationError(w, r, "/library", err)
		return
	}
	if n == 0 {
		redirectMessage(w, r, "/library", "", "Датасет используется в заказе и его нельзя удалить.")
		return
	}
	redirectMessage(w, r, "/library", "Датасет удалён", "")
}

func (s *server) archiveMusic(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	n, err := s.db.ArchiveOwnedMusic(r.Context(), dbgen.ArchiveOwnedMusicParams{ID: id, UserID: &u.ID})
	if err != nil {
		s.operationError(w, r, "/library", err)
		return
	}
	if n == 0 {
		http.NotFound(w, r)
		return
	}
	redirectMessage(w, r, "/library", "Трек убран из библиотеки", "")
}

func (s *server) audio(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r, false)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var track dbgen.MusicTrack
	if u.Role == dbgen.UserRoleADMIN {
		track, err = s.db.GetMusicTrackByID(r.Context(), id)
	} else {
		track, err = s.db.GetMusicForUser(r.Context(), dbgen.GetMusicForUserParams{ID: id, ViewerID: u.ID})
	}
	if err != nil {
		http.NotFound(w, r)
		return
	}
	s.serveFile(w, r, track.FilePath, false)
}

func (s *server) serveFile(w http.ResponseWriter, r *http.Request, key string, download bool) {
	path, err := service.SafePath(s.opts.DataDir, key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	if download {
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": "contgen-" + r.PathValue("id") + filepath.Ext(key)}))
	}
	http.ServeContent(w, r, filepath.Base(key), info.ModTime(), f)
}

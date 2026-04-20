package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"seriestracker/internal/models"
)

type SeriesHandler struct {
	DB *sql.DB
}

func (h *SeriesHandler) List(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, current_episode, total_episodes, image_path, created_at, updated_at
		FROM series
	`
	args := []any{}

	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		query += ` WHERE LOWER(name) LIKE LOWER(?)`
		args = append(args, "%"+q+"%")
	}

	query += ` ORDER BY id DESC`

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		log.Printf("list query: %v", err)
		writeError(w, http.StatusInternalServerError, "error al consultar series")
		return
	}
	defer rows.Close()

	list := []models.Series{}
	for rows.Next() {
		var s models.Series
		if err := rows.Scan(&s.ID, &s.Name, &s.CurrentEpisode, &s.TotalEpisodes,
			&s.ImagePath, &s.CreatedAt, &s.UpdatedAt); err != nil {
			log.Printf("list scan: %v", err)
			writeError(w, http.StatusInternalServerError, "error al leer serie")
			return
		}
		list = append(list, s)
	}

	writeJSON(w, http.StatusOK, list)
}

func (h *SeriesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	s, err := h.findByID(id)
	if errors.Is(err, models.ErrNotFound) {
		writeError(w, http.StatusNotFound, "serie no encontrada")
		return
	}
	if err != nil {
		log.Printf("get series: %v", err)
		writeError(w, http.StatusInternalServerError, "error al consultar serie")
		return
	}

	writeJSON(w, http.StatusOK, s)
}

func (h *SeriesHandler) Create(w http.ResponseWriter, r *http.Request) {
	
	if r.Header.Get("Content-Type") != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "content-type debe ser application/json")
		return
	}
	
	var input models.SeriesInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("create decode: %v", err)
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}

	if errs := input.Validate(); errs != nil {
		log.Printf("create validation: %v", errs)
		writeValidationError(w, errs)
		return
	}

	log.Printf("create insert: name=%q current=%d total=%d", input.Name, input.CurrentEpisode, input.TotalEpisodes)

	res, err := h.DB.Exec(`
		INSERT INTO series (name, current_episode, total_episodes)
		VALUES (?, ?, ?)
	`, input.Name, input.CurrentEpisode, input.TotalEpisodes)
	if err != nil {
		log.Printf("create insert error: %v", err)
		writeError(w, http.StatusInternalServerError, "error al crear serie")
		return
	}

	id, _ := res.LastInsertId()
	log.Printf("create inserted id=%d", id)

	s, err := h.findByID(id)
	if err != nil {
		log.Printf("create findByID error: %v", err)
		writeError(w, http.StatusInternalServerError, "error al crear serie")
		return
	}

	writeJSON(w, http.StatusCreated, s)
}

func (h *SeriesHandler) Update(w http.ResponseWriter, r *http.Request) {

	if r.Header.Get("Content-Type") != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "content-type debe ser application/json")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var input models.SeriesInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}

	if errs := input.Validate(); errs != nil {
		writeValidationError(w, errs)
		return
	}

	res, err := h.DB.Exec(`
		UPDATE series
		SET name = ?, current_episode = ?, total_episodes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, input.Name, input.CurrentEpisode, input.TotalEpisodes, id)
	if err != nil {
		log.Printf("update: %v", err)
		writeError(w, http.StatusInternalServerError, "error al actualizar serie")
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeError(w, http.StatusNotFound, "serie no encontrada")
		return
	}

	s, err := h.findByID(id)
	if err != nil {
		log.Printf("update findByID: %v", err)
		writeError(w, http.StatusInternalServerError, "error al leer serie")
		return
	}

	writeJSON(w, http.StatusOK, s)
}

func (h *SeriesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	res, err := h.DB.Exec(`DELETE FROM series WHERE id = ?`, id)
	if err != nil {
		log.Printf("delete: %v", err)
		writeError(w, http.StatusInternalServerError, "error al eliminar serie")
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeError(w, http.StatusNotFound, "serie no encontrada")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SeriesHandler) findByID(id int64) (*models.Series, error) {
	var s models.Series
	err := h.DB.QueryRow(`
		SELECT id, name, current_episode, total_episodes, image_path, created_at, updated_at
		FROM series
		WHERE id = ?
	`, id).Scan(&s.ID, &s.Name, &s.CurrentEpisode, &s.TotalEpisodes,
		&s.ImagePath, &s.CreatedAt, &s.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}
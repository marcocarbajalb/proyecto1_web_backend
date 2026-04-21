package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"seriestracker/internal/models"
)

type RatingHandler struct {
	DB *sql.DB
}

func (h *RatingHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	if err := seriesExists(h.DB, id); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			writeError(w, http.StatusNotFound, "serie no encontrada")
			return
		}
		log.Printf("rating get check: %v", err)
		writeError(w, http.StatusInternalServerError, "error al verificar serie")
		return
	}

	var rating models.Rating
	err = h.DB.QueryRow(`
		SELECT series_id, rating, created_at, updated_at
		FROM ratings
		WHERE series_id = ?
	`, id).Scan(&rating.SeriesID, &rating.Rating, &rating.CreatedAt, &rating.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "la serie no tiene rating")
		return
	}
	if err != nil {
		log.Printf("rating get: %v", err)
		writeError(w, http.StatusInternalServerError, "error al obtener rating")
		return
	}

	writeJSON(w, http.StatusOK, rating)
}

func (h *RatingHandler) Set(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "content-type debe ser application/json")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	if err := seriesExists(h.DB, id); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			writeError(w, http.StatusNotFound, "serie no encontrada")
			return
		}
		log.Printf("rating set check: %v", err)
		writeError(w, http.StatusInternalServerError, "error al verificar serie")
		return
	}

	var input models.RatingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}

	if errs := input.Validate(); errs != nil {
		writeValidationError(w, errs)
		return
	}

	_, err = h.DB.Exec(`
		INSERT INTO ratings (series_id, rating) VALUES (?, ?)
		ON CONFLICT(series_id) DO UPDATE SET
			rating = excluded.rating,
			updated_at = CURRENT_TIMESTAMP
	`, id, input.Rating)
	if err != nil {
		log.Printf("rating set: %v", err)
		writeError(w, http.StatusInternalServerError, "error al guardar rating")
		return
	}

	var rating models.Rating
	if err := h.DB.QueryRow(`
		SELECT series_id, rating, created_at, updated_at
		FROM ratings WHERE series_id = ?
	`, id).Scan(&rating.SeriesID, &rating.Rating, &rating.CreatedAt, &rating.UpdatedAt); err != nil {
		log.Printf("rating set read: %v", err)
		writeError(w, http.StatusInternalServerError, "error al leer rating")
		return
	}

	writeJSON(w, http.StatusOK, rating)
}

func (h *RatingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	res, err := h.DB.Exec(`DELETE FROM ratings WHERE series_id = ?`, id)
	if err != nil {
		log.Printf("rating delete: %v", err)
		writeError(w, http.StatusInternalServerError, "error al eliminar rating")
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeError(w, http.StatusNotFound, "la serie no tiene rating")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
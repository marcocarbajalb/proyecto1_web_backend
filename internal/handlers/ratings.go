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
		writeError(w, http.StatusBadRequest, "El ID proporcionado no es válido.")
		return
	}

	if err := seriesExists(h.DB, id); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			writeError(w, http.StatusNotFound, "No se encontró una serie con ese ID.")
			return
		}
		log.Printf("rating get check: %v", err)
		writeError(w, http.StatusInternalServerError, "No se pudo verificar la serie.")
		return
	}

	var rating models.Rating
	err = h.DB.QueryRow(`
		SELECT series_id, rating, created_at, updated_at
		FROM ratings
		WHERE series_id = ?
	`, id).Scan(&rating.SeriesID, &rating.Rating, &rating.CreatedAt, &rating.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Esta serie todavía no tiene rating asignado.")
		return
	}
	if err != nil {
		log.Printf("rating get: %v", err)
		writeError(w, http.StatusInternalServerError, "No se pudo obtener el rating de la serie.")
		return
	}

	writeJSON(w, http.StatusOK, rating)
}

func (h *RatingHandler) Set(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "El Content-Type debe ser application/json.")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "El ID proporcionado no es válido.")
		return
	}

	if err := seriesExists(h.DB, id); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			writeError(w, http.StatusNotFound, "No se encontró una serie con ese ID.")
			return
		}
		log.Printf("rating set check: %v", err)
		writeError(w, http.StatusInternalServerError, "No se pudo verificar la serie.")
		return
	}

	var input models.RatingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "El cuerpo de la petición no es un JSON válido.")
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
		writeError(w, http.StatusInternalServerError, "No se pudo guardar el rating.")
		return
	}

	var rating models.Rating
	if err := h.DB.QueryRow(`
		SELECT series_id, rating, created_at, updated_at
		FROM ratings WHERE series_id = ?
	`, id).Scan(&rating.SeriesID, &rating.Rating, &rating.CreatedAt, &rating.UpdatedAt); err != nil {
		log.Printf("rating set read: %v", err)
		writeError(w, http.StatusInternalServerError, "No se pudo leer el rating guardado.")
		return
	}

	writeJSON(w, http.StatusOK, rating)
}

func (h *RatingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "El ID proporcionado no es válido.")
		return
	}

	res, err := h.DB.Exec(`DELETE FROM ratings WHERE series_id = ?`, id)
	if err != nil {
		log.Printf("rating delete: %v", err)
		writeError(w, http.StatusInternalServerError, "No se pudo eliminar el rating.")
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeError(w, http.StatusNotFound, "Esta serie no tiene rating asignado.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
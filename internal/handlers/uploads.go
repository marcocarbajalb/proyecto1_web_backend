package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"seriestracker/internal/models"
)

const maxImageSize = 1 << 20

var uploadsDir = getUploadsDir()

func getUploadsDir() string {
	if v := os.Getenv("UPLOADS_DIR"); v != "" {
		return v
	}
	return "uploads"
}

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type UploadHandler struct {
	DB *sql.DB
}

func (h *UploadHandler) UploadSeriesImage(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("upload check: %v", err)
		writeError(w, http.StatusInternalServerError, "error al verificar serie")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImageSize)
	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "el archivo supera 1MB")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "falta el campo 'image'")
		return
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		log.Printf("upload detect: %v", err)
		writeError(w, http.StatusInternalServerError, "error al leer archivo")
		return
	}
	contentType := http.DetectContentType(buf[:n])

	ext, ok := allowedImageTypes[contentType]
	if !ok {
		writeError(w, http.StatusUnsupportedMediaType, "solo se aceptan jpg, png o webp")
		return
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		log.Printf("upload seek: %v", err)
		writeError(w, http.StatusInternalServerError, "error al procesar archivo")
		return
	}

	filename := randomFilename() + ext
	destPath := filepath.Join(uploadsDir, filename)
	publicPath := "/uploads/" + filename

	dest, err := os.Create(destPath)
	if err != nil {
		log.Printf("upload create: %v", err)
		writeError(w, http.StatusInternalServerError, "error al guardar archivo")
		return
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		log.Printf("upload copy: %v", err)
		os.Remove(destPath)
		writeError(w, http.StatusInternalServerError, "error al guardar archivo")
		return
	}

	var oldImage sql.NullString
	if err := h.DB.QueryRow(`SELECT image_path FROM series WHERE id = ?`, id).Scan(&oldImage); err == nil && oldImage.Valid {
		os.Remove("." + oldImage.String)
	}

	if _, err := h.DB.Exec(
		`UPDATE series SET image_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		publicPath, id,
	); err != nil {
		log.Printf("upload update db: %v", err)
		os.Remove(destPath)
		writeError(w, http.StatusInternalServerError, "error al actualizar serie")
		return
	}

	log.Printf("upload ok: serie=%d archivo=%s tamaño=%d", id, filename, header.Size)

	writeJSON(w, http.StatusOK, map[string]string{
		"image_path": publicPath,
	})
}

func seriesExists(db *sql.DB, id int64) error {
	var exists int
	err := db.QueryRow(`SELECT 1 FROM series WHERE id = ?`, id).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return models.ErrNotFound
	}
	return err
}

func randomFilename() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "fallback"
	}
	return hex.EncodeToString(b)
}

func sanitizeStaticPath(requested string) (string, bool) {
	requested = strings.TrimPrefix(requested, "/uploads/")
	clean := filepath.Base(requested)
	if clean == "." || clean == "/" || strings.Contains(clean, "..") {
		return "", false
	}
	return filepath.Join(uploadsDir, clean), true
}

func ServeUpload(w http.ResponseWriter, r *http.Request) {
	path, ok := sanitizeStaticPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusBadRequest, "ruta inválida")
		return
	}

	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}

	http.ServeFile(w, r, path)
}
package models

import (
	"errors"
	"strings"
	"time"
)

type Series struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	CurrentEpisode int       `json:"current_episode"`
	TotalEpisodes  int       `json:"total_episodes"`
	ImagePath      *string   `json:"image_path"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SeriesInput struct {
	Name           string `json:"name"`
	CurrentEpisode int    `json:"current_episode"`
	TotalEpisodes  int    `json:"total_episodes"`
}

func (s *SeriesInput) Validate() map[string]string {
	errs := map[string]string{}

	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		errs["name"] = "el nombre es obligatorio"
	} else if len(s.Name) > 200 {
		errs["name"] = "el nombre no puede superar 200 caracteres"
	}

	if s.TotalEpisodes < 1 {
		errs["total_episodes"] = "debe ser al menos 1"
	}
	if s.CurrentEpisode < 0 {
		errs["current_episode"] = "no puede ser negativo"
	}
	if s.CurrentEpisode > s.TotalEpisodes {
		errs["current_episode"] = "no puede ser mayor al total de episodios"
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

var ErrNotFound = errors.New("not found")

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type PaginatedSeries struct {
	Data       []Series   `json:"data"`
	Pagination Pagination `json:"pagination"`
}
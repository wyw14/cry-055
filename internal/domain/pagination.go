package domain

import (
	"strings"
	"time"
)

type PageRequest struct {
	Page    int
	Size    int
	Sort    string
	Desc    bool
	Filters map[string]string
	AsOf    time.Time
}

func (p PageRequest) Normalize(allowedSort, allowedFilters map[string]struct{}) (PageRequest, error) {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Size <= 0 {
		p.Size = 20
	}
	if p.Size > 100 {
		p.Size = 100
	}
	if p.Sort == "" {
		p.Sort = "created_at"
	}
	if _, ok := allowedSort[p.Sort]; !ok {
		return PageRequest{}, NewValidationError("sort", "unsupported sort field")
	}
	clean := make(map[string]string, len(p.Filters))
	for key, value := range p.Filters {
		if _, ok := allowedFilters[key]; !ok {
			return PageRequest{}, NewValidationError("filter", "unsupported filter: "+key)
		}
		if value = strings.TrimSpace(value); value != "" {
			clean[key] = value
		}
	}
	p.Filters = clean
	return p, nil
}

type Page[T any] struct {
	Items []T `json:"items"`
	Page  int `json:"page"`
	Size  int `json:"size"`
	Total int `json:"total"`
}

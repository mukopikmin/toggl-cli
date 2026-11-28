package model

import (
	"slices"
	"sort"
)

type Project struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Alias        string   `json:"alias,omitempty"`
	DisplayName  string   `json:"displayName"`
	Active       bool     `json:"active"`
	Hidden       bool     `json:"hidden"`
	DisplayOrder *float64 `json:"displayOrder,omitempty"`
}

func (p Project) WithSettings(settings ProjectSettings) Project {
	p.Alias = settings.DisplayName
	p.DisplayName = p.Alias
	if p.DisplayName == "" {
		p.DisplayName = p.Name
	}
	p.Hidden = settings.Hidden
	p.DisplayOrder = settings.DisplayOrder
	return p
}

func SortedProjects(projects []Project) []Project {
	r := slices.Clone(projects)
	sort.SliceStable(r, func(i, j int) bool {
		a, b := r[i].DisplayOrder, r[j].DisplayOrder
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		return *a < *b
	})
	return r
}

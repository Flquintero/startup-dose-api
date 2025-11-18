package models

import "time"

// Company represents a company entity from the database
type Company struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description"`
	Excerpt     string     `json:"excerpt"`
	Appeal      string     `json:"appeal"`
	Website     string     `json:"website"`
	CoverImage  string     `json:"cover_image"`
	Twitter     *string    `json:"twitter"`
	LinkedIn    *string    `json:"linkedin"`
	Facebook    *string    `json:"facebook"`
	Instagram   *string    `json:"instagram"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

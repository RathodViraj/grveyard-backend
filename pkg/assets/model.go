package assets

import "time"

type Asset struct {
	ID           int64     `json:"id"`
	StartupID    int64     `json:"startup_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	AssetType    string    `json:"asset_type"`
	ImageURL     string    `json:"image_url"`
	Price        float64   `json:"price"`
	IsNegotiable bool      `json:"is_negotiable"`
	IsSold       bool      `json:"is_sold"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

type AssetList struct {
	Items []Asset `json:"items"`
	Total int64   `json:"total"`
	Page  int     `json:"page"`
	Limit int     `json:"limit"`
}

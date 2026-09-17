package dto

import "time"

type CreateOrgRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type AddMemberRequest struct {
	UserID string `json:"user_id"`
}

type OrgResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

type SelectOrgResponse struct {
	Token     string      `json:"token"`
	TokenType string      `json:"token_type"`
	Org       OrgResponse `json:"organization"`
}

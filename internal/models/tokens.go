package models

import "github.com/golang-jwt/jwt/v5"

// not entity
type JWTAccess struct {
	jwt.RegisteredClaims
	Sign    string `json:"sign"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	TokenID string `json:"token_id"`
}

// not entity
type JWTRefresh struct {
	jwt.RegisteredClaims
	Sign               string `json:"sign"`
	Type               string `json:"type"`
	Role               string `json:"role"`
	TokenID            string `json:"token_id"`
	VersionCredentials int    `json:"version_credentials"`
}
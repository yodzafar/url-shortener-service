package domain

type TokenClaims struct {
	UserID int64
	Role   Role
}

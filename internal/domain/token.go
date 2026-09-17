package domain

type TokenClaims struct {
	UserID int64
	Role   Role
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

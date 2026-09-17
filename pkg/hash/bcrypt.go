package hash

import "golang.org/x/crypto/bcrypt"

type Bcrypt struct {
	cost int
}

func NewBcrypt(cost int) *Bcrypt {
	return &Bcrypt{cost: cost}
}

func (b *Bcrypt) Hash(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	if err != nil {
		return "", err
	}

	return string(h), nil
}

func (b *Bcrypt) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

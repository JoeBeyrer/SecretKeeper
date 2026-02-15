package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(plain string) ([]byte, error) {
    return bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
}

func CheckPasswordHash(hash []byte, plain string) error {
    return bcrypt.CompareHashAndPassword(hash, []byte(plain))
}
package auth

import "golang.org/x/crypto/bcrypt"

var BcryptCost = bcrypt.DefaultCost

func HashPassword(plain string) ([]byte, error) {
    return bcrypt.GenerateFromPassword([]byte(plain), BcryptCost)
}

func CheckPasswordHash(hash []byte, plain string) error {
    return bcrypt.CompareHashAndPassword(hash, []byte(plain))
}
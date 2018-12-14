package main

import (
	"io/ioutil"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func tokenForUser() {
	privKeyPath := "../keys/app.rsa"

	signBytes, err := ioutil.ReadFile(privKeyPath)
	if err != nil {
		log.Panic(err)
	}

	signKey, err := jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	if err != nil {
		log.Panic(err)
	}

	claims := jwt.MapClaims{
		"aud":     "catchcath",
		"iss":     "login.catchcatch.club",
		"jti":     "player@gmail.com",
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(time.Minute).Unix(),
		"nbf":     time.Now().Unix(),
		"email":   "player@gmail.com",
		"name":    "Joao das Coves",
		"picture": "https://www.xxx.com/img.jpg",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(signKey)
	if err != nil {
		log.Panic(err)
	}
	log.Println("signed key:", signed)
}

func validateToken(signed string) bool {
	pubKeyPath := "../keys/app.rsa.pub"

	verifyBytes, err := ioutil.ReadFile(pubKeyPath)
	if err != nil {
		log.Panic(err)
	}

	verifyKey, err := jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
	if err != nil {
		log.Panic(err)
	}

	token, err := jwt.Parse(signed, func(token *jwt.Token) (interface{}, error) {
		return verifyKey, nil
	})
	if err != nil {
		log.Panic(err)
	}

	log.Println(token.Claims, token.Valid)

	return token.Valid
}

func main() {
	tokenForUser()

	validateToken("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJjYXRjaGNhdGgiLCJlbWFpbCI6InBsYXllckBnbWFpbC5jb20iLCJleHAiOjE1NDM2MTA4NDMsImlhdCI6MTU0MzYxMDc4MywiaXNzIjoibG9naW4uY2F0Y2hjYXRjaC5jbHViIiwianRpIjoicGxheWVyQGdtYWlsLmNvbSIsIm5hbWUiOiJKb2FvIGRhcyBDb3ZlcyIsIm5iZiI6MTU0MzYxMDc4MywicGljdHVyZSI6Imh0dHBzOi8vd3d3Lnh4eC5jb20vaW1nLmpwZyJ9.YyC28FMUCcTgMQ98BOfQWfglObcUZLShZz6p9mEzxCm5pFVVSWzS_Xbj7j2MRv2RbJaXx8mQAEkpFT0ckURUhA")
}

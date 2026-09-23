package logic

import "web_app/07-pkg/jwt"

func VerifyToken(token string) (*jwt.MyClaims, error) {
	return jwt.ParseToken(token)
}

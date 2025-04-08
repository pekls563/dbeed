package jwt_op

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"log"
	"time"
)

const (
	TokenExpired     = "Token已过期"
	TokenNotValidYet = "Token不再有效"
	TokenMalformed   = "Token非法"
	TokenInvalid     = "Token无效"
)

type CustonClaims struct {
	jwt.StandardClaims
	ID          int32
	NickName    string
	AuthorityId int32
}

type JWT struct {
	SigninKey []byte
}

func NewJWT() *JWT {
	//return &JWT{SigninKey: []byte(conf.AppConf.JWTConfig.SingingKey)}
	return &JWT{SigninKey: []byte("4524532816")}
}

func (j *JWT) GenerateJWT(claims CustonClaims) (string, error) {
	//生成token对象
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	//j.SigninKey是相同的，但由于claims可能不同，得到的token可能不同，进而得到的tokenStr(真正的token）可能不同
	//得到token对象对应的token字符串
	tokenStr, err := token.SignedString(j.SigninKey)
	if err != nil {
		//log.Logger.Error("生成JWT错误:" + err.Error())
		log.Println("生成JWT错误:" + err.Error())

		return "", err
	}
	return tokenStr, nil
}

func (j *JWT) ParseToken(tokenStr string) (*CustonClaims, error) {

	//利用tokenStr(token对应的字符串）解密出token对象
	token, err := jwt.ParseWithClaims(tokenStr, &CustonClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.SigninKey, nil
	})
	if err != nil {
		if result, ok := err.(jwt.ValidationError); ok {
			if result.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, errors.New(TokenMalformed)
			} else if result.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, errors.New(TokenExpired)
			} else if result.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, errors.New(TokenNotValidYet)
			} else {
				return nil, errors.New(TokenInvalid)
			}
		}
	}
	if token != nil {
		if claims, ok := token.Claims.(*CustonClaims); ok && token.Valid {
			return claims, nil
		}
		return nil, errors.New(TokenInvalid)
	} else {
		return nil, errors.New(TokenInvalid)
	}
}

func (j *JWT) RefreshToken(tokenStr string) (string, error) {

	jwt.TimeFunc = func() time.Time {
		return time.Unix(0, 0)
	}

	token, err := jwt.ParseWithClaims(tokenStr, &CustonClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.SigninKey, nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(*CustonClaims); ok && token.Valid {
		jwt.TimeFunc = time.Now
		claims.StandardClaims.ExpiresAt = time.Now().Add(7 * 24 * time.Hour).Unix()
		return j.GenerateJWT(*claims)
	}
	return "", errors.New(TokenInvalid)
}

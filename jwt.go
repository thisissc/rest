package rest

import (
	"fmt"
	"log"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/thisissc/crypto"
)

type myClaims struct {
	jwt.StandardClaims

	Uid    string `json:"uid,omitempty"`
	RoleId string `json:"rid",omitempty"`
}

func GenTokenWithExpire(userid, roleid string, expire int64) string {
	mySigningKey := []byte(globalAppConfig.JWTTokenSecret)

	nowTS := time.Now().Unix()
	claims := &myClaims{
		Uid:    userid,
		RoleId: roleid,
	}
	claims.Subject = userid                 // FIXME: compatible with old token parser
	claims.IssuedAt = nowTS - 180           // XXX: for 3 minutes error
	claims.ExpiresAt = nowTS + expire + 180 // XXX: for 3 minutes error

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(mySigningKey)
	if err != nil {
		log.Println(err)
	}

	result, _ := crypto.NaclEncrypt([]byte(globalAppConfig.AESSecret), ss[len(globalAppConfig.JWTTokenPrefix):])

	return result
}

func GetAuthToken(req *http.Request) string {
	token := req.Header.Get("Authorization") // Authorization: Bearer <token>
	if len(token) > 7 {
		return token[7:]
	} else {
		return ""
	}
}

func ParseToken(tokenStr string) (string, string, bool) {
	origData, err := crypto.NaclDecrypt([]byte(globalAppConfig.AESSecret), tokenStr)
	if err != nil {
		log.Println(err)
	} else {
		origData = fmt.Sprintf("%s%s", globalAppConfig.JWTTokenPrefix, origData)

		token, err := jwt.ParseWithClaims(origData, &myClaims{},
			func(token *jwt.Token) (interface{}, error) {
				return []byte(globalAppConfig.JWTTokenSecret), nil
			})

		if err == nil && token.Valid {
			claims := token.Claims.(*myClaims)

			// FIXME: compatible with old token
			uid := claims.Uid
			if len(uid) == 0 {
				uid = claims.Subject
			}
			rid := claims.RoleId

			return uid, rid, true
		}
	}

	return "", "", false
}

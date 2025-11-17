// package auth

// import (
//     "crypto/rsa"
//     "errors"
//     "io/ioutil"

//     "github.com/golang-jwt/jwt/v5"
// )

// type JWTValidator struct {
//     key *rsa.PublicKey
// }

// func NewJWTValidator(pubPath string) (*JWTValidator, error) {
//     b, err := ioutil.ReadFile(pubPath)
//     if err != nil {
//         return nil, err
//     }
//     pub, err := jwt.ParseRSAPublicKeyFromPEM(b)
//     if err != nil {
//         return nil, err
//     }
//     return &JWTValidator{key: pub}, nil
// }

// // Validate returns subject (user id) if token valid
// func (j *JWTValidator) Validate(tokenString string) (string, error) {
//     t, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
//         if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
//             return nil, errors.New("invalid signing method")
//         }
//         return j.key, nil
//     })
//     if err != nil {
//         return "", err
//     }
//     if !t.Valid {
//         return "", errors.New("invalid token")
//     }
//     claims, ok := t.Claims.(jwt.MapClaims)
//     if !ok {
//         return "", errors.New("invalid claims")
//     }
//     sub, _ := claims["sub"].(string)
//     if sub == "" {
//         return "", errors.New("sub (user id) missing in token")
//     }
//     return sub, nil
// }

package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type JWTValidator struct {
	pub *rsa.PublicKey
}

func NewJWTValidator(path string) (*JWTValidator, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("failed to decode public key")
	}
	pubIfc, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := pubIfc.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not rsa public key")
	}
	return &JWTValidator{pub: pub}, nil
}

func (j *JWTValidator) Validate(tokenStr string) (string, error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return j.pub, nil
	}, jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		return "", err
	}
	if claims, ok := tok.Claims.(jwt.MapClaims); ok && tok.Valid {
		if sub, ok := claims["sub"].(string); ok {
			return sub, nil
		}
		if userID, ok := claims["user_id"].(string); ok {
			return userID, nil
		}
	}
	return "", errors.New("invalid token")
}

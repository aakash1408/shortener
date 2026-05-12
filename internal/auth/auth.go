// Header paylod Signature 
//Header = Algorithm that will be used to sign 
// Payload = Claims = userid, username, expirytime
// signature = signed of payload nad jwt_secret


package auth

import(
	"fmt"
	"time"
	
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct{
	UserID string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error){
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil{
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

func CheckPassword(password, hash string) bool{
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateToken(userID, username, secret string)(string, error){
	claims := Claims{
		UserID: userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil{
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return signed, nil
}

func ValidateToken(tokenStr, secret string) (*Claims, error){
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error){
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok{
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret),nil
	})

	if err != nil{
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid{
		return nil, fmt.Errorf("token is not valid")
	}

	return claims, nil

}
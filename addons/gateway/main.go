package main

import (
	"crypto/rsa"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/scrypt"
)

type AuthReq struct {
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	AccessToken  string `json:"accessToken,omitempty"`
}

var (
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
	clientCreds map[string]string
)

func issueAccessToken(req AuthReq) (*string, error) {

	// create token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": req.ClientId,
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	accessToken, err := token.SignedString(privateKey)
	if err != nil {
		return nil, err
	}

	return &accessToken, err

}

func loadKeys(privateKeyPath string, publicKeyPath string) error {
	privateKey_file, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return err
	}

	// Parse RSA private key
	privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(privateKey_file)
	if err != nil {
		return err
	}

	// read RSA public key from file
	publicKey_file, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return err
	}

	// Parse RSA public key
	publicKey, err = jwt.ParseRSAPublicKeyFromPEM(publicKey_file)
	if err != nil {
		return err
	}

	return nil
}

func parseClients(clients string) error {
	// parse clients of form c1,c2,c3 where cX = clientId:clientSecret
	clientCreds = make(map[string]string, 0)

	splitClients := strings.Split(clients, "&")

	for _, c := range splitClients {
		creds := strings.Split(c, ":")
		if len(creds) != 2 {
			return fmt.Errorf("client creds formatted improperly")
		}
		clientCreds[creds[0]] = creds[1]
	}
	return nil
}

func validateClientCredentials(authReq AuthReq) error {
	// base 64 decode the request secret
	base64DecodedSecret, err := base64.URLEncoding.DecodeString(authReq.ClientSecret)
	if err != nil {
		return err
	}

	hashedSecret, ok := clientCreds[authReq.ClientId]
	if !ok {
		return fmt.Errorf("client credentials incorrect")
	}

	parts := strings.Split(hashedSecret, "?")

	salt, err := base64.StdEncoding.DecodeString(parts[4])
	if err != nil {
		return err
	}

	hashedKey, err := base64.StdEncoding.DecodeString(parts[5])
	if err != nil {
		return err
	}

	N, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}
	r, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}
	p, err := strconv.Atoi(parts[2])
	if err != nil {
		return err
	}
	keyLen, err := strconv.Atoi(parts[3])
	if err != nil {
		return err
	}

	derivedKey, err := scrypt.Key(base64DecodedSecret, salt, N, r, p, keyLen)
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare(hashedKey, derivedKey) == 1 {
		return nil
	}

	return fmt.Errorf("client credentials missing or invalid")
}

func verifyAccessToken(authReq AuthReq) (int, error) {
	var claims jwt.MapClaims
	tkn, err := jwt.ParseWithClaims(authReq.AccessToken, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("invalid signing method")
		}

		return publicKey, nil
	})
	if err != nil {
		return http.StatusBadRequest, err
	}

	if tkn == nil || !tkn.Valid {
		return http.StatusUnauthorized, err
	}

	return http.StatusOK, nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("env file not found")
	}

	// verify env variables
	PORT, isSet := os.LookupEnv("PORT")
	if !isSet {
		panic(fmt.Errorf("PORT variable not defined"))
	}

	PRIVATE_KEY_PATH, isSet := os.LookupEnv("PRIVATE_KEY_PATH")
	if !isSet {
		panic(fmt.Errorf("PRIVATE_KEY_PATH variable not defined"))
	}

	PUBLIC_KEY_PATH, isSet := os.LookupEnv("PUBLIC_KEY_PATH")
	if !isSet {
		panic(fmt.Errorf("PUBLIC_KEY_PATH variable not defined"))
	}

	CLIENTS, isSet := os.LookupEnv("CLIENTS")
	if !isSet {
		panic(fmt.Errorf("CLIENTS variable not defined"))
	}

	ALLOWED_ORIGINS, isSet := os.LookupEnv("ALLOWED_ORIGINS")
	if !isSet {
		panic(fmt.Errorf("ALLOWED_ORIGINS variable not defined"))
	}

	_, isSet = os.LookupEnv("DEBUG")
	gin.SetMode(gin.ReleaseMode)
	if isSet {
		gin.SetMode(gin.DebugMode)
	}

	// parse client credentials
	err = parseClients(CLIENTS)
	if err != nil {
		panic(err)
	}

	// load keys
	err = loadKeys(PRIVATE_KEY_PATH, PUBLIC_KEY_PATH)
	if err != nil {
		panic(err)
	}

	r := gin.Default()

	// cors
	r.Use(cors.New(cors.Config{
		AllowOrigins: strings.Split(ALLOWED_ORIGINS, "&"),
		AllowMethods: []string{"GET", "POST"},
		AllowHeaders: []string{"Content-Type"},
	}))

	// health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
		})
	})

	// generate oauth2 & refresh token from client credentials
	r.POST("/auth", func(c *gin.Context) {
		var authReq AuthReq

		// validate req
		if err := c.BindJSON(&authReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		// validate client credentials
		err := validateClientCredentials(authReq)
		if err != nil {
			fmt.Println(err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		// generate token
		accessToken, err := issueAccessToken(authReq)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		// send token back
		c.JSON(http.StatusOK, gin.H{
			"accessToken": *accessToken,
		})

	})

	// validate oauth2 token
	r.POST("/verify", func(c *gin.Context) {
		var authReq AuthReq

		// validate req
		if err := c.BindJSON(&authReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		if authReq.AccessToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Errorf("AccessToken payload paramater not set"),
			})
			return
		}

		// validate client credentials
		err := validateClientCredentials(authReq)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		// generate token
		responseCode, err := verifyAccessToken(authReq)
		if err != nil {
			c.JSON(responseCode, gin.H{
				"error": err.Error(),
			})
			return
		}
		// send token back
		c.JSON(responseCode, gin.H{})

	})

	http.ListenAndServe(fmt.Sprintf(":%s", PORT), r)
}

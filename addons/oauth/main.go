package main

import (
	"fmt"
	"net/http"
	"os"
	"github.com/gin-gonic/gin"
	"errors"
	"crypto/rsa"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"io/ioutil"
	"time"
	"strings"
	"golang.org/x/crypto/bcrypt"
	"github.com/gin-contrib/cors"
	"log"
	"context"
)

type Account struct {
	Uid uuid.UUID
	Email string
	Password string
	SecurityQuestion string
	SecurityAnswer string
}

type AuthRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
	NewPassword string `json:"new_password,omitempty"`
	ConfirmPassword string `json:"confirm_password,omitempty"`
	ConfirmNewPassword string `json:"confirm_new_password,omitempty"`
	SecurityQuestion string `json:"security_question,omitempty"`
	SecurityAnswer string `json:"security_answer,omitempty"`
}

var (
	// RSA private key
	privateKey *rsa.PrivateKey
	publicKey *rsa.PublicKey
	GIN_MODE string
	db DB
)

type Statement struct {
	Query string
	Args []interface{}
}

type DB interface {
	QueryRow(ctx context.Context, stmt Statement, dest ...any) error
	Close()
	Exec(ctx context.Context, stmt Statement) (err error)
}

func issueAccessToken(uid string) (error, *string) {
	// Create a new token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": uid,
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	// Sign the token & return
	accessToken, err := token.SignedString(privateKey)
	if err != nil {
		return err, nil
	}

	return nil, &accessToken
}

func verifyToken(token string) error {
	// Parse the token
	var claims jwt.MapClaims
	tkn, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("Invalid signing method")
		}
		return publicKey, nil
	})
	if err != nil {
		return err
	}

	if !tkn.Valid {
		return errors.New("Invalid token")
	}

	return nil
}

func create_account(data *AuthRequest) (error, *string, *string, *uuid.UUID) {
	// verify length of password
	if len(data.Password) < 8 {
		return errors.New("Password must be at least 8 characters"), nil, nil, nil
	}

	// verify if passwords match
	if data.Password != data.ConfirmPassword {
		return errors.New("Passwords do not match"), nil, nil, nil
	}

	// hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		return err, nil, nil, nil
	}

	// Create a new account
	account := Account{
		Uid: uuid.New(),
		Email: data.Email,
		Password: string(hashedPassword),
		SecurityQuestion: data.SecurityQuestion,
		SecurityAnswer: data.SecurityAnswer,
	}

	var refreshToken string
	var response *string
	var email string

	// Check if account already exists
	stmt := Statement{
		Query: "SELECT email from accounts WHERE email = $1",
		Args: []interface{}{account.Email},
	}
	err = db.QueryRow(context.Background(), stmt, &email)
	if err == nil {
		return errors.New("Account already exists"), nil, nil, nil
	}

	// Save account to accounts table
	stmt = Statement{
		Query: "INSERT INTO accounts (uid, email, password, securityQuestion, securityAnswer) VALUES ($1, $2, $3, $4, $5)",
		Args: []interface{}{account.Uid, account.Email, account.Password, account.SecurityQuestion, account.SecurityAnswer},
	}
	err = db.Exec(context.Background(), stmt)
	if err != nil {
		return err, nil, nil, nil
	}

	err, response = issueAccessToken(account.Uid.String())
	if err != nil {
		return err, nil, nil, nil
	}

	refreshToken = uuid.New().String()

	// save refresh token to refreshTokens table
	stmt = Statement{
		Query: "INSERT INTO refreshTokens (uid, token) VALUES ($1, $2)",
		Args: []interface{}{account.Uid, refreshToken},
	}
	err = db.Exec(context.Background(), stmt)
	if err != nil {
		return err, nil, nil, nil
	}

	return nil, response, &refreshToken, &account.Uid
}

func login(data *AuthRequest) (error, *string, *string) {
	// Find account
	var account Account
	var refreshToken string

	// get account
	stmt := Statement{
		Query: "SELECT uid, email, password, securityQuestion, securityAnswer from accounts WHERE email = $1",
		Args: []interface{}{data.Email},
	}
	err := db.QueryRow(context.Background(), stmt, &account.Uid, &account.Email, &account.Password, &account.SecurityQuestion, &account.SecurityAnswer)
	
	if err != nil {
		return err, nil, nil
	}

	// get refresh token
	stmt = Statement{
		Query: "SELECT token from refreshTokens WHERE uid = $1",
		Args: []interface{}{account.Uid},
	}
	err = db.QueryRow(context.Background(), stmt, &refreshToken)
	
	if err != nil {
		return err, nil, nil
	}

	// Create a new token
	err, response := issueAccessToken(account.Uid.String())
	if err != nil {
		return err, nil, nil
	}
	
	return nil, response, &refreshToken 
}

func delete(uid string) error {
	// delete account
	stmt := Statement{
		Query: "DELETE FROM accounts WHERE uid = $1",
		Args: []interface{}{uid},
	}
	err := db.Exec(context.Background(), stmt)
	if err != nil {
		return err
	}

	// delete refresh token
	stmt = Statement{
		Query: "DELETE FROM refreshTokens WHERE uid = $1",
		Args: []interface{}{uid},
	}
	err = db.Exec(context.Background(), stmt)
	if err != nil {
		return err
	}

	return nil
}

func refresh(refreshToken string, uid string) (error, *string){
	// verify refresh token
	var response *string
	var currentToken string

	// verify refresh token & issue new access token
	stmt := Statement{
		Query: "SELECT token from refreshTokens WHERE uid = $1",
		Args: []interface{}{uid},
	}
	err := db.QueryRow(context.Background(), stmt, &currentToken)
	if err != nil {
		return err, nil
	}

	if currentToken != refreshToken {
		return errors.New("Invalid refresh token"), nil
	}

	// Create a new token
	err, response = issueAccessToken(uid)
	if err != nil {
		return err, nil
	}

	return nil, response

}

func resetPassword(data *AuthRequest) error {
	// verify if passwords match
	if data.NewPassword != data.ConfirmNewPassword {
		return errors.New("Passwords do not match")
	}

	var account Account
	// Find account
	stmt := Statement{
		Query: "SELECT uid, email, password, securityQuestion, securityAnswer from accounts WHERE email = $1",
		Args: []interface{}{data.Email},
	}
	err := db.QueryRow(context.Background(), stmt, &account.Uid, &account.Email, &account.Password, &account.SecurityQuestion, &account.SecurityAnswer)
	if err != nil {
		return err
	}

	// Check security question
	if account.SecurityAnswer != data.SecurityAnswer || account.SecurityQuestion != data.SecurityQuestion {
		return errors.New("Invalid security question or answer")
	}

	// verify length of password && not same as old password
	if len(data.NewPassword) < 8 {
		return errors.New("Password must be at least 8 characters")
	}

	if data.NewPassword == account.Password {
		return errors.New("New password cannot be the same as old password")
	}
	

	// Save new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	stmt = Statement{
		Query: "UPDATE accounts SET password = $1 WHERE email = $2",
		Args: []interface{}{string(hashedPassword), account.Email},
	}
	err = db.Exec(context.Background(), stmt)
	if err != nil {
		return err
	}

	return nil
}

func initTables() error {
	// create accounts table
	stmt := Statement{
		Query: "CREATE TABLE IF NOT EXISTS accounts (uid UUID PRIMARY KEY, email STRING, password STRING, securityQuestion STRING, securityAnswer STRING)",
		Args: []interface{}{},
	}
	err := db.Exec(context.Background(), stmt)
	if err != nil {
		return err
	}

	// create refreshTokens table
	stmt = Statement{
		Query: "CREATE TABLE IF NOT EXISTS refreshTokens (uid UUID PRIMARY KEY, token STRING)",
		Args: []interface{}{},
	}
	err = db.Exec(context.Background(), stmt)
	if err != nil {
		return err
	}
	return nil

}

func main() {
	r := gin.Default()
	
	// set port
	PORT := os.Getenv("PORT")
	if PORT == "" {
		PORT = "3000"
	}

	ALLOWED_ORIGINS := os.Getenv("ALLOWED_ORIGINS")
	if ALLOWED_ORIGINS == "" {
		panic("allowed orgins not set")
	}

	var connectionString string

	GIN_MODE = os.Getenv("GIN_MODE")
	if GIN_MODE == "release" {
		log.Println(fmt.Sprintf("GIN_MODE = %v", GIN_MODE))
		log.Println(fmt.Sprintf("PORT = %v", PORT))
		log.Println(fmt.Sprintf("ALLOWED_ORIGINS = %v", ALLOWED_ORIGINS))
		
		gin.SetMode(gin.ReleaseMode)
		
		connectionString = os.Getenv("CONNECTION_STRING")
		if connectionString == "" {
			panic("connection string not set")
		}
	}

	// cors
	r.Use(cors.New(cors.Config{
		AllowOrigins: strings.Split(ALLOWED_ORIGINS, ","),
		AllowMethods: []string{"GET", "POST", "DELETE"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
	}))

	// read RSA private key from file	
	privateKey_file, err := ioutil.ReadFile("/etc/ssl/private/private_key.pem")
	if err != nil {
		panic(err)
	}

	// Parse RSA private key
	privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(privateKey_file)
	if err != nil {
		panic(err)
	}

	// read RSA public key from file
	publicKey_file, err := ioutil.ReadFile("/etc/ssl/public/public_key.pem")
	if err != nil {
		panic(err)
	}

	// Parse RSA public key
	publicKey, err = jwt.ParseRSAPublicKeyFromPEM(publicKey_file)
	if err != nil {
		panic(err)
	}

	if GIN_MODE == "release" {
		// connect to cockroach db instance
		err, db = NewPostgres(context.Background(), connectionString)
		if err != nil {
			panic(err)
		}

		// Close the database
		defer db.Close()

		// init tables
		if os.Getenv("INIT_CONTAINER") == "true" {
			log.Println("Initializing tables")
			err = initTables()
			if err != nil {
				panic(err)
			}
			return 
		}
	}else{
		// connect to sqlite db
		err, db = NewSQLite()
		if err != nil {
			panic(err)
		}

		// Close the database
		defer db.Close()

		// init tables
		log.Println("Initializing tables")
		err = initTables()
		if err != nil {
			panic(err)
		}
	}
	
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
		})
	})

	r.POST("/signup", func(c *gin.Context) {
		var data AuthRequest
		err := c.BindJSON(&data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		
		err, response, refreshToken, uid := create_account(&data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token": response,
			"refresh_token": refreshToken,
			"uid": uid.String(),
		})
		return
	})

	r.POST("/login", func(c *gin.Context) {
		var data AuthRequest
		err := c.BindJSON(&data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		err, response, refreshToken := login(&data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token": response,
			"refresh_token": refreshToken,
		})
		return
	})

	r.POST("/resetpassword", func(c *gin.Context) {
		var data AuthRequest
		err := c.BindJSON(&data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		err = resetPassword(&data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{})
		return
	})

	r.DELETE("/delete/:uid", func(c *gin.Context) {
		uid := c.Param("uid")
		if uid == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "No uid provided",
			})
			return
		}

		err = delete(uid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{})
	})

	r.GET("/refresh/:uid", func(c *gin.Context) {
		uid := c.Param("uid")
		refreshToken := c.GetHeader("Authorization")
		if refreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "No refresh token provided",
			})
			return
		}

		parsedRefreshToken := strings.Split(refreshToken, "Bearer ")[1]

		err, response := refresh(parsedRefreshToken, uid)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token": response,
		})
	})

	r.GET("/verify/:token", func(c *gin.Context) {
		// verify token
		token := c.Param("token")
		err := verifyToken(token)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{})
	})
	
	http.ListenAndServe(fmt.Sprintf(":%s", PORT), r)
}
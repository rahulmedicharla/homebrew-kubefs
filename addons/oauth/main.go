package main

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

type Account struct {
	Uid      uuid.UUID
	Email    string
	Password string
	Secret   string
}

type AuthRequest struct {
	Email              string `json:"email"`
	Password           string `json:"password"`
	NewPassword        string `json:"new_password,omitempty"`
	ConfirmPassword    string `json:"confirm_password,omitempty"`
	ConfirmNewPassword string `json:"confirm_new_password,omitempty"`
}

var (
	// RSA private key
	NAME            string
	TWO_FACTOR_AUTH bool
	privateKey      *rsa.PrivateKey
	publicKey       *rsa.PublicKey
	db              DB
)

type Statement struct {
	Query string
	Args  []interface{}
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

func verifyToken(token string) (error, int) {
	// Parse the token
	var claims jwt.MapClaims
	tkn, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("Invalid signing method")
		}

		return publicKey, nil
	})
	if err != nil {
		return err, http.StatusBadRequest
	}

	if tkn == nil || !tkn.Valid {
		return errors.New("Invalid token"), http.StatusUnauthorized
	}

	return nil, http.StatusOK
}

func create_account(data *AuthRequest, tfa_code string) (error, int, *string, *string, *uuid.UUID) {
	// verify length of password
	if len(data.Password) < 8 {
		return errors.New("Password must be at least 8 characters"), http.StatusBadRequest, nil, nil, nil
	}

	// verify if passwords match
	if data.Password != data.ConfirmPassword {
		return errors.New("Passwords do not match"), http.StatusBadRequest, nil, nil, nil
	}

	// hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		return err, http.StatusBadRequest, nil, nil, nil
	}

	// check if account already exists
	var email string
	stmt := Statement{
		Query: "SELECT email FROM accounts WHERE email = $1",
		Args:  []interface{}{data.Email},
	}
	err = db.QueryRow(context.Background(), stmt, &email)
	if err == nil {
		return errors.New("Account already exists"), http.StatusConflict, nil, nil, nil
	}

	// verify tfa code
	var totpSecret string
	if TWO_FACTOR_AUTH {
		// get TOTP secret for user
		stmt := Statement{
			Query: "SELECT secret FROM twoFactorAuth WHERE email = $1",
			Args:  []interface{}{data.Email},
		}
		err := db.QueryRow(context.Background(), stmt, &totpSecret)
		if err != nil {
			return err, http.StatusInternalServerError, nil, nil, nil
		}

		validTotp := totp.Validate(tfa_code, totpSecret)
		if !validTotp {
			return errors.New("Invalid 2FA code"), http.StatusBadRequest, nil, nil, nil
		}

		// delete secret from twoFactorAuth table
		stmt = Statement{
			Query: "DELETE FROM twoFactorAuth WHERE email = $1",
			Args:  []interface{}{data.Email},
		}
		err = db.Exec(context.Background(), stmt)
		if err != nil {
			return err, http.StatusInternalServerError, nil, nil, nil
		}
	}

	// Create a new account
	account := Account{
		Uid:      uuid.New(),
		Email:    data.Email,
		Password: string(hashedPassword),
		Secret:   totpSecret,
	}

	var refreshToken string
	var response *string

	// Save account to accounts table
	stmt = Statement{
		Query: "INSERT INTO accounts (uid, email, password, secret) VALUES ($1, $2, $3, $4)",
		Args:  []interface{}{account.Uid, account.Email, account.Password, account.Secret},
	}
	err = db.Exec(context.Background(), stmt)
	if err != nil {
		return err, http.StatusInternalServerError, nil, nil, nil
	}

	err, response = issueAccessToken(account.Uid.String())
	if err != nil {
		return err, http.StatusInternalServerError, nil, nil, nil
	}

	refreshToken = uuid.New().String()

	// save refresh token to refreshTokens table
	stmt = Statement{
		Query: "INSERT INTO refreshTokens (uid, token) VALUES ($1, $2)",
		Args:  []interface{}{account.Uid, refreshToken},
	}
	err = db.Exec(context.Background(), stmt)
	if err != nil {
		return err, http.StatusInternalServerError, nil, nil, nil
	}

	return nil, http.StatusOK, response, &refreshToken, &account.Uid
}

func login(data *AuthRequest, tfa_code string) (error, int, *string, *string, *uuid.UUID) {
	// Find account
	var account Account
	var refreshToken string

	// verify tfa code
	if TWO_FACTOR_AUTH {
		// get TOTP secret for user
		stmt := Statement{
			Query: "SELECT secret FROM accounts WHERE email = $1",
			Args:  []interface{}{data.Email},
		}
		var secret string
		err := db.QueryRow(context.Background(), stmt, &secret)
		if err != nil {
			return err, http.StatusInternalServerError, nil, nil, nil
		}

		log.Println("secret: ", secret)
		log.Println("tfa_code: ", tfa_code)
		validTotp := totp.Validate(tfa_code, secret)
		if !validTotp {
			return errors.New("Invalid 2FA code"), http.StatusBadRequest, nil, nil, nil
		}
	}

	// get account
	stmt := Statement{
		Query: "SELECT uid, email, password, secret from accounts WHERE email = $1",
		Args:  []interface{}{data.Email},
	}
	err := db.QueryRow(context.Background(), stmt, &account.Uid, &account.Email, &account.Password, &account.Secret)

	if err != nil {
		return err, http.StatusInternalServerError, nil, nil, nil
	}

	// verify password
	err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(data.Password))
	if err != nil {
		return errors.New("Invalid password"), http.StatusBadRequest, nil, nil, nil
	}

	// get refresh token
	stmt = Statement{
		Query: "SELECT token from refreshTokens WHERE uid = $1",
		Args:  []interface{}{account.Uid},
	}
	err = db.QueryRow(context.Background(), stmt, &refreshToken)

	if err != nil {
		return err, http.StatusInternalServerError, nil, nil, nil
	}

	// Create a new token
	err, response := issueAccessToken(account.Uid.String())
	if err != nil {
		return err, http.StatusInternalServerError, nil, nil, nil
	}

	return nil, http.StatusOK, response, &refreshToken, &account.Uid
}

func delete(uid string, tfa_code string) (error, int) {
	// verify tfa code
	if TWO_FACTOR_AUTH {
		// get TOTP secret for user
		stmt := Statement{
			Query: "SELECT secret FROM accounts WHERE uid = $1",
			Args:  []interface{}{uid},
		}
		var secret string
		err := db.QueryRow(context.Background(), stmt, &secret)
		if err != nil {
			return err, http.StatusInternalServerError
		}

		validTotp := totp.Validate(tfa_code, secret)
		if !validTotp {
			return errors.New("Invalid 2FA code"), http.StatusBadRequest
		}
	}

	// delete account
	stmt := Statement{
		Query: "DELETE FROM accounts WHERE uid = $1",
		Args:  []interface{}{uid},
	}
	err := db.Exec(context.Background(), stmt)
	if err != nil {
		return err, http.StatusInternalServerError
	}

	// delete refresh token
	stmt = Statement{
		Query: "DELETE FROM refreshTokens WHERE uid = $1",
		Args:  []interface{}{uid},
	}
	err = db.Exec(context.Background(), stmt)
	if err != nil {
		return err, http.StatusInternalServerError
	}

	return nil, http.StatusOK
}

func refresh(refreshToken string, uid string) (error, int, *string) {
	// verify refresh token
	var response *string
	var currentToken string

	// verify refresh token & issue new access token
	stmt := Statement{
		Query: "SELECT token FROM refreshTokens WHERE uid = $1",
		Args:  []interface{}{uid},
	}
	err := db.QueryRow(context.Background(), stmt, &currentToken)
	if err != nil {
		return err, http.StatusInternalServerError, nil
	}

	if currentToken != refreshToken {
		return errors.New("Invalid refresh token"), http.StatusUnauthorized, nil
	}

	// Create a new token
	err, response = issueAccessToken(uid)
	if err != nil {
		return err, http.StatusInternalServerError, nil
	}

	return nil, http.StatusOK, response

}

func resetPassword(data *AuthRequest, tfa_code string) (error, int) {
	// verify tfa code
	if TWO_FACTOR_AUTH {
		// get TOTP secret for user
		stmt := Statement{
			Query: "SELECT secret FROM accounts WHERE email = $1",
			Args:  []interface{}{data.Email},
		}
		var secret string
		err := db.QueryRow(context.Background(), stmt, &secret)
		if err != nil {
			return err, http.StatusInternalServerError
		}

		validTotp := totp.Validate(tfa_code, secret)
		if !validTotp {
			return errors.New("Invalid 2FA code"), http.StatusBadRequest
		}
	}

	// verify if passwords match
	if data.NewPassword != data.ConfirmNewPassword {
		return errors.New("Passwords do not match"), http.StatusBadRequest
	}

	var account Account
	// Find account
	stmt := Statement{
		Query: "SELECT uid, email, password, secret from accounts WHERE email = $1",
		Args:  []interface{}{data.Email},
	}
	err := db.QueryRow(context.Background(), stmt, &account.Uid, &account.Email, &account.Password, &account.Secret)
	if err != nil {
		return err, http.StatusInternalServerError
	}

	// verify length of password && not same as old password
	if len(data.NewPassword) < 8 {
		return errors.New("Password must be at least 8 characters"), http.StatusBadRequest
	}

	if data.NewPassword == account.Password {
		return errors.New("New password cannot be the same as old password"), http.StatusBadRequest
	}

	// Save new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err, http.StatusInternalServerError
	}

	stmt = Statement{
		Query: "UPDATE accounts SET password = $1 WHERE email = $2",
		Args:  []interface{}{string(hashedPassword), account.Email},
	}
	err = db.Exec(context.Background(), stmt)
	if err != nil {
		return err, http.StatusInternalServerError
	}

	return nil, http.StatusOK
}

func initTables() error {
	// create accounts table
	stmt := Statement{
		Query: "CREATE TABLE IF NOT EXISTS accounts (uid UUID PRIMARY KEY, email TEXT, password TEXT, secret TEXT)",
		Args:  []interface{}{},
	}
	err := db.Exec(context.Background(), stmt)
	if err != nil {
		return err
	}

	// create refreshTokens table
	stmt = Statement{
		Query: "CREATE TABLE IF NOT EXISTS refreshTokens (uid UUID PRIMARY KEY, token TEXT)",
		Args:  []interface{}{},
	}
	err = db.Exec(context.Background(), stmt)
	if err != nil {
		return err
	}

	if TWO_FACTOR_AUTH {
		stmt = Statement{
			Query: "CREATE TABLE IF NOT EXISTS twoFactorAuth (email TEXT PRIMARY KEY, secret TEXT)",
			Args:  []interface{}{},
		}
		err = db.Exec(context.Background(), stmt)
		if err != nil {
			return err
		}
	}

	return nil

}

func create2FAAccount(email string) (error, int, *string) {
	if !TWO_FACTOR_AUTH {
		return errors.New("2FA is not enabled"), http.StatusBadRequest, nil
	}

	// Create a new key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      NAME + "2FA",
		AccountName: email,
	})

	if err != nil {
		return err, http.StatusInternalServerError, nil
	}

	url := key.URL()
	// see if key already exists
	var secret string
	stmt := Statement{
		Query: "SELECT secret from twoFactorAuth WHERE email = $1",
		Args:  []interface{}{email},
	}
	err = db.QueryRow(context.Background(), stmt, &secret)
	if err == nil {
		stmt = Statement{
			Query: "DELETE FROM twoFactorAuth WHERE email = $1",
			Args:  []interface{}{email},
		}
		err = db.Exec(context.Background(), stmt)
		if err != nil {
			return err, http.StatusInternalServerError, nil
		}
	}

	// save key to twoFactorAuth table
	stmt = Statement{
		Query: "INSERT INTO twoFactorAuth (email, secret) VALUES ($1, $2)",
		Args:  []interface{}{email, key.Secret()},
	}
	err = db.Exec(context.Background(), stmt)
	if err != nil {
		return err, http.StatusInternalServerError, nil
	}

	return nil, http.StatusOK, &url

}

func main() {
	r := gin.Default()

	var writeConnectionString string
	var readConnectionString string
	var err error

	// read if 2FA is enabled
	TWO_FACTOR_AUTH = false
	if os.Getenv("TWO_FACTOR_AUTH") == "true" {
		log.Println("2FA enabled")
		TWO_FACTOR_AUTH = true
	}

	// set mode & establish connection
	MODE := os.Getenv("MODE")
	if MODE == "release" || MODE == "init" {
		gin.SetMode(gin.ReleaseMode)

		writeConnectionString = os.Getenv("WRITE_CONNECTION_STRING")
		if writeConnectionString == "" {
			panic("write connection string not set")
		}

		readConnectionString = os.Getenv("READ_CONNECTION_STRING")
		if readConnectionString == "" {
			panic("read connection string not set")
		}

		// connect to postgresql db instance
		err, db = NewPostgres(context.Background(), writeConnectionString, readConnectionString)
		if err != nil {
			panic(err)
		}

		// Close the database
		defer db.Close()

		if MODE == "init" {
			log.Println("MODE = init")

			log.Println("Initializing tables")
			err = initTables()
			if err != nil {
				panic(err)
			}
			return
		}

	} else {
		log.Println("MODE = dev")
		MODE = "dev"

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

	// set port
	PORT := os.Getenv("PORT")
	if PORT == "" {
		PORT = "3000"
	}

	NAME = os.Getenv("NAME")
	if NAME == "" {
		panic("requesting resource not set")
	}

	// set allowed origins
	ALLOWED_ORIGINS := os.Getenv("ALLOWED_ORIGINS")
	if ALLOWED_ORIGINS == "" {
		panic("allowed orgins not set")
	}

	// cors
	r.Use(cors.New(cors.Config{
		AllowOrigins: strings.Split(ALLOWED_ORIGINS, ","),
		AllowMethods: []string{"GET", "POST", "DELETE"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
	}))

	// read RSA private key from file
	privateKey_file, err := os.ReadFile("/etc/ssl/private/private_key.pem")
	if err != nil {
		panic(err)
	}

	// Parse RSA private key
	privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(privateKey_file)
	if err != nil {
		panic(err)
	}

	// read RSA public key from file
	publicKey_file, err := os.ReadFile("/etc/ssl/public/public_key.pem")
	if err != nil {
		panic(err)
	}

	// Parse RSA public key
	publicKey, err = jwt.ParseRSAPublicKeyFromPEM(publicKey_file)
	if err != nil {
		panic(err)
	}

	if MODE == "dev" {
		// serve docs/index.html as static file at /docs
		r.LoadHTMLFiles("docs/index.html")
		r.GET("/docs", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", gin.H{
				"TWO_FACTOR_AUTH":         TWO_FACTOR_AUTH,
				"MODE":                    MODE,
				"NAME":                    NAME,
				"WRITE CONNECTION_STRING": writeConnectionString,
				"READ CONNECTION_STRING":  readConnectionString,
				"PORT":                    PORT,
				"ALLOWED_ORIGINS":         ALLOWED_ORIGINS,
			})
		})

	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
		})
	})

	r.GET("/generateTOTPAccount/:email", func(c *gin.Context) {
		email := c.Param("email")
		if email == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "No email provided",
			})
			return
		}

		err, code, key := create2FAAccount(email)
		if err != nil {
			c.JSON(code, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(code, gin.H{
			"url": key,
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

		var tfa_code string
		if TWO_FACTOR_AUTH {
			tfa_code = c.GetHeader("Authorization")
			if tfa_code == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "No 2FA code provided",
				})
				return
			}
			tfa_code = strings.Split(tfa_code, "Bearer ")[1]
		}

		err, code, response, refreshToken, uid := create_account(&data, tfa_code)
		if err != nil {
			c.JSON(code, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(code, gin.H{
			"access_token":  response,
			"refresh_token": refreshToken,
			"uid":           uid.String(),
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

		var tfa_code string
		if TWO_FACTOR_AUTH {
			tfa_code = c.GetHeader("Authorization")
			if tfa_code == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "No 2FA code provided",
				})
				return
			}
			tfa_code = strings.Split(tfa_code, "Bearer ")[1]
		}

		err, code, response, refreshToken, uid := login(&data, tfa_code)
		if err != nil {
			c.JSON(code, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(code, gin.H{
			"access_token":  response,
			"refresh_token": refreshToken,
			"uid":           uid.String(),
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

		var tfa_code string
		if TWO_FACTOR_AUTH {
			tfa_code = c.GetHeader("Authorization")
			if tfa_code == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "No 2FA code provided",
				})
				return
			}
			tfa_code = strings.Split(tfa_code, "Bearer ")[1]
		}

		err, code := resetPassword(&data, tfa_code)
		if err != nil {
			c.JSON(code, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(code, gin.H{})
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

		var tfa_code string
		if TWO_FACTOR_AUTH {
			tfa_code = c.GetHeader("Authorization")
			if tfa_code == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "No 2FA code provided",
				})
				return
			}
			tfa_code = strings.Split(tfa_code, "Bearer ")[1]
		}

		err, code := delete(uid, tfa_code)
		if err != nil {
			c.JSON(code, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(code, gin.H{})
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

		err, code, response := refresh(parsedRefreshToken, uid)

		if err != nil {
			c.JSON(code, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(code, gin.H{
			"access_token": response,
		})
	})

	r.GET("/verify/:token", func(c *gin.Context) {
		// verify token
		token := c.Param("token")
		err, code := verifyToken(token)
		if err != nil {
			c.JSON(code, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(code, gin.H{})
	})

	http.ListenAndServe(fmt.Sprintf(":%s", PORT), r)
}

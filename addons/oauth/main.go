package main

import (
	"fmt"
	"net/http"
	"os"
	"github.com/gin-gonic/gin"
	"errors"
	"crypto/rsa"
	"github.com/golang-jwt/jwt/v5"
	"encoding/pem"
	"crypto/x509"
	"github.com/google/uuid"
	"io/ioutil"
	"github.com/dgraph-io/badger/v4"
	"encoding/json"
	"time"
	"strings"
	"golang.org/x/crypto/bcrypt"
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
	ConfirmPassword string `json:"confirm_password omitempty"`
	SecurityQuestion string `json:"security_question omitempty"`
	SecurityAnswer string `json:"security_answer omitempty"`
}

const SUCCESS = 1
const ERROR = 0

var (
	// RSA private key
	privateKey *rsa.PrivateKey
	db *badger.DB
)

func issueAccessToken(uid string) (int, string) {
	// Create a new token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": uid,
		"exp": 3600,
		"iat": time.Now().Unix(),
	})

	// Sign the token & return
	accessToken, err := token.SignedString(privateKey)
	if err != nil {
		return ERROR, err.Error()
	}

	return SUCCESS, accessToken
}

func create_account(data *AuthRequest) (int, string, string) {
	// verify length of password
	if len(data.Password) < 8 {
		return ERROR, "Password must be at least 8 characters", ""
	}

	// verify if passwords match
	if data.Password != data.ConfirmPassword {
		return ERROR, "Passwords do not match", ""
	}

	// hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		return ERROR, err.Error(), ""
	}

	// Create a new account
	account := Account{
		Uid: uuid.New(),
		Email: data.Email,
		Password: string(hashedPassword),
		SecurityQuestion: data.SecurityQuestion,
		SecurityAnswer: data.SecurityAnswer,
	}

	// Check if account already exists
	dbErr := db.View(func(txn *badger.Txn) error {
		item, _ := txn.Get([]byte(account.Email))
		if item != nil {
			return errors.New("Account already exists")
		}

		return nil
	})
	if dbErr != nil {
		return ERROR, dbErr.Error(), ""
	}

	refreshToken := uuid.New().String()

	// Save account to database
	dbErr = db.Update(func(txn *badger.Txn) error {
		accountBytes, err := json.Marshal(account)
		if err != nil {
			return err
		}

		err = txn.Set([]byte(account.Email), accountBytes)
		if err != nil {
			return err
		}

		err = txn.Set([]byte("refreshToken/"+account.Uid.String()), []byte(refreshToken))
		if err != nil {
			return err
		}

		return nil
	})

	if dbErr != nil {
		return ERROR, dbErr.Error(), ""
	}

	// Create a new token
	status, response := issueAccessToken(account.Uid.String())
	if status == ERROR {
		return ERROR, response, ""
	}

	return SUCCESS, response, refreshToken
}

func login(data *AuthRequest) (int, string, string) {
	// Find account
	var account Account
	var refreshToken string
	dbErr := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(data.Email))

		if err != nil {
			return errors.New("Account not found")
		}

		err = item.Value(func(val []byte) error {
			err := json.Unmarshal(val, &account)
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return err
		}

		item, err = txn.Get([]byte("refreshToken/" + account.Uid.String()))
		if err != nil {
			return err 
		}

		err = item.Value(func(val []byte) error {
			refreshToken = string(val)
			return nil
		})

		if err != nil {
			return err
		}

		return nil
	})
	if dbErr != nil {
		return ERROR, dbErr.Error(), ""
	}

	// Check password
	err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(data.Password))
	if err != nil {
		return ERROR, "Invalid password", ""
	}

	// Create a new token
	status, response := issueAccessToken(account.Uid.String())
	if status == ERROR {
		return ERROR, response, ""
	}
	
	return SUCCESS, response, refreshToken 
}

func delete(email string) (int, string) {
	// Find account
	dbErr := db.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(email))
		if err != nil {
			return err
		}

		err = txn.Delete([]byte("refreshToken/" + email))
		if err != nil {
			return err
		}

		return nil
	})
	if dbErr != nil {
		return ERROR, dbErr.Error()
	}

	return SUCCESS, "Account deleted"
}

func refresh(refreshToken string, uid string) (int, string){
	// verify refresh token
	var response string
	dbErr := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("refreshToken/"+uid))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			currentToken := string(val)
			if currentToken != refreshToken {
				return errors.New("Invalid refresh token")
			}

			// Create a new token
			var status int
			status, response = issueAccessToken(uid)
			if status == ERROR {
				return errors.New(response)
			}

			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})
	if dbErr != nil {
		return ERROR, dbErr.Error()
	}

	return SUCCESS, response

}

func resetPassword(email string, newPassword string, confirmNewPassword string, securityQuestion, securityAnswer string) (int, string) {
	// verify if passwords match
	if newPassword != confirmNewPassword {
		return ERROR, "Passwords do not match"
	}

	// Find account
	var account Account
	dbErr := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(email))

		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			err := json.Unmarshal(val, &account)
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})
	if dbErr != nil {
		return ERROR, dbErr.Error()
	}

	// Check security question
	if account.SecurityAnswer != securityAnswer || account.SecurityQuestion != securityQuestion {
		return ERROR, "Invalid security question/answer"
	}

	// verify length of password && not same as old password
	if len(newPassword) < 8 {
		return ERROR, "Password must be at least 8 characters"
	}

	if newPassword == account.Password {
		return ERROR, "New password cannot be the same as old password"
	}
	

	// Save new password
	dbErr = db.Update(func(txn *badger.Txn) error {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		
		account.Password = string(hashedPassword)

		accountBytes, err := json.Marshal(account)
		if err != nil {
			return err
		}

		err = txn.Set([]byte(account.Email), accountBytes)
		if err != nil {
			return err
		}

		return nil
	})
	if dbErr != nil {
		return ERROR, dbErr.Error()
	}

	return SUCCESS, "Password reset successful"
}

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("public/*")
	
	// set port
	PORT := os.Getenv("PORT")
	if PORT == "" {
		PORT = "3000"
	}

	REDIRECT_URL := os.Getenv("REDIRECT_URL")
	if REDIRECT_URL == "" {
		panic("REDIRECT_URL not set")
	}

	// read RSA private key from file	
	privateKey_file, err := ioutil.ReadFile("/etc/ssl/private/private_key.pem")
	if err != nil {
		panic(err)
	}

	// Parse RSA private key
	block, _ := pem.Decode(privateKey_file)
	if block == nil {
		panic(errors.New("failed to parse PEM block containing the key"))
	}

	// get private key
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		panic(err)
	}

	var ok bool
	privateKey, ok = key.(*rsa.PrivateKey)
	if !ok {
		panic(errors.New("failed to parse private key"))
	}

	// Create a new badger database
	opts := badger.DefaultOptions("/app/store")
	db, err = badger.Open(opts)
	if err != nil {
		panic(err)
	}

	// Close the database
	defer db.Close()

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{})
	})

	r.GET("/login", func(c *gin.Context) {
		errMsg := c.Query("error")
		successMsg := c.Query("success")		
		c.HTML(http.StatusOK, "login.html", gin.H{
			"error": errMsg,
			"success": successMsg,
		})
	})

	r.GET("/signup", func(c *gin.Context) {
		errMsg := c.Query("error")
		c.HTML(http.StatusOK, "signup.html", gin.H{
			"error": errMsg,
		})
	})

	r.GET("/forgotpassword", func(c *gin.Context) {
		errMsg := c.Query("error")
		c.HTML(http.StatusOK, "forgotpassword.html", gin.H{
			"error": errMsg,
		})
	})
	
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
		})
	})

	r.POST("/signup", func(c *gin.Context) {
		email := c.PostForm("email")
		password := c.PostForm("password")
		confirmPassword := c.PostForm("confirmPassword")
		securityQuestion := c.PostForm("securityQuestion")
		securityAnswer := c.PostForm("securityAnswer")

		data := AuthRequest{
			Email: email,
			Password: password,
			ConfirmPassword: confirmPassword,
			SecurityQuestion: securityQuestion,
			SecurityAnswer: securityAnswer,
		}

		status, response, refreshToken := create_account(&data)
		if status == ERROR {
			c.Redirect(http.StatusFound, "/signup?error=" + response)
			return
		}

		c.Redirect(http.StatusFound, REDIRECT_URL + "?access_token=" + response + "&refresh_token=" + refreshToken)
		return
	})

	r.POST("/login", func(c *gin.Context) {
		email := c.PostForm("email")
		password := c.PostForm("password")

		data := AuthRequest{
			Email: email,
			Password: password,
		}

		status, response, refreshToken := login(&data)
		if status == ERROR {
			c.Redirect(http.StatusFound, "/login?error=" + response)
			return
		}

		c.Redirect(http.StatusFound, REDIRECT_URL + "?access_token=" + response + "&refresh_token=" + refreshToken)
		return
	})

	r.POST("/forgotpassword", func(c *gin.Context) {
		email := c.PostForm("email")
		newPassword := c.PostForm("newPassword")
		confirmNewPassword := c.PostForm("confirmNewPassword")
		security_question := c.PostForm("securityQuestion")
		securityAnswer := c.PostForm("securityAnswer")

		status, response := resetPassword(email, newPassword, confirmNewPassword, security_question, securityAnswer)
		if status == ERROR {
			c.Redirect(http.StatusFound, "/forgotpassword?error=" + response)
			return
		}

		c.Redirect(http.StatusFound, "/login?success=" + response)
		return
	})

	r.DELETE("/delete/:email", func(c *gin.Context) {
		email := c.Param("email")
		status, response := delete(email)

		if status == ERROR {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "error",
				"message": response,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"access_token": response,
		})
	})

	r.GET("/refresh/:uid", func(c *gin.Context) {
		uid := c.Param("uid")
		refreshToken := c.GetHeader("Authorization")
		if refreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "error",
				"message": "No refresh token provided",
			})
			return
		}

		parsedRefreshToken := strings.Split(refreshToken, "Bearer ")[1]

		status, response := refresh(parsedRefreshToken, uid)

		if status == ERROR {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "error",
				"message": response,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"access_token": response,
		})
	})
	
	http.ListenAndServe(fmt.Sprintf(":%s", PORT), r)
}
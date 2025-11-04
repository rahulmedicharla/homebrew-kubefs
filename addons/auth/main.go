package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type Account struct {
	Uid      uuid.UUID
	Email    string
	Password string
}

type AuthRequest struct {
	Email              string `json:"email"`
	Password           string `json:"password"`
	NewPassword        string `json:"newPassword,omitempty"`
	ConfirmPassword    string `json:"confirmPassword,omitempty"`
	ConfirmNewPassword string `json:"confirmNewPassword,omitempty"`
}

var (
	db DB
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

func create_account(data *AuthRequest) (int, *uuid.UUID, error) {
	// verify length of password
	if len(data.Password) < 8 {
		return http.StatusBadRequest, nil, fmt.Errorf("password must be at least 8 characters")
	}

	// verify if passwords match
	if data.Password != data.ConfirmPassword {
		return http.StatusBadRequest, nil, fmt.Errorf("passwords do not match")
	}

	// hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	// check if account already exists
	var email string
	stmt := Statement{
		Query: "SELECT email FROM accounts WHERE email = $1",
		Args:  []interface{}{data.Email},
	}
	err = db.QueryRow(context.Background(), stmt, &email)
	if err == nil {
		return http.StatusConflict, nil, fmt.Errorf("account already exists")
	}

	// Create a new account
	account := Account{
		Uid:      uuid.New(),
		Email:    data.Email,
		Password: string(hashedPassword),
	}

	// Save account to accounts table
	stmt = Statement{
		Query: "INSERT INTO accounts (uid, email, password) VALUES ($1, $2, $3)",
		Args:  []interface{}{account.Uid, account.Email, account.Password},
	}
	err = db.Exec(context.Background(), stmt)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusOK, &account.Uid, nil
}

func login(data *AuthRequest) (int, *uuid.UUID, error) {
	// Find account
	var account Account

	// get account
	stmt := Statement{
		Query: "SELECT uid, email, password from accounts WHERE email = $1",
		Args:  []interface{}{data.Email},
	}
	err := db.QueryRow(context.Background(), stmt, &account.Uid, &account.Email, &account.Password)

	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	// verify password
	err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(data.Password))
	if err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("invalid password")
	}

	return http.StatusOK, &account.Uid, nil
}

func delete(uid string) (int, error) {
	// verify tfa code

	// delete account
	stmt := Statement{
		Query: "DELETE FROM accounts WHERE uid = $1",
		Args:  []interface{}{uid},
	}
	err := db.Exec(context.Background(), stmt)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func resetPassword(data *AuthRequest) (int, error) {
	// verify if passwords match
	if data.NewPassword != data.ConfirmNewPassword {
		return http.StatusBadRequest, fmt.Errorf("passwords do not match")
	}

	var account Account
	// Find account
	stmt := Statement{
		Query: "SELECT uid, email, password from accounts WHERE email = $1",
		Args:  []interface{}{data.Email},
	}
	err := db.QueryRow(context.Background(), stmt, &account.Uid, &account.Email, &account.Password)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// verify length of password && not same as old password
	if len(data.NewPassword) < 8 {
		return http.StatusBadRequest, fmt.Errorf("password must be at least 8 characters")
	}

	if data.NewPassword == account.Password {
		return http.StatusBadRequest, errors.New("new password cannot be the same as old password")
	}

	// Save new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	stmt = Statement{
		Query: "UPDATE accounts SET password = $1 WHERE email = $2",
		Args:  []interface{}{string(hashedPassword), account.Email},
	}
	err = db.Exec(context.Background(), stmt)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func initTables() error {
	// create accounts table
	stmt := Statement{
		Query: "CREATE TABLE IF NOT EXISTS accounts (uid UUID PRIMARY KEY, email TEXT, password TEXT)",
		Args:  []interface{}{},
	}
	err := db.Exec(context.Background(), stmt)
	if err != nil {
		return err
	}

	return nil

}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("env file not found")
	}

	r := gin.Default()

	var writeConnectionString string
	var readConnectionString string

	// set mode & establish connection
	DEBUG, ok := os.LookupEnv("DEBUG")
	if ok {
		gin.SetMode(gin.DebugMode)
		// connect to sqlite db
		db, err = NewSQLite()
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
	} else {
		gin.SetMode(gin.ReleaseMode)

		writeConnectionString, ok = os.LookupEnv("WRITE_CONNECTION_STRING")
		if !ok {
			panic("write connection string not set")
		}

		readConnectionString, ok = os.LookupEnv("READ_CONNECTION_STRING")
		if !ok {
			panic("read connection string not set")
		}

		// connect to postgresql db instance
		db, err = NewPostgres(context.Background(), writeConnectionString, readConnectionString)
		if err != nil {
			panic(err)
		}

		// Close the database
		defer db.Close()
	}

	// set port
	PORT, ok := os.LookupEnv("PORT")
	if !ok {
		panic("port not defined")
	}

	// set allowed origins
	ALLOWED_ORIGINS, ok := os.LookupEnv("ALLOWED_ORIGINS")
	if !ok {
		panic("allowed orgins not set")
	}
	// cors
	r.Use(cors.New(cors.Config{
		AllowOrigins: strings.Split(ALLOWED_ORIGINS, "&"),
		AllowMethods: []string{"GET", "POST", "DELETE"},
		AllowHeaders: []string{"Content-Type"},
	}))

	if DEBUG == "1" {
		// serve docs/index.html as static file at /docs
		r.LoadHTMLFiles("docs/index.html")
		r.GET("/docs", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", gin.H{
				"DEBUG":                   DEBUG,
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

	r.POST("/signup", func(c *gin.Context) {
		var data AuthRequest
		err := c.BindJSON(&data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		code, uid, err := create_account(&data)
		if err != nil {
			c.JSON(code, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(code, gin.H{
			"uid": uid.String(),
		})
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

		code, uid, err := login(&data)
		if err != nil {
			c.JSON(code, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(code, gin.H{
			"uid": uid.String(),
		})
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

		code, err := resetPassword(&data)
		if err != nil {
			c.JSON(code, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(code, gin.H{})
	})

	r.DELETE("/delete/:uid", func(c *gin.Context) {
		uid := c.Param("uid")
		if uid == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "No uid provided",
			})
			return
		}

		code, err := delete(uid)
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

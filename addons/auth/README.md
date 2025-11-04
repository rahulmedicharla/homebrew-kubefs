
# Headless OAuth2 Server with Optional TOTP 2FA

This project implements a headless authentication server using Go and the Gin framework. It provides endpoints for user authentication, account creation, password reset, and account deletion.

## Overview

The authentication server is designed to be self-contained, meaning it does not rely on any external services for user authentication and token management. It uses PostgreSQL for storing user accounts and maintaing encryption at rest. 
## Endpoints

### Health Check

- **URL:** `/health`
- **Method:** `GET`
- **Description:** Checks the health of the server.
- **Response:**
  - `200 OK` with JSON `{ "status": "success" }`

### Sign Up

- **URL:** `/signup`
- **Method:** `POST`
- **Description:** Creates a new user account.
- **Request Body:**
  ```json
  {
    "email": "user@example.com",
    "password": "password123",
    "confirmPassword": "password123"
  }
  ```
- **Headers**
  - `Content-Type: application/json`
- **Response:**
  - `200 OK` with JSON `{ "uid": "user_id" }`
  - `400 Bad Request` with JSON `{ "error": "error message" }`
  - `409 Conflict` with JSON `{ "error": "Account already exists" }`

### Login

- **URL:** `/login`
- **Method:** `POST`
- **Description:** Authenticates a user and returns access and refresh tokens.
- **Request Body:**
  ```json
  {
    "email": "user@example.com",
    "password": "password123"
  }
  ```
- **Headers**
  - `Content-Type: application/json`
- **Response:**
  - `200 OK` with JSON `{ "uid": "user_id" }`
  - `400 Bad Request` with JSON `{ "error": "error message" }`

### Reset Password

- **URL:** `/resetpassword`
- **Method:** `POST`
- **Description:** Resets the user's password.
- **Request Body:**
  ```json
  {
    "email": "user@example.com",
    "newPassword": "newpassword123",
    "confirmNewPassword": "newpassword123"
  }
  ```
- **Headers**
  - `Content-Type: application/json`
- **Response:**
  - `200 OK` with JSON `{}`
  - `400 Bad Request` with JSON `{ "error": "error message" }`

### Delete Account

- **URL:** `/delete/:uid`
- **Method:** `DELETE`
- **Description:** Deletes a user account.
- **Headers**
  - `Content-Type: application/json`
- **Response:**
  - `200 OK` with JSON `{}`
  - `400 Bad Request` with JSON `{ "error": "error message" }`

## Environment Variables

- `PORT`: The port on which the server will run (default: `3000`).
- `ALLOWED_ORIGINS`: Comma-separated list of allowed origins for CORS.
- `WRITE_CONNECTION_STRING`: Connection string for writing to the PostgreSQL database.
- `READ_CONNECTION_STRING`: Connection string for reading from the PostgreSQL database.
- `DEBUG`: A flag for whether to run in debug mode or not

## Running the Server

1. Ensure you have Go installed.
2. Set up the required environment variables.
3. Run the server:
   ```sh
   go run main.go
   ```

## Dependencies

- [Gin](https://github.com/gin-gonic/gin)
- [bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
- [cors](https://github.com/gin-contrib/cors)
- [PostgreSQL](https://github.com/jackc/pgx)

## License

This project is licensed under the MIT License.

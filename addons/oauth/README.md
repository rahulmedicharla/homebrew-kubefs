
# Self-Contained Headless OAuth2 Server

This project implements a self-contained headless OAuth2 server using Go and the Gin framework. It provides endpoints for user authentication, account creation, password reset, token verification, and token refresh.

## Overview

The OAuth2 server is designed to be self-contained, meaning it does not rely on any external services for user authentication and token management. It uses RSA keys for signing and verifying JWT tokens, and BadgerDB for storing user accounts and refresh tokens.

The issued JWT tokens have two claims. sub (a unique uuid) which you can use as a unique id for your user, and exp (the expiration time, 1 hour). 

This server has two required params passed in as environment variables. 1. PORT is the specified port to run on. 2. ALLOWED_ORIGINS. The server sets up a default cors policy that only allows request from these allowed origins, ie. the domain requesting the JWT and the protected resource.

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
    "confirm_password": "password123",
    "security_question": "Your favorite color?",
    "security_answer": "Blue"
  }
  ```
- **Response:**
  - `200 OK` with JSON `{ "status": "success", "access_token": "token", "refresh_token": "token" }`
  - `400 Bad Request` with JSON `{ "status": "error", "message": "error message" }`

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
- **Response:**
  - `200 OK` with JSON `{ "status": "success", "access_token": "token", "refresh_token": "token" }`
  - `400 Bad Request` with JSON `{ "status": "error", "message": "error message" }`

### Forgot Password

- **URL:** `/forgotpassword`
- **Method:** `POST`
- **Description:** Resets the user's password.
- **Request Body:**
  ```json
  {
    "email": "user@example.com",
    "security_question": "Your favorite color?",
    "security_answer": "Blue",
    "new_password": "newpassword123",
    "confirm_new_password": "newpassword123"
  }
  ```
- **Response:**
  - `200 OK` with JSON `{ "status": "success", "message": "Password reset successful" }`
  - `400 Bad Request` with JSON `{ "status": "error", "message": "error message" }`

### Delete Account

- **URL:** `/delete/:email`
- **Method:** `DELETE`
- **Description:** Deletes a user account.
- **Response:**
  - `200 OK` with JSON `{ "status": "success" }`
  - `400 Bad Request` with JSON `{ "status": "error", "message": "error message" }`

### Refresh Token

- **URL:** `/refresh/:uid`
- **Method:** `GET`
- **Description:** Refreshes the access token using the refresh token.
- **Headers:**
  - `Authorization: Bearer <refresh_token>`
- **Response:**
  - `200 OK` with JSON `{ "status": "success", "access_token": "new_access_token" }`
  - `400 Bad Request` with JSON `{ "status": "error", "message": "error message" }`

### Verify Token

- **URL:** `/verify/:token`
- **Method:** `GET`
- **Description:** Verifies the validity of an access token.
- **Response:**
  - `200 OK` with JSON `{ "status": "success", "message": "Token is valid" }`
  - `400 Bad Request` with JSON `{ "status": "error", "message": "error message" }`

## Environment Variables

- `PORT`: The port on which the server will run (default: `3000`).
- `ALLOWED_ORIGINS`: Comma-separated list of allowed origins for CORS.

## Running the Server

1. Ensure you have Go installed.
2. Set up the required environment variables.
3. Run the server:
   ```sh
   go run main.go
   ```

## Dependencies

- [Gin](https://github.com/gin-gonic/gin)
- [Badger](https://github.com/dgraph-io/badger)
- [JWT](https://github.com/golang-jwt/jwt)
- [bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
- [cors](https://github.com/gin-contrib/cors)

## License

This project is licensed under the MIT License.

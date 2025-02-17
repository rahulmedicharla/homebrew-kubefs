
# Headless OAuth2 Server with Optional TOTP 2FA

This project implements a headless OAuth2 server using Go and the Gin framework. It provides endpoints for user authentication, account creation, password reset, token verification, and token refresh. Additionally, it supports optional TOTP-based two-factor authentication (2FA).

## Overview

The OAuth2 server is designed to be self-contained, meaning it does not rely on any external services for user authentication and token management. It uses RSA keys for signing and verifying JWT tokens, and PostgreSQL for storing user accounts and refresh tokens. Optional TOTP 2FA can be enabled for added security.

## Endpoints

### Health Check

- **URL:** `/health`
- **Method:** `GET`
- **Description:** Checks the health of the server.
- **Response:**
  - `200 OK` with JSON `{ "status": "success" }`

### Generate TOTP Account

- **URL:** `/generateTOTPAccount/:email`
- **Method:** `GET`
- **Description:** Generates a TOTP account for the given email.
- **Response:**
  - `200 OK` with JSON `{ "url": "totp_url" }`
  - `400 Bad Request` with JSON `{ "error": "error message" }`

### Sign Up

- **URL:** `/signup`
- **Method:** `POST`
- **Description:** Creates a new user account.
- **Request Body:**
  ```json
  {
    "email": "user@example.com",
    "password": "password123",
    "confirm_password": "password123"
  }
  ```
- **Headers (if 2FA is enabled):**
  - `Authorization: Bearer <2FA_code>`
- **Response:**
  - `200 OK` with JSON `{ "access_token": "token", "refresh_token": "token", "uid": "user_id" }`
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
- **Headers (if 2FA is enabled):**
  - `Authorization: Bearer <2FA_code>`
- **Response:**
  - `200 OK` with JSON `{ "access_token": "token", "refresh_token": "token", "uid": "user_id" }`
  - `400 Bad Request` with JSON `{ "error": "error message" }`

### Reset Password

- **URL:** `/resetpassword`
- **Method:** `POST`
- **Description:** Resets the user's password.
- **Request Body:**
  ```json
  {
    "email": "user@example.com",
    "new_password": "newpassword123",
    "confirm_new_password": "newpassword123"
  }
  ```
- **Headers (if 2FA is enabled):**
  - `Authorization: Bearer <2FA_code>`
- **Response:**
  - `200 OK` with JSON `{}`
  - `400 Bad Request` with JSON `{ "error": "error message" }`

### Delete Account

- **URL:** `/delete/:uid`
- **Method:** `DELETE`
- **Description:** Deletes a user account.
- **Headers (if 2FA is enabled):**
  - `Authorization: Bearer <2FA_code>`
- **Response:**
  - `200 OK` with JSON `{}`
  - `400 Bad Request` with JSON `{ "error": "error message" }`

### Refresh Token

- **URL:** `/refresh/:uid`
- **Method:** `GET`
- **Description:** Refreshes the access token using the refresh token.
- **Headers:**
  - `Authorization: Bearer <refresh_token>`
- **Response:**
  - `200 OK` with JSON `{ "access_token": "new_access_token" }`
  - `400 Bad Request` with JSON `{ "error": "error message" }`

### Verify Token

- **URL:** `/verify/:token`
- **Method:** `GET`
- **Description:** Verifies the validity of an access token.
- **Response:**
  - `200 OK` with JSON `{}`
  - `400 Bad Request` with JSON `{ "error": "error message" }`

## Environment Variables

- `PORT`: The port on which the server will run (default: `3000`).
- `ALLOWED_ORIGINS`: Comma-separated list of allowed origins for CORS.
- `WRITE_CONNECTION_STRING`: Connection string for writing to the PostgreSQL database.
- `READ_CONNECTION_STRING`: Connection string for reading from the PostgreSQL database.
- `MODE`: The mode in which the server runs (`release`, `init`, or `dev`).
- `TWO_FACTOR_AUTH`: Enable or disable TOTP 2FA (`true` or `false`).
- `NAME`: The name of the service.

## Running the Server

1. Ensure you have Go installed.
2. Set up the required environment variables.
3. Run the server:
   ```sh
   go run main.go
   ```

## Dependencies

- [Gin](https://github.com/gin-gonic/gin)
- [JWT](https://github.com/golang-jwt/jwt)
- [bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
- [cors](https://github.com/gin-contrib/cors)
- [TOTP](https://github.com/pquerna/otp)
- [PostgreSQL](https://github.com/jackc/pgx)

## License

This project is licensed under the MIT License.

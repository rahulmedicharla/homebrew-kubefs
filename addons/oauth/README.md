
# Headless OAuth2 Server

This is a headless OAuth2 server implemented using Go and the Gin framework. It provides endpoints for user authentication, account creation, password reset, and token refresh.

Note. This addon mounts a public_key.pem file into the specified verification resource at the path /app/public_key.pem in the container for you to verify the JWT Tokens with 

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

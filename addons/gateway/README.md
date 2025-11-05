
# Centralized Client Manager

This addon implements a centralized client credentials manager using Go and the Gin framework. It provides endpoints for JWT signed Oauth2 tokens and oauth2 token verification.

## Overview

The client credentials manager acts a centralized api that attached resources can make requests to using their client credentials passed in as env variables. The manager uses scrypt to keep the client information encrypted at rest and stores them as a secret in kubernetes.
## Endpoints

### Health Check

- **URL:** `/health`
- **Method:** `GET`
- **Description:** Checks the health of the server.
- **Response:**
  - `200 OK` with JSON `{ "status": "success" }`

### Auth

- **URL:** `/auth`
- **Method:** `POST`
- **Description:** issues a new access token.
- **Request Body:**
  ```json
  {
    "clientID": "client-id",
    "clientSecret": "client-secret",
  }
  ```
- **Headers**
  - `Content-Type: application/json`
- **Response:**
  - `200 OK` with JSON `{ "accessToken": "accessToken" }`
  - `400 Bad Request` with JSON `{ "error": "error message" }`
  - `401 Unauthorized` with JSON `{ "error": "error message" }`

### Verify

- **URL:** `/verify`
- **Method:** `POST`
- **Description:** validates provided access token.
- **Request Body:**
  ```json
  {
    "clientID": "client-id",
    "clientSecret": "client-secret",
    "accessToken": "access-token"
  }
  ```
- **Headers**
  - `Content-Type: application/json`
- **Response:**
  - `200 OK` with JSON `{}`
  - `400 Bad Request` with JSON `{ "error": "error message" }`
  - `401 Unauthorized` with JSON `{ "error": "error message" }`

## Environment Variables

- `PORT`: The port on which the server will run (default: `3000`).
- `PRIVATE_KEY_PATH`: The path for where the openssl private key.pem file resides.
- `PUBLIC_KEY_PATH`: The path for where the openssl public key.pem file resides.
- `CLIENTS`: A string of clients ids & secrets for validation in the format clientId1:encryptedClientSecret1&clientId2encryptedClientSecret2.
- `ALLOWED_ORIGINS`: Comma-separated list of allowed origins for CORS.
- `DEBUG`: A flag for whether to run in debug mode or not

## Running the Server

1. Ensure you have Go installed.
2. Set up the required environment variables.
3. Run the server:
   ```sh
   go run main.go
   ```

## Dependencies

- [cors](https://github.com/gin-contrib/cors)
- [Gin](https://github.com/gin-gonic/gin)
- [jwt]("github.com/golang-jwt/jwt/v5")
- [dotenv]("github.com/joho/godotenv")
- [scrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt)

## License

This project is licensed under the MIT License.


# OAuth Service

This service provides OAuth functionalities including user signup, login, password reset, and token refresh using JWT and BadgerDB.

## Endpoints

### GET /

Serves the login page.

### GET /login

Serves the login page with optional error and success messages.

### GET /signup

Serves the signup page with optional error messages.

### GET /forgotpassword

Serves the forgot password page with optional error messages.

### GET /health

Returns the health status of the service.

### POST /signup

Creates a new user account.

**Request Parameters:**
- `email`: User's email address.
- `password`: User's password.
- `confirmPassword`: Confirmation of the user's password.
- `securityQuestion`: Security question for password recovery.
- `securityAnswer`: Answer to the security question.

**Response:**
- Redirects to the specified `REDIRECT_URL` with access and refresh tokens on success.
- Redirects to `/signup` with an error message on failure.

### POST /login

Logs in an existing user.

**Request Parameters:**
- `email`: User's email address.
- `password`: User's password.

**Response:**
- Redirects to the specified `REDIRECT_URL` with access and refresh tokens on success.
- Redirects to `/login` with an error message on failure.

### POST /forgotpassword

Resets the user's password.

**Request Parameters:**
- `email`: User's email address.
- `newPassword`: New password.
- `confirmNewPassword`: Confirmation of the new password.
- `securityQuestion`: Security question for password recovery.
- `securityAnswer`: Answer to the security question.

**Response:**
- Redirects to `/login` with a success message on success.
- Redirects to `/forgotpassword` with an error message on failure.

### DELETE /delete/:email

Deletes a user account.

**Request Parameters:**
- `email`: User's email address.

**Response:**
- JSON response with status and message.

### GET /refresh/:uid

Refreshes the access token using the refresh token.

**Request Headers:**
- `Authorization`: Bearer token containing the refresh token.

**Response:**
- JSON response with status and new access token.

## Functions

### issueAccessToken(uid string) (int, string)

Issues a new JWT access token for the given user ID.

### create_account(data *AuthRequest) (int, string, string)

Creates a new user account.

### login(data *AuthRequest) (int, string, string)

Logs in an existing user.

### delete(email string) (int, string)

Deletes a user account.

### refresh(refreshToken string, uid string) (int, string)

Refreshes the access token using the refresh token.

### resetPassword(email string, newPassword string, confirmNewPassword string, securityQuestion, securityAnswer string) (int, string)

Resets the user's password.

# API Testing Instructions

The server is now running with the following authentication APIs implemented:

## Available APIs

### 1. POST /auth/login
**Description**: Login with username and password
**Body**:
```json
{
  "username": "admin",
  "password": "admin123",
  "rememberMe": true
}
```

### 2. POST /auth/refresh
**Description**: Refresh access token using refresh token
**Body**:
```json
{
  "refresh_token": "your_refresh_token_here"
}
```

### 3. POST /auth/logout
**Description**: Logout and invalidate refresh token
**Body**:
```json
{
  "refresh_token": "your_refresh_token_here"
}
```

### 4. GET /user
**Description**: Get current user information
**Headers**: 
```
Authorization: Bearer your_access_token_here
```

### 5. GET /user/menu
**Description**: Get user menu structure
**Headers**: 
```
authorization: Bearer your_access_token_here
```

## Test with curl

### Login:
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123","rememberMe":true}'
```

### Get User Info (replace YOUR_TOKEN with the access_token from login):
```bash
curl -X GET http://localhost:8080/user \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Get Menu (replace YOUR_TOKEN with the access_token from login):
```bash
curl -X GET http://localhost:8080/user/menu \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Logout:
```bash
curl -X POST http://localhost:8080/auth/logout \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"YOUR_REFRESH_TOKEN"}'
```

## Default Login Credentials
- **Username**: admin
- **Password**: admin123

## Database
The server uses SQLite database (`webadmin.db`) to store:
- User information
- Refresh tokens

## Features Implemented
- ✅ JWT-based authentication
- ✅ Access tokens (24-hour expiry)
- ✅ Refresh tokens (7-day expiry)
- ✅ Password hashing with bcrypt
- ✅ SQLite database integration
- ✅ CORS support
- ✅ Protected routes with middleware
- ✅ Default admin user creation
- ✅ Menu structure API

## Angular Frontend Integration
The Angular Matero app's `LoginService` will automatically work with these APIs since they match the expected endpoints:
- `/auth/login`
- `/auth/refresh`
- `/auth/logout`
- `/user`
- `/user/menu`

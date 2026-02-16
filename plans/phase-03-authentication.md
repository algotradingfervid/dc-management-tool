# Phase 3: User Authentication

## Overview

Implement username/password authentication using a focused library stack:
- **alexedwards/scs/v2** — server-side session management with SQLite-backed storage
- **golang.org/x/crypto/bcrypt** — password hashing
- **gorilla/csrf** — CSRF protection for all forms

Sessions are stored in the SQLite database (not cookies), providing server-side control over session lifecycle, easy invalidation, and no cookie size limits.

## Prerequisites

- Phase 1 completed (project scaffolding)
- Phase 2 completed (database schema with users table)
- Libraries installed:
  - `github.com/alexedwards/scs/v2`
  - `github.com/alexedwards/scs/sqlite3store`
  - `github.com/gorilla/csrf`
  - `golang.org/x/crypto`

## Goals

- Implement secure password hashing with bcrypt
- Create server-side session management using SCS with SQLite store
- Add CSRF protection to all POST forms
- Build login page with form validation
- Implement logout functionality with session cleanup
- Create authentication middleware for route protection
- Handle session timeout and expiration
- Display user information in application header
- No role-based access control (all authenticated users have equal access)
- Redirect unauthenticated users to login page

## Detailed Implementation Steps

### 1. Create Sessions Table

1.1. Add sessions table migration (SCS requires this):
```sql
CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    data BLOB NOT NULL,
    expiry REAL NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions(expiry);
```

### 2. Create Authentication Utilities

2.1. Create password hashing utilities in `internal/auth/password.go`
- Hash password using bcrypt
- Verify password against hash

2.2. Create session manager in `internal/auth/session.go`
- Initialize SCS session manager with SQLite store
- Configure session lifetime, cookie options
- Provide helper functions for session data

### 3. Create User Model and Repository

3.1. Create user model in `internal/models/user.go`
- User struct matching database schema

3.2. Create user repository in `internal/database/users.go`
- GetUserByUsername (for login)
- GetUserByID (for session restoration)
- CreateUser (for future user management)

### 4. Create Authentication Handlers

4.1. Create login handler in `internal/handlers/auth.go`
- GET /login — show login form with CSRF token
- POST /login — process login credentials
- GET /logout — destroy session and redirect to login

### 5. Create Authentication Middleware

5.1. Create auth middleware in `internal/middleware/auth.go`
- Check if user is authenticated (session has user_id)
- Load user data from database
- Store user in Gin context
- Redirect to login if not authenticated

### 6. Create Login Page Template

6.1. Create login template with CSRF token field in all forms

### 7. Update Main Application

7.1. Update `cmd/server/main.go`
- Initialize SCS session manager with SQLite store
- Wrap router with SCS middleware (via `scs.LoadAndSave`)
- Wrap router with CSRF middleware
- Set up public vs protected route groups

## Files to Create/Modify

### `/Users/narendhupati/Documents/ProjectManagementTool/migrations/003_sessions_table.sql`
```sql
-- Sessions table for SCS session manager
CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    data BLOB NOT NULL,
    expiry REAL NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions(expiry);
```

### `/Users/narendhupati/Documents/ProjectManagementTool/internal/auth/password.go`
```go
package auth

import (
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 10

// HashPassword generates bcrypt hash from password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword checks if password matches hash
func VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
```

### `/Users/narendhupati/Documents/ProjectManagementTool/internal/auth/session.go`
```go
package auth

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
)

const (
	sessionUserIDKey = "user_id"
	sessionFlashKey  = "flash_message"
	sessionFlashType = "flash_type"
)

// SessionManager is the global SCS session manager
var SessionManager *scs.SessionManager

// InitSessionManager creates and configures the SCS session manager with SQLite store
func InitSessionManager(db *sql.DB, secure bool) {
	SessionManager = scs.New()
	SessionManager.Store = sqlite3store.New(db)
	SessionManager.Lifetime = 7 * 24 * time.Hour // 7 days
	SessionManager.IdleTimeout = 24 * time.Hour   // 24 hours inactivity
	SessionManager.Cookie.Name = "dc_session"
	SessionManager.Cookie.HttpOnly = true
	SessionManager.Cookie.SameSite = http.SameSiteLaxMode
	SessionManager.Cookie.Secure = secure // true in production with HTTPS
}

// SetUserID stores the authenticated user ID in the session
func SetUserID(r *http.Request, userID int) {
	SessionManager.Put(r.Context(), sessionUserIDKey, userID)
}

// GetUserID retrieves the authenticated user ID from the session
func GetUserID(r *http.Request) int {
	return SessionManager.GetInt(r.Context(), sessionUserIDKey)
}

// SetFlash sets a flash message in the session
func SetFlash(r *http.Request, msgType, message string) {
	SessionManager.Put(r.Context(), sessionFlashKey, message)
	SessionManager.Put(r.Context(), sessionFlashType, msgType)
}

// PopFlash retrieves and clears the flash message from the session
func PopFlash(r *http.Request) (string, string) {
	message := SessionManager.PopString(r.Context(), sessionFlashKey)
	msgType := SessionManager.PopString(r.Context(), sessionFlashType)
	return msgType, message
}

// DestroySession destroys the current session
func DestroySession(r *http.Request) error {
	return SessionManager.Destroy(r.Context())
}

// RenewToken regenerates the session token (use after login to prevent session fixation)
func RenewToken(r *http.Request) error {
	return SessionManager.RenewToken(r.Context())
}
```

### `/Users/narendhupati/Documents/ProjectManagementTool/internal/auth/context.go`
```go
package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

const contextUserKey = "current_user"

// GetCurrentUser retrieves the current user from Gin context
func GetCurrentUser(c *gin.Context) *models.User {
	if user, exists := c.Get(contextUserKey); exists {
		if u, ok := user.(*models.User); ok {
			return u
		}
	}
	return nil
}

// SetCurrentUser stores the current user in Gin context
func SetCurrentUser(c *gin.Context, user *models.User) {
	c.Set(contextUserKey, user)
}
```

### `/Users/narendhupati/Documents/ProjectManagementTool/internal/models/user.go`
```go
package models

import (
	"time"
)

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never serialize password hash
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
```

### `/Users/narendhupati/Documents/ProjectManagementTool/internal/database/users.go`
```go
package database

import (
	"database/sql"
	"fmt"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

// GetUserByUsername retrieves a user by username
func GetUserByUsername(username string) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, full_name, email, created_at, updated_at
		FROM users
		WHERE username = ?
	`

	user := &models.User{}
	err := DB.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.FullName,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func GetUserByID(id int) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, full_name, email, created_at, updated_at
		FROM users
		WHERE id = ?
	`

	user := &models.User{}
	err := DB.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.FullName,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// CreateUser creates a new user
func CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (username, password_hash, full_name, email)
		VALUES (?, ?, ?, ?)
	`

	result, err := DB.Exec(query, user.Username, user.PasswordHash, user.FullName, user.Email)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	user.ID = int(id)
	return nil
}
```

### `/Users/narendhupati/Documents/ProjectManagementTool/internal/handlers/auth.go`
```go
package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
)

// ShowLogin displays the login page
func ShowLogin(c *gin.Context) {
	// If already logged in, redirect to dashboard
	if userID := auth.GetUserID(c.Request); userID != 0 {
		c.Redirect(http.StatusFound, "/")
		return
	}

	// Get flash message if any
	flashType, flashMsg := auth.PopFlash(c.Request)

	c.HTML(http.StatusOK, "login.html", gin.H{
		"csrfField":  csrf.TemplateField(c.Request),
		"flashType":  flashType,
		"flashMsg":   flashMsg,
	})
}

// ProcessLogin handles login form submission
func ProcessLogin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Validate input
	if username == "" || password == "" {
		auth.SetFlash(c.Request, "error", "Username and password are required")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Get user from database
	user, err := database.GetUserByUsername(username)
	if err != nil {
		log.Printf("Login failed for username %s: %v", username, err)
		auth.SetFlash(c.Request, "error", "Invalid username or password")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Verify password
	if !auth.VerifyPassword(user.PasswordHash, password) {
		log.Printf("Invalid password for username %s", username)
		auth.SetFlash(c.Request, "error", "Invalid username or password")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Renew session token to prevent session fixation
	if err := auth.RenewToken(c.Request); err != nil {
		log.Printf("Failed to renew session token: %v", err)
	}

	// Store user ID in session
	auth.SetUserID(c.Request, user.ID)
	auth.SetFlash(c.Request, "success", "Login successful")

	log.Printf("User %s logged in successfully", username)
	c.Redirect(http.StatusFound, "/")
}

// Logout handles user logout
func Logout(c *gin.Context) {
	userID := auth.GetUserID(c.Request)

	// Destroy session
	if err := auth.DestroySession(c.Request); err != nil {
		log.Printf("Failed to destroy session: %v", err)
	}

	log.Printf("User ID %d logged out", userID)
	c.Redirect(http.StatusFound, "/login")
}
```

### `/Users/narendhupati/Documents/ProjectManagementTool/internal/middleware/auth.go`
```go
package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
)

// RequireAuth is middleware that requires user to be authenticated
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if user ID exists in session
		userID := auth.GetUserID(c.Request)
		if userID == 0 {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Load user from database
		user, err := database.GetUserByID(userID)
		if err != nil {
			log.Printf("Failed to load user from session: %v", err)
			// Destroy invalid session
			auth.DestroySession(c.Request)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Store user in context
		auth.SetCurrentUser(c, user)

		c.Next()
	}
}
```

### Update `/Users/narendhupati/Documents/ProjectManagementTool/cmd/server/main.go`
```go
package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/config"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/handlers"
	"github.com/narendhupati/dc-management-tool/internal/middleware"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Init(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db, "./migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize SCS session manager with SQLite store
	isSecure := cfg.Environment == "production"
	auth.InitSessionManager(db, isSecure)

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.Default()

	// Load templates
	router.LoadHTMLGlob("templates/*")

	// Serve static files
	router.Static("/static", "./static")

	// Public routes
	router.GET("/login", handlers.ShowLogin)
	router.POST("/login", handlers.ProcessLogin)
	router.GET("/logout", handlers.Logout)
	router.GET("/health", handlers.HealthCheck)

	// Protected routes
	protected := router.Group("/")
	protected.Use(middleware.RequireAuth())
	{
		protected.GET("/", func(c *gin.Context) {
			user := auth.GetCurrentUser(c)
			c.HTML(http.StatusOK, "base.html", gin.H{
				"user":  user,
				"title": "Dashboard",
			})
		})
	}

	// Wrap with SCS session middleware + CSRF middleware
	csrfMiddleware := csrf.Protect(
		[]byte(cfg.SessionSecret),
		csrf.Secure(isSecure),
		csrf.SameSite(csrf.SameSiteLaxMode),
	)

	// Stack: CSRF wraps SCS wraps Gin router
	handler := csrfMiddleware(auth.SessionManager.LoadAndSave(router))

	// Start server
	log.Printf("Starting server on %s in %s mode", cfg.ServerAddress, cfg.Environment)
	if err := http.ListenAndServe(cfg.ServerAddress, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

### `/Users/narendhupati/Documents/ProjectManagementTool/templates/login.html`
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login - DC Management Tool</title>

    <!-- Tailwind CSS CDN -->
    <script src="https://cdn.tailwindcss.com"></script>

    <!-- Tailwind Config -->
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        brand: {
                            50: '#eff6ff',
                            100: '#dbeafe',
                            200: '#bfdbfe',
                            300: '#93c5fd',
                            400: '#60a5fa',
                            500: '#3b82f6',
                            600: '#2563eb',
                            700: '#1d4ed8',
                            800: '#1e40af',
                            900: '#1e3a8a',
                        }
                    }
                }
            }
        }
    </script>
</head>
<body class="bg-gradient-to-br from-brand-50 to-brand-100 min-h-screen">
    <div class="min-h-screen flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div class="max-w-md w-full space-y-8">
            <!-- Logo/Header -->
            <div class="text-center">
                <h1 class="text-4xl font-bold text-brand-900 mb-2">
                    DC Management
                </h1>
                <p class="text-gray-600">
                    Delivery Challan Management System
                </p>
            </div>

            <!-- Login Card -->
            <div class="bg-white rounded-lg shadow-xl p-8">
                <h2 class="text-2xl font-semibold text-gray-800 mb-6 text-center">
                    Sign In
                </h2>

                <!-- Flash Messages -->
                {{ if .flashMsg }}
                    <div class="mb-4 p-4 rounded-lg {{ if eq .flashType "error" }}bg-red-50 text-red-800 border border-red-200{{ else if eq .flashType "success" }}bg-green-50 text-green-800 border border-green-200{{ else }}bg-blue-50 text-blue-800 border border-blue-200{{ end }}">
                        <p class="text-sm">{{ .flashMsg }}</p>
                    </div>
                {{ end }}

                <!-- Login Form -->
                <form method="POST" action="/login" class="space-y-6">
                    <!-- CSRF Token -->
                    {{ .csrfField }}

                    <!-- Username -->
                    <div>
                        <label for="username" class="block text-sm font-medium text-gray-700 mb-2">
                            Username
                        </label>
                        <input
                            type="text"
                            id="username"
                            name="username"
                            required
                            autofocus
                            autocomplete="username"
                            class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-transparent transition duration-200"
                            placeholder="Enter your username"
                        >
                    </div>

                    <!-- Password -->
                    <div>
                        <label for="password" class="block text-sm font-medium text-gray-700 mb-2">
                            Password
                        </label>
                        <input
                            type="password"
                            id="password"
                            name="password"
                            required
                            autocomplete="current-password"
                            class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-transparent transition duration-200"
                            placeholder="Enter your password"
                        >
                    </div>

                    <!-- Submit Button -->
                    <button
                        type="submit"
                        class="w-full bg-brand-600 hover:bg-brand-700 text-white font-semibold py-3 px-4 rounded-lg transition duration-200 shadow-lg hover:shadow-xl"
                    >
                        Sign In
                    </button>
                </form>

                <!-- Test Credentials Info -->
                <div class="mt-6 p-4 bg-gray-50 rounded-lg border border-gray-200">
                    <p class="text-xs text-gray-600 mb-2 font-semibold">Test Credentials:</p>
                    <p class="text-xs text-gray-500">Username: <span class="font-mono">admin</span></p>
                    <p class="text-xs text-gray-500">Password: <span class="font-mono">password123</span></p>
                </div>
            </div>

            <!-- Footer -->
            <div class="text-center text-sm text-gray-600">
                <p>Internal Use Only</p>
            </div>
        </div>
    </div>
</body>
</html>
```

## API Routes / Endpoints

| Method | Path | Handler | Auth Required | Description |
|--------|------|---------|---------------|-------------|
| GET | /login | handlers.ShowLogin | No | Display login form |
| POST | /login | handlers.ProcessLogin | No | Process login credentials (CSRF protected) |
| GET | /logout | handlers.Logout | No | Destroy session and redirect |
| GET | / | Dashboard placeholder | Yes | Protected home page |
| GET | /health | handlers.HealthCheck | No | Health check endpoint |

## Database Queries

```sql
-- Sessions table (managed by SCS automatically)
CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    data BLOB NOT NULL,
    expiry REAL NOT NULL
);

-- Get user by username (login)
SELECT id, username, password_hash, full_name, email, created_at, updated_at
FROM users
WHERE username = ?;

-- Get user by ID (session restoration)
SELECT id, username, password_hash, full_name, email, created_at, updated_at
FROM users
WHERE id = ?;

-- Create new user (future user management)
INSERT INTO users (username, password_hash, full_name, email)
VALUES (?, ?, ?, ?);
```

## Key Architecture Decisions

### Why SCS over gorilla/sessions

| Feature | gorilla/sessions | SCS |
|---------|-----------------|-----|
| Session storage | Cookie (client-side) | SQLite database (server-side) |
| Session size limit | ~4KB (cookie limit) | Unlimited |
| Server-side invalidation | Not possible | Yes — delete from DB |
| Session fixation protection | Manual | Built-in `RenewToken()` |
| Idle timeout | Not supported | Built-in |
| Automatic cleanup | No | Yes — expired sessions auto-purged |
| Gin compatibility | Native | Via `http.Handler` wrapping |

### Why gorilla/csrf

- Drop-in CSRF middleware compatible with `net/http`
- Generates per-request tokens embedded in forms via `csrf.TemplateField()`
- Validates automatically on POST/PUT/DELETE
- Works seamlessly with SCS sessions

### Middleware Stack Order

```
HTTP Request → CSRF Middleware → SCS Session Middleware → Gin Router → Handlers
```

SCS wraps the Gin router as `http.Handler`, and CSRF wraps that. This means:
- Sessions are loaded/saved automatically per request
- CSRF tokens are validated before reaching handlers
- No manual `session.Save()` calls needed

## UI Components

### Login Page
- **Route**: GET /login
- **Layout**: Full-page centered card with gradient background
- **Components**:
  - Application logo/title
  - Hidden CSRF token field (auto-injected via `{{ .csrfField }}`)
  - Username and password fields
  - Submit button (brand-colored)
  - Flash message display area (error/success)
  - Test credentials info box (development only)
  - Footer with "Internal Use Only"

### CSRF Token in Forms
All POST forms must include `{{ .csrfField }}` which renders a hidden input:
```html
<input type="hidden" name="gorilla.csrf.Token" value="...">
```

For HTMX requests (later phases), include the CSRF token as a header:
```html
<meta name="csrf-token" content="{{ .csrfToken }}">
```
```javascript
document.body.addEventListener('htmx:configRequest', function(evt) {
    evt.detail.headers['X-CSRF-Token'] = document.querySelector('meta[name="csrf-token"]').content;
});
```

## Testing Checklist

### Manual Testing

- [ ] Access / without login redirects to /login
- [ ] Login page displays correctly with proper styling
- [ ] Login with valid credentials (admin/password123) succeeds
- [ ] Successful login redirects to /
- [ ] Login with invalid username shows error message
- [ ] Login with invalid password shows error message
- [ ] Login with empty fields shows error message
- [ ] Flash messages display correctly and clear after one view
- [ ] Session persists across page refreshes
- [ ] Access /logout destroys session and redirects to /login
- [ ] After logout, accessing / redirects back to /login
- [ ] Already logged-in user accessing /login redirects to /
- [ ] Session expires after 7 days
- [ ] Idle session expires after 24 hours of inactivity
- [ ] Multiple browser tabs share same session

### Security Testing

- [ ] Password is hashed with bcrypt (check database)
- [ ] Password hash never appears in logs or responses
- [ ] Session cookie has HttpOnly flag set
- [ ] Session cookie has SameSite=Lax set
- [ ] CSRF token is present in login form HTML
- [ ] POST /login without CSRF token returns 403 Forbidden
- [ ] Sessions are stored in SQLite (check sessions table)
- [ ] Session token changes after login (session fixation prevention)
- [ ] Invalid session token doesn't crash application
- [ ] SQL injection attempts in username/password are handled safely
- [ ] Error messages don't leak user existence info

### Integration Testing

- [ ] Auth middleware correctly protects routes
- [ ] Current user is available in protected route handlers
- [ ] User data is correctly loaded from session
- [ ] Invalid user ID in session destroys session and redirects
- [ ] Database errors during login are handled gracefully
- [ ] SCS auto-cleans expired sessions from database

## Acceptance Criteria

- [x] Password hashing utility using bcrypt implemented
- [x] SCS session manager initialized with SQLite store
- [x] Sessions table created in database
- [x] CSRF protection enabled on all POST routes
- [x] User model and repository functions created
- [x] Login handler (GET and POST) implemented with CSRF token
- [x] Logout handler implemented with session destruction
- [x] Auth middleware protects routes
- [x] Current user accessible via context helpers
- [x] Login page styled with Tailwind (centered card)
- [x] Flash messages for errors and success
- [x] Session lifetime: 7 days, idle timeout: 24 hours
- [x] HttpOnly and SameSite flags set on session cookie
- [x] Session token renewed after login (prevent fixation)
- [x] Redirect to dashboard after successful login
- [x] Redirect to login when accessing protected routes unauthenticated
- [x] Error messages don't leak sensitive information

## Security Best Practices Implemented

1. **Password Security**
   - Bcrypt hashing with cost factor 10
   - Password hash never logged or returned in API responses
   - Password field excluded from JSON serialization

2. **Session Security**
   - Server-side session storage in SQLite (not in cookies)
   - Session token regeneration after login (prevents session fixation)
   - HttpOnly cookies (not accessible via JavaScript)
   - SameSite=Lax (CSRF defense-in-depth)
   - Secure flag ready for HTTPS in production
   - 7-day lifetime with 24-hour idle timeout
   - Automatic cleanup of expired sessions

3. **CSRF Protection**
   - Per-request CSRF tokens via gorilla/csrf
   - Automatic validation on all state-changing requests (POST/PUT/DELETE)
   - Token injected into forms via template helper

4. **Error Handling**
   - Generic error messages (don't leak user existence)
   - Failed login attempts logged server-side
   - Database errors don't expose schema information

## Configuration

### Environment Variables

```env
SESSION_SECRET=your-random-secret-key-minimum-32-characters
APP_ENV=production  # Set to production for HTTPS-only cookies
```

### Session Configuration

- **Lifetime**: 7 days
- **Idle Timeout**: 24 hours
- **Cookie Name**: dc_session
- **HttpOnly**: true
- **Secure**: false (development), true (production with HTTPS)
- **SameSite**: Lax

## Notes

- Test user credentials in seed data: admin/password123, john/password123, jane/password123
- Session secret should be 32+ characters in production
- SCS automatically cleans up expired sessions from the database
- No manual `session.Save()` calls needed — SCS handles this via middleware
- For HTMX requests in later phases, CSRF token must be sent as `X-CSRF-Token` header
- No "remember me" functionality in this phase (future enhancement)
- No password reset functionality (future enhancement)
- No user registration UI (users created manually for now)

## Future Enhancements

- Password reset via email
- User registration form
- Remember me checkbox (extend session duration)
- Two-factor authentication
- Rate limiting on login attempts
- Session management UI (view active sessions, revoke)
- Password complexity requirements
- Password change functionality
- Account lockout after failed attempts

## Next Steps

After completing Phase 3, proceed to:
- **Phase 4**: Shared UI Layout & Navigation Shell — create base template with sidebar navigation

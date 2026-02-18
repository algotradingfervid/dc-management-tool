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

var SessionManager *scs.SessionManager

func InitSessionManager(db *sql.DB, secure bool) {
	SessionManager = scs.New()
	SessionManager.Store = sqlite3store.New(db)
	SessionManager.Lifetime = 7 * 24 * time.Hour
	SessionManager.IdleTimeout = 24 * time.Hour
	SessionManager.Cookie.Name = "dc_session"
	SessionManager.Cookie.HttpOnly = true
	SessionManager.Cookie.SameSite = http.SameSiteLaxMode
	SessionManager.Cookie.Secure = secure
}

func SetUserID(r *http.Request, userID int) {
	SessionManager.Put(r.Context(), sessionUserIDKey, userID)
}

func GetUserID(r *http.Request) int {
	return SessionManager.GetInt(r.Context(), sessionUserIDKey)
}

func SetFlash(r *http.Request, msgType, message string) {
	SessionManager.Put(r.Context(), sessionFlashKey, message)
	SessionManager.Put(r.Context(), sessionFlashType, msgType)
}

func PopFlash(r *http.Request) (flashType, flashMsg string) {
	flashMsg = SessionManager.PopString(r.Context(), sessionFlashKey)
	flashType = SessionManager.PopString(r.Context(), sessionFlashType)
	return flashType, flashMsg
}

func DestroySession(r *http.Request) error {
	return SessionManager.Destroy(r.Context())
}

func RenewToken(r *http.Request) error {
	return SessionManager.RenewToken(r.Context())
}

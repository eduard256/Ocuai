package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

const (
	SessionName        = "ocuai_session"
	SessionMaxAge      = 24 * 3600 * 7 // 7 дней
	SessionUserIDKey   = "user_id"
	SessionUsernameKey = "username"
	SessionRoleKey     = "role"
	SessionCreatedKey  = "created_at"
)

// User представляет пользователя в системе
type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// Session представляет сессию пользователя
type Session struct {
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	CreatedAt int64  `json:"created_at"`
}

// AuthService предоставляет сервисы авторизации
type AuthService struct {
	db    *sql.DB
	store *sessions.CookieStore
}

// New создает новый сервис авторизации
func New(db *sql.DB, secretKey string) (*AuthService, error) {
	if secretKey == "" {
		// Используем фиксированный ключ для стабильности сессий
		secretKey = "ocuai-session-key-2024-stable-key-for-development-and-production-use"
	}

	store := sessions.NewCookieStore([]byte(secretKey))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   SessionMaxAge,
		HttpOnly: false,                // Отключаем для возможности доступа из JS
		Secure:   false,                // false для HTTP localhost
		SameSite: http.SameSiteLaxMode, // Lax для localhost
		Domain:   "",                   // Пустой домен для localhost
	}

	fmt.Printf("Session store configured: MaxAge=%d, HttpOnly=false, Secure=false\n", SessionMaxAge)

	service := &AuthService{
		db:    db,
		store: store,
	}

	// Создаем таблицу пользователей
	if err := service.createUsersTable(); err != nil {
		return nil, fmt.Errorf("failed to create users table: %w", err)
	}

	return service, nil
}

// createUsersTable создает таблицу пользователей
func (s *AuthService) createUsersTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
		CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
	`

	_, err := s.db.Exec(query)
	return err
}

// Register регистрирует нового пользователя
func (s *AuthService) Register(username, password string) (*User, error) {
	fmt.Printf("Registration attempt: username='%s', password_length=%d\n", username, len(password))

	// Проверяем, есть ли уже пользователи в системе
	hasUsers, err := s.HasUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to check existing users: %w", err)
	}

	if hasUsers {
		return nil, errors.New("registration is closed - admin already exists")
	}

	// Хэшируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	fmt.Printf("Password hashed successfully for user '%s'\n", username)

	// Первый пользователь всегда администратор
	role := "admin"

	// Создаем пользователя
	query := `INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`
	result, err := s.db.Exec(query, username, string(hashedPassword), role)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	fmt.Printf("User created successfully: id=%d, username='%s', role='%s'\n", userID, username, role)

	return &User{
		ID:       int(userID),
		Username: username,
		Role:     role,
	}, nil
}

// Login проверяет учетные данные и создает сессию
func (s *AuthService) Login(w http.ResponseWriter, r *http.Request, username, password string) (*Session, error) {
	fmt.Printf("=== AuthService.Login called ===\n")
	fmt.Printf("Login attempt: username='%s', password_length=%d\n", username, len(password))

	// Получаем пользователя из БД
	user, err := s.GetUserByUsername(username)
	if err != nil {
		fmt.Printf("Login failed: user not found for username '%s': %v\n", username, err)
		return nil, fmt.Errorf("invalid credentials")
	}

	fmt.Printf("User found in database: id=%d, username='%s', role='%s'\n", user.ID, user.Username, user.Role)

	// Проверяем пароль
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		fmt.Printf("Login failed: password mismatch for username '%s': %v\n", username, err)
		return nil, fmt.Errorf("invalid credentials")
	}

	fmt.Printf("Password verified successfully for user '%s'\n", username)

	// Создаем новую сессию с фиксированным ключом
	session := sessions.NewSession(s.store, SessionName)
	session.Options = s.store.Options
	session.IsNew = true

	sessionData := &Session{
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: time.Now().Unix(),
	}

	fmt.Printf("Setting session values...\n")
	session.Values[SessionUserIDKey] = sessionData.UserID
	session.Values[SessionUsernameKey] = sessionData.Username
	session.Values[SessionRoleKey] = sessionData.Role
	session.Values[SessionCreatedKey] = sessionData.CreatedAt

	fmt.Printf("Session values set, saving session...\n")
	if err := session.Save(r, w); err != nil {
		fmt.Printf("Failed to save session: %v\n", err)
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	fmt.Printf("Session saved successfully for user '%s'\n", username)
	return sessionData, nil
}

// Logout удаляет сессию пользователя
func (s *AuthService) Logout(w http.ResponseWriter, r *http.Request) error {
	session, err := s.store.Get(r, SessionName)
	if err != nil {
		return nil // Игнорируем ошибки при выходе
	}

	session.Options.MaxAge = -1
	return session.Save(r, w)
}

// GetSession получает текущую сессию пользователя
func (s *AuthService) GetSession(r *http.Request) (*Session, error) {
	fmt.Printf("=== GetSession called ===\n")

	session, err := s.store.Get(r, SessionName)
	if err != nil {
		fmt.Printf("GetSession: Failed to get session from store: %v\n", err)
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	fmt.Printf("GetSession: Session retrieved, checking values...\n")
	fmt.Printf("GetSession: Session.Values length: %d\n", len(session.Values))

	for key, value := range session.Values {
		fmt.Printf("GetSession: Session[%v] = %v (type: %T)\n", key, value, value)
	}

	userID, ok := session.Values[SessionUserIDKey].(int)
	if !ok {
		fmt.Printf("GetSession: UserID not found or invalid type in session\n")
		return nil, errors.New("invalid session")
	}

	username, ok := session.Values[SessionUsernameKey].(string)
	if !ok {
		fmt.Printf("GetSession: Username not found or invalid type in session\n")
		return nil, errors.New("invalid session")
	}

	role, ok := session.Values[SessionRoleKey].(string)
	if !ok {
		fmt.Printf("GetSession: Role not found or invalid type in session\n")
		return nil, errors.New("invalid session")
	}

	createdAt, ok := session.Values[SessionCreatedKey].(int64)
	if !ok {
		fmt.Printf("GetSession: CreatedAt not found or invalid type in session\n")
		return nil, errors.New("invalid session")
	}

	sessionData := &Session{
		UserID:    userID,
		Username:  username,
		Role:      role,
		CreatedAt: createdAt,
	}

	fmt.Printf("GetSession: Session data extracted successfully for user '%s'\n", username)
	fmt.Printf("=== GetSession completed successfully ===\n")

	return sessionData, nil
}

// RequireAuth middleware для проверки авторизации
func (s *AuthService) RequireAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := s.GetSession(r)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Добавляем сессию в контекст запроса
			ctx := r.Context()
			ctx = SetSessionContext(ctx, session)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole middleware для проверки роли пользователя
func (s *AuthService) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session := GetSessionFromContext(r.Context())
			if session == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if session.Role != role {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserByUsername получает пользователя по имени
func (s *AuthService) GetUserByUsername(username string) (*User, error) {
	query := `SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?`

	var user User
	err := s.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// HasUsers проверяет, есть ли пользователи в системе
func (s *AuthService) HasUsers() (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to count users: %w", err)
	}

	return count > 0, nil
}

// GetUserByID получает пользователя по ID
func (s *AuthService) GetUserByID(id int) (*User, error) {
	query := `SELECT id, username, password_hash, role, created_at FROM users WHERE id = ?`

	var user User
	err := s.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

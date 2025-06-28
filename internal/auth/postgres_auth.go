package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// PostgresAuthService предоставляет сервисы авторизации для PostgreSQL
type PostgresAuthService struct {
	pool  *pgxpool.Pool
	store *sessions.CookieStore
}

// NewPostgres создает новый сервис авторизации для PostgreSQL
func NewPostgres(pool *pgxpool.Pool, secretKey string) (*PostgresAuthService, error) {
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

	fmt.Printf("PostgreSQL Session store configured: MaxAge=%d, HttpOnly=false, Secure=false\n", SessionMaxAge)

	service := &PostgresAuthService{
		pool:  pool,
		store: store,
	}

	return service, nil
}

// Register регистрирует нового пользователя
func (s *PostgresAuthService) Register(username, password string) (*User, error) {
	ctx := context.Background()
	fmt.Printf("PostgreSQL Registration attempt: username='%s', password_length=%d\n", username, len(password))

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

	// Создаем пользователя (PostgreSQL синтаксис)
	query := `INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3) RETURNING id, created_at`
	var userID int
	var createdAt time.Time

	err = s.pool.QueryRow(ctx, query, username, string(hashedPassword), role).Scan(&userID, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("User created successfully: id=%d, username='%s', role='%s'\n", userID, username, role)

	return &User{
		ID:        userID,
		Username:  username,
		Role:      role,
		CreatedAt: createdAt,
	}, nil
}

// Login проверяет учетные данные и создает сессию
func (s *PostgresAuthService) Login(w http.ResponseWriter, r *http.Request, username, password string) (*Session, error) {
	fmt.Printf("=== PostgreSQL AuthService.Login called ===\n")
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
func (s *PostgresAuthService) Logout(w http.ResponseWriter, r *http.Request) error {
	session, err := s.store.Get(r, SessionName)
	if err != nil {
		return nil // Игнорируем ошибки при выходе
	}

	session.Options.MaxAge = -1
	return session.Save(r, w)
}

// GetSession возвращает текущую сессию пользователя
func (s *PostgresAuthService) GetSession(r *http.Request) (*Session, error) {
	session, err := s.store.Get(r, SessionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session.IsNew {
		return nil, errors.New("no active session")
	}

	userID, ok := session.Values[SessionUserIDKey].(int)
	if !ok {
		return nil, errors.New("invalid session: user_id not found")
	}

	username, ok := session.Values[SessionUsernameKey].(string)
	if !ok {
		return nil, errors.New("invalid session: username not found")
	}

	role, ok := session.Values[SessionRoleKey].(string)
	if !ok {
		return nil, errors.New("invalid session: role not found")
	}

	createdAt, ok := session.Values[SessionCreatedKey].(int64)
	if !ok {
		return nil, errors.New("invalid session: created_at not found")
	}

	return &Session{
		UserID:    userID,
		Username:  username,
		Role:      role,
		CreatedAt: createdAt,
	}, nil
}

// RequireAuth middleware для проверки авторизации
func (s *PostgresAuthService) RequireAuth() func(http.Handler) http.Handler {
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
func (s *PostgresAuthService) RequireRole(role string) func(http.Handler) http.Handler {
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

// GetUserByUsername получает пользователя по имени (PostgreSQL синтаксис)
func (s *PostgresAuthService) GetUserByUsername(username string) (*User, error) {
	ctx := context.Background()
	query := `SELECT id, username, password_hash, role, created_at FROM users WHERE username = $1`

	var user User
	err := s.pool.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// HasUsers проверяет, есть ли пользователи в системе (PostgreSQL синтаксис)
func (s *PostgresAuthService) HasUsers() (bool, error) {
	ctx := context.Background()
	var count int
	err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to count users: %w", err)
	}

	return count > 0, nil
}

// GetUserByID получает пользователя по ID (PostgreSQL синтаксис)
func (s *PostgresAuthService) GetUserByID(id int) (*User, error) {
	ctx := context.Background()
	query := `SELECT id, username, password_hash, role, created_at FROM users WHERE id = $1`

	var user User
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

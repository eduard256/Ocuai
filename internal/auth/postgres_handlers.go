package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/render"
)

// PostgresAuthHandlers предоставляет HTTP обработчики для авторизации с PostgreSQL
type PostgresAuthHandlers struct {
	service *PostgresAuthService
}

// NewPostgresHandlers создает новые обработчики авторизации для PostgreSQL
func NewPostgresHandlers(service *PostgresAuthService) *PostgresAuthHandlers {
	return &PostgresAuthHandlers{
		service: service,
	}
}

// RegisterHandler обрабатывает регистрацию нового пользователя
func (h *PostgresAuthHandlers) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("=== PostgreSQL RegisterHandler called ===\n")

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("PostgreSQL RegisterHandler: JSON decode error: %v\n", err)
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	fmt.Printf("PostgreSQL RegisterHandler: Decoded request - username='%s', password_length=%d\n", req.Username, len(req.Password))

	// Валидация входных данных
	if strings.TrimSpace(req.Username) == "" {
		fmt.Printf("PostgreSQL RegisterHandler: Validation failed - empty username\n")
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Username is required",
		})
		return
	}

	if len(req.Password) < 6 {
		fmt.Printf("PostgreSQL RegisterHandler: Validation failed - password too short\n")
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Password must be at least 6 characters long",
		})
		return
	}

	fmt.Printf("PostgreSQL RegisterHandler: Validation passed, calling service.Register\n")

	// Создаем пользователя
	user, err := h.service.Register(req.Username, req.Password)
	if err != nil {
		fmt.Printf("PostgreSQL RegisterHandler: service.Register failed: %v\n", err)
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "registration is closed") {
			status = http.StatusForbidden
		}

		w.WriteHeader(status)
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	fmt.Printf("PostgreSQL RegisterHandler: service.Register SUCCESS - user created: id=%d, username='%s', role='%s'\n",
		user.ID, user.Username, user.Role)

	// После успешной регистрации создаем сессию для автоматического входа
	fmt.Printf("PostgreSQL RegisterHandler: Creating session for auto-login\n")
	session, err := h.service.Login(w, r, req.Username, req.Password)
	if err != nil {
		fmt.Printf("PostgreSQL RegisterHandler: Auto-login failed: %v\n", err)
		// Регистрация прошла успешно, но автологин не удался
		render.JSON(w, r, AuthResponse{
			Success: true,
			Data: map[string]interface{}{
				"user": map[string]interface{}{
					"id":       user.ID,
					"username": user.Username,
					"role":     user.Role,
				},
				"message":    "Registration successful, please login manually",
				"auto_login": false,
			},
		})
		return
	}

	fmt.Printf("PostgreSQL RegisterHandler: Auto-login SUCCESS - session created for user '%s'\n", session.Username)

	render.JSON(w, r, AuthResponse{
		Success: true,
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"id":       session.UserID,
				"username": session.Username,
				"role":     session.Role,
			},
			"message":    "Registration and login successful",
			"auto_login": true,
		},
	})

	fmt.Printf("PostgreSQL RegisterHandler: Complete SUCCESS - user registered and logged in\n")
}

// LoginHandler обрабатывает вход пользователя
func (h *PostgresAuthHandlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("=== PostgreSQL LoginHandler called ===\n")

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("PostgreSQL LoginHandler: JSON decode error: %v\n", err)
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	fmt.Printf("PostgreSQL LoginHandler: Decoded request - username='%s', password_length=%d\n", req.Username, len(req.Password))

	// Валидация входных данных
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		fmt.Printf("PostgreSQL LoginHandler: Validation failed - empty username or password\n")
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Username and password are required",
		})
		return
	}

	fmt.Printf("PostgreSQL LoginHandler: Validation passed, calling service.Login\n")

	// Выполняем вход
	session, err := h.service.Login(w, r, req.Username, req.Password)
	if err != nil {
		fmt.Printf("PostgreSQL LoginHandler: service.Login failed: %v\n", err)
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Invalid username or password",
		})
		return
	}

	fmt.Printf("PostgreSQL LoginHandler: service.Login SUCCESS - session created for user '%s'\n", session.Username)

	render.JSON(w, r, AuthResponse{
		Success: true,
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"id":       session.UserID,
				"username": session.Username,
				"role":     session.Role,
			},
			"message": "Login successful",
		},
	})

	fmt.Printf("PostgreSQL LoginHandler: Complete SUCCESS - user logged in\n")
}

// LogoutHandler обрабатывает выход пользователя
func (h *PostgresAuthHandlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("=== PostgreSQL LogoutHandler called ===\n")

	if err := h.service.Logout(w, r); err != nil {
		fmt.Printf("PostgreSQL LogoutHandler: service.Logout failed: %v\n", err)
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Failed to logout",
		})
		return
	}

	fmt.Printf("PostgreSQL LogoutHandler: SUCCESS - user logged out\n")

	render.JSON(w, r, AuthResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Logout successful",
		},
	})
}

// StatusHandler проверяет статус авторизации
func (h *PostgresAuthHandlers) StatusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("=== PostgreSQL StatusHandler called ===\n")

	session, err := h.service.GetSession(r)
	if err != nil {
		fmt.Printf("PostgreSQL StatusHandler: No active session: %v\n", err)
		render.JSON(w, r, AuthResponse{
			Success: true,
			Data: map[string]interface{}{
				"authenticated": false,
			},
		})
		return
	}

	fmt.Printf("PostgreSQL StatusHandler: Active session found for user '%s'\n", session.Username)

	render.JSON(w, r, AuthResponse{
		Success: true,
		Data: map[string]interface{}{
			"authenticated": true,
			"user": map[string]interface{}{
				"id":       session.UserID,
				"username": session.Username,
				"role":     session.Role,
			},
		},
	})
}

// CheckSetupHandler проверяет, нужна ли начальная настройка
func (h *PostgresAuthHandlers) CheckSetupHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("=== PostgreSQL CheckSetupHandler called ===\n")

	hasUsers, err := h.service.HasUsers()
	if err != nil {
		fmt.Printf("PostgreSQL CheckSetupHandler: HasUsers failed: %v\n", err)
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Failed to check system status",
		})
		return
	}

	fmt.Printf("PostgreSQL CheckSetupHandler: HasUsers = %v\n", hasUsers)

	render.JSON(w, r, AuthResponse{
		Success: true,
		Data: map[string]interface{}{
			"setup_required": !hasUsers,
			"has_users":      hasUsers,
		},
	})
}

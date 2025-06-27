package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/render"
)

// RegisterRequest представляет запрос на регистрацию
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest представляет запрос на вход
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse представляет ответ авторизации
type AuthResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// AuthHandlers предоставляет HTTP обработчики для авторизации
type AuthHandlers struct {
	service *AuthService
}

// NewHandlers создает новые обработчики авторизации
func NewHandlers(service *AuthService) *AuthHandlers {
	return &AuthHandlers{
		service: service,
	}
}

// RegisterHandler обрабатывает регистрацию нового пользователя
func (h *AuthHandlers) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("=== RegisterHandler called ===\n")

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("RegisterHandler: JSON decode error: %v\n", err)
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	fmt.Printf("RegisterHandler: Decoded request - username='%s', password_length=%d\n", req.Username, len(req.Password))

	// Валидация входных данных
	if strings.TrimSpace(req.Username) == "" {
		fmt.Printf("RegisterHandler: Validation failed - empty username\n")
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Username is required",
		})
		return
	}

	if len(req.Password) < 6 {
		fmt.Printf("RegisterHandler: Validation failed - password too short\n")
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Password must be at least 6 characters long",
		})
		return
	}

	fmt.Printf("RegisterHandler: Validation passed, calling service.Register\n")

	// Создаем пользователя
	user, err := h.service.Register(req.Username, req.Password)
	if err != nil {
		fmt.Printf("RegisterHandler: service.Register failed: %v\n", err)
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

	fmt.Printf("RegisterHandler: service.Register SUCCESS - user created: id=%d, username='%s', role='%s'\n",
		user.ID, user.Username, user.Role)

	// После успешной регистрации создаем сессию для автоматического входа
	fmt.Printf("RegisterHandler: Creating session for auto-login\n")
	session, err := h.service.Login(w, r, req.Username, req.Password)
	if err != nil {
		fmt.Printf("RegisterHandler: Auto-login failed: %v\n", err)
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

	fmt.Printf("RegisterHandler: Auto-login SUCCESS - session created for user '%s'\n", session.Username)

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

	fmt.Printf("RegisterHandler: Complete SUCCESS - user registered and logged in\n")
}

// LoginHandler обрабатывает вход пользователя
func (h *AuthHandlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("=== LoginHandler called ===\n")

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("LoginHandler: JSON decode error: %v\n", err)
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	fmt.Printf("LoginHandler: Decoded request - username='%s', password_length=%d\n", req.Username, len(req.Password))

	// Валидация входных данных
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		fmt.Printf("LoginHandler: Validation failed - empty username or password\n")
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Username and password are required",
		})
		return
	}

	fmt.Printf("LoginHandler: Validation passed, calling service.Login\n")

	// Авторизация пользователя
	session, err := h.service.Login(w, r, req.Username, req.Password)
	if err != nil {
		fmt.Printf("LoginHandler: service.Login returned error: %v\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	fmt.Printf("LoginHandler: service.Login SUCCESS - returning session for user '%s'\n", session.Username)

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

	fmt.Printf("LoginHandler: Response sent successfully\n")
}

// LogoutHandler обрабатывает выход пользователя
func (h *AuthHandlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Logout(w, r); err != nil {
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Failed to logout",
		})
		return
	}

	render.JSON(w, r, AuthResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Logout successful",
		},
	})
}

// StatusHandler проверяет статус авторизации
func (h *AuthHandlers) StatusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("=== StatusHandler called ===\n")

	session, err := h.service.GetSession(r)
	if err != nil {
		fmt.Printf("StatusHandler: GetSession failed: %v\n", err)
		render.JSON(w, r, AuthResponse{
			Success: false,
			Data: map[string]interface{}{
				"authenticated": false,
			},
		})
		return
	}

	fmt.Printf("StatusHandler: Session found for user '%s'\n", session.Username)

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

	fmt.Printf("StatusHandler: Success response sent for user '%s'\n", session.Username)
}

// CheckSetupHandler проверяет, нужна ли начальная настройка
func (h *AuthHandlers) CheckSetupHandler(w http.ResponseWriter, r *http.Request) {
	hasUsers, err := h.service.HasUsers()
	if err != nil {
		render.JSON(w, r, AuthResponse{
			Success: false,
			Error:   "Failed to check setup status",
		})
		return
	}

	render.JSON(w, r, AuthResponse{
		Success: true,
		Data: map[string]interface{}{
			"setup_required": !hasUsers,
			"has_users":      hasUsers,
		},
	})
}

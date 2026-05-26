package server

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"employee-api/internal/crypto"
	"employee-api/internal/handler"
	"employee-api/internal/middleware"
	"employee-api/internal/store"
)

func New() http.Handler {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/tmp/employees.db"
	}
	s, err := store.New(dbPath)
	if err != nil {
		panic(err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-jwt-secret" //nolint:gosec
	}

	encKey := os.Getenv("SALARY_ENCRYPTION_KEY")
	if encKey == "" {
		encKey = jwtSecret
	}
	crypter := crypto.New(encKey)

	h := handler.New(s, crypter)

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Get("/health", h.HealthCheck)

	r.Route("/employees", func(r chi.Router) {
		r.Use(middleware.Authenticator(jwtSecret))
		r.With(middleware.RequireRole("admin")).Post("/", h.CreateEmployee)
		r.With(middleware.RequireRole("admin", "manager")).Get("/", h.ListEmployees)
		r.With(middleware.RequireRole("admin", "manager", "employee")).Get("/{employeeID}", h.GetEmployee)
		r.With(middleware.RequireRole("admin")).Put("/{employeeID}", h.UpdateEmployee)
		r.With(middleware.RequireRole("admin")).Delete("/{employeeID}", h.SoftDeleteEmployee)
		r.With(middleware.RequireRole("admin")).Get("/{employeeID}/audit", h.GetAuditLog)
		r.With(middleware.RequireRole("admin")).Post("/{employeeID}/erasure", h.Erasure)
	})

	return r
}

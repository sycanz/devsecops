package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"employee-api/internal/model"
)

func (h *Handler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func (h *Handler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	var req model.EmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	encrypted, err := h.crypter.Encrypt(req.Salary)
	if err != nil {
		http.Error(w, `{"error":"encryption failed"}`, http.StatusInternalServerError)
		return
	}

	emp := model.Employee{
		EmployeeID:      uuid.New().String(),
		Name:            req.Name,
		Email:           req.Email,
		Department:      req.Department,
		Role:            req.Role,
		SalaryEncrypted: encrypted,
		DateHired:       req.DateHired,
		IsActive:        true,
	}

	if err := h.store.CreateEmployee(r.Context(), emp); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, h.toResponse(emp))
}

func (h *Handler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	employees, err := h.store.ListEmployees(r.Context())
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	var resp []model.EmployeeResponse
	for _, e := range employees {
		resp = append(resp, h.toResponse(e))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetEmployee(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "employeeID")
	e, err := h.store.GetEmployee(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"employee not found"}`, http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, h.toResponse(*e))
}

func (h *Handler) UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "employeeID")
	var req model.EmployeeUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Department != nil {
		updates["department"] = *req.Department
	}
	if req.Role != nil {
		updates["role"] = *req.Role
	}
	if req.Salary != nil {
		encrypted, err := h.crypter.Encrypt(*req.Salary)
		if err != nil {
			http.Error(w, `{"error":"encryption failed"}`, http.StatusInternalServerError)
			return
		}
		updates["salary_encrypted"] = encrypted
	}
	if req.DateHired != nil {
		updates["date_hired"] = *req.DateHired
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := h.store.UpdateEmployee(r.Context(), id, updates); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	e, _ := h.store.GetEmployee(r.Context(), id)
	writeJSON(w, http.StatusOK, h.toResponse(*e))
}

func (h *Handler) SoftDeleteEmployee(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "employeeID")
	if err := h.store.SoftDeleteEmployee(r.Context(), id); err != nil {
		http.Error(w, `{"error":"employee not found"}`, http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"detail": "Employee soft-deleted (GDPR erasure trigger)"})
}

func (h *Handler) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "employeeID")
	entries, err := h.store.GetAuditLog(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, entries)
}

func (h *Handler) Erasure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "employeeID")
	if err := h.store.HardDeleteEmployee(r.Context(), id); err != nil {
		http.Error(w, `{"error":"employee not found"}`, http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"detail":      "GDPR right-to-erasure completed",
		"employee_id": id,
		"action":      "hard-deleted with audit trail",
	})
}

func (h *Handler) toResponse(e model.Employee) model.EmployeeResponse {
	salary, _ := h.crypter.Decrypt(e.SalaryEncrypted)
	return model.EmployeeResponse{
		EmployeeID: e.EmployeeID,
		Name:       e.Name,
		Email:      e.Email,
		Department: e.Department,
		Role:       e.Role,
		Salary:     salary,
		DateHired:  e.DateHired,
		IsActive:   e.IsActive,
	}
}

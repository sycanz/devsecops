package model

type Employee struct {
	EmployeeID      string `json:"employee_id"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	Department      string `json:"department"`
	Role            string `json:"role"`
	SalaryEncrypted string `json:"-"` // never exposed in JSON
	DateHired       string `json:"date_hired"`
	IsActive        bool   `json:"is_active"`
}

type EmployeeRequest struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Department string `json:"department"`
	Role       string `json:"role"`
	Salary     int    `json:"salary"`
	DateHired  string `json:"date_hired"`
}

type EmployeeUpdate struct {
	Name       *string `json:"name,omitempty"`
	Email      *string `json:"email,omitempty"`
	Department *string `json:"department,omitempty"`
	Role       *string `json:"role,omitempty"`
	Salary     *int    `json:"salary,omitempty"`
	DateHired  *string `json:"date_hired,omitempty"`
	IsActive   *bool   `json:"is_active,omitempty"`
}

type EmployeeResponse struct {
	EmployeeID string `json:"employee_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Department string `json:"department"`
	Role       string `json:"role"`
	Salary     int    `json:"salary"`
	DateHired  string `json:"date_hired"`
	IsActive   bool   `json:"is_active"`
}

type AuditLog struct {
	ID         int    `json:"id"`
	EmployeeID string `json:"employee_id"`
	Action     string `json:"action"`
	ActorRole  string `json:"actor_role"`
	Timestamp  string `json:"timestamp"`
	Details    string `json:"details,omitempty"`
}

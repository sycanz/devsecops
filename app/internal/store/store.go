package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"employee-api/internal/model"

	_ "modernc.org/sqlite"
)

type Store interface {
	CreateEmployee(ctx context.Context, e model.Employee) error
	GetEmployee(ctx context.Context, id string) (*model.Employee, error)
	ListEmployees(ctx context.Context) ([]model.Employee, error)
	UpdateEmployee(ctx context.Context, id string, updates map[string]any) error
	SoftDeleteEmployee(ctx context.Context, id string) error
	HardDeleteEmployee(ctx context.Context, id string) error
	EmailExists(ctx context.Context, email string) (bool, error)

	LogAudit(ctx context.Context, entry model.AuditLog) error
	GetAuditLog(ctx context.Context, employeeID string) ([]model.AuditLog, error)
}

type sqliteStore struct {
	db *sql.DB
}

func New(dbPath string) (Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if err = migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &sqliteStore{db: db}, nil
}

func migrate(db *sql.DB) error {
	query := `
        CREATE TABLE IF NOT EXISTS employees (
            employee_id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            email TEXT NOT NULL UNIQUE,
            department TEXT NOT NULL,
            role TEXT NOT NULL CHECK(role IN ('employee', 'manager', 'admin')),
            salary_encrypted TEXT NOT NULL,
            date_hired TEXT NOT NULL,
            is_active INTEGER NOT NULL DEFAULT 1
        );
        CREATE TABLE IF NOT EXISTS audit_logs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            employee_id TEXT NOT NULL,
            action TEXT NOT NULL,
            actor_role TEXT NOT NULL,
            timestamp TEXT NOT NULL,
            details TEXT,
            FOREIGN KEY (employee_id) REFERENCES employees(employee_id)
        );
    `
	_, err := db.ExecContext(context.Background(), query)
	return err
}

func (s *sqliteStore) CreateEmployee(ctx context.Context, e model.Employee) error {
	query := `INSERT INTO employees (employee_id, name, email, department, role, salary_encrypted, date_hired, is_active)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(
		ctx,
		query,
		e.EmployeeID,
		e.Name,
		e.Email,
		e.Department,
		e.Role,
		e.SalaryEncrypted,
		e.DateHired,
		e.IsActive,
	)
	return err
}

func (s *sqliteStore) GetEmployee(ctx context.Context, id string) (*model.Employee, error) {
	query := `SELECT employee_id, name, email, department, role, salary_encrypted, date_hired, is_active
              FROM employees WHERE employee_id = ?`
	row := s.db.QueryRowContext(ctx, query, id)
	var e model.Employee
	err := row.Scan(
		&e.EmployeeID,
		&e.Name,
		&e.Email,
		&e.Department,
		&e.Role,
		&e.SalaryEncrypted,
		&e.DateHired,
		&e.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil // not found is not an error — caller decides
	}
	if err != nil {
		return nil, fmt.Errorf("scan employee: %w", err)
	}
	return &e, nil
}

func (s *sqliteStore) ListEmployees(ctx context.Context) ([]model.Employee, error) {
	query := `SELECT employee_id, name, email, department, role, salary_encrypted, date_hired, is_active
              FROM employees WHERE is_active = 1`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list employees: %w", err)
	}
	defer rows.Close()

	var employees []model.Employee
	for rows.Next() {
		var e model.Employee
		if err = rows.Scan(
			&e.EmployeeID,
			&e.Name,
			&e.Email,
			&e.Department,
			&e.Role,
			&e.SalaryEncrypted,
			&e.DateHired,
			&e.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan employee: %w", err)
		}
		employees = append(employees, e)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	if employees == nil {
		employees = []model.Employee{}
	}
	return employees, nil
}

func (s *sqliteStore) UpdateEmployee(ctx context.Context, id string, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	set := ""
	var args []any
	for k, v := range updates {
		if set != "" {
			set += ", "
		}
		set += k + " = ?"
		args = append(args, v)
	}
	args = append(args, id)
	query := "UPDATE employees SET " + set + " WHERE employee_id = ?"
	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update employee: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("employee not found")
	}
	return nil
}

func (s *sqliteStore) SoftDeleteEmployee(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE employees SET is_active = 0 WHERE employee_id = ?", id)
	if err != nil {
		return fmt.Errorf("soft delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("employee not found")
	}
	return nil
}

func (s *sqliteStore) HardDeleteEmployee(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM audit_logs WHERE employee_id = ?", id)
	if err != nil {
		return fmt.Errorf("delete audit logs: %w", err)
	}
	res, err := s.db.ExecContext(ctx, "DELETE FROM employees WHERE employee_id = ?", id)
	if err != nil {
		return fmt.Errorf("hard delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("employee not found")
	}
	return nil
}

func (s *sqliteStore) EmailExists(ctx context.Context, email string) (bool, error) {
	row := s.db.QueryRowContext(ctx, "SELECT 1 FROM employees WHERE email = ?", email)
	var exists int
	err := row.Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check email: %w", err)
	}
	return true, nil
}

func (s *sqliteStore) LogAudit(ctx context.Context, entry model.AuditLog) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO audit_logs (employee_id, action, actor_role, timestamp, details) VALUES (?, ?, ?, ?, ?)",
		entry.EmployeeID, entry.Action, entry.ActorRole, entry.Timestamp, entry.Details,
	)
	return err
}

func (s *sqliteStore) GetAuditLog(ctx context.Context, employeeID string) ([]model.AuditLog, error) {
	rows, err := s.db.QueryContext(
		ctx,
		"SELECT id, employee_id, action, actor_role, timestamp, details FROM audit_logs WHERE employee_id = ? ORDER BY id DESC",
		employeeID,
	)
	if err != nil {
		return nil, fmt.Errorf("get audit log: %w", err)
	}
	defer rows.Close()

	var entries []model.AuditLog
	for rows.Next() {
		var entry model.AuditLog
		if err = rows.Scan(
			&entry.ID,
			&entry.EmployeeID,
			&entry.Action,
			&entry.ActorRole,
			&entry.Timestamp,
			&entry.Details,
		); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		entries = append(entries, entry)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	if entries == nil {
		entries = []model.AuditLog{}
	}
	return entries, nil
}

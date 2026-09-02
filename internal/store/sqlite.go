package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func DefaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".rhino/history.db"
	}
	return filepath.Join(home, ".rhino", "history.db")
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent access
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return s, nil
}

func (s *SQLiteStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS workflow_runs (
		id TEXT PRIMARY KEY,
		workflow_name TEXT NOT NULL,
		workflow_hash TEXT NOT NULL DEFAULT '',
		workflow_yaml TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL,
		trigger_type TEXT NOT NULL DEFAULT '',
		started_at DATETIME NOT NULL,
		completed_at DATETIME,
		error TEXT DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_runs_workflow ON workflow_runs(workflow_name);
	CREATE INDEX IF NOT EXISTS idx_runs_status ON workflow_runs(status);
	CREATE INDEX IF NOT EXISTS idx_runs_started ON workflow_runs(started_at);

	CREATE TABLE IF NOT EXISTS task_executions (
		id TEXT PRIMARY KEY,
		run_id TEXT NOT NULL,
		task_name TEXT NOT NULL,
		provider TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL,
		started_at DATETIME NOT NULL,
		completed_at DATETIME,
		output TEXT DEFAULT '',
		error TEXT DEFAULT '',
		retries INTEGER DEFAULT 0,
		duration_ms INTEGER DEFAULT 0,
		FOREIGN KEY (run_id) REFERENCES workflow_runs(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_tasks_run ON task_executions(run_id);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	return s.addColumnIfMissing("workflow_runs", "inputs", "TEXT NOT NULL DEFAULT ''")
}

// addColumnIfMissing applies an additive migration to a table created by an
// older version of the schema.
func (s *SQLiteStore) addColumnIfMissing(table, column, definition string) error {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid        int
			name, typ  string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultVal, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}

// encodeInputs stores run inputs as JSON; an empty map stays an empty string.
func encodeInputs(inputs map[string]string) string {
	if len(inputs) == 0 {
		return ""
	}
	b, err := json.Marshal(inputs)
	if err != nil {
		return ""
	}
	return string(b)
}

func decodeInputs(raw string) map[string]string {
	if raw == "" {
		return nil
	}
	var inputs map[string]string
	if err := json.Unmarshal([]byte(raw), &inputs); err != nil {
		return nil
	}
	return inputs
}

// nullTime stores an unset completion time as NULL rather than year 1.
func nullTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}

func (s *SQLiteStore) SaveRun(run *WorkflowRun) error {
	_, err := s.db.Exec(
		`INSERT INTO workflow_runs (id, workflow_name, workflow_hash, workflow_yaml, status, trigger_type, inputs, started_at, completed_at, error)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.WorkflowName, run.WorkflowHash, run.WorkflowYAML,
		run.Status, run.TriggerType, encodeInputs(run.Inputs), run.StartedAt, nullTime(run.CompletedAt), run.Error,
	)
	return err
}

func (s *SQLiteStore) UpdateRun(run *WorkflowRun) error {
	_, err := s.db.Exec(
		`UPDATE workflow_runs SET status = ?, completed_at = ?, error = ? WHERE id = ?`,
		run.Status, nullTime(run.CompletedAt), run.Error, run.ID,
	)
	return err
}

func (s *SQLiteStore) GetRun(id string) (*WorkflowRun, error) {
	row := s.db.QueryRow(
		`SELECT id, workflow_name, workflow_hash, workflow_yaml, status, trigger_type, inputs, started_at, completed_at, error
		 FROM workflow_runs WHERE id = ?`, id,
	)

	var run WorkflowRun
	var completedAt sql.NullTime
	var inputs string
	err := row.Scan(&run.ID, &run.WorkflowName, &run.WorkflowHash, &run.WorkflowYAML,
		&run.Status, &run.TriggerType, &inputs, &run.StartedAt, &completedAt, &run.Error)
	run.Inputs = decodeInputs(inputs)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("run '%s' not found", id)
	}
	if err != nil {
		return nil, err
	}
	if completedAt.Valid {
		run.CompletedAt = completedAt.Time
	}
	return &run, nil
}

func (s *SQLiteStore) ListRuns(filter RunFilter) ([]*WorkflowRun, error) {
	query := `SELECT id, workflow_name, workflow_hash, status, trigger_type, inputs, started_at, completed_at, error
	          FROM workflow_runs WHERE 1=1`
	var args []interface{}

	if filter.WorkflowName != "" {
		query += " AND workflow_name = ?"
		args = append(args, filter.WorkflowName)
	}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.Since > 0 {
		query += " AND started_at >= ?"
		args = append(args, time.Now().Add(-filter.Since))
	}

	query += " ORDER BY started_at DESC"

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	query += " LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*WorkflowRun
	for rows.Next() {
		var run WorkflowRun
		var completedAt sql.NullTime
		var inputs string
		if err := rows.Scan(&run.ID, &run.WorkflowName, &run.WorkflowHash, &run.Status,
			&run.TriggerType, &inputs, &run.StartedAt, &completedAt, &run.Error); err != nil {
			return nil, err
		}
		run.Inputs = decodeInputs(inputs)
		if completedAt.Valid {
			run.CompletedAt = completedAt.Time
		}
		runs = append(runs, &run)
	}
	return runs, rows.Err()
}

func (s *SQLiteStore) SaveTaskExecution(exec *TaskExecution) error {
	_, err := s.db.Exec(
		`INSERT INTO task_executions (id, run_id, task_name, provider, status, started_at, completed_at, output, error, retries, duration_ms)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		exec.ID, exec.RunID, exec.TaskName, exec.Provider, exec.Status,
		exec.StartedAt, nullTime(exec.CompletedAt), exec.Output, exec.Error, exec.Retries, exec.DurationMs,
	)
	return err
}

func (s *SQLiteStore) GetTaskExecutions(runID string) ([]*TaskExecution, error) {
	rows, err := s.db.Query(
		`SELECT id, run_id, task_name, provider, status, started_at, completed_at, output, error, retries, duration_ms
		 FROM task_executions WHERE run_id = ? ORDER BY started_at`, runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var execs []*TaskExecution
	for rows.Next() {
		var exec TaskExecution
		var completedAt sql.NullTime
		if err := rows.Scan(&exec.ID, &exec.RunID, &exec.TaskName, &exec.Provider, &exec.Status,
			&exec.StartedAt, &completedAt, &exec.Output, &exec.Error, &exec.Retries, &exec.DurationMs); err != nil {
			return nil, err
		}
		if completedAt.Valid {
			exec.CompletedAt = completedAt.Time
		}
		execs = append(execs, &exec)
	}
	return execs, rows.Err()
}

func (s *SQLiteStore) DeleteRunsBefore(before time.Time) (int64, error) {
	// Delete task executions first (FK constraint)
	_, err := s.db.Exec(
		`DELETE FROM task_executions WHERE run_id IN (SELECT id FROM workflow_runs WHERE started_at < ?)`, before,
	)
	if err != nil {
		return 0, err
	}

	result, err := s.db.Exec(`DELETE FROM workflow_runs WHERE started_at < ?`, before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

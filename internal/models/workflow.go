package models

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/y0anfa/rhino/internal/config"
	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/providers"
	"github.com/y0anfa/rhino/internal/store"
	"github.com/y0anfa/rhino/internal/template"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Workflow struct {
	Name          string              `yaml:"name"`
	Description   string              `yaml:"description"`
	Settings      Settings            `yaml:"settings"`
	Notifications *NotificationConfig `yaml:"notifications,omitempty"`
	Trigger       Trigger             `yaml:"trigger"`
	Tasks         []Task              `yaml:"tasks"`
	Order         [][]string          `yaml:"order"`
}

func NewWorkflow(name string, desc string) *Workflow {
	return &Workflow{Name: name, Description: desc, Settings: *NewSettings(MaxTriesDefault, TimeoutDefault)}
}

func DeleteWorkflow(name string) error {
	dir := config.GetString("workflows-dir")
	return os.Remove(filepath.Join(dir, name+".yaml"))
}

func (w *Workflow) Describe() string {
	desc := "Workflow: " + w.Name + "\n"
	desc += "Description: " + w.Description + "\n"
	desc += "\nSettings:\n"
	desc += fmt.Sprintf("  Max Tries: %d\n", w.Settings.MaxTries)
	desc += fmt.Sprintf("  Timeout: %s\n", w.Settings.Timeout)
	desc += "\nTrigger:\n"
	desc += fmt.Sprintf("  Name: %s\n", w.Trigger.Name)
	desc += fmt.Sprintf("  Type: %s\n", w.Trigger.Type)
	if w.Trigger.Schedule != "" {
		desc += fmt.Sprintf("  Schedule: %s\n", w.Trigger.Schedule)
	}
	desc += "\nTasks:\n"
	for _, t := range w.Tasks {
		desc += fmt.Sprintf("  - %s (provider: %s)\n", t.Name, t.Provider)
		if t.Description != "" {
			desc += fmt.Sprintf("    Description: %s\n", t.Description)
		}
		for k, v := range t.Params {
			desc += fmt.Sprintf("    %s: %v\n", k, v)
		}
	}
	desc += "\nOrder:\n"
	for i, group := range w.Order {
		desc += fmt.Sprintf("  %d: %v\n", i+1, group)
	}
	return desc
}

func ListWorkflows() ([]string, error) {
	dir := config.GetString("workflows-dir")

	var workflows []string
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		ext := filepath.Ext(f.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		workflows = append(workflows, strings.TrimSuffix(f.Name(), ext))
	}
	return workflows, nil
}

func (w *Workflow) SetTrigger(trigger Trigger) {
	w.Trigger = trigger
}

func (w *Workflow) AddTask(task Task) {
	w.Tasks = append(w.Tasks, task)
}

func (w *Workflow) RemoveTask(task Task) (string, error) {
	for i, t := range w.Tasks {
		if t.Name == task.Name {
			w.Tasks = append(w.Tasks[:i], w.Tasks[i+1:]...)
			return t.Name, nil
		}
	}
	return "", fmt.Errorf("task %s not found", task.Name)
}

func (w *Workflow) GetTask(name string) *Task {
	for i := range w.Tasks {
		if w.Tasks[i].Name == name {
			return &w.Tasks[i]
		}
	}
	return nil
}

func (w *Workflow) Validate() error {
	if w.Name == "" {
		return fmt.Errorf("workflow validation failed: name is empty")
	}
	if w.Settings.MaxTries <= 0 {
		return fmt.Errorf("workflow validation failed: max tries must be greater than 0, got %d", w.Settings.MaxTries)
	}
	if w.Settings.Timeout == "" {
		return fmt.Errorf("workflow validation failed: timeout is empty")
	}
	if _, err := time.ParseDuration(w.Settings.Timeout); err != nil {
		return fmt.Errorf("workflow validation failed: invalid timeout format '%s': %w", w.Settings.Timeout, err)
	}
	if err := w.Settings.validateRetry(); err != nil {
		return fmt.Errorf("workflow validation failed: %w", err)
	}
	if w.Settings.MaxConcurrentRuns < 0 {
		return fmt.Errorf("workflow validation failed: max-concurrent-runs must be >= 0, got %d", w.Settings.MaxConcurrentRuns)
	}
	if w.Settings.MaxOutputSize < 0 {
		return fmt.Errorf("workflow validation failed: max-output-size must be >= 0, got %d", w.Settings.MaxOutputSize)
	}
	if w.Trigger.Name == "" {
		return fmt.Errorf("workflow validation failed: trigger name is empty")
	}
	if w.Trigger.Type == "" {
		return fmt.Errorf("workflow validation failed: trigger type is empty")
	}
	switch w.Trigger.Type {
	case TriggerManual, TriggerScheduled, TriggerWebhook, TriggerWatch:
	default:
		return fmt.Errorf("workflow validation failed: unknown trigger type '%s'", w.Trigger.Type)
	}
	if w.Trigger.Type == TriggerWatch && w.Trigger.WatchPath == "" {
		return fmt.Errorf("workflow validation failed: watch-path is empty for watch trigger")
	}
	if w.Trigger.Debounce != "" {
		if _, err := time.ParseDuration(w.Trigger.Debounce); err != nil {
			return fmt.Errorf("workflow validation failed: invalid debounce '%s': %w", w.Trigger.Debounce, err)
		}
	}
	if w.Trigger.Type == TriggerScheduled && w.Trigger.Schedule == "" {
		return fmt.Errorf("workflow validation failed: trigger schedule is empty for cron trigger")
	}
	if w.Trigger.Type == TriggerScheduled {
		if _, err := cron.ParseStandard(w.Trigger.Schedule); err != nil {
			return fmt.Errorf("workflow validation failed: invalid cron schedule '%s': %w", w.Trigger.Schedule, err)
		}
	}
	if len(w.Tasks) == 0 {
		return fmt.Errorf("workflow validation failed: tasks list is empty")
	}
	seenTasks := make(map[string]bool, len(w.Tasks))
	for _, t := range w.Tasks {
		if t.Name == "" {
			return fmt.Errorf("workflow validation failed: task name is empty")
		}
		if seenTasks[t.Name] {
			return fmt.Errorf("workflow validation failed: duplicate task name '%s'", t.Name)
		}
		seenTasks[t.Name] = true
		if t.MaxTries < 0 {
			return fmt.Errorf("workflow validation failed: task '%s' max-tries must be >= 0, got %d", t.Name, t.MaxTries)
		}
		if t.Timeout != "" {
			if _, err := time.ParseDuration(t.Timeout); err != nil {
				return fmt.Errorf("workflow validation failed: task '%s' has invalid timeout '%s': %w", t.Name, t.Timeout, err)
			}
		}
		if err := validateCondition(t.Condition); err != nil {
			return fmt.Errorf("workflow validation failed: task '%s': %w", t.Name, err)
		}
		if t.Provider == "" {
			return fmt.Errorf("workflow validation failed: task '%s' provider is empty", t.Name)
		}
		if len(t.Params) == 0 {
			return fmt.Errorf("workflow validation failed: task '%s' params are empty", t.Name)
		}
		// Validate task provider
		provider, err := providers.Get(t.Provider)
		if err != nil {
			return fmt.Errorf("workflow validation failed: task '%s' has unknown provider '%s': %w", t.Name, t.Provider, err)
		}
		if err := provider.Validate(t.Params); err != nil {
			return fmt.Errorf("workflow validation failed: task '%s' validation failed: %w", t.Name, err)
		}
	}
	// Check if using depends-on DAG mode
	useDAG := false
	for _, t := range w.Tasks {
		if len(t.DependsOn) > 0 {
			useDAG = true
			break
		}
	}

	if useDAG {
		// Validate depends-on references and detect cycles
		order, err := w.buildDAGOrder()
		if err != nil {
			return fmt.Errorf("workflow validation failed: %w", err)
		}
		w.Order = order
	} else {
		if len(w.Order) == 0 {
			return fmt.Errorf("workflow validation failed: order is empty")
		}
		ordered := make(map[string]bool, len(w.Tasks))
		for _, group := range w.Order {
			if len(group) == 0 {
				return fmt.Errorf("workflow validation failed: order group is empty")
			}
			for _, taskName := range group {
				task := w.GetTask(taskName)
				if task == nil {
					return fmt.Errorf("workflow validation failed: task '%s' not found in order", taskName)
				}
				if ordered[taskName] {
					return fmt.Errorf("workflow validation failed: task '%s' appears more than once in order", taskName)
				}
				ordered[taskName] = true
			}
		}
		for _, t := range w.Tasks {
			if !ordered[t.Name] {
				return fmt.Errorf("workflow validation failed: task '%s' is not listed in order", t.Name)
			}
		}
	}
	return nil
}

func (w *Workflow) buildDAGOrder() ([][]string, error) {
	// Build adjacency: task -> tasks it depends on
	taskNames := make(map[string]bool)
	deps := make(map[string][]string)
	for _, t := range w.Tasks {
		taskNames[t.Name] = true
		deps[t.Name] = t.DependsOn
	}

	// Validate references
	for name, depList := range deps {
		for _, dep := range depList {
			if !taskNames[dep] {
				return nil, fmt.Errorf("task '%s' depends on unknown task '%s'", name, dep)
			}
			if dep == name {
				return nil, fmt.Errorf("task '%s' depends on itself", name)
			}
		}
	}

	// Topological sort (Kahn's algorithm) into groups
	inDegree := make(map[string]int)
	for name := range taskNames {
		inDegree[name] = 0
	}
	for _, depList := range deps {
		for _, dep := range depList {
			_ = dep // inDegree counts how many tasks depend on this one
		}
	}
	// Actually: inDegree[X] = number of dependencies X has
	for name, depList := range deps {
		inDegree[name] = len(depList)
	}

	var order [][]string
	resolved := make(map[string]bool)

	for len(resolved) < len(taskNames) {
		// Find all tasks with no unresolved dependencies
		var group []string
		for name := range taskNames {
			if resolved[name] {
				continue
			}
			allResolved := true
			for _, dep := range deps[name] {
				if !resolved[dep] {
					allResolved = false
					break
				}
			}
			if allResolved {
				group = append(group, name)
			}
		}

		if len(group) == 0 {
			return nil, fmt.Errorf("circular dependency detected")
		}

		order = append(order, group)
		for _, name := range group {
			resolved[name] = true
		}
	}

	return order, nil
}

// notificationTimeout bounds each notification delivery so a hung channel
// cannot keep a finished run alive indefinitely.
const notificationTimeout = 30 * time.Second

// sendNotifications delivers on-success / on-failure channels. Params are
// resolved against the run's template context, so they can reference task
// outputs, {{run.id}}, and {{workflow.error}}.
func (w *Workflow) sendNotifications(runErr error, tmplCtx *template.Context) {
	if w.Notifications == nil {
		return
	}

	var channels []NotificationChannel
	if runErr == nil {
		channels = w.Notifications.OnSuccess
	} else {
		channels = w.Notifications.OnFailure
	}
	tmplCtx.SetRunError(runErr)

	for _, ch := range channels {
		provider, err := providers.Get(ch.Provider)
		if err != nil {
			logger.Error("notification provider not found",
				zap.String("provider", ch.Provider), zap.Error(err))
			continue
		}
		resolvedParams := template.ResolveParams(ch.Params, tmplCtx)

		ctx, cancel := context.WithTimeout(context.Background(), notificationTimeout)
		_, err = provider.Run(ctx, resolvedParams)
		cancel()
		if err != nil {
			logger.Error("notification failed",
				zap.String("provider", ch.Provider), zap.Error(err))
		}
	}
}

func (w *Workflow) Save() error {
	dir := config.GetString("workflows-dir")
	path := filepath.Join(dir, w.Name+".yaml")

	data, err := yaml.Marshal(w)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	logger.Info("saving workflow", zap.String("path", path))
	_, err = file.Write(data)
	return err
}

func LoadWorkflow(name string) (Workflow, error) {
	dir := config.GetString("workflows-dir")
	path := filepath.Join(dir, name+".yaml")

	file, err := os.ReadFile(path)
	if err != nil {
		return Workflow{}, err
	}

	var workflow Workflow
	if err = yaml.Unmarshal(file, &workflow); err != nil {
		logger.Error("error decoding workflow", zap.String("workflow", name), zap.Error(err))
		return Workflow{}, err
	}
	return workflow, nil
}

func LoadWorkflows() ([]Workflow, error) {
	var workflows []Workflow
	workflowsList, err := ListWorkflows()
	if err != nil {
		return nil, err
	}
	for _, w := range workflowsList {
		workflow, err := LoadWorkflow(w)
		if err != nil {
			return nil, err
		}
		if err = workflow.Validate(); err != nil {
			return nil, fmt.Errorf("workflow '%s' is invalid: %w", workflow.Name, err)
		}
		workflows = append(workflows, workflow)
	}
	return workflows, nil
}

func init() {
	providers.WorkflowExecutor = func(ctx context.Context, name string) (map[string]*providers.TaskResult, error) {
		w, err := LoadWorkflow(name)
		if err != nil {
			return nil, fmt.Errorf("failed to load workflow '%s': %w", name, err)
		}
		if err := w.Validate(); err != nil {
			return nil, fmt.Errorf("workflow '%s' validation failed: %w", name, err)
		}
		return w.Run(ctx)
	}
}

type taskRunResult struct {
	result *providers.TaskResult
	err    error
}

func (w *Workflow) resolveTimeout(t *Task) (time.Duration, error) {
	timeoutStr := w.Settings.Timeout
	if t.Timeout != "" {
		timeoutStr = t.Timeout
	}
	return time.ParseDuration(timeoutStr)
}

func (w *Workflow) retryDelay(attempt int) time.Duration {
	backoff := w.Settings.RetryBackoff
	if backoff == "" || backoff == "none" {
		return 0
	}

	baseDelay, err := time.ParseDuration(w.Settings.RetryBaseDelay)
	if err != nil {
		baseDelay = time.Second
	}
	maxDelay, err := time.ParseDuration(w.Settings.RetryMaxDelay)
	if err != nil {
		maxDelay = time.Minute
	}

	if baseDelay <= 0 || maxDelay <= 0 {
		return 0
	}

	var scaled float64
	switch backoff {
	case "linear":
		scaled = float64(baseDelay) * float64(attempt+1)
	case "exponential":
		scaled = float64(baseDelay) * math.Pow(2, float64(attempt))
	default:
		return 0
	}

	// Cap before adding jitter: a large attempt count overflows time.Duration,
	// which would make the delay (and the jitter bound) negative.
	delay := maxDelay
	if scaled > 0 && scaled < float64(maxDelay) {
		delay = time.Duration(scaled)
	}

	// Add jitter: 0-25% of computed delay
	delay += time.Duration(rand.Int63n(int64(delay/4) + 1))
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}

// conditionStatusRef matches "{{task.NAME.status}}" references inside a condition.
var conditionStatusRef = regexp.MustCompile(`\{\{\s*task\.([^.\s}]+)\.status\s*\}\}`)

// validateCondition accepts "", "always", "never", or a single comparison of a
// task status reference: "{{task.X.status}} == success" / "!= failed".
func validateCondition(cond string) error {
	cond = strings.TrimSpace(cond)
	if cond == "" || cond == "always" || cond == "never" {
		return nil
	}
	if !conditionStatusRef.MatchString(cond) {
		return fmt.Errorf("invalid condition '%s': expected always, never, or a {{task.NAME.status}} comparison", cond)
	}
	op := "=="
	if strings.Contains(cond, "!=") {
		op = "!="
	}
	parts := strings.SplitN(cond, op, 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid condition '%s': expected '<lhs> == <rhs>' or '<lhs> != <rhs>'", cond)
	}
	rhs := strings.TrimSpace(parts[1])
	switch rhs {
	case "success", "failed", "skipped":
		return nil
	}
	return fmt.Errorf("invalid condition '%s': status must be success, failed, or skipped", cond)
}

func (w *Workflow) evaluateCondition(t *Task, taskStatuses map[string]string) bool {
	cond := strings.TrimSpace(t.Condition)
	if cond == "" {
		return true
	}
	if cond == "always" {
		return true
	}
	if cond == "never" {
		return false
	}

	// Resolve "{{task.TASKNAME.status}}"; a task that never ran has no status,
	// so the reference resolves to an empty string.
	cond = conditionStatusRef.ReplaceAllStringFunc(cond, func(match string) string {
		name := conditionStatusRef.FindStringSubmatch(match)[1]
		return taskStatuses[name]
	})

	if parts := strings.SplitN(cond, "!=", 2); len(parts) == 2 {
		return strings.TrimSpace(parts[0]) != strings.TrimSpace(parts[1])
	}
	if parts := strings.SplitN(cond, "==", 2); len(parts) == 2 {
		return strings.TrimSpace(parts[0]) == strings.TrimSpace(parts[1])
	}

	return true
}

// resolveOrder returns the execution order, deriving it from depends-on when no
// explicit order is set (Validate does the same, but Run may be called directly).
func (w *Workflow) resolveOrder() ([][]string, error) {
	if len(w.Order) > 0 {
		return w.Order, nil
	}
	for _, t := range w.Tasks {
		if len(t.DependsOn) > 0 {
			return w.buildDAGOrder()
		}
	}
	return nil, fmt.Errorf("no execution order: set 'order' or 'depends-on'")
}

// runIDKey carries a caller-chosen run ID so the caller can hand it out (for
// example in an HTTP response) before the run finishes.
type runIDKey struct{}

// WithRunID returns a context that makes Run record the workflow run under the
// given ID instead of generating one.
func WithRunID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, runIDKey{}, id)
}

func runIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(runIDKey{}).(string)
	return id
}

// ErrTooManyRuns is returned when a workflow already has max-concurrent-runs
// executions in flight.
var ErrTooManyRuns = errors.New("max concurrent runs reached")

var (
	runSlotsMu sync.Mutex
	runSlots   = make(map[string]int)
)

// acquireRunSlot reserves an execution slot for the workflow. It fails without
// blocking when the limit is reached, so overlapping cron ticks or webhook
// bursts are dropped instead of piling up behind each other.
func acquireRunSlot(name string, limit int) (release func(), err error) {
	if limit <= 0 {
		return func() {}, nil
	}
	runSlotsMu.Lock()
	defer runSlotsMu.Unlock()
	if runSlots[name] >= limit {
		return nil, fmt.Errorf("workflow '%s': %w (limit %d)", name, ErrTooManyRuns, limit)
	}
	runSlots[name]++
	var once sync.Once
	return func() {
		once.Do(func() {
			runSlotsMu.Lock()
			defer runSlotsMu.Unlock()
			if runSlots[name] <= 1 {
				delete(runSlots, name)
				return
			}
			runSlots[name]--
		})
	}, nil
}

// truncateOutput enforces max-output-size on a task result so a chatty task
// cannot bloat memory, templates, or the history database.
func (w *Workflow) truncateOutput(res *providers.TaskResult) {
	limit := w.Settings.MaxOutputSize
	if res == nil || limit <= 0 || len(res.Output) <= limit {
		return
	}
	res.Output = res.Output[:limit]
	if res.Metadata == nil {
		res.Metadata = make(map[string]string)
	}
	res.Metadata["output_truncated"] = "true"
}

// taskOutcome is what a single task attempt sequence produced, used both for
// the in-memory results and the persisted execution record.
type taskOutcome struct {
	result   *providers.TaskResult
	err      error
	attempts int
	started  time.Time
	finished time.Time
}

// runTask executes a task with retries and per-attempt timeouts.
func (w *Workflow) runTask(ctx context.Context, t *Task, maxTries int, timeout time.Duration, params map[string]interface{}) (out taskOutcome) {
	out.started = time.Now()
	// Named result: the deferred stamp must land on the value being returned.
	defer func() { out.finished = time.Now() }()

	for try := 0; try < maxTries; try++ {
		if try > 0 {
			delay := w.retryDelay(try - 1)
			if delay > 0 {
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					out.err = ctx.Err()
					return out
				case <-timer.C:
				}
			}
		}
		out.attempts++

		taskCtx, cancel := context.WithTimeout(ctx, timeout)
		ch := make(chan taskRunResult, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					ch <- taskRunResult{err: fmt.Errorf("task '%s' panicked: %v", t.Name, r)}
				}
			}()
			res, err := t.RunWithParams(taskCtx, params)
			ch <- taskRunResult{result: res, err: err}
		}()

		select {
		case <-taskCtx.Done():
			out.err = taskCtx.Err()
			logger.Error("task execution failed: timeout reached", zap.String("task", t.Name), zap.Error(out.err))
		case tr := <-ch:
			out.err = tr.err
			out.result = tr.result
		}
		cancel()

		if out.err == nil {
			logger.Info("task execution succeeded", zap.String("task", t.Name))
			return out
		}
		logger.Error("task execution failed", zap.String("task", t.Name), zap.Error(out.err), zap.Int("attempt", out.attempts))

		// A cancelled workflow must not burn its remaining attempts.
		if ctx.Err() != nil {
			return out
		}
	}
	return out
}

func (w *Workflow) Run(ctx context.Context) (map[string]*providers.TaskResult, error) {
	results := make(map[string]*providers.TaskResult)
	var resultsMu sync.Mutex
	taskStatuses := make(map[string]string)
	var statusMu sync.Mutex
	tmplCtx := template.NewContext(w.Name, w.Description, w.Trigger.Name, string(w.Trigger.Type))

	order, orderErr := w.resolveOrder()
	if orderErr != nil {
		return results, fmt.Errorf("workflow '%s' failed: %w", w.Name, orderErr)
	}

	release, err := acquireRunSlot(w.Name, w.Settings.MaxConcurrentRuns)
	if err != nil {
		return results, err
	}
	defer release()

	runID := runIDFromContext(ctx)
	if runID == "" {
		runID = store.NewID()
	}
	tmplCtx.RunID = runID

	// Record run in history store
	var runRecord *store.WorkflowRun
	if s := store.Global(); s != nil {
		yamlBytes, _ := yaml.Marshal(w)
		hash := fmt.Sprintf("%x", sha256.Sum256(yamlBytes))
		runRecord = &store.WorkflowRun{
			ID:           runID,
			WorkflowName: w.Name,
			WorkflowHash: hash[:16],
			WorkflowYAML: string(yamlBytes),
			Status:       store.RunStatusRunning,
			TriggerType:  string(w.Trigger.Type),
			StartedAt:    time.Now(),
		}
		if err := s.SaveRun(runRecord); err != nil {
			logger.Error("failed to save run record", zap.Error(err))
			runRecord = nil
		}
	}

	recordTask := func(t *Task, status store.TaskStatus, out taskOutcome) {
		if runRecord == nil {
			return
		}
		s := store.Global()
		if s == nil {
			return
		}
		exec := &store.TaskExecution{
			ID:          store.NewID(),
			RunID:       runRecord.ID,
			TaskName:    t.Name,
			Provider:    t.Provider,
			Status:      status,
			StartedAt:   out.started,
			CompletedAt: out.finished,
			DurationMs:  out.finished.Sub(out.started).Milliseconds(),
		}
		if out.attempts > 0 {
			exec.Retries = out.attempts - 1
		}
		if out.result != nil {
			exec.Output = out.result.Output
		}
		if out.err != nil {
			exec.Error = out.err.Error()
		}
		if err := s.SaveTaskExecution(exec); err != nil {
			logger.Error("failed to save task execution", zap.String("task", t.Name), zap.Error(err))
		}
	}

	finishRun := func(runErr error) {
		if runRecord != nil {
			if s := store.Global(); s != nil {
				runRecord.CompletedAt = time.Now()
				if runErr != nil {
					runRecord.Status = store.RunStatusFailed
					runRecord.Error = runErr.Error()
				} else {
					runRecord.Status = store.RunStatusSuccess
				}
				if err := s.UpdateRun(runRecord); err != nil {
					logger.Error("failed to update run record", zap.Error(err))
				}
			}
		}

		w.sendNotifications(runErr, tmplCtx)
	}

	for _, group := range order {
		if err := ctx.Err(); err != nil {
			runErr := fmt.Errorf("workflow '%s' cancelled: %w", w.Name, err)
			finishRun(runErr)
			return results, runErr
		}

		var wg sync.WaitGroup
		var mu sync.Mutex
		var errs []error

		for _, taskName := range group {
			task := w.GetTask(taskName)
			if task == nil {
				runErr := fmt.Errorf("workflow '%s' failed: task '%s' in order not found", w.Name, taskName)
				finishRun(runErr)
				return results, runErr
			}
			// Resolve per-task settings locally: the Task slice is shared by
			// concurrent runs of the same workflow, so it must not be mutated.
			maxTries := task.MaxTries
			if maxTries <= 0 {
				maxTries = w.Settings.MaxTries
			}

			// Evaluate condition
			statusMu.Lock()
			shouldRun := w.evaluateCondition(task, taskStatuses)
			statusMu.Unlock()
			if !shouldRun {
				logger.Info("task skipped: condition not met", zap.String("task", task.Name))
				statusMu.Lock()
				taskStatuses[task.Name] = "skipped"
				statusMu.Unlock()
				now := time.Now()
				recordTask(task, store.TaskStatusSkipped, taskOutcome{started: now, finished: now})
				continue
			}

			timeout, parseErr := w.resolveTimeout(task)
			if parseErr != nil {
				runErr := fmt.Errorf("workflow '%s' failed: invalid timeout for task '%s': %w", w.Name, task.Name, parseErr)
				finishRun(runErr)
				return results, runErr
			}

			wg.Add(1)

			go func(t *Task, maxTries int, timeout time.Duration) {
				defer wg.Done()

				// Resolve template expressions in params
				resolvedParams := template.ResolveParams(t.Params, tmplCtx)

				out := w.runTask(ctx, t, maxTries, timeout, resolvedParams)

				status := store.TaskStatusSuccess
				if out.err == nil {
					w.truncateOutput(out.result)
					resultsMu.Lock()
					results[t.Name] = out.result
					tmplCtx.SetTaskResult(t.Name, out.result)
					resultsMu.Unlock()
				} else {
					status = store.TaskStatusFailed
				}
				recordTask(t, status, out)

				statusMu.Lock()
				taskStatuses[t.Name] = string(status)
				statusMu.Unlock()

				if out.err != nil {
					logger.Error("task execution failed: max retries reached", zap.String("task", t.Name), zap.Error(out.err))
					if !t.ContinueOnError {
						mu.Lock()
						errs = append(errs, fmt.Errorf("task '%s' failed: %w", t.Name, out.err))
						mu.Unlock()
					}
				}
			}(task, maxTries, timeout)
		}

		wg.Wait()

		if len(errs) > 0 {
			runErr := fmt.Errorf("workflow '%s' failed: %v", w.Name, errs)
			finishRun(runErr)
			return results, runErr
		}
	}

	finishRun(nil)
	return results, nil
}

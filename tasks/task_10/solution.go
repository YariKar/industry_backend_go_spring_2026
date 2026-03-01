package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Clock interface {
	Now() time.Time
}

type Task struct {
	ID        string
	Title     string
	Done      bool
	UpdatedAt time.Time
}
type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List() []Task
	SetDone(id string, done bool) (Task, error)
}

type InMemoryTaskRepo struct {
	mu     sync.RWMutex
	tasks  map[string]Task
	clock  Clock
	nextID int64
}

func NewInMemoryTaskRepo(clock Clock) *InMemoryTaskRepo {
	return &InMemoryTaskRepo{
		tasks:  make(map[string]Task),
		clock:  clock,
		nextID: 1,
	}
}

func (r *InMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, errors.New("title cannot be empty")
	}

	id := atomic.AddInt64(&r.nextID, 1)
	task := Task{
		ID:        formatID(id),
		Title:     title,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}

	r.mu.Lock()
	r.tasks[task.ID] = task
	r.mu.Unlock()

	return task, nil
}

func (r *InMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	task, ok := r.tasks[id]
	r.mu.RUnlock()
	return task, ok
}

func (r *InMemoryTaskRepo) List() []Task {
	r.mu.RLock()
	tasks := make([]Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		tasks = append(tasks, t)
	}
	r.mu.RUnlock()
	return tasks
}

func (r *InMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return Task{}, errors.New("task not found")
	}

	task.Done = done
	task.UpdatedAt = r.clock.Now()
	r.tasks[id] = task

	return task, nil
}

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

type createRequest struct {
	Title string `json:"title"`
}

type patchRequest struct {
	Done *bool `json:"done"`
}

func NewHTTPHandler(repo TaskRepo) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		var req createRequest
		if err := parseJSON(r, &req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		task, err := repo.Create(req.Title)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(toDTO(task))
	})

	mux.HandleFunc("GET /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		task, ok := repo.Get(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(toDTO(task))
	})

	mux.HandleFunc("GET /tasks", func(w http.ResponseWriter, r *http.Request) {
		tasks := repo.List()
		sort.Slice(tasks, func(i, j int) bool {
			if tasks[i].UpdatedAt.Equal(tasks[j].UpdatedAt) {
				return tasks[i].ID < tasks[j].ID
			}
			return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
		})

		dtos := make([]taskDTO, len(tasks))
		for i, t := range tasks {
			dtos[i] = toDTO(t)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dtos)
	})

	mux.HandleFunc("PATCH /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var req patchRequest
		if err := parseJSON(r, &req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Done == nil {
			http.Error(w, "missing field 'done'", http.StatusBadRequest)
			return
		}

		task, err := repo.SetDone(id, *req.Done)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(toDTO(task))
	})

	return mux
}

func parseJSON(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func toDTO(t Task) taskDTO {
	return taskDTO{
		ID:        t.ID,
		Title:     t.Title,
		Done:      t.Done,
		UpdatedAt: t.UpdatedAt,
	}
}
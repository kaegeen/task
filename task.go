package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Task represents a task object
type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

// In-memory "database"
var taskDB = struct {
	sync.Mutex
	tasks map[int]Task
}{tasks: make(map[int]Task)}

var taskIDCounter = 1

// Function to respond with JSON
func respondWithJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Handler to list all tasks
func getTasksHandler(w http.ResponseWriter, r *http.Request) {
	taskDB.Lock()
	defer taskDB.Unlock()

	var tasks []Task
	for _, task := range taskDB.tasks {
		tasks = append(tasks, task)
	}

	respondWithJSON(w, http.StatusOK, tasks)
}

// Handler to create a new task
func createTaskHandler(w http.ResponseWriter, r *http.Request) {
	var newTask Task
	if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	taskDB.Lock()
	defer taskDB.Unlock()

	newTask.ID = taskIDCounter
	newTask.CreatedAt = time.Now()
	taskDB.tasks[taskIDCounter] = newTask
	taskIDCounter++

	respondWithJSON(w, http.StatusCreated, newTask)
}

// Handler to update an existing task
func updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var updatedTask Task
	if err := json.NewDecoder(r.Body).Decode(&updatedTask); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	taskDB.Lock()
	defer taskDB.Unlock()

	// Check if the task exists
	task, exists := taskDB.tasks[updatedTask.ID]
	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Update task fields
	task.Title = updatedTask.Title
	task.Completed = updatedTask.Completed
	taskDB.tasks[updatedTask.ID] = task

	respondWithJSON(w, http.StatusOK, task)
}

// Handler to delete a task
func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	taskDB.Lock()
	defer taskDB.Unlock()

	// Convert taskID to integer
	id, err := fmt.Sscanf(taskID, "%d")
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	// Delete the task from the map
	_, exists := taskDB.tasks[id]
	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	delete(taskDB.tasks, id)

	w.WriteHeader(http.StatusNoContent)
}

// Concurrent task processing
func processTasksConcurrently(tasks []Task, ch chan Task) {
	for _, task := range tasks {
		// Simulating some processing time
		go func(task Task) {
			time.Sleep(1 * time.Second)
			task.Completed = true
			ch <- task // Send back processed task to channel
		}(task)
	}
}

func main() {
	// Setup HTTP routes
	http.HandleFunc("/tasks", getTasksHandler)
	http.HandleFunc("/tasks/create", createTaskHandler)
	http.HandleFunc("/tasks/update", updateTaskHandler)
	http.HandleFunc("/tasks/delete", deleteTaskHandler)

	// Concurrent task processing (mocked)
	tasks := []Task{
		{ID: 1, Title: "Task 1", Completed: false, CreatedAt: time.Now()},
		{ID: 2, Title: "Task 2", Completed: false, CreatedAt: time.Now()},
		{ID: 3, Title: "Task 3", Completed: false, CreatedAt: time.Now()},
	}

	// Channel for task results
	ch := make(chan Task)
	go processTasksConcurrently(tasks, ch)

	// Handling concurrent task results
	go func() {
		for task := range ch {
			taskDB.Lock()
			taskDB.tasks[task.ID] = task
			taskDB.Unlock()
			log.Printf("Task %d processed: %s", task.ID, task.Title)
		}
	}()

	// Start the server
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

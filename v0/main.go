package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	_ "github.com/mattn/go-sqlite3"
)

type Service struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	URL           string `json:"url"`
	Status        bool   `json:"status"`
	StatusHistory []bool `json:"statusHistory"`
}

var (
	services     []Service
	clients      = make(map[chan []Service]bool)
	clientsMutex sync.Mutex
	db           *sql.DB
)

func initDB() error {
	var err error
	db, err = sql.Open("sqlite3", "./status.db")
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS services (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			url TEXT
		);
		CREATE TABLE IF NOT EXISTS status_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			service_id INTEGER,
			status BOOLEAN,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (service_id) REFERENCES services(id)
		);
	`)
	return err
}

func loadConfig() error {
	file, err := os.ReadFile("config.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(file, &services)
	if err != nil {
		return err
	}

	for i, service := range services {
		var id int
		err := db.QueryRow("SELECT id FROM services WHERE name = ? AND url = ?", service.Name, service.URL).Scan(&id)
		if err == sql.ErrNoRows {
			result, err := db.Exec("INSERT INTO services (name, url) VALUES (?, ?)", service.Name, service.URL)
			if err != nil {
				return err
			}
			id64, _ := result.LastInsertId()
			id = int(id64)
		} else if err != nil {
			return err
		}
		services[i].ID = id
		services[i].StatusHistory, err = getStatusHistory(id)
		if err != nil {
			return err
		}
		// Ensure StatusHistory has at least 30 elements
		for len(services[i].StatusHistory) < 30 {
			services[i].StatusHistory = append(services[i].StatusHistory, false)
		}
	}
	return nil
}

func getStatusHistory(serviceID int) ([]bool, error) {
	rows, err := db.Query("SELECT status FROM status_history WHERE service_id = ? ORDER BY timestamp DESC LIMIT 30", serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []bool
	for rows.Next() {
		var status bool
		if err := rows.Scan(&status); err != nil {
			return nil, err
		}
		history = append(history, status)
	}
	return history, nil
}

func watchConfig() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("Error creating watcher:", err)
		return
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					fmt.Println("Config file modified. Reloading...")
					loadConfig()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Println("Error:", err)
			}
		}
	}()

	err = watcher.Add("config.json")
	if err != nil {
		fmt.Println("Error adding config file to watcher:", err)
	}
}

func checkStatus(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func updateStatuses() {
	for i, service := range services {
		currentStatus := checkStatus(service.URL)
		services[i].Status = currentStatus
		services[i].StatusHistory = append([]bool{currentStatus}, services[i].StatusHistory[:29]...)

		_, err := db.Exec("INSERT INTO status_history (service_id, status) VALUES (?, ?)", service.ID, currentStatus)
		if err != nil {
			fmt.Println("Error inserting status history:", err)
		}
	}
}

func broadcastServices() {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	for clientChan := range clients {
		clientChan <- services
	}
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	clientChan := make(chan []Service)
	clientsMutex.Lock()
	clients[clientChan] = true
	clientsMutex.Unlock()

	defer func() {
		clientsMutex.Lock()
		delete(clients, clientChan)
		clientsMutex.Unlock()
	}()

	for {
		select {
		case <-r.Context().Done():
			return
		case services := <-clientChan:
			data, _ := json.Marshal(services)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.(http.Flusher).Flush()
		}
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, services)
}

func main() {
	err := initDB()
	if err != nil {
		fmt.Println("Error initializing database:", err)
		return
	}
	defer db.Close()

	err = loadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	watchConfig()

	go func() {
		for {
			updateStatuses()
			broadcastServices()
			time.Sleep(1* time.Minute)
		}
	}()

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/sse", handleSSE)

	fmt.Println("Server starting on :8080")
	http.ListenAndServe(":8080", nil)
}

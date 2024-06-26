package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Site struct {
	ID  string `json:"id" gorm:"primaryKey"`
	URL string `json:"url"`
}

type Config struct {
	Sites []Site `json:"sites"`
}

type StatusData struct {
	ID     uint      `gorm:"primaryKey"`
	SiteID string    `json:"site_id"`
	Status int       `json:"status"`
	Time   time.Time `json:"time"`
	Msg    string    `json:"msg"`
	Ping   int       `json:"ping"`
}

type HeartbeatList map[string][]StatusData

type UptimeList map[string]float64

type ResponseData struct {
	HeartbeatList HeartbeatList `json:"heartbeatList"`
	UptimeList    UptimeList    `json:"uptimeList"`
}

var (
	config Config
	db     *gorm.DB
)

func main() {
	var err error
	db, err = gorm.Open(sqlite.Open("status.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&Site{}, &StatusData{})
	if err != nil {
		log.Fatalf("Failed to migrate database schema: %v", err)
	}

	err = loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	http.HandleFunc("/status", handleStatus)
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func loadConfig() error {
	file, err := os.ReadFile("config.json")
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	err = json.Unmarshal(file, &config)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Upsert sites from config into database
	for _, site := range config.Sites {
		result := db.Where(Site{ID: site.ID}).Assign(Site{URL: site.URL}).FirstOrCreate(&site)
		if result.Error != nil {
			return fmt.Errorf("failed to upsert site %s: %w", site.ID, result.Error)
		}
	}

	return nil
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	for {
		data, err := getStatusData()
		if err != nil {
			log.Printf("Error getting status data: %v", err)
			// Send an error event to the client
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		} else {
			jsonData, err := json.Marshal(data)
			if err != nil {
				log.Printf("Error marshaling JSON: %v", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
		}
		w.(http.Flusher).Flush()

		time.Sleep(30 * time.Second)
	}
}


func getStatusData() (ResponseData, error) {
    heartbeatList := make(HeartbeatList)
    uptimeList := make(UptimeList)

    var sites []Site
    result := db.Find(&sites)
    if result.Error != nil {
        return ResponseData{}, fmt.Errorf("failed to fetch sites: %w", result.Error)
    }

    for _, site := range sites {
        status, err := checkSiteStatus(site.URL)
        if err != nil {
            log.Printf("Error checking status for site %s: %v", site.ID, err)
            continue
        }
        status.SiteID = site.ID
        result = db.Create(&status)
        if result.Error != nil {
            log.Printf("Error saving status for site %s: %v", site.ID, result.Error)
            continue
        }

        var statuses []StatusData
        result = db.Where("site_id = ?", site.ID).Order("time desc").Limit(50).Find(&statuses)
        if result.Error != nil {
            log.Printf("Error fetching statuses for site %s: %v", site.ID, result.Error)
            continue
        }
        heartbeatList[site.ID] = statuses

        uptime, err := calculateUptime(site.ID)
        if err != nil {
            log.Printf("Error calculating uptime for site %s: %v", site.ID, err)
            continue
        }
        uptimeList[site.ID+"_24"] = uptime
    }

    return ResponseData{
        HeartbeatList: heartbeatList,
        UptimeList:    uptimeList,
    }, nil
}

// ... (rest of the code remains the same)
func checkSiteStatus(url string) (StatusData, error) {
	start := time.Now()
	resp, err := http.Get(url)
	if err != nil {
		return StatusData{
			Status: 0,
			Time:   time.Now(),
			Msg:    err.Error(),
			Ping:   0,
		}, nil
	}
	defer resp.Body.Close()

	ping := int(time.Since(start).Milliseconds())

	status := 1
	if resp.StatusCode != http.StatusOK {
		status = 0
	}

	return StatusData{
		Status: status,
		Time:   time.Now(),
		Msg:    resp.Status,
		Ping:   ping,
	}, nil
}

func calculateUptime(siteID string) (float64, error) {
	var count int64
	result := db.Model(&StatusData{}).Where("site_id = ? AND status = 1 AND time > ?", siteID, time.Now().Add(-24*time.Hour)).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to calculate uptime: %w", result.Error)
	}

	total := 24 * 60 * 60 / 30 // Assuming we check every 30 seconds
	return float64(count) / float64(total), nil
}

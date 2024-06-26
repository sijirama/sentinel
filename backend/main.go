package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"
	"time"
)

type Site struct {
	ID   string `json:"id" gorm:"primaryKey"`
	URL  string `json:"url"`
	Name string `json:"name"`
}

type StatusData struct {
	ID     string    `gorm:"primaryKey"`
	Time   time.Time `json:"time"`
	SiteId string    `gorm:"foreignKey:ID" json:"site_id"`
	Status int       `json:"status"`
	Msg    string    `json:"msg"`
	Ping   int       `json:"ping"`
}

type Config struct {
	Sites []Site `json:"sites"`
}

type SiteStatus struct {
	Site     Site         `json:"site"`
	Statuses []StatusData `json:"statuses"`
	Uptime   float64      `json:"uptime"`
}

type ResponseData struct {
	SiteStatuses map[string]SiteStatus `json:"siteStatuses"`
}

var (
	config  Config
	storage *gorm.DB // lord forgive me, i declared global and non global db
)

func connectToDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("main.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&Site{}, &StatusData{})

	storage = db
	return db
}

func Load_Config(db *gorm.DB) {
	configFile, err := os.ReadFile("config.json")
	if err != nil {
		log.Panicln("Failed to load config file")
	}
	json.Unmarshal(configFile, &config)

	for _, site := range config.Sites {
		result := db.Where(&Site{ID: site.ID}).Assign(Site{Name: site.Name, URL: site.URL}).FirstOrCreate(&site)

		if result.Error != nil {
			log.Panicln("Failed to update server")
		}

	}
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	for {
		data, err := getStatusData()
		if err != nil {
			log.Printf("Error fetching status data: %v", err)
		} else {
			jsonData, err := json.Marshal(data)
			if err != nil {
				log.Printf("Error marshalling data: %v", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
		}
		w.(http.Flusher).Flush()
		time.Sleep(30 * time.Second)
	}

}

func getStatusData() (ResponseData, error) {
	siteStatuses := make(map[string]SiteStatus)
	var sites []Site
	query := storage.Find(&sites)
	if query.Error != nil {
		return ResponseData{}, fmt.Errorf("Failed to fetch sites: %v", query.Error)
	}

	fmt.Println(sites)

	for _, site := range sites {

		stat, err := checkSiteStatus(site.URL)
		if err != nil {
			log.Printf("Error checking status for site %s: %v", site.ID, err)
			continue
		}

		stat.SiteId = site.ID
		stat.ID = uuid.New().String()
		result := storage.Create(&stat)
		if result.Error != nil {
			log.Printf("Error saving status for site %s: %v", site.ID, result.Error)
			continue
		}

		var statuses []StatusData
		result = storage.Where("site_id = ?", site.ID).Order("time desc").Limit(60).Find(&statuses)
		if result.Error != nil {
			log.Printf("Error fetching status for site %s: %v", site.ID, result.Error)
			continue
		}

		uptime, error := calculateUptime(site.ID)
		if error != nil {
			log.Panicf("Error saving status for site %s: %v", site.ID, error)
			continue
		}

		siteStatuses[site.ID] = SiteStatus{
			Site:     site,
			Uptime:   uptime,
			Statuses: statuses,
		}

	}

	return ResponseData{SiteStatuses: siteStatuses}, nil
}

func calculateUptime(siteID string) (float64, error) {
	var count int64
	result := storage.Model(&StatusData{}).Where("site_id = ? AND status = 1 AND time > ?", siteID, time.Now().Add(-24*time.Hour)).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to calculate uptime: %w", result.Error)
	}
	total := 24 * 60 * 60 / 30 // Assuming we check every 30 seconds
	return (float64(count) / float64(total)) * 100, nil
}

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

func main() {
	db := connectToDB()
	Load_Config(db)

	http.HandleFunc("/status", handleStatus)

	fmt.Println("Server is running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

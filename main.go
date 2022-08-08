// Testing program.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// Server config.
type Config struct {
	Port     int
	Host     string
	Username string
	Password string
	Database string
}

// Load and parse server config file.
func loadConfig() (*Config, error) {
	data, err := os.ReadFile("config.json")
	var config Config
	if err != nil {
		return nil, err
	}
	json.Unmarshal(data, &config)
	return &config, nil
}

func toLog(message string) {
	year, month, day := time.Now().Date()
	f, err := os.OpenFile(fmt.Sprintf("tester_%4d-%0.2d-%0.2d.log", year, month, day), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)
	log.Println(message)
}

func main() {
	config, err := loadConfig()
	if err != nil {
		panic(err)
	}
	databaseConnect(config.Host, config.Username, config.Password, config.Database)

	startHttpServer(config.Port)
}

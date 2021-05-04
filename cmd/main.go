package main

import (
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.TraceLevel)
	err := godotenv.Load()
	if err != nil {
		log.Infof("Error loading .env file, assuming production: %s", err.Error())
	}
}

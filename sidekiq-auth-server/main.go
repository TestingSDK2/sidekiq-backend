package main

import (
	"fmt"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-auth-server/cmd"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-auth-server/util"
)

func main() {
	currentTime := time.Now()
	istLocation, err := time.LoadLocation("Asia/Kolkata") // IST timezone
	if err != nil {
		fmt.Println("Error loading IST timezone:", err)
		return
	}
	data := map[string]interface{}{
		"startTime":   currentTime.In(istLocation).Format("January 02, 2006 - 03:04:05 PM MST (") + istLocation.String() + ")",
		"message":     "Starting authentication backend server . . .",
		"codeVersion": "1.1.2",
		"repo":        "sidekiq-server",
	}
	util.PrettyPrint(data)
	cmd.New().Execute()
}

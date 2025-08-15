package main

import (
	"flag"
	"log"
	"manindexer/basicprotocols/group_chat"
	"os"
)

// @title Group Chat API
// @version 1.0
// @description Group Chat Service API documentation, including database queries, group management, community management and other functions
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host 0.0.0.0:7568
// @BasePath /

// @tag.name Database Query
// @tag.description Database query related APIs for viewing data in Pebble database

// @tag.name Group Management
// @tag.description Group management related APIs, including group information, member management, etc.

// @tag.name Community Management
// @tag.description Community management related APIs, including community information, member management, etc.

// @tag.name Chat Function
// @tag.description Chat function related APIs, including messages, queues, etc.

// @tag.name User Management
// @tag.description User management related APIs, including user information, group lists, etc.

func main() {
	// Parse command line arguments
	var (
		host = flag.String("host", "0.0.0.0", "Server listening address")
		port = flag.String("port", "8080", "Server listening port")
	)
	flag.Parse()

	// Set log format
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Group Chat Service...")

	// Create server configuration
	config := &group_chat.ServerConfig{
		Host: *host,
		Port: *port,
	}

	// Run server
	err := group_chat.RunWithConfig(config, nil)
	if err != nil {
		log.Printf("Failed to start server: %v", err)
		os.Exit(1)
	}
}

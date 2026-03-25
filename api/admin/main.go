package main

import (
	"fmt"
	"os"

	"github.com/photoview/photoview/api/admin/migration"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: admin <command>")
		fmt.Println("Commands:")
		fmt.Println("  clean-path    Clean the photo directory path")
		fmt.Println("  clean-users   Clear all users from database")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "clean-path":
		if err := migration.CleanPath(os.Args[2]); err != nil {
			fmt.Printf("Error cleaning path: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Path cleaned successfully")

	case "clean-users":
		if err := migration.ClearUsers(); err != nil {
			fmt.Printf("Error clearing users: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Users cleared successfully")

	case "clean-albums":
		if err := migration.CleanAlbums(); err != nil {
			fmt.Printf("Error clearing albums: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Albums cleared successfully")

	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

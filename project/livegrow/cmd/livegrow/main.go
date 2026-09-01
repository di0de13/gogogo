package main

import "fmt"

// Version is deliberately small in the bootstrap unit; service behavior will be added incrementally.
const Version = "0.1.0"

func main() {
	fmt.Printf("livegrow %s\n", Version)
}

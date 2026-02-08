package main

import "fmt"

func greet(name string) string {
	if name == "" {
		return "Hello, World!"
	} else {
		return fmt.Sprintf("Hello, %s!", name)
	}
}

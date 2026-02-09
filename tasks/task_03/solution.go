package main

import (
	"fmt"
	"strconv"
)

func fizzBuzz(n int) (string, error) {

	if n < 0 {
		return "", fmt.Errorf("")
	}
	var result string = ""
	if n%3 == 0 {
		result += "Fizz"
	}
	if n%5 == 0 {
		result += "Buzz"
	}
	if result == "" {
		return strconv.Itoa(n), nil
	} else {
		return result, nil
	}
}

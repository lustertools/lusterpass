package main

import (
	"fmt"
	"os"
	"sort"
)

func main() {
	keys := []string{"APP_NAME", "ENV", "LOG_LEVEL", "TEST_SECRET_A", "TEST_SECRET_B"}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%s=%s\n", k, os.Getenv(k))
	}
}

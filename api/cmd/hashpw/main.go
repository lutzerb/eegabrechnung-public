// hashpw bcrypt-hashes a password for manual INSERT statements (e.g. bootstrapping
// a new organization's first admin user — see scripts/bootstrap-customer-org.sql).
// Usage: go run ./cmd/hashpw <plaintext-password>
package main

import (
	"fmt"
	"os"

	"github.com/lutzerb/eegabrechnung/internal/auth"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: hashpw <plaintext-password>")
		os.Exit(1)
	}
	hash, err := auth.HashPassword(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println(hash)
}

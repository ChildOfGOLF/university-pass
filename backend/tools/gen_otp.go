package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/pquerna/otp/totp"
)

func main() {
	secret := flag.String("secret", "", "secret")
	flag.Parse()

	if *secret == "" {
		fmt.Println("Less secret")
		return
	}

	code, err := totp.GenerateCode(*secret, time.Now().UTC())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("OTP: %s\n", code)
}

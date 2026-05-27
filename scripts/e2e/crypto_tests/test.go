package main

import (
	"fmt"
	"net/http"
	"io"
)

func main() {
	req, err := http.NewRequest("GET", "http://localhost:8000/api/crypto/dashboard", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.Status)
	for k, v := range resp.Header {
		fmt.Printf("%s: %v\n", k, v)
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Body:", string(body))
}

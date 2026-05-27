package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
)

func main() {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Backend-Header", "yes")
		w.WriteHeader(200)
		w.Write([]byte("backend"))
	}))
	defer backend.Close()

	target, _ := url.Parse(backend.URL)
	proxy := httputil.NewSingleHostReverseProxy(target)

	frontend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "test", Value: "123"})
		proxy.ServeHTTP(w, r)
	}))
	defer frontend.Close()

	resp, _ := http.Get(frontend.URL)
	fmt.Println("Cookies:", resp.Cookies())
	fmt.Println("Headers:", resp.Header)
}

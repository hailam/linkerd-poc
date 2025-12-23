package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type SelfInfo struct {
	AppName  string `json:"app_name"`
	Hostname string `json:"hostname"`
}

type CallerInfo struct {
	L5dClientID string `json:"l5d_client_id,omitempty"`
	RawHeader   string `json:"raw_header,omitempty"`
}

type Response struct {
	Self           SelfInfo          `json:"self"`
	Caller         CallerInfo        `json:"caller"`
	LinkerdHeaders map[string]string `json:"linkerd_headers"`
}

type CallEchoResponse struct {
	Self         SelfInfo `json:"self"`
	EchoResponse Response `json:"echo_response"`
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getSelfInfo() SelfInfo {
	hostname, _ := os.Hostname()
	return SelfInfo{
		AppName:  getEnv("APP_NAME", "unknown"),
		Hostname: hostname,
	}
}

func extractLinkerdHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)
	for name, values := range r.Header {
		lowerName := strings.ToLower(name)
		if strings.HasPrefix(lowerName, "l5d-") {
			headers[lowerName] = strings.Join(values, ", ")
		}
	}
	return headers
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	linkerdHeaders := extractLinkerdHeaders(r)
	clientID := r.Header.Get("l5d-client-id")

	resp := Response{
		Self: getSelfInfo(),
		Caller: CallerInfo{
			L5dClientID: clientID,
			RawHeader:   clientID,
		},
		LinkerdHeaders: linkerdHeaders,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func callEchoHandler(w http.ResponseWriter, r *http.Request) {
	echoURL := getEnv("ECHO_SERVICE_URL", "http://echo-service")

	resp, err := http.Get(echoURL)
	if err != nil {
		http.Error(w, "Failed to call echo-service: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read echo response: "+err.Error(), http.StatusBadGateway)
		return
	}

	var echoResp Response
	if err := json.Unmarshal(body, &echoResp); err != nil {
		http.Error(w, "Failed to parse echo response: "+err.Error(), http.StatusBadGateway)
		return
	}

	callResp := CallEchoResponse{
		Self:         getSelfInfo(),
		EchoResponse: echoResp,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(callResp)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func main() {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/call-echo", callEchoHandler)
	http.HandleFunc("/health", healthHandler)

	port := getEnv("PORT", "8080")
	log.Printf("Starting server on :%s (APP_NAME=%s)", port, getEnv("APP_NAME", "unknown"))
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

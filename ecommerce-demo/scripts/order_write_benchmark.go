// Concurrent local benchmark for POST /api/order/create.
// It creates a dedicated test user and prints aggregate latency/status results.
package main

import (
	"bytes"
	"crypto/rand"
	mathrand "math/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

type apiResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type result struct {
	latency time.Duration
	status  int
	success bool
	err     string
}

func main() {
	gateway := flag.String("gateway", envOr("GATEWAY", "http://localhost:30088"), "gateway URL")
	requests := flag.Int("requests", envInt("REQUESTS", 100), "total order requests")
	concurrency := flag.Int("concurrency", envInt("CONCURRENCY", 4), "parallel workers")
	rate := flag.Int("rate", envInt("RATE", 0), "global request rate; 0 sends as fast as possible")
	productID := flag.Int64("product-id", int64(envInt("PRODUCT_ID", 900001)), "seeded benchmark product ID")
	productMax := flag.Int64("product-max", int64(envInt("PRODUCT_MAX", 0)), "if >0, random product id in [1, productMax]")
	flag.Parse()

	if *requests <= 0 || *concurrency <= 0 {
		fmt.Fprintln(os.Stderr, "requests and concurrency must be positive")
		os.Exit(2)
	}

	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{
		MaxIdleConns:        *concurrency * 2,
		MaxIdleConnsPerHost: *concurrency * 2,
	}}
	token, err := createTestUser(client, *gateway)
	if err != nil {
		fmt.Fprintln(os.Stderr, "test-user setup failed:", err)
		os.Exit(1)
	}

	fmt.Printf("Order-write benchmark: requests=%d concurrency=%d rate=%d req/s product_id=%d\n", *requests, *concurrency, *rate, *productID)
	started := time.Now()
	jobs := make(chan struct{})
	results := make(chan result, *requests)
	var workers sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range jobs {
				pid := *productID
			if *productMax > 0 {
				pid = mathrand.Int63n(*productMax) + 1
			}
			results <- createOrder(client, *gateway, token, pid)
			}
		}()
	}
	if *rate > 0 {
		interval := time.Second / time.Duration(*rate)
		ticker := time.NewTicker(interval)
		for i := 0; i < *requests; i++ {
			jobs <- struct{}{}
			if i+1 < *requests {
				<-ticker.C
			}
		}
		ticker.Stop()
	} else {
		for i := 0; i < *requests; i++ {
			jobs <- struct{}{}
		}
	}
	close(jobs)
	workers.Wait()
	close(results)
	elapsed := time.Since(started)

	var latencies []time.Duration
	statusCounts := map[int]int{}
	errorCounts := map[string]int{}
	successes := 0
	for r := range results {
		statusCounts[r.status]++
		if r.success {
			successes++
			latencies = append(latencies, r.latency)
		} else if r.err != "" {
			errorCounts[r.err]++
		}
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	fmt.Printf("success=%d failed=%d success_rate=%.3f%% throughput=%.2f req/s\n", successes, *requests-successes, float64(successes)*100/float64(*requests), float64(successes)/elapsed.Seconds())
	if len(latencies) > 0 {
		fmt.Printf("latency p50=%s p90=%s p99=%s max=%s\n", percentile(latencies, .50), percentile(latencies, .90), percentile(latencies, .99), latencies[len(latencies)-1])
	}
	fmt.Printf("api_code=%v\n", statusCounts)
	if len(errorCounts) > 0 {
		fmt.Printf("errors=%v\n", errorCounts)
	}
}

func createTestUser(client *http.Client, gateway string) (string, error) {
	suffix := make([]byte, 3)
	if _, err := rand.Read(suffix); err != nil {
		return "", err
	}
	username := fmt.Sprintf("ob_%d_%x", time.Now().Unix(), suffix) // <= 20 chars
	password := "test123456"
	registered, err := callAPI(client, gateway+"/api/user/register", "", map[string]string{"username": username, "password": password})
	if err != nil {
		return "", err
	}
	if registered.Code != 0 {
		return "", fmt.Errorf("register: %s", registered.Msg)
	}
	response, err := callAPI(client, gateway+"/api/user/login", "", map[string]string{"username": username, "password": password})
	if err != nil {
		return "", err
	}
	if response.Code != 0 {
		return "", fmt.Errorf("login: %s", response.Msg)
	}
	var data struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(response.Data, &data); err != nil {
		return "", err
	}
	if data.AccessToken == "" {
		return "", fmt.Errorf("login returned no access token")
	}
	return data.AccessToken, nil
}

func createOrder(client *http.Client, gateway, token string, productID int64) result {
	started := time.Now()
	response, err := callAPI(client, gateway+"/api/order/create", token, map[string]any{"productId": productID, "count": 1})
	r := result{latency: time.Since(started)}
	if err != nil {
		r.err = err.Error()
		return r
	}
	r.status = response.Code
	r.success = response.Code == 0
	if !r.success {
		r.err = response.Msg
	}
	return r
}

func callAPI(client *http.Client, url, token string, payload any) (apiResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return apiResponse{}, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return apiResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return apiResponse{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiResponse{}, fmt.Errorf("http %d", resp.StatusCode)
	}
	var response apiResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return apiResponse{}, err
	}
	return response, nil
}

func percentile(values []time.Duration, p float64) time.Duration {
	index := int(float64(len(values)-1) * p)
	return values[index]
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func envInt(key string, fallback int) int {
	var value int
	if _, err := fmt.Sscanf(os.Getenv(key), "%d", &value); err == nil && value > 0 {
		return value
	}
	return fallback
}

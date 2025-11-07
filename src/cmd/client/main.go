package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/petz1209/goratelimiter/client"
)

func main() {

	//rl := client.NewRateLimitClient("localhost:8000")
	pool, err := client.NewPool("localhost:8000", 10)
	if err != nil {
	}
	defer pool.Close()
	mw := &RateLimitMw{client: pool}

	mux := http.NewServeMux()
	mux.HandleFunc("/{user}", LogMiddleware(mw.middleware(handler)))

	srv := http.Server{
		Addr:    ":3000",
		Handler: mux,
	}

	fmt.Println("running webserver on port 3000")
	srv.ListenAndServe()

}

func handler(w http.ResponseWriter, r *http.Request) {

	metrics := r.Context().Value("metrics").(*RequestMetrics)
	msg := map[string]string{"message": "ok"}

	time.Sleep(5 * time.Second)
	metrics.StatusCode = 200
	metrics.VolumeRows = 10
	metrics.VolumeByte = 100
	w.WriteHeader(metrics.StatusCode)
	json.NewEncoder(w).Encode(msg)

}

const MetricKey = "metrics"

type RequestMetrics struct {
	Start      time.Time
	End        time.Time
	StatusCode int
	VolumeRows int
	VolumeByte int
}

type RateLimitMw struct {
	client client.RateLimiter
}

func (rt *RateLimitMw) middleware(next func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {

	TO_MANY_TOTAL_REQUESTS := map[string]string{"message": "to many total requests"}
	TO_MANY_USER_REQUESTS := map[string]string{"message": "Currently your usergroup is running to many requests."}
	MAX_VOLUME_REACHED_MSG := map[string]string{"message": "your daily data volume is exhausted for today."}
	MAX_CONCURRENCY_BY_USER := 5
	MAX_VOLUME := 100000

	return func(w http.ResponseWriter, r *http.Request) {
		user := r.PathValue("user")
		metrics := r.Context().Value(MetricKey).(*RequestMetrics)
		code, e := rt.client.Aquire(user, MAX_CONCURRENCY_BY_USER, MAX_VOLUME)
		if e != nil {
			slog.Error("Error in RateLimiitMW in getStatus call with: " + e.Error())
		}
		if e != nil {
			slog.Error("getting ratelimit status failed with:  " + e.Error())
			next(w, r)
			return
		}

		switch code {
		case client.OK:
			next(w, r)
			rt.client.Return(user, metrics.VolumeRows)
		case client.MAX_USER_CONCURRENCY_REACHED:
			metrics.StatusCode = 429
			w.WriteHeader(metrics.StatusCode)
			json.NewEncoder(w).Encode(TO_MANY_USER_REQUESTS)
			return
		case client.MAX_TOTAL_CONCURRENCY_REACHED:
			metrics.StatusCode = 429
			w.WriteHeader(metrics.StatusCode)
			json.NewEncoder(w).Encode(TO_MANY_TOTAL_REQUESTS)
			return
		case client.MAX_VOLUME_REACHED:
			metrics.StatusCode = 429
			w.WriteHeader(metrics.StatusCode)
			json.NewEncoder(w).Encode(MAX_VOLUME_REACHED_MSG)
			return

			// we default to run it anyways
		default:
			next(w, r)
			rt.client.Return(user, 30)
		}

	}

}

// LogMiddleware
//
//	takes care of logging
func LogMiddleware(next func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		// instanciate the metrics. we use this to pass its pointer to the context so we can read it later.
		metrics := &RequestMetrics{
			Start: time.Now(),
		}

		ctx := context.WithValue(r.Context(), MetricKey, metrics)
		next(w, r.WithContext(ctx))
		metrics.End = time.Now()
		dur := time.Since(metrics.Start)
		fmt.Printf("[%d] %s  start: %s  end: %s duration: %s volume: %d\n", metrics.StatusCode, r.URL, metrics.Start, metrics.End, dur, metrics.VolumeRows)

	}

}

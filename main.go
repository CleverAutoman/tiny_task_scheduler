package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Emotion string

const (
	PLEASANT Emotion = "PLEASANT"
	NEUTRAL  Emotion = "NEUTRAL"
	AVERSIVE Emotion = "AVERSIVE"
)

type Task struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Emotion       Emotion `json:"emotion"`
	MinutesNeeded int     `json:"minutesNeeded"`
	Importance    int     `json:"importance"` // 1~5
	DueAt         *string `json:"dueAt"`      // RFC3339 or null
}

var (
	dataFile = "tasks.json"
	mu       sync.RWMutex
	tasks    = map[string]Task{}
)

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func urgencyScore(t Task, now time.Time) float64 {
	if t.DueAt == nil {
		return clamp(0.15*float64(t.Importance), 0, 0.8)
	}
	d, err := time.Parse(time.RFC3339, *t.DueAt)
	if err != nil {
		return clamp(0.15*float64(t.Importance), 0, 0.8)
	}
	minLeft := time.Until(d).Minutes()
	if minLeft < 0 {
		minLeft = 0
	}
	timePressure := 1.0 / math.Log10(10+minLeft)
	importanceBoost := 0.2 * float64(t.Importance-1)
	return clamp(timePressure+importanceBoost, 0, 1.5)
}

func fitScore(t Task, freeMin int) float64 {
	if freeMin <= 0 {
		return 0
	}
	if t.MinutesNeeded <= freeMin {
		return clamp(0.6+0.4*(1.0/(1+math.Log10(1+float64(t.MinutesNeeded)))), 0, 1)
	}
	ratio := float64(freeMin) / float64(t.MinutesNeeded)
	return clamp(0.3*ratio, 0, 1)
}

func emotionScore(t Task, stress int) float64 {
	switch t.Emotion {
	case PLEASANT:
		if stress >= 4 {
			return 0.30
		}
		return 0.15
	case NEUTRAL:
		return 0.10
	case AVERSIVE:
		if stress >= 4 {
			return -0.10
		}
		return 0.05
	default:
		return 0
	}
}

func stressMatchScore(t Task, stress int) float64 {
	if stress >= 4 {
		return clamp(1.0/(1+math.Log10(5+float64(t.MinutesNeeded))), 0, 1)
	} else if stress <= 2 {
		return clamp(math.Log10(10+float64(t.MinutesNeeded))/3.0, 0, 1)
	}
	return 0.5
}

func score(t Task, now time.Time, freeMin, stress int) float64 {
	// 权重：可调整
	urgencyW, fitW, emotionW, stressW := 0.50, 0.25, 0.15, 0.10
	return urgencyW*urgencyScore(t, now) +
		fitW*fitScore(t, freeMin) +
		emotionW*emotionScore(t, stress) +
		stressW*stressMatchScore(t, stress)
}

func load() {
	b, err := os.ReadFile(dataFile)
	if err != nil {
		return
	}
	var arr []Task
	if json.Unmarshal(b, &arr) == nil {
		mu.Lock()
		defer mu.Unlock()
		tasks = map[string]Task{}
		for _, t := range arr {
			tasks[t.ID] = t
		}
	}
}

func save() {
	tmp := dataFile + ".tmp"
	mu.RLock()
	arr := make([]Task, 0, len(tasks))
	for _, t := range tasks {
		arr = append(arr, t)
	}
	mu.RUnlock()
	b, _ := json.MarshalIndent(arr, "", "  ")
	_ = os.WriteFile(tmp, b, 0644)
	_ = os.Rename(tmp, dataFile)
}

func withCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func main() {

	if p, err := os.Executable(); err == nil {
		os.Chdir(filepath.Dir(p))
	}
	load()
	go func() {
		t := time.NewTicker(30 * time.Second)
		for range t.C {
			save()
		}
	}()

	http.HandleFunc("/health", withCORS(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))

	http.HandleFunc("/tasks", withCORS(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			mu.RLock()
			arr := make([]Task, 0, len(tasks))
			for _, t := range tasks {
				arr = append(arr, t)
			}
			mu.RUnlock()
			writeJSON(w, http.StatusOK, arr)
		case http.MethodPost:
			var t Task
			if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			if t.ID == "" {
				http.Error(w, "id required", 400)
				return
			}
			mu.Lock()
			tasks[t.ID] = t
			mu.Unlock()
			save()
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "count": len(tasks)})
		default:
			http.Error(w, "method not allowed", 405)
		}
	}))

	http.HandleFunc("/tasks/", withCORS(func(w http.ResponseWriter, r *http.Request) {
		// DELETE /tasks/:id
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", 405)
			return
		}
		id := r.URL.Path[len("/tasks/"):]
		mu.Lock()
		delete(tasks, id)
		mu.Unlock()
		save()
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "count": len(tasks)})
	}))

	http.HandleFunc("/order", withCORS(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		freeMin, _ := strconv.Atoi(q.Get("freeMin"))
		if freeMin == 0 {
			freeMin = 30
		}
		stress, _ := strconv.Atoi(q.Get("stress"))
		if stress == 0 {
			stress = 3
		}
		now := time.Now()

		mu.RLock()
		arr := make([]Task, 0, len(tasks))
		for _, t := range tasks {
			arr = append(arr, t)
		}
		mu.RUnlock()

		sort.Slice(arr, func(i, j int) bool {
			si := score(arr[i], now, freeMin, stress)
			sj := score(arr[j], now, freeMin, stress)
			if si != sj {
				return si > sj
			}

			di, dj := int64(1<<62), int64(1<<62)
			if arr[i].DueAt != nil {
				if tt, err := time.Parse(time.RFC3339, *arr[i].DueAt); err == nil {
					di = tt.UnixNano()
				}
			}
			if arr[j].DueAt != nil {
				if tt, err := time.Parse(time.RFC3339, *arr[j].DueAt); err == nil {
					dj = tt.UnixNano()
				}
			}
			if di != dj {
				return di < dj
			}
			return arr[i].MinutesNeeded < arr[j].MinutesNeeded
		})
		writeJSON(w, http.StatusOK, arr)
	}))

	http.HandleFunc("/next", withCORS(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		freeMin, _ := strconv.Atoi(q.Get("freeMin"))
		if freeMin == 0 {
			freeMin = 30
		}
		stress, _ := strconv.Atoi(q.Get("stress"))
		if stress == 0 {
			stress = 3
		}
		now := time.Now()

		mu.RLock()
		var best *Task
		bestScore := -1.0
		for _, t := range tasks {
			s := score(t, now, freeMin, stress)
			if s > bestScore {
				tt := t
				best = &tt
				bestScore = s
			}
		}
		mu.RUnlock()

		if best == nil {
			w.WriteHeader(204)
			return
		}
		writeJSON(w, http.StatusOK, best)
	}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

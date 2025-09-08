# Smart To-Do Scheduler · Emotion × Stress × Time

A lightweight task scheduler that recommends what to do **next** by blending **deadline pressure**, **available time**, **emotion valence**, and **current stress**.

- **Frontend:** single HTML file (vanilla JS, no deps)
- **Backend:** Go (`net/http`), in-memory store with periodic JSON persistence
- **Core:** `/order` (rank tasks) and `/next` (pick one task)

---

## ✨ Features
- Add / Update / Delete tasks; list all tasks
- Rank by context (`freeMin`, `stress`) and suggest next task
- RFC3339 timestamps; permissive CORS for easy local dev
- Auto-persist to `tasks.json` every 30s

---

## 🗂 Data Model
```json
{
  "id": "task-001",
  "title": "Write weekly report",
  "emotion": "NEUTRAL | PLEASANT | AVERSIVE",
  "minutesNeeded": 25,
  "importance": 3,
  "dueAt": "2025-09-08T03:00:00Z"   // or null
}
```

## 🚀 Quick Start
### Server
```json
go run server/main.go   # PORT defaults to 8080
```
### Client

- Open frontend/index.html in a browser
- Set “Backend base URL” to http://localhost:8080 → click Set

## 🔌 API

- GET /health → "OK"
- GET /tasks → Task[]
- POST /tasks → upsert by id (body = Task JSON)
- DELETE /tasks/:id 
- GET /order?freeMin=30&stress=3 → ranked Task[]
- GET /next?freeMin=30&stress=3 → best Task or 204

### Example

```bash
curl -X POST http://localhost:8080/tasks -H 'Content-Type: application/json' \
  -d '{"id":"task-001","title":"Write weekly report","emotion":"NEUTRAL","minutesNeeded":25,"importance":3,"dueAt":"2025-09-08T03:00:00Z"}'
```

## 🧠 Scoring
```ini 
score = 0.50*urgency + 0.25*fit + 0.15*emotion + 0.10*stressMatch
```

- urgency: deadline proximity + importance boost 
- fit: how well duration fits freeMin 
- emotion: prefer pleasant tasks under high stress; penalize aversive 
- stressMatch: short tasks under high stress; longer under low stress 
- Ties → earlier dueAt, then shorter minutesNeeded

## 💾 Persistence
- In-memory map, flushed to tasks.json every 30s 
- Loaded on startup, atomic write (tmp → rename)

## 🐳 Deploy (example)
```dockerfile
FROM golang:1.22 AS build
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o todo-server server/main.go

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /app/todo-server .
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/todo-server"]

docker build -t smart-todo .
docker run -p 8080:8080 smart-todo
```

## 🛣 Roadmap
- Config & Explainability: tune weights, show per-factor scores - 
- Scheduling: task splitting, basic deadline/calendar constraints 
- Learning: personalize weights via user feedback 
- Storage: SQLite/Postgres option 
- Multi-user: JWT auth, user data isolation 
- DevOps: OpenAPI docs, tests, CI/CD, Prometheus metrics 
- UX: i18n, dark mode, accessibility, PWA/offline

## 📜 License
MIT

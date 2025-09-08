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

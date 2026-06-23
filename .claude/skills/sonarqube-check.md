---
name: sonarqube-check
description: Checklist otomatis berdasarkan rule SonarQube untuk full stack
---

# 🛡️ Standar SonarQube (Full Stack)

**Semua kode WAJIB mematuhi checklist ini sebelum di-commit / direview:**

✅ AUTOMATIC CODE REVIEW CHECKLIST (FULL STACK)

🟢 1. QUALITY GATE (AUTO BLOCK MERGE)
PR WAJIB FAIL jika:
❌ Bug > 0
❌ Vulnerability > 0
❌ Code Smell Critical
❌ Coverage < 75%
❌ Duplication > 3%
❌ Security hotspot belum di-review

🟡 2. BACKEND (GOLANG + POSTGRESQL)
🧠 Clean Architecture
- Handler tidak berisi business logic
- Logic dipisah ke service layer
- Repository layer untuk DB query
🧾 Golang Code Style
- Gunakan gofmt / golangci-lint
- Error selalu di-handle (if err != nil)
- Tidak ada panic di production flow
- Struct tidak terlalu besar (anti God Struct)
🗄 PostgreSQL Rules
- Query pakai parameter binding (anti SQL injection)
- Index digunakan untuk kolom sering di-filter
- Hindari SELECT *
- Gunakan pagination untuk data besar
- Relasi jelas (FK constraint aktif)
⚡ Performance Backend
- Tidak ada N+1 query
- Connection pool digunakan
- Response time API < 300–500ms (ideal)

🟢 3. FRONTEND (VUE + TAILWIND)
⚙ Vue.js Rules
- Komponen reusable (tidak duplikasi UI)
- Tidak ada logic berat di template
- Composition API digunakan konsisten
- Props validation aktif
🎨 Tailwind CSS Rules
- Tidak ada inline style manual
- Tidak ada class berlebihan/duplikasi
- Gunakan design system (spacing, color konsisten)
- Responsive design wajib (mobile-first)
🧹 Frontend Cleanliness
- Tidak ada console.log
- Tidak ada unused component/import
- State management jelas (Pinia/Vuex jika perlu)

🟣 4. n8n WORKFLOW (AUTOMATION)
⚙ Workflow Rules
- Node tidak hardcode credential
- Error handling di setiap workflow
- Retry mechanism untuk API failure
- Tidak ada infinite loop workflow
- Logging aktif di setiap node penting
🔐 Security n8n
- Credential disimpan di vault n8n
- Webhook memiliki auth/token
- Tidak expose internal endpoint publik

🔴 5. SECURITY CHECK (ALL STACK)
- Tidak ada hardcoded secret / API key
- Env variable digunakan semua service
- Input validation di backend & frontend
- CORS dikontrol dengan benar
- Rate limiting aktif untuk API

🔵 6. TESTING CHECK
Backend (Go)
- Unit test service layer
- Mock database repository
- Coverage ≥ 75%
Frontend (Vue)
- Component test untuk UI critical
- API call di-test (mocked)

🟠 7. ARCHITECTURE RULE
- Separation of concerns (strict layer)
- Tidak ada direct DB call dari handler/frontend
- Shared logic dipindahkan ke common package
- API response standardized (format konsisten)

⚙️ 8. CI/CD AUTOMATION RULE
Pipeline Rule:
if sonar_quality_gate == "FAILED":
    block_merge()
Wajib PASS:
✔ Build success
✔ SonarQube PASS
✔ Test PASS
✔ Lint PASS

📊 9. FINAL CODE SCORE
Score    Status
90–100    Production Ready
80–89    Good
70–79    Needs Improvement
<70    Reject

🧠 SUMMARY RULE
"Tidak boleh ada kode masuk production yang tidak clean, secure, tested, dan scalable."

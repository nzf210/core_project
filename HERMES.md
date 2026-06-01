# Hermes Framework — WCH Platform AI-Driven Development

> **Dokumen ini adalah master guide untuk semua AI agent yang bekerja di repo ini.**
> Setiap AI agent WAJIB membaca dokumen ini di awal sesi sebelum mengerjakan task.
> Owner (manusia) hanya memberikan direction/arah — AI yang eksekusi, design, dan implement.

---

## Apa Itu Hermes?

**Hermes** adalah framework AI-driven development untuk WCH Platform.
Alih-alih owner menulis kode, owner hanya memberikan **arah/visi**:
- "Kita mau add fitur X"
- "Kita mau optimize bagian Y"
- "Kita mau refactor component Z"

**Hermes AI Agents** yang mengeksekusi semua pekerjaan teknis.
Owner mereview hasil, memberikan feedback, dan mengarahkan ulang.

---

## Arsitektur Hermes

```
Owner (Human)
    │
    ├── Goal / Direction (via /goal atau chat)
    │
    ├── Hermes Orchestrator (AI) — pemahaman + breakdown task
    │       │
    │       ├── Hermes Workflows (.hermes/workflows/) — AI agents spesifik domain
    │       │
    │       └── Hermes Coordination (.hermes/workflows/coordination/)
    │               ├── task-router.yml     — routing task ke workflow yang tepat
    │               ├── sprint-planner.yml  — planning sprint backlog
    │               └── ai-handoff.yml     — perpindahan antar domain agent
    │
    └── CI/CD Automation (.github/hermes/) — auto-run quality gates
```

---

## Jenis AI Agents

### 1. Domain Agents (Skill-based)

| Agent | Scope | Trigger |
|:------|:------|:--------|
| `analysis/*` agents | Code understanding, dependency mapping | /model, /claude-api |
| `development/*` agents | Feature implementation, refactoring | /new-feature, /build-service |
| `research/*` agents | Tech research, benchmarking | /deep-research |
| `coordination/*` agents | Sprint planning, task routing | /loop |

### 2. Built-in Claude Code Agents

| Agent | Pakai untuk |
|:------|:------------|
| `/model` Explore | Mencari kode, memahami struktur |
| `/model` Plan | Desain arsitektur baru |
| `/model` General Purpose | Task kompleks multi-step |
| `/code-review` | Security + correctness review |
| `/simplify` | Refactoring + cleanup |
| `/deep-research` | Research panjang dengan web search |

---

## Workflow Files (.hermes/workflows/)

Setiap workflow YAML mendefinisikan:
- **Trigger**: Kapan workflow ini dipanggil
- **Input**: Apa yang dibutuhkan
- **Steps**: Urutan AI agents yang dijalankan
- **Output**: Hasil akhir yang diharapkan

### Naming Convention Workflow Files

```
{agent-category}-{workflow-name}.yml

Contoh:
  analysis-code-understanding.yml
  development-new-feature.yml
  research-tech-benchmark.yml
  coordination-sprint-planner.yml
```

---

## Task Breakdown Standard

Setiap task yang masuk WAJIB di-breakdown menggunakan pattern ini:

```
TASK: [Judul Singkat]
TIPE: [feature | bugfix | refactor | research | chore]
PRIORITAS: [critical | high | medium | low]
DOMAIN: [umkm | crypto | campaign | platform | infra]
BATCH: [group task yang bisa dikerjakan paralel]

STEP 1 — Analysis
  └─ Apa yang sudah ada?
  └─ Apa yang perlu diubah?
  └─ Apa resikonya?

STEP 2 — Design
  └─ Approach mana yang dipilih? (2-3 opsi, keuntungan/kerugian masing-masing)
  └─ Keputusan: [Pilih approach X karena ...]

STEP 3 — Implementation
  └─ File mana yang diubah?
  └─ Urutan perubahan
  └─ Depedencies antar perubahan

STEP 4 — Verification
  └─ Bagaimana cara test?
  └─ Kriteria sukses

STEP 5 — Documentation
  └─ Update file mana?
```

---

## CI/CD Automation (.github/hermes/)

```
.github/hermes/
├── hermes-ci.yml        — Auto quality gate: lint + vet + test + build
├── hermes-pr-review.yml — Auto PR review oleh AI
├── hermes-task-gate.yml — Task completion gate sebelum merge
└── scripts/
    ├── pre-commit-ai.sh  — Pre-commit quality check (go vet, gofmt)
    └── post-merge-ai.sh — Post-merge automation (migration, deploy signal)
```

---

## Konvensi Commit untuk AI Agents

```
{fنوع}: {short description}

{fنوع} Jenis:
  feat    — feature baru
  fix     — bug fix
  refactor — refactoring
  perf    — performance improvement
  docs    — dokumentasi
  test    — test
  chore   — maintenance, deps
  infra   — infra, CI/CD
  security — security fix

Contoh:
  feat(crypto): add DCA order executor
  fix(umkm/accounting): correct tax calculation edge case
  refactor(auth): extract JWT middleware to shared/sdk
  infra(ci): add hermes quality gate to PR checks
```

---

## Memory System (auto-saved oleh AI)

| Type | Lokasi | Isi |
|:-----|:-------|:----|
| `user` | `/home/syahril/.claude/projects/.../memory/` | Profile owner, preferences |
| `feedback` | `.../memory/` | Corrections, confirmed approaches |
| `project` | `.../memory/` | Current goals, initiatives, deadlines |
| `reference` | `.../memory/` | Link ke sistem eksternal (Linear, Grafana) |

**Setiap AI session otomatis baca memory sebelum bekerja.**
Owner bisa bilang `/remember` untuk menyimpan informasi baru ke memory.

---

## Sprint & Backlog Automation

1. **Task masuk** → `coordination/task-router.yml` → route ke domain agent yang tepat
2. **Domain agent** → analyze + design → generate implementation plan
3. **Parallel work** → multiple domain agents bisa kerja paralel jika task independent
4. **Review gate** → owner review hasil, give feedback
5. **Iterate** → AI refine berdasarkan feedback

### Sprint Cycle (1 week default)

```
Monday    — Sprint planning (owner gives direction)
Tuesday   — AI agents implement features
Wednesday — AI agents continue + internal review
Thursday  — Owner review + feedback
Friday    — AI refine + merge + deploy signal
```

---

## Quick Commands untuk Owner

| Command | Fungsi |
|:--------|:-------|
| `/goal [arah]` | Set direction untuk sprint ini |
| `/model` | Pergi ke sub-agent untuk task spesifik |
| `/deep-research [question]` | Research panjang via web search |
| `/new-feature [deskripsi]` | Start new feature workflow |
| `/build-service [nama]` | Build specific service |
| `/db-migrate [deskripsi]` | Generate + execute migration |
| `/code-review` | AI review kode yang berubah |
| `/simplify` | AI refactor kode yang berubah |
| `/loop [interval] [prompt]` | Run task recursively |

---

## Qualitas Standard

Semua code yang di-generate oleh AI WAJIB:

- [ ] **Compile** — `go build ./...` passes
- [ ] **Lint** — `go vet ./...` clean
- [ ] **Test** — `go test ./...` passes (unit test wajib untuk logic baru)
- [ ] **Type-safe** — tidak ada `interface{}` tanpa alasan
- [ ] **Documented** — setiap exported function ada comment
- [ ] **Secure** — tidak ada hardcoded secret, parameterized queries
- [ ] **Multi-tenant aware** — semua query pakai `tenant_id`

---

## Resiko & Mitigasi

| Resiko | Mitigasi |
|:-------|:---------|
| AI generate code yang salah | Wajib CI gate + owner review |
| AI overwrite work yang sudah benar | Git branch per feature, PR review sebelum merge |
| AI bikin breaking change | Migration backward-compatible, feature flag |
| AI abuse API key | Semua secret di .env, tidak pernah di-commit |
| Context window overflow | Compact history secara berkala |

---

*Last updated: 2026-06-01 — Initial Hermes Framework setup*
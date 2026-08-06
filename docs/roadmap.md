# Platform Roadmap & Milestones

This roadmap outlines the phased development timeline for the WCH Multi-Product Platform. For a detailed architecture and product specification, please refer to the [Master Plan](docs/master_plan.md).

---

## 📅 Chronological Timeline

### Phase 1: Foundational Infra & UMKM Bookkeeping MVP ✅
*   **Month 1**: Shared Infrastructure & Core Identity
    *   Monorepo, Docker environments, central databases.
    *   `services/auth-service` (JWT, RBAC), shared SDK (`shared/sdk`), superadmin web.
*   **Month 2**: UMKM MVP & Automated Bookkeeping
    *   Double-Entry Bookkeeping core (`apps/umkm/accounting`).
    *   WhatsApp chatbot via n8n/webhooks.
    *   `frontend/umkm-web` dashboard.

### Phase 2: Monetization, AI Orchestration & Subscription Engine ✅
*   **Month 3**: AI Gateway & Billing/Monetization Engine
    *   `services/billing-service` (Xendit subscription, wallet, voucher — Lite/Pro/Ultimate tiers).
    *   `services/ai-gateway` for LLM routing, semantic caching, and per-tenant billing.
    *   AI OCR receipt scanner dan conversational accounting via WhatsApp.
*   **Month 4**: Platform Expansion *(Crypto ARCHIVED — replaced with platform hardening)*
    *   Hybrid WhatsApp architecture (whatsmeow + Meta Cloud API).
    *   Referral & affiliate system (F036, F054).
    *   Dynamic feature gating & zero-hardcode feature toggle (F066).

### Phase 3: Campaign & Election Winning Platform ✅
*   **Month 5**: Campaign Management MVP
    *   `apps/campaign` API: volunteer onboarding, geofenced surveys, voter mapping.
    *   Quick Count / Real Count system with C1 Plano OCR validation.
    *   GIS & map (Leaflet), Voter CRM, coordinator hierarchy.
*   **Month 6**: Production-Ready Infrastructure
    *   N8N Queue Mode dengan Redis worker scaling.
    *   Prometheus + Grafana monitoring — 8 dashboards (F067).
    *   Nginx reverse proxy, SSL, multi-container Docker orchestration.

### Phase 4: Quality & Security Hardening (2026-07 — ongoing)
*   Code quality compliance — SonarQube standards, file size limits, test coverage.
*   Rate limiting semua critical endpoints (auth, OTP, billing).
*   PgBouncer connection pooling untuk production scale.
*   Superadmin impersonate, dynamic feature matrix, landing page CMS (F058, F065, F066).

---

*Tier langganan aktif: **Lite / Pro / Ultimate** (lihat `services/billing-service` untuk detail quota).*

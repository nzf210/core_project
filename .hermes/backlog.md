# WCH Platform — Sprint Backlog

> Managed by Hermes AI agents. Owner sets direction; AI executes.

---

## Current Sprint

**Goal:** (set via `/goal`)
**Started:** 2026-06-01
**Status:** 🟡 Planning

---

## Backlog (Prioritized)

| # | Task | Type | Domain | Priority | Status | Notes |
|---|:-----|:-----|:-------|:---------|:-------|:------|
| 1 | Setup Hermes AI workspace | infra | platform | critical | ✅ Done | Framework created 2026-06-01 |
| 2 | SaaS Plans v2 (Lite/Pro/Business) + Voucher + Auto-ticket notification | feature | umkm | critical | ✅ Done | Migration 000025, billing-service updated |
| 3 | Superadmin UI: Kelola Harga Paket SaaS | feature | umkm | high | ✅ Done | billing PUT /admin/plans, api-gateway route, superadminApi, SuperAdminDashboard plan editor |
| 4 | (next task) | | | | | |

---

## Icebox (Future)

- [ ] Platform-wide observability: structured logging standardization
- [ ] Cross-product unified auth token exchange (UMKM ↔ Crypto ↔ Campaign)
- [ ] Shared SDK v2: extract common patterns into typed packages
- [ ] End-to-end test suite (Playwright or similar)
- [ ] Performance profiling baseline for all services

---

## Done

| # | Task | Completed | Notes |
|---|:-----|:----------|:------|
| 1 | Hermes Framework v1 | 2026-06-01 | .hermes/, .github/hermes/, HERMES.md |
| 2 | SaaS Plans + Voucher + Auto-ticket Notification | 2026-06-01 | Migration 000025, billing-service, notification-service |
| 3 | Superadmin Plan Price Editor UI | 2026-06-01 | billing PUT /admin/plans + api-gateway route + Vue component |

---

## Sprint Cadence

| Day | Activity |
|:----|:---------|
| Monday | Sprint planning (owner sets goal, AI generates backlog) |
| Tue–Wed | AI agents implement |
| Thursday | Owner review + feedback |
| Friday | AI refine + merge |

---

*Last updated: 2026-06-01*
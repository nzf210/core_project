# Platform Roadmap & Milestones

This roadmap outlines the phased development timeline for the WCH Multi-Product Platform. For a detailed architecture and product specification, please refer to the [Master Plan](file:///home/syahril/Desktop/dev/core_project/docs/master_plan.md).

---

## 📅 Chronological Timeline

### Phase 1: Foundational Infra & UMKM Bookkeeping MVP (Month 1-2)
*   **Month 1**: Shared Infrastructure & Core Identity
    *   Initialize monorepo, Docker environments, and central databases.
    *   Deploy `services/auth-service` (JWT, SSO, RBAC) and `services/tenant-service`.
    *   Initialize shared SDK (`shared/sdk`) and admin panel template (`frontend/admin-web`).
*   **Month 2**: UMKM MVP & Automated Bookkeeping
    *   Build Double-Entry Bookkeeping core in `apps/umkm/accounting`.
    *   Integrate WhatsApp API chatbot via n8n/webhooks.
    *   Launch `frontend/umkm-web` dashboard.

### Phase 2: Monetization, AI Orchestration & Crypto Trading Bot (Month 3-4)
*   **Month 3**: AI Gateway & Billing/Monetization Engine
    *   Build `services/billing-service` integrated with Stripe/Xendit.
    *   Launch `services/ai-gateway` for LLM routing, semantic caching, and cost limiting.
    *   Add AI OCR receipt scanner and conversational accounting voice/text in `apps/umkm`.
*   **Month 4**: SaaS Crypto Trading Bot
    *   Integrate `CCXT` multi-exchange trading library inside `apps/crypto`.
    *   Build the asynchronous high-frequency/scheduler bot execution worker (`apps/crypto/worker`).
    *   Launch `frontend/crypto-web` dashboard with TradingView charts and real-time PnL.
    *   Enforce billing plan restrictions on bot execution quotas.

### Phase 3: Campaign & Election Winning Platform (Month 5-6)
*   **Month 5**: Campaign Management MVP
    *   Build `apps/campaign` API for volunteer onboarding, geofenced surveys, and voter mapping.
    *   Integrate visual demographics map using Leaflet/Mapbox in `frontend/campaign-web`.
    *   Implement Quick Count / Real Count system with C1 Plano OCR validation.
*   **Month 6**: n8n Workflow Automation, Security Audits, & Production Deployment
    *   Set up global automation flows in `services/workflow-service` using self-hosted n8n.
    *   Conduct end-to-end data encryption verification and stress-testing of WebSocket tickers.
    *   Configure Nginx reverse proxies, SSL certificates, and deploy multi-container ecosystem.

---

*For monetization strategies, please see [Monetization Plan](file:///home/syahril/Desktop/dev/core_project/docs/monetization.md).*

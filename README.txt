# WCH Platform — Multi Product Architecture

platform/
├── apps/
│   ├── crypto/
│   │   ├── api/
│   │   ├── domain/
│   │   ├── worker/
│   │   └── configs/
│   ├── umkm/
│   │   ├── api/
│   │   ├── chatbot/
│   │   ├── accounting/
│   │   └── automation/
│   └── campaign/
│       ├── api/
│       ├── volunteer/
│       ├── analytics/
│       └── premium/
│
├── services/
│   ├── auth-service/
│   ├── billing-service/
│   ├── ai-gateway/
│   ├── notification-service/
│   ├── workflow-service/
│   ├── file-service/
│   └── tenant-service/
│
├── frontend/
│   ├── crypto-web/
│   ├── umkm-web/
│   ├── campaign-web/
│   └── admin-web/
│
├── infra/
│   ├── docker/
│   ├── nginx/
│   ├── postgres/
│   ├── redis/
│   ├── n8n/
│   └── deploy/
│
├── shared/
│   ├── sdk/
│   ├── ui/
│   ├── config/
│   └── types/
│
└── docs/
    ├── roadmap.md
    ├── monetization.md
    └── deployment.md

Roadmap:
Phase 1 → Shared Platform + UMKM MVP
Phase 2 → Crypto + WCH
Phase 3 → Campaign

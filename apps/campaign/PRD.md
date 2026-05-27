# Product Requirements Document (PRD)

## 1. Meta Information
- **Project Name:** Aplikasi Manajemen Pemenangan Pemilu (`apps/campaign`)
- **Target Audience:** Calon legislatif (Caleg), kepala daerah (Cakada), partai politik, dan tim sukses (Timses).
- **Status:** Draft
- **Last Updated:** 2026-05-25

## 2. Product Overview
Platform taktis dan komprehensif untuk mengorganisir dan memenangkan kampanye politik. Menyediakan alat untuk manajemen relawan, pemetaan demografis konstituen, sentimen analitik, serta sistem rekapitulasi suara (Real Count) berbasis bukti yang andal untuk mencegah kecurangan.

## 3. Goals & Objectives
- **Business Goals:** Menjadi platform premium utama bagi politisi dalam musim pemilu (SaaS B2B/B2G).
- **User Goals:** Memiliki "War Room" digital yang akurat untuk melacak relawan, survei, logistik, dan memastikan suara aman di TPS.
- **Non-Goals:** Bukan platform media sosial, melainkan alat manajemen internal tim kampanye.

## 4. Key Features & Requirements

### Feature 1: Volunteer & Timses Management
- **Description:** Pendaftaran dan pelacakan kinerja relawan lapangan.
- **Acceptance Criteria:** Relawan dapat check-in berbasis GPS, mengisi survei door-to-door, dan terdapat sistem leaderboard/poin.
- **Priority:** P1 (High)

### Feature 2: Voter Database & Demographic Mapping
- **Description:** CRM khusus untuk pemilih berdasarkan geografi.
- **Acceptance Criteria:** Visualisasi peta dukungan (GIS) menggunakan Leaflet/Mapbox per Dapil/Kecamatan/Kelurahan/TPS.
- **Priority:** P1 (High)

### Feature 3: Quick Count & Real Count System (AI OCR)
- **Description:** Input perolehan suara per TPS beserta bukti foto C1 Plano.
- **Acceptance Criteria:** Relawan menginput angka suara TPS, dan sistem AI (via ai-gateway) memvalidasi kesesuaian input dengan foto formulir C1 untuk mitigasi error/kecurangan.
- **Priority:** P1 (High)

### Feature 4: Budget & Logistics Management
- **Description:** Pelacakan pengeluaran kampanye (kaos, sembako, operasional).
- **Acceptance Criteria:** Timses dapat mengetahui ROI dukungan pemilih vs biaya logistik per area.
- **Priority:** P2 (Medium)

### Feature 5: Real-Time Sentiment Analysis
- **Description:** Memantau tren kandidat di berita atau media sosial.
- **Acceptance Criteria:** Dasbor sentimen publik (Positif/Netral/Negatif) untuk penyesuaian strategi kampanye.
- **Priority:** P3 (Low)

## 5. Technical Considerations (Architecture & Integrations)
- **Dependencies:** Leaflet/Mapbox untuk GIS, `ai-gateway` untuk validasi OCR C1 dan Sentiment Analysis.
- **Database:** PostgreSQL (skema spesifik untuk relasi demografis/hierarki wilayah Indonesia).
- **Security & Performance:** Kebutuhan sistem *high-availability* dan sanggup menahan beban *traffic spike* ekstrem pada hari-H pemilihan (Real count input dari ribuan TPS secara bersamaan).

## 6. User Experience & Design
- **Platform:** Web Admin (War Room) & Mobile-friendly Web (untuk relawan di lapangan).
- **Key Flows:** Setup Wilayah -> Daftarkan Relawan -> Relawan Input Data Pemilih -> Admin Pantau Peta Dukungan -> Hari H (Input C1 Real Count).

## 7. Metrics & Analytics
- **Success Metrics:** Jumlah relawan aktif, Jumlah pemilih terpetakan, Uptime sistem pada hari pemilu.

## 8. Release Phases
- **Phase 1 (MVP):** Database Pemilih, Manajemen Relawan Dasar, Form Survei, Input Real Count tanpa OCR.
- **Phase 2:** Validasi OCR C1 Plano, Peta GIS Interaktif, Sentiment Analysis, Modul logistik premium.

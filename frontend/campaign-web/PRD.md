# Product Requirements Document (PRD)

## 1. Meta Information
- **Project Name:** Campaign Web App (`frontend/campaign-web`)
- **Target Audience:** Admin Timses Kampanye, Relawan Lapangan.
- **Status:** Draft
- **Last Updated:** 2026-05-25

## 2. Product Overview
Antarmuka pengguna (Frontend) untuk platform Manajemen Kampanye & Pemenangan Pemilu. Didesain untuk menampilkan Peta Dukungan demografis raksasa bagi pimpinan kampanye (War Room) dan juga sebagai form survei cepat bagi relawan di lapangan.

## 3. Goals & Objectives
- **Business Goals:** Menyediakan visualisasi data spasial (peta) yang menakjubkan bagi calon legislatif/kepala daerah untuk membantu keputusan alokasi dana kampanye.
- **User Goals:** (Bagi Admin) Melihat peta kantong suara. (Bagi Relawan) Mengisi form C1 atau form relawan dengan cepat di HP.
- **Non-Goals:** Bukan aplikasi media sosial kampanye untuk dibagikan ke masyarakat umum.

## 4. Key Features & Requirements

### Feature 1: War Room Map Dashboard (GIS)
- **Description:** Peta interaktif demografi dukungan.
- **Acceptance Criteria:** Terintegrasi dengan Leaflet/Mapbox, menampilkan pin/heat-map wilayah kecamatan mana yang banyak pemilih potensialnya berdasarkan hasil survei.
- **Priority:** P1 (High)

### Feature 2: Real Count C1 Form & OCR Upload
- **Description:** Antarmuka khusus saksi di hari pemilu.
- **Acceptance Criteria:** Tombol "Input Hasil TPS". Mengandung upload kamera untuk foto C1 Plano dan form isian angka.
- **Priority:** P1 (High)

### Feature 3: Volunteer Leaderboard
- **Description:** Gamifikasi untuk kinerja relawan lapangan.
- **Acceptance Criteria:** Menampilkan top 10 relawan yang paling banyak mendaftarkan konstituen/pemilih baru.
- **Priority:** P2 (Medium)

## 5. Technical Considerations (Architecture & Integrations)
- **Dependencies:** React / Next.js, Leaflet.js / Mapbox-GL, Chart.js.
- **Integrations:** `apps/campaign` backend API.

## 6. User Experience & Design
- **Platform:** 
  - Desktop Web (Untuk Admin War Room).
  - Mobile Web App (Untuk Relawan Lapangan).
- **Vibe:** "Profesional, Taktis, Data-Driven".

## 7. Metrics & Analytics
- **Success Metrics:** Kecepatan rendering Peta dengan banyak pin, Load time pada koneksi buruk (saat hari H di TPS desa terpencil).

## 8. Release Phases
- **Phase 1 (MVP):** Form Relawan Dasar, Form C1 Manual (tanpa OCR), Tabel Rekap Suara.
- **Phase 2:** Peta Heatmap GIS interaktif, Dashboard OCR Scanner Terintegrasi.

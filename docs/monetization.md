# WCH Platform — Monetization Strategy & Tiering Model

Dokumen ini merinci strategi monetisasi, model penetapan harga (*pricing model*), dan tiering fitur untuk ketiga produk dalam **WCH Platform**. Penggabungan strategi ini dirancang untuk memaksimalkan Pendapatan Berulang Bulanan (*Monthly Recurring Revenue - MRR*) serta mengoptimalkan biaya infrastruktur (terutama konsumsi API LLM).

---

## 🪙 1. ~~SaaS Crypto Trading Bot (`apps/crypto`)~~ *(ARCHIVED)*
*Produk ini telah di-archive dan tidak lagi dikembangkan.*

---

## 💼 2. AI Agent UMKM & Pembukuan (`apps/umkm`)
Monetisasi menggabungkan model **Freemium SaaS** untuk pembukuan dengan biaya berbasis konsumsi (Pay-as-you-go) untuk AI Agent.

### 📊 Skema Paket Langganan:

| Fitur | **Starter (UMKM Mikro)** | **Grow (UMKM Berkembang)** | **Scale (UMKM Skala Menengah)** |
| :--- | :--- | :--- | :--- |
| **Harga Bulanan** | Rp 0 (Gratis Selamanya) | Rp 99.000 / bln | Rp 299.000 / bln |
| **Pengguna (Staff/Admin)** | 1 Pengguna | Maks. 5 Pengguna | Unlimited |
| **Fitur Pembukuan** | Pencatatan manual dasar & Laba Rugi | Akuntansi Lengkap (Neraca, Arus Kas) | Multi-cabang & Konsolidasi Keuangan |
| **AI OCR Receipt Scanner** | 5 scan / bulan | 50 scan / bulan | Unlimited (Fair Use) |
| **Conversational Chatbot** | Tidak aktif | Aktif (Maks. 250 chat/bln) | Aktif (Maks. 1.500 chat/bln) |
| **WhatsApp/IG Integration** | Tidak tersedia | Tersedia (1 nomor WA) | Tersedia (Multi-kanal) |
| **AI Business Analyst** | Tidak tersedia | Laporan bulanan dasar | Analitis Real-time & Prediksi Stok |

### 💡 Skema Add-on / Pay-as-you-go:
1.  **AI OCR Extra Pack**: Rp 20.000 per 50 scan nota tambahan.
2.  **WhatsApp Chat Credit**: Karena WhatsApp API mengenakan biaya per percakapan (session-based), platform mengenakan margin sekitar 20% di atas harga dasar WhatsApp Business API untuk pesan keluar yang dipicu chatbot.
3.  **Laporan Pajak Tahunan Otomatis**: Rp 150.000 per laporan SPT tahunan yang di-generate langsung oleh sistem akuntansi.

---

## 🗳️ 3. Aplikasi Manajemen Pemenangan Pemilu (`apps/campaign`)
Model monetisasi berbasis **Value-Based Pricing (B2G/B2B Event-driven)** karena kampanye pemilu memiliki batas waktu musiman dan anggaran yang besar.

### 📊 Paket Lisensi Campaign (Satu Kali Bayar / Kontrak Musiman):

| Fitur | **Paket DPRD (Kab/Kota)** | **Paket DPRD Provinsi** | **Paket DPR RI / Pilkada** |
| :--- | :--- | :--- | :--- |
| **Estimasi Harga Lisensi** | Rp 15.000.000 / kampanye | Rp 45.000.000 / kampanye | Rp 120.000.000+ / kampanye |
| **Masa Aktif Sistem** | 6 Bulan (s/d Pemilu selesai) | 9 Bulan (s/d Pemilu selesai) | 12 Bulan (s/d Pemilu selesai) |
| **Kuota Relawan Terdaftar** | Maks. 250 Relawan | Maks. 1.000 Relawan | Unlimited |
| **Pemetaan DPT Pemilih** | 1 Dapil Tingkat II | 1 Dapil Tingkat I | 1 Dapil Nasional / Wilayah Pilkada |
| **Real Count TPS (Saksi)** | Maks. 500 TPS | Maks. 2.000 TPS | Unlimited TPS |
| **WhatsApp Blast Relawan** | 5.000 pesan / bulan | 25.000 pesan / bulan | 100.000 pesan / bulan |
| **AI C1 Plano OCR Scan** | Tersedia | Tersedia | Tersedia + Validasi Ganda |
| **Dashboard Pemenangan** | Standard Web App | Advanced Dashboard + Map | Executive Command Center App |

### 💡 Modul Premium Tambahan (Upselling):
1.  **Sentiment Monitoring & Social Analytics**: Tambahan Rp 5.000.000/bulan untuk memantau penyebaran isu negatif/positif di media sosial secara real-time.
2.  **Ajudan AI Politik (AI Political Advisor)**: Chatbot khusus berpengetahuan undang-undang pemilu dan isu daerah pemilihan setempat untuk membantu Caleg menyusun draf pidato politik yang relevan dengan kebutuhan dapil.
3.  **Audit Forensik C1**: Layanan verifikasi manual oleh tim analis jika terdapat selisih mencolok antara Real Count sistem dengan data KPU.

---

## 🛡️ 4. Pengendalian Biaya Operasional AI (AI Cost Optimization)
Menggunakan LLM untuk jutaan pesan chatbot dan ratusan ribu scan OCR berpotensi memakan biaya besar. Kita menerapkan strategi penekanan biaya melalui **AI Gateway**:

1.  **Semantic Caching (Redis)**: Untuk pertanyaan chatbot yang sering diulang pelanggan UMKM (misalnya: *"Alamat toko di mana?"* atau *"Apakah barang ini ready?"*), respons disimpan dalam Redis Cache. Prompt tidak dikirim ulang ke API OpenAI/Gemini, menekan biaya token hingga **40% - 60%**.
2.  **Model Tiering**:
    *   *Task Sederhana (OCR & Klasifikasi)*: Menggunakan model murah dan cepat seperti **Gemini 1.5 Flash** atau **GPT-4o mini**.
    *   *Task Kompleks (Analisis Keuangan & Strategi Kampanye)*: Menggunakan **Gemini 1.5 Pro** atau **GPT-4o**.
3.  **Edge Compute**: Penerapan OCR berbasis pustaka klien (Tesseract.js) untuk pra-proses gambar sebelum diunggah, memotong resolusi gambar yang terlalu besar agar menghemat token input gambar.

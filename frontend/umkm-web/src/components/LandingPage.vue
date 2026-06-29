<template>
  <div class="landing" :class="themeClass" lang="id">
    <!-- Navbar -->
    <nav class="navbar">
      <div class="nav-inner">
        <a href="/" class="nav-logo">
          <svg width="28" height="28" viewBox="0 0 28 28" fill="none">
            <rect width="28" height="28" rx="8" fill="#f59e0b"/>
            <path d="M7 9h14M7 14h9M7 19h12" stroke="#1a1a2e" stroke-width="2.2" stroke-linecap="round"/>
          </svg>
          <span>WCH Platform</span>
        </a>
        <div class="nav-links">
          <a href="#fitur">Fitur</a>
          <a href="#harga">Harga</a>
          <a href="#testimoni">Testimoni</a>
        </div>
        <div class="nav-actions">
          <a href="/login" class="btn-ghost">Masuk</a>
          <a href="/register" class="btn-primary">Daftar Gratis</a>
        </div>
        <!-- Mobile toggle -->
        <button class="nav-mobile-toggle" @click="mobileOpen = !mobileOpen" aria-label="Menu">
          <span></span><span></span><span></span>
        </button>
      </div>
      <!-- Mobile menu -->
      <div v-if="mobileOpen" class="nav-mobile-menu">
        <a href="#fitur" @click="mobileOpen = false">Fitur</a>
        <a href="#harga" @click="mobileOpen = false">Harga</a>
        <a href="#testimoni" @click="mobileOpen = false">Testimoni</a>
        <hr>
        <a href="/login" @click="mobileOpen = false">Masuk</a>
        <a href="/register" class="cta-mobile" @click="mobileOpen = false">Daftar Gratis</a>
      </div>
    </nav>

    <!-- Hero -->
    <section class="hero">
      <div class="hero-bg">
        <div class="hero-mesh"></div>
        <div class="hero-glow"></div>
      </div>
      <div class="hero-inner reveal">
        <div class="hero-badge">
          <span class="badge-dot"></span>
          Kasir · Pembukuan · AI Chatbot — dalam satu platform
        </div>
        <h1 class="hero-title">
          Kelola Usaha<br>
          <span class="title-accent">Tanpa Ribet,</span><br>
          Tanpa Accountant
        </h1>
        <p class="hero-sub">
          WCH Platform adalah all-in-one aplikasi kasir, pembukuan double-entry,
          dan AI Customer Service untuk UMKM Indonesia. Mulai gratis, upgrade kapan saja.
        </p>
        <div class="hero-cta">
          <a href="/register" class="cta-main">
            <span>Mulai Gratis</span>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
          </a>
          <a href="#fitur" class="cta-secondary">Lihat Fitur</a>
        </div>
        <div class="hero-stats">
          <div class="stat">
            <strong>100+</strong>
            <span>UMKM Aktif</span>
          </div>
          <div class="stat-divider"></div>
          <div class="stat">
            <strong>50K+</strong>
            <span>Transaksi/Bulan</span>
          </div>
          <div class="stat-divider"></div>
          <div class="stat">
            <strong>24/7</strong>
            <span>AI Chatbot</span>
          </div>
        </div>
      </div>
    </section>

    <!-- Logos / Trust bar -->
    <div class="trust-bar reveal">
      <p class="trust-label">Telah dipercaya oleh pelaku UMKM di</p>
      <div class="trust-badges">
        <span>Jakarta</span>
        <span>Surabaya</span>
        <span>Bandung</span>
        <span>Medan</span>
        <span>Makassar</span>
        <span>Semarang</span>
      </div>
    </div>

    <!-- Features -->
    <section id="fitur" class="section-features reveal">
      <div class="section-inner">
        <div class="section-label">Fitur Unggulan</div>
        <h2 class="section-title">Semua yang Anda butuhkan,<br>dalam satu aplikasi</h2>
        <div class="features-grid">
          <div v-for="(f, i) in features" :key="i" class="feature-card" :style="{ '--delay': `${i * 80}ms` }">
            <div class="feature-icon">{{ f.icon }}</div>
            <h3>{{ f.title }}</h3>
            <p>{{ f.desc }}</p>
          </div>
        </div>
      </div>
    </section>

    <!-- How it works -->
    <section class="section-how reveal">
      <div class="section-inner">
        <div class="section-label">Cara Kerja</div>
        <h2 class="section-title">Dari nol ke operasional<br>dalam 3 menit</h2>
        <div class="steps">
          <div v-for="(step, i) in steps" :key="i" class="step" :style="{ '--delay': `${i * 120}ms` }">
            <div class="step-num">{{ String(i + 1).padStart(2, '0') }}</div>
            <div class="step-body">
              <h3>{{ step.title }}</h3>
              <p>{{ step.desc }}</p>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Pricing -->
    <section id="harga" class="section-pricing reveal">
      <div class="section-inner">
        <div class="section-label">Harga Transparan</div>
        <h2 class="section-title">Paket yang cocok untuk<br>semua ukuran usaha</h2>
        <p class="section-sub">Tidak ada biaya tersembunyi. Mulai gratis, upgrade sesuai kebutuhan.</p>

        <!-- Billing cycle toggle -->
        <div class="billing-toggle">
          <button :class="['toggle-btn', billingCycle === 'monthly' ? 'active' : '']"
            @click="billingCycle = 'monthly'">Bulanan</button>
          <button :class="['toggle-btn', billingCycle === 'yearly' ? 'active' : '']"
            @click="billingCycle = 'yearly'">Tahunan <span class="save-badge">Hemat</span></button>
        </div>

        <div class="pricing-grid">
          <div
            v-for="plan in plans"
            :key="plan.id"
            class="pricing-card"
            :class="{ 'pricing-featured': plan.sort_order === 2 }"
          >
            <div v-if="plan.sort_order === 2" class="pricing-badge">Paling Populer</div>
            <div class="plan-name">{{ plan.name }}</div>
            <div class="plan-price">
              <span v-if="billingCycle === 'monthly'">
                <span v-if="plan.price_monthly === 0" class="price-amount">Gratis</span>
                <span v-else>
                  <span class="price-currency">Rp</span>
                  <span class="price-amount">{{ formatPrice(plan.price_monthly) }}</span>
                  <span class="price-period">/bulan</span>
                </span>
              </span>
              <span v-else>
                <span v-if="plan.price_yearly === 0" class="price-amount">Gratis</span>
                <span v-else>
                  <span class="price-currency">Rp</span>
                  <span class="price-amount">{{ formatPrice(plan.price_yearly) }}</span>
                  <span class="price-period">/tahun</span>
                </span>
              </span>
            </div>
            <p class="plan-desc">{{ plan.description }}</p>
            <ul class="plan-features">
              <li v-for="f in plan.features" :key="f.feature_key">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
                {{ f.feature_name }}
              </li>
            </ul>
            <a :href="'/register?plan=' + plan.id + '&cycle=' + billingCycle" class="plan-cta" :class="{ 'cta-active': plan.sort_order === 2 }">
              {{ plan.price_monthly === 0 ? 'Mulai Gratis' : 'Mulai ' + plan.name }}
            </a>
          </div>
        </div>
      </div>
    </section>

    <!-- Testimonials -->
    <section id="testimoni" class="section-testimoni reveal">
      <div class="section-inner">
        <div class="section-label">Cerita User</div>
        <h2 class="section-title">Dipercaya oleh pelaku usaha<br>seperti Anda</h2>
        <div class="testimoni-grid">
          <div v-for="(t, i) in testimonials" :key="i" class="testimoni-card" :style="{ '--delay': `${i * 100}ms` }">
            <div class="testimoni-stars">
              <span v-for="n in 5" :key="n">★</span>
            </div>
            <blockquote>{{ t.quote }}</blockquote>
            <div class="testimoni-author">
              <div class="author-avatar">{{ t.name.charAt(0) }}</div>
              <div>
                <strong>{{ t.name }}</strong>
                <span>{{ t.role }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- CTA Banner -->
    <section class="section-cta reveal">
      <div class="cta-inner">
        <div class="cta-glow"></div>
        <h2>Siap mengelola usaha dengan lebih cerdas?</h2>
        <p>Bergabung dengan 100+ UMKM Indonesia hari ini. Gratis selamanya.</p>
        <a href="/register" class="cta-main cta-large">
          <span>Daftar Sekarang — Gratis</span>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
        </a>
      </div>
    </section>

    <!-- Footer -->
    <footer class="footer">
      <div class="footer-inner">
        <div class="footer-brand">
          <div class="footer-logo">
            <svg width="24" height="24" viewBox="0 0 28 28" fill="none">
              <rect width="28" height="28" rx="8" fill="#f59e0b"/>
              <path d="M7 9h14M7 14h9M7 19h12" stroke="#1a1a2e" stroke-width="2.2" stroke-linecap="round"/>
            </svg>
            <span>WCH Platform</span>
          </div>
          <p>All-in-one aplikasi kasir, pembukuan, dan AI chatbot untuk UMKM Indonesia.</p>
        </div>
        <div class="footer-links">
          <div class="footer-col">
            <h4>Produk</h4>
            <a href="#fitur">Fitur</a>
            <a href="#harga">Harga</a>
            <a href="/register">Daftar</a>
          </div>
          <div class="footer-col">
            <h4>Perusahaan</h4>
            <a href="#">Tentang Kami</a>
            <a href="#">Blog</a>
            <a href="#">Karir</a>
          </div>
          <div class="footer-col">
            <h4>Bantuan</h4>
            <a href="#">Pusat Bantuan</a>
            <a href="#">Syarat & Ketentuan</a>
            <a href="#">Kebijakan Privasi</a>
          </div>
        </div>
      </div>
      <div class="footer-bottom">
        <p>© 2026 WCH Platform. Hak cipta dilindungi.</p>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useHead } from '@unhead/vue'
import { api } from '../api'

useHead({
  title: 'WCH Platform — Kasir, Pembukuan & AI Chatbot untuk UMKM Indonesia',
  meta: [
    { name: 'description', content: 'WCH Platform adalah all-in-one aplikasi kasir, pembukuan double-entry, dan AI Customer Service untuk UMKM Indonesia. Mulai gratis, upgrade kapan saja.' },
    { property: 'og:title', content: 'WCH Platform — Kasir, Pembukuan & AI Chatbot' },
    { property: 'og:description', content: 'All-in-one aplikasi kasir, pembukuan double-entry, dan AI chatbot untuk UMKM Indonesia. Gratis selamanya.' },
    { property: 'og:type', content: 'website' },
    { name: 'twitter:card', content: 'summary_large_image' },
    { name: 'twitter:title', content: 'WCH Platform — Kasir, Pembukuan & AI Chatbot' },
    { name: 'twitter:description', content: 'All-in-one aplikasi kasir, pembukuan, dan AI chatbot untuk UMKM Indonesia.' },
    { name: 'robots', content: 'index, follow' },
  ],
  link: [
    { rel: 'canonical', href: 'https://wch.id' },
  ],
})

const mobileOpen = ref(false)

// F059: Dynamic plans from backend (public endpoint, no auth)
const plans = ref<any[]>([])
const plansLoading = ref(false)
const billingCycle = ref<'monthly' | 'yearly'>('monthly')

const fetchPlans = async () => {
  plansLoading.value = true
  try {
    const res = await api.getPublicPlans()
    if (res?.success && Array.isArray(res.data)) {
      plans.value = res.data
    }
  } catch (e) {
    console.error('Failed to fetch plans', e)
  } finally {
    plansLoading.value = false
  }
}

// Dark landing always — matches existing WCH dark theme
const themeClass = computed(() => 'theme-dark')

const formatPrice = (sen: number) => {
  return (sen / 1000).toLocaleString('id-ID') + 'K'
}

const features = [
  {
    icon: '💰',
    title: 'Kasir POS',
    desc: 'Catat transaksi jual-beli dengan cepat. Dukung multi-pembayaran: Tunai, QRIS, Transfer, E-Wallet.'
  },
  {
    icon: '📒',
    title: 'Pembukuan Otomatis',
    desc: 'Double-entry accounting. Setiap transaksi POS langsung tercatat di jurnal — tidak perlu accountant.'
  },
  {
    icon: '🤖',
    title: 'AI Customer Service',
    desc: 'Bot WhatsApp otomatis jawab pertanyaan pelanggan 24/7. Bisa di-training dengan FAQ toko Anda.'
  },
  {
    icon: '📊',
    title: 'Laporan Keuangan',
    desc: 'Laba rugi, arus kas, neraca — siap pakai untuk pajak dan pengajuan kredit bank.'
  },
  {
    icon: '🏪',
    title: 'Multi-Toko',
    desc: 'Kelola beberapa cabang dalam satu akun. Cocok untuk franchise dan warung kopi.',
  },
  {
    icon: '🔗',
    title: 'Integrasi Marketplace',
    desc: 'Hubungkan dengan GoFood, Grab, Shopee. Stok tersinkron otomatis di satu dashboard.'
  }
]

const steps = [
  {
    title: 'Daftar dalam 30 detik',
    desc: 'Masukkan nomor WhatsApp, terima OTP, langsung masuk. Tanpa verifikasi email.'
  },
  {
    title: 'Pilih paket atau mulai gratis',
    desc: 'Lite gratis selamanya. Atau pilih Pro/Ultimate untuk fitur AI dan multi-user.'
  },
  {
    title: 'Mulai berjualan',
    desc: 'Kasir langsung bisa dipakai. Pembukuan berjalan otomatis di belakang layar.'
  }
]

const testimonials = [
  {
    quote: 'Dulu pembukuan pakai Excel, sekarang semua otomatis. Owner warung kopi saya bisa fokus layani pelanggan.',
    name: 'Budi Santoso',
    role: 'Pemilik Warung Kopi, Bandung'
  },
  {
    quote: 'AI chatbot-nya jawab pertanyaan pelanggan di malam hari. Servis tetap prima meski saya lagi tutup.',
    name: 'Siti Rahayu',
    role: 'Pemilik Toko Fashion, Surabaya'
  },
  {
    quote: 'Laporan keuangan langsung jadi, tinggal export ke PDF untuk pengajuan KUR ke bank.',
    name: 'Ahmad Hidayat',
    role: 'Pemilik Klinik Pratama, Medan'
  }
]

// Scroll reveal via IntersectionObserver
onMounted(() => {
  fetchPlans()
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          entry.target.classList.add('revealed')
          observer.unobserve(entry.target)
        }
      })
    },
    { threshold: 0.12 }
  )
  document.querySelectorAll('.reveal').forEach((el) => observer.observe(el))
})
</script>

<style scoped>
/* ── Theme: always dark, reuses WCH CSS vars ── */
.landing {
  background: #0d0d14;
  color: #e2e8f0;
  min-height: 100vh;
  overflow-x: hidden;
}

/* ── Navbar ── */
.navbar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  background: rgba(13, 13, 20, 0.85);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.nav-inner {
  max-width: 1120px;
  margin: 0 auto;
  padding: 0 1.5rem;
  height: 64px;
  display: flex;
  align-items: center;
  gap: 2rem;
}

.nav-logo {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  text-decoration: none;
  font-size: 1.1rem;
  font-weight: 700;
  color: #f1f5f9;
  letter-spacing: -0.02em;
  flex-shrink: 0;
}

.nav-links {
  display: flex;
  gap: 1.75rem;
  margin-left: auto;
}

.nav-links a {
  text-decoration: none;
  color: #94a3b8;
  font-size: 0.9rem;
  font-weight: 500;
  transition: color 0.2s;
}

.nav-links a:hover { color: #e2e8f0; }

.nav-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.btn-ghost {
  text-decoration: none;
  color: #94a3b8;
  font-size: 0.9rem;
  font-weight: 500;
  padding: 0.4rem 1rem;
  border-radius: 8px;
  transition: color 0.2s, background 0.2s;
}
.btn-ghost:hover { color: #e2e8f0; background: rgba(255,255,255,0.05); }

.btn-primary {
  text-decoration: none;
  background: #f59e0b;
  color: #1a1a2e;
  font-size: 0.9rem;
  font-weight: 700;
  padding: 0.45rem 1.1rem;
  border-radius: 8px;
  transition: background 0.2s, transform 0.15s;
}
.btn-primary:hover { background: #fbbf24; transform: translateY(-1px); }

.nav-mobile-toggle {
  display: none;
  flex-direction: column;
  gap: 5px;
  background: none;
  border: none;
  cursor: pointer;
  padding: 4px;
  margin-left: auto;
}
.nav-mobile-toggle span {
  display: block;
  width: 22px;
  height: 2px;
  background: #e2e8f0;
  border-radius: 2px;
  transition: 0.2s;
}

.nav-mobile-menu {
  display: flex;
  flex-direction: column;
  padding: 1rem 1.5rem 1.5rem;
  gap: 0.5rem;
  background: rgba(13, 13, 20, 0.97);
  border-top: 1px solid rgba(255,255,255,0.06);
}
.nav-mobile-menu a {
  text-decoration: none;
  color: #94a3b8;
  font-size: 1rem;
  padding: 0.5rem 0;
  border-bottom: 1px solid rgba(255,255,255,0.04);
}
.cta-mobile {
  color: #f59e0b !important;
  font-weight: 700;
  border-bottom: none !important;
  margin-top: 0.5rem;
}

/* ── Hero ── */
.hero {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  padding: 8rem 1.5rem 6rem;
}

.hero-bg {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
}

.hero-mesh {
  position: absolute;
  inset: 0;
  background-image:
    radial-gradient(ellipse 80% 60% at 50% 0%, rgba(245, 158, 11, 0.12) 0%, transparent 70%),
    radial-gradient(ellipse 40% 40% at 80% 60%, rgba(20, 184, 166, 0.07) 0%, transparent 60%);
}

.hero-glow {
  position: absolute;
  top: 20%;
  left: 50%;
  transform: translateX(-50%);
  width: 600px;
  height: 400px;
  background: radial-gradient(ellipse, rgba(245, 158, 11, 0.08) 0%, transparent 70%);
  filter: blur(40px);
}

.hero-inner {
  position: relative;
  max-width: 720px;
  margin: 0 auto;
  text-align: center;
}

.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  background: rgba(245, 158, 11, 0.1);
  border: 1px solid rgba(245, 158, 11, 0.25);
  color: #fbbf24;
  font-size: 0.82rem;
  font-weight: 600;
  padding: 0.35rem 0.9rem;
  border-radius: 100px;
  margin-bottom: 1.75rem;
  letter-spacing: 0.01em;
}

.badge-dot {
  width: 7px;
  height: 7px;
  background: #22c55e;
  border-radius: 50%;
  animation: pulse-dot 2s infinite;
}

@keyframes pulse-dot {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.6; transform: scale(0.8); }
}

.hero-title {
  font-size: clamp(2.4rem, 6vw, 4rem);
  font-weight: 800;
  line-height: 1.1;
  letter-spacing: -0.03em;
  color: #f1f5f9;
  margin-bottom: 1.5rem;
}

.title-accent {
  background: linear-gradient(135deg, #f59e0b, #fbbf24);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.hero-sub {
  font-size: 1.1rem;
  line-height: 1.7;
  color: #94a3b8;
  max-width: 560px;
  margin: 0 auto 2.5rem;
}

.hero-cta {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  margin-bottom: 3.5rem;
  flex-wrap: wrap;
}

.cta-main {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  background: #f59e0b;
  color: #1a1a2e;
  font-weight: 700;
  font-size: 1rem;
  padding: 0.8rem 1.75rem;
  border-radius: 12px;
  text-decoration: none;
  transition: background 0.2s, transform 0.15s, box-shadow 0.2s;
  box-shadow: 0 0 0 0 rgba(245, 158, 11, 0);
}
.cta-main:hover {
  background: #fbbf24;
  transform: translateY(-2px);
  box-shadow: 0 8px 30px rgba(245, 158, 11, 0.3);
}
.cta-main svg { transition: transform 0.2s; }
.cta-main:hover svg { transform: translateX(3px); }

.cta-secondary {
  display: inline-flex;
  align-items: center;
  color: #94a3b8;
  font-weight: 600;
  font-size: 0.95rem;
  text-decoration: none;
  padding: 0.8rem 1.5rem;
  border-radius: 12px;
  border: 1px solid rgba(255,255,255,0.1);
  transition: color 0.2s, border-color 0.2s, background 0.2s;
}
.cta-secondary:hover {
  color: #e2e8f0;
  border-color: rgba(255,255,255,0.2);
  background: rgba(255,255,255,0.04);
}

.hero-stats {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 2rem;
  flex-wrap: wrap;
}

.stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.2rem;
}

.stat strong {
  font-size: 1.6rem;
  font-weight: 800;
  color: #f59e0b;
  letter-spacing: -0.03em;
}

.stat span {
  font-size: 0.78rem;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.stat-divider {
  width: 1px;
  height: 36px;
  background: rgba(255,255,255,0.08);
}

/* ── Trust bar ── */
.trust-bar {
  padding: 2rem 1.5rem;
  border-top: 1px solid rgba(255,255,255,0.05);
  border-bottom: 1px solid rgba(255,255,255,0.05);
  text-align: center;
}

.trust-label {
  font-size: 0.78rem;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: #475569;
  margin-bottom: 1rem;
}

.trust-badges {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.trust-badges span {
  background: rgba(255,255,255,0.04);
  border: 1px solid rgba(255,255,255,0.08);
  color: #64748b;
  font-size: 0.82rem;
  font-weight: 600;
  padding: 0.3rem 0.8rem;
  border-radius: 100px;
  letter-spacing: 0.03em;
}

/* ── Sections shared ── */
.section-inner {
  max-width: 1100px;
  margin: 0 auto;
  padding: 0 1.5rem;
}

.section-label {
  display: inline-block;
  font-size: 0.78rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: #f59e0b;
  margin-bottom: 0.75rem;
}

.section-title {
  font-size: clamp(1.8rem, 4vw, 2.6rem);
  font-weight: 800;
  line-height: 1.15;
  letter-spacing: -0.03em;
  color: #f1f5f9;
  margin-bottom: 1rem;
}

.section-sub {
  font-size: 1rem;
  color: #64748b;
  max-width: 500px;
  line-height: 1.6;
  margin-bottom: 3rem;
}

/* ── Scroll reveal ── */
.reveal {
  opacity: 0;
  transform: translateY(24px);
  transition: opacity 0.6s ease, transform 0.6s ease;
}
.reveal.revealed {
  opacity: 1;
  transform: translateY(0);
}

/* ── Features ── */
.section-features {
  padding: 6rem 0;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1.25rem;
}

.feature-card {
  background: rgba(255,255,255,0.03);
  border: 1px solid rgba(255,255,255,0.07);
  border-radius: 16px;
  padding: 1.75rem;
  transition: border-color 0.25s, transform 0.25s, background 0.25s;
  transition-delay: var(--delay, 0ms);
}
.feature-card:hover {
  border-color: rgba(245, 158, 11, 0.3);
  background: rgba(245, 158, 11, 0.04);
  transform: translateY(-3px);
}

.feature-icon {
  font-size: 2rem;
  margin-bottom: 1rem;
}

.feature-card h3 {
  font-size: 1rem;
  font-weight: 700;
  color: #f1f5f9;
  margin-bottom: 0.5rem;
}

.feature-card p {
  font-size: 0.88rem;
  color: #64748b;
  line-height: 1.6;
}

/* ── How it works ── */
.section-how {
  padding: 6rem 0;
  background: rgba(255,255,255,0.015);
}

.steps {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  margin-top: 3rem;
  position: relative;
}

.steps::before {
  content: '';
  position: absolute;
  left: 1.4rem;
  top: 2.5rem;
  bottom: 2.5rem;
  width: 2px;
  background: linear-gradient(to bottom, rgba(245,158,11,0.4), rgba(245,158,11,0.05));
  border-radius: 2px;
}

.step {
  display: flex;
  gap: 1.5rem;
  align-items: flex-start;
  padding-left: 0.5rem;
}

.step-num {
  flex-shrink: 0;
  width: 3rem;
  height: 3rem;
  background: linear-gradient(135deg, #f59e0b, #d97706);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
  font-weight: 800;
  color: #1a1a2e;
  letter-spacing: 0.05em;
  position: relative;
  z-index: 1;
}

.step-body h3 {
  font-size: 1.05rem;
  font-weight: 700;
  color: #f1f5f9;
  margin-bottom: 0.3rem;
  padding-top: 0.5rem;
}

.step-body p {
  font-size: 0.88rem;
  color: #64748b;
  line-height: 1.6;
}

/* ── Pricing ── */
.section-pricing {
  padding: 6rem 0;
}

.pricing-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1.25rem;
  align-items: start;
}

/* Billing cycle toggle */
.billing-toggle {
  display: flex;
  justify-content: center;
  gap: 0.5rem;
  margin-bottom: 2.5rem;
}

.billing-toggle .toggle-btn {
  background: rgba(255,255,255,0.05);
  border: 1px solid rgba(255,255,255,0.12);
  color: #94a3b8;
  font-size: 0.9rem;
  font-weight: 600;
  padding: 0.5rem 1.5rem;
  border-radius: 100px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.billing-toggle .toggle-btn.active {
  background: #f59e0b;
  border-color: #f59e0b;
  color: #1a1a2e;
}

.billing-toggle .toggle-btn .save-badge {
  background: rgba(34, 197, 94, 0.2);
  color: #22c55e;
  font-size: 0.7rem;
  font-weight: 700;
  padding: 0.1rem 0.4rem;
  border-radius: 100px;
}

.billing-toggle .toggle-btn.active .save-badge {
  background: rgba(0,0,0,0.2);
  color: #1a1a2e;
}

.pricing-card {
  background: rgba(255,255,255,0.03);
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 20px;
  padding: 2rem;
  position: relative;
  transition: border-color 0.25s;
}

.pricing-featured {
  background: rgba(245, 158, 11, 0.06);
  border-color: rgba(245, 158, 11, 0.35);
}

.pricing-badge {
  position: absolute;
  top: -12px;
  left: 50%;
  transform: translateX(-50%);
  background: linear-gradient(135deg, #f59e0b, #d97706);
  color: #1a1a2e;
  font-size: 0.72rem;
  font-weight: 800;
  padding: 0.25rem 0.9rem;
  border-radius: 100px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  white-space: nowrap;
}

.plan-name {
  font-size: 0.9rem;
  font-weight: 700;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: 0.75rem;
}

.plan-price {
  margin-bottom: 0.75rem;
}

.price-currency {
  font-size: 1.1rem;
  font-weight: 700;
  color: #64748b;
  vertical-align: super;
}

.price-amount {
  font-size: 2.4rem;
  font-weight: 800;
  color: #f1f5f9;
  letter-spacing: -0.04em;
}

.price-period {
  font-size: 0.85rem;
  color: #64748b;
  margin-left: 0.2rem;
}

.plan-desc {
  font-size: 0.85rem;
  color: #64748b;
  margin-bottom: 1.5rem;
  line-height: 1.5;
}

.plan-features {
  list-style: none;
  padding: 0;
  margin: 0 0 2rem;
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.plan-features li {
  display: flex;
  align-items: flex-start;
  gap: 0.6rem;
  font-size: 0.88rem;
  color: #cbd5e1;
}

.plan-features li svg {
  color: #22c55e;
  flex-shrink: 0;
  margin-top: 2px;
}

.plan-cta {
  display: block;
  text-align: center;
  text-decoration: none;
  font-weight: 700;
  font-size: 0.92rem;
  padding: 0.75rem;
  border-radius: 10px;
  border: 1px solid rgba(255,255,255,0.12);
  color: #94a3b8;
  transition: all 0.2s;
}

.plan-cta:hover {
  border-color: rgba(245,158,11,0.4);
  color: #f59e0b;
}

.cta-active {
  background: linear-gradient(135deg, #f59e0b, #d97706);
  color: #1a1a2e !important;
  border-color: transparent !important;
}
.cta-active:hover { background: linear-gradient(135deg, #fbbf24, #f59e0b); }

/* ── Testimonials ── */
.section-testimoni {
  padding: 6rem 0;
  background: rgba(255,255,255,0.015);
}

.testimoni-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1.25rem;
  margin-top: 3rem;
}

.testimoni-card {
  background: rgba(255,255,255,0.03);
  border: 1px solid rgba(255,255,255,0.07);
  border-radius: 16px;
  padding: 1.75rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  transition: border-color 0.25s, transform 0.25s;
}
.testimoni-card:hover {
  border-color: rgba(245,158,11,0.2);
  transform: translateY(-2px);
}

.testimoni-stars {
  color: #f59e0b;
  font-size: 0.9rem;
  letter-spacing: 2px;
}

.testimoni-card blockquote {
  font-size: 0.92rem;
  color: #94a3b8;
  line-height: 1.7;
  font-style: italic;
  flex: 1;
}

.testimoni-author {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.author-avatar {
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #f59e0b, #d97706);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.85rem;
  font-weight: 800;
  color: #1a1a2e;
  flex-shrink: 0;
}

.author-name strong {
  display: block;
  font-size: 0.88rem;
  color: #e2e8f0;
}

.author-name span {
  display: block;
  font-size: 0.78rem;
  color: #475569;
}

/* ── CTA Banner ── */
.section-cta {
  padding: 6rem 1.5rem;
}

.cta-inner {
  position: relative;
  max-width: 680px;
  margin: 0 auto;
  text-align: center;
  background: linear-gradient(135deg, rgba(245,158,11,0.08), rgba(217,119,6,0.04));
  border: 1px solid rgba(245,158,11,0.2);
  border-radius: 24px;
  padding: 4rem 2rem;
  overflow: hidden;
}

.cta-glow {
  position: absolute;
  top: -60px;
  left: 50%;
  transform: translateX(-50%);
  width: 400px;
  height: 200px;
  background: radial-gradient(ellipse, rgba(245,158,11,0.15), transparent 70%);
  filter: blur(30px);
  pointer-events: none;
}

.cta-inner h2 {
  font-size: clamp(1.5rem, 3.5vw, 2.2rem);
  font-weight: 800;
  color: #f1f5f9;
  letter-spacing: -0.03em;
  margin-bottom: 0.75rem;
  position: relative;
}

.cta-inner p {
  font-size: 1rem;
  color: #64748b;
  margin-bottom: 2rem;
  position: relative;
}

.cta-large {
  font-size: 1.05rem;
  padding: 0.9rem 2rem;
  position: relative;
}

/* ── Footer ── */
.footer {
  border-top: 1px solid rgba(255,255,255,0.06);
  padding: 4rem 0 0;
}

.footer-inner {
  max-width: 1100px;
  margin: 0 auto;
  padding: 0 1.5rem;
  display: grid;
  grid-template-columns: 1.5fr 2fr;
  gap: 4rem;
}

.footer-logo {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1rem;
  font-weight: 700;
  color: #f1f5f9;
  margin-bottom: 0.75rem;
}

.footer-brand p {
  font-size: 0.85rem;
  color: #475569;
  line-height: 1.6;
  max-width: 280px;
}

.footer-links {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 2rem;
}

.footer-col h4 {
  font-size: 0.78rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: #f59e0b;
  margin-bottom: 1rem;
}

.footer-col a {
  display: block;
  text-decoration: none;
  font-size: 0.88rem;
  color: #475569;
  padding: 0.25rem 0;
  transition: color 0.2s;
}

.footer-col a:hover { color: #94a3b8; }

.footer-bottom {
  border-top: 1px solid rgba(255,255,255,0.04);
  padding: 1.5rem;
  text-align: center;
}

.footer-bottom p {
  font-size: 0.78rem;
  color: #334155;
}

/* ── Responsive ── */
@media (max-width: 900px) {
  .features-grid,
  .pricing-grid,
  .testimoni-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .footer-inner {
    grid-template-columns: 1fr;
    gap: 2rem;
  }
}

@media (max-width: 640px) {
  .nav-links, .nav-actions { display: none; }
  .nav-mobile-toggle { display: flex; }

  .features-grid,
  .pricing-grid,
  .testimoni-grid {
    grid-template-columns: 1fr;
  }

  .hero-stats { gap: 1rem; }
  .stat-divider { display: none; }

  .footer-links {
    grid-template-columns: repeat(2, 1fr);
  }

  .steps::before { display: none; }
  .step { padding-left: 0; }
}

/* ── Staggered reveal for grid items ── */
.features-grid .feature-card,
.testimoni-grid .testimoni-card {
  transition-delay: var(--delay, 0ms);
}
</style>

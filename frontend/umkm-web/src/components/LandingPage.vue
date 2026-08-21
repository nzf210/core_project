<template>
  <div class="landing" :class="themeClass" lang="id">
    <NavBar />
    <HeroSection :hero="hero" />

    <div class="trust-bar reveal">
      <p>{{ trust.label }}</p>
      <div class="trust-badges">
        <span v-for="city in trust.cities" :key="city">{{ city }}</span>
      </div>
    </div>

    <FeaturesSection :features="features" />
    <HowItWorksSection :steps="steps" />
    <PricingSection :plans="plans" :formatRupiah="formatRupiah" />

    <section id="testimoni" class="section-testimoni reveal">
      <div class="section-inner">
        <div class="section-label">Testimoni</div>
        <h2 class="section-title">Dipercaya oleh<br>ratusan pemilik usaha</h2>
        <div class="testimoni-grid">
          <div v-for="(t, i) in testimonials" :key="i" class="testimoni-card">
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

    <section class="section-cta reveal">
      <div class="cta-inner">
        <div class="cta-glow"></div>
        <h2>{{ cta.title }}</h2>
        <p>{{ cta.subtitle }}</p>
        <a href="/register" class="cta-main cta-large">
          <span>{{ cta.button }}</span>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
        </a>
      </div>
    </section>

    <FooterSection />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useHead } from '@unhead/vue'
import { api } from '../api'
import { formatRupiah } from '../composables/useCurrency'
import NavBar from './landing/NavBar.vue'
import HeroSection from './landing/HeroSection.vue'
import FeaturesSection from './landing/FeaturesSection.vue'
import HowItWorksSection from './landing/HowItWorksSection.vue'
import PricingSection from './landing/PricingSection.vue'
import FooterSection from './landing/FooterSection.vue'

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
  link: [{ rel: 'canonical', href: 'https://wch.id' }],
})

const plans = ref<any[]>([])
const landingContent = ref<Record<string, any>>({})

const fallbackLandingContent: Record<string, any> = {
  hero: {
    badge: 'Kasir · Pembukuan · AI Chatbot — dalam satu platform',
    title_line1: 'Kelola Usaha',
    title_line2: 'Tanpa Ribet,',
    title_line3: 'Tanpa Accountant',
    subtitle: 'WCH Platform adalah all-in-one aplikasi kasir, pembukuan double-entry, dan AI Customer Service untuk UMKM Indonesia. Mulai gratis, upgrade kapan saja.',
    cta_primary: 'Mulai Gratis',
    cta_secondary: 'Lihat Fitur',
    stats: [
      { value: '100+', label: 'UMKM Aktif' },
      { value: '50K+', label: 'Transaksi/Bulan' },
      { value: '24/7', label: 'AI Chatbot' },
    ],
  },
  features: [
    { icon: '💰', title: 'Kasir POS', desc: 'Catat transaksi jual-beli dengan cepat. Dukung multi-pembayaran: Tunai, QRIS, Transfer, E-Wallet.' },
    { icon: '📒', title: 'Pembukuan Otomatis', desc: 'Double-entry accounting. Setiap transaksi POS langsung tercatat di jurnal — tidak perlu accountant.' },
    { icon: '🤖', title: 'AI Customer Service', desc: 'Bot WhatsApp otomatis jawab pertanyaan pelanggan 24/7. Bisa di-training dengan FAQ toko Anda.' },
    { icon: '📊', title: 'Laporan Keuangan', desc: 'Laba rugi, arus kas, neraca — siap pakai untuk pajak dan pengajuan kredit bank.' },
    { icon: '🏪', title: 'Multi-Toko', desc: 'Kelola beberapa cabang dalam satu akun. Cocok untuk franchise dan warung kopi.' },
    { icon: '🔗', title: 'Integrasi Marketplace', desc: 'Hubungkan dengan GoFood, Grab, Shopee. Stok tersinkron otomatis di satu dashboard.' },
  ],
  steps: [
    { title: 'Daftar dalam 30 detik', desc: 'Masukkan nomor WhatsApp, terima OTP, langsung masuk. Tanpa verifikasi email.' },
    { title: 'Pilih paket atau mulai gratis', desc: 'Lite gratis selamanya. Atau pilih Pro/Ultimate untuk fitur AI dan multi-user.' },
    { title: 'Mulai berjualan', desc: 'Kasir langsung bisa dipakai. Pembukuan berjalan otomatis di belakang layar.' },
  ],
  testimonials: [
    { quote: 'Dulu pembukuan pakai Excel, sekarang semua otomatis. Owner warung kopi saya bisa fokus layani pelanggan.', name: 'Budi Santoso', role: 'Pemilik Warung Kopi, Bandung' },
    { quote: 'AI chatbot-nya jawab pertanyaan pelanggan di malam hari. Servis tetap prima meski saya lagi tutup.', name: 'Siti Rahayu', role: 'Pemilik Toko Fashion, Surabaya' },
    { quote: 'Laporan keuangan langsung jadi, tinggal export ke PDF untuk pengajuan KUR ke bank.', name: 'Ahmad Hidayat', role: 'Pemilik Klinik Pratama, Medan' },
  ],
  cta: {
    title: 'Siap mengelola usaha dengan lebih cerdas?',
    subtitle: 'Bergabung dengan 100+ UMKM Indonesia hari ini. Gratis selamanya.',
    button: 'Daftar Sekarang — Gratis',
  },
  trust: {
    label: 'Telah dipercaya oleh pelaku UMKM di',
    cities: ['Jakarta', 'Surabaya', 'Bandung', 'Medan', 'Makassar', 'Semarang'],
  },
}

const fallbackPlans = [
  { id: 'lite', name: 'Lite', sort_order: 1, price_monthly: 3500000, price_yearly: 35000000, description: 'Untuk bisnis kecil yang baru memulai.', features: [{ feature_key: 'pos', feature_name: 'Kasir POS Dasar' }, { feature_key: 'journal', feature_name: '100 Transaksi/bulan' }, { feature_key: 'reports', feature_name: 'Laporan Keuangan Dasar' }, { feature_key: 'chatbot', feature_name: 'AI Chatbot WhatsApp (50 pesan)' }, { feature_key: 'users', feature_name: '1 Pengguna' }] },
  { id: 'pro', name: 'Pro', sort_order: 2, price_monthly: 15000000, price_yearly: 150000000, description: 'Untuk bisnis berkembang yang butuh lebih.', features: [{ feature_key: 'pos', feature_name: 'Kasir POS Lengkap' }, { feature_key: 'journal', feature_name: '10.000 Transaksi/bulan' }, { feature_key: 'reports', feature_name: 'Semua Laporan Keuangan' }, { feature_key: 'chatbot', feature_name: 'AI Chatbot Unlimited' }, { feature_key: 'users', feature_name: '5 Pengguna' }, { feature_key: 'ai_vision', feature_name: 'AI Vision (foto produk)' }, { feature_key: 'marketplace', feature_name: 'Integrasi Marketplace' }] },
  { id: 'ultimate', name: 'Ultimate', sort_order: 3, price_monthly: 30000000, price_yearly: 300000000, description: 'Untuk bisnis menengah dan franchise.', features: [{ feature_key: 'pos', feature_name: 'Kasir POS Lengkap' }, { feature_key: 'journal', feature_name: 'Transaksi Unlimited' }, { feature_key: 'reports', feature_name: 'Semua Laporan Keuangan' }, { feature_key: 'chatbot', feature_name: 'AI Chatbot Unlimited' }, { feature_key: 'users', feature_name: 'User Unlimited' }, { feature_key: 'multi_branch', feature_name: 'Multi-Toko (5 Cabang)' }, { feature_key: 'ai_multimodal', feature_name: 'AI Multimodal (vision + audio)' }, { feature_key: 'custom_branding', feature_name: 'Custom Branding' }, { feature_key: 'priority_support', feature_name: 'Priority Support' }] },
]

const hero = computed(() => (landingContent.value.hero || fallbackLandingContent.hero) as typeof fallbackLandingContent.hero)
const features = computed(() => (landingContent.value.features || fallbackLandingContent.features) as typeof fallbackLandingContent.features)
const steps = computed(() => (landingContent.value.steps || fallbackLandingContent.steps) as typeof fallbackLandingContent.steps)
const testimonials = computed(() => (landingContent.value.testimonials || fallbackLandingContent.testimonials) as typeof fallbackLandingContent.testimonials)
const cta = computed(() => (landingContent.value.cta || fallbackLandingContent.cta) as typeof fallbackLandingContent.cta)
const trust = computed(() => (landingContent.value.trust || fallbackLandingContent.trust) as typeof fallbackLandingContent.trust)
const themeClass = computed(() => 'dark')

onMounted(async () => {
  await Promise.all([
    (async () => {
      try {
        const res = await api.getPublicPlans()
        plans.value = (res?.success && Array.isArray(res.data) && res.data.length > 0) ? res.data : fallbackPlans
      } catch (e) {
        plans.value = fallbackPlans
      }
    })(),
    (async () => {
      try {
        const res = await api.getLandingConfigs()
        landingContent.value = (res?.success && res.data) ? res.data : fallbackLandingContent
      } catch (e) {
        landingContent.value = fallbackLandingContent
      }
    })(),
  ])

  const observer = new IntersectionObserver((entries) => {
    entries.forEach((entry) => {
      if (entry.isIntersecting) {
        entry.target.classList.add('revealed')
      }
    })
  }, { threshold: 0.1 })

  document.querySelectorAll('.reveal').forEach((el) => observer.observe(el))
})
</script>

<style scoped>
.landing {
  background: #0d0d14;
  color: #e2e8f0;
  min-height: 100vh;
  overflow-x: hidden;
}

.trust-bar {
  padding: 60px 1.5rem;
  text-align: center;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  opacity: 0;
  transform: translateY(20px);
  transition: all 0.8s ease;
}

.trust-bar.revealed {
  opacity: 1;
  transform: translateY(0);
}

.trust-bar p {
  font-size: 0.875rem;
  color: #94a3b8;
  margin: 0 0 1rem;
}

.trust-badges {
  display: flex;
  gap: 1.5rem;
  justify-content: center;
  flex-wrap: wrap;
}

.trust-badges span {
  font-size: 0.9375rem;
  font-weight: 600;
  color: #cbd5e1;
  padding: 0.5rem 1rem;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 6px;
}

.section-testimoni {
  padding: 100px 1.5rem;
  background: rgba(255, 255, 255, 0.01);
}

.section-inner {
  max-width: 1120px;
  margin: 0 auto;
}

.section-label {
  text-align: center;
  font-size: 0.875rem;
  font-weight: 600;
  color: #fbbf24;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 1rem;
}

.section-title {
  text-align: center;
  font-size: clamp(2rem, 4vw, 2.75rem);
  font-weight: 800;
  line-height: 1.2;
  color: #f1f5f9;
  margin: 0 0 3rem;
}

.testimoni-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 2rem;
}

.testimoni-card {
  padding: 2rem;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 12px;
}

.testimoni-stars {
  display: flex;
  gap: 4px;
  margin-bottom: 1rem;
  color: #fbbf24;
  font-size: 1.25rem;
}

.testimoni-card blockquote {
  font-size: 1rem;
  line-height: 1.6;
  color: #cbd5e1;
  margin: 0 0 1.5rem;
  font-style: italic;
}

.testimoni-author {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.author-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: #1a1a2e;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.25rem;
}

.testimoni-author strong {
  display: block;
  font-size: 1rem;
  color: #f1f5f9;
  margin-bottom: 2px;
}

.testimoni-author span {
  font-size: 0.875rem;
  color: #94a3b8;
}

.section-cta {
  padding: 100px 1.5rem;
  position: relative;
  overflow: hidden;
}

.cta-inner {
  max-width: 800px;
  margin: 0 auto;
  text-align: center;
  position: relative;
  padding: 4rem 2rem;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 20px;
}

.cta-glow {
  position: absolute;
  top: -100px;
  left: 50%;
  transform: translateX(-50%);
  width: 400px;
  height: 400px;
  background: radial-gradient(circle, rgba(245, 158, 11, 0.15), transparent 70%);
  filter: blur(80px);
  pointer-events: none;
}

.cta-inner h2 {
  font-size: clamp(2rem, 4vw, 2.5rem);
  font-weight: 800;
  color: #f1f5f9;
  margin: 0 0 1rem;
}

.cta-inner p {
  font-size: 1.125rem;
  color: #94a3b8;
  margin: 0 0 2rem;
}

.cta-main {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem 2rem;
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: #1a1a2e;
  font-weight: 600;
  font-size: 1.125rem;
  text-decoration: none;
  border-radius: 8px;
  transition: all 0.3s ease;
  box-shadow: 0 4px 16px rgba(245, 158, 11, 0.3);
}

.cta-main:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(245, 158, 11, 0.4);
}

.reveal {
  opacity: 0;
  transform: translateY(20px);
  transition: all 0.8s cubic-bezier(0.16, 1, 0.3, 1);
}

.reveal.revealed {
  opacity: 1;
  transform: translateY(0);
}

@media (max-width: 768px) {
  .trust-bar {
    padding: 40px 1.25rem;
  }

  .section-testimoni {
    padding: 60px 1.25rem;
  }

  .testimoni-grid {
    grid-template-columns: 1fr;
  }

  .section-cta {
    padding: 60px 1.25rem;
  }

  .cta-inner {
    padding: 3rem 1.5rem;
  }
}
</style>

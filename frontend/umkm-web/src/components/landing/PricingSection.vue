<template>
  <section id="harga" class="section-pricing reveal">
    <div class="section-inner">
      <div class="section-label">Harga Transparan</div>
      <h2 class="section-title">Paket yang cocok untuk<br>semua ukuran usaha</h2>
      <p class="section-sub">Tidak ada biaya tersembunyi. Mulai gratis, upgrade sesuai kebutuhan.</p>

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
              <span v-else class="price-amount">{{ formatRupiah(plan.price_monthly) }}<span class="price-period">/bulan</span></span>
            </span>
            <span v-else>
              <span v-if="plan.price_yearly === 0" class="price-amount">Gratis</span>
              <span v-else class="price-amount">{{ formatRupiah(plan.price_yearly) }}<span class="price-period">/tahun</span></span>
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
</template>

<script setup lang="ts">
import { ref } from 'vue'

defineProps<{
  plans: Array<{
    id: string
    name: string
    description: string
    price_monthly: number
    price_yearly: number
    sort_order: number
    features: Array<{
      feature_key: string
      feature_name: string
    }>
  }>
  formatRupiah: (value: number) => string
}>()

const billingCycle = ref<'monthly' | 'yearly'>('monthly')
</script>

<style scoped>
.section-pricing {
  padding: 100px 1.5rem;
  background: linear-gradient(180deg, #0d0d14 0%, #16161f 100%);
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
  margin: 0 0 1rem;
}

.section-sub {
  text-align: center;
  font-size: 1.125rem;
  color: #94a3b8;
  margin: 0 0 3rem;
  max-width: 600px;
  margin-left: auto;
  margin-right: auto;
}

.billing-toggle {
  display: flex;
  justify-content: center;
  gap: 0.5rem;
  margin-bottom: 3rem;
  padding: 4px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 12px;
  width: fit-content;
  margin-left: auto;
  margin-right: auto;
}

.toggle-btn {
  padding: 0.625rem 1.5rem;
  background: transparent;
  border: none;
  border-radius: 8px;
  font-size: 0.9375rem;
  font-weight: 600;
  color: #94a3b8;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.toggle-btn.active {
  background: #f59e0b;
  color: #1a1a2e;
}

.save-badge {
  padding: 2px 8px;
  background: rgba(34, 197, 94, 0.2);
  color: #22c55e;
  font-size: 0.75rem;
  border-radius: 4px;
  font-weight: 700;
}

.pricing-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 2rem;
  max-width: 960px;
  margin: 0 auto;
}

.pricing-card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 16px;
  padding: 2rem;
  position: relative;
  transition: all 0.3s ease;
}

.pricing-card:hover {
  transform: translateY(-4px);
  border-color: rgba(245, 158, 11, 0.3);
  box-shadow: 0 12px 40px rgba(245, 158, 11, 0.1);
}

.pricing-featured {
  border-color: rgba(245, 158, 11, 0.4);
  background: rgba(245, 158, 11, 0.05);
}

.pricing-badge {
  position: absolute;
  top: -12px;
  left: 50%;
  transform: translateX(-50%);
  padding: 4px 16px;
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: #1a1a2e;
  font-size: 0.75rem;
  font-weight: 700;
  border-radius: 999px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.plan-name {
  font-size: 1.5rem;
  font-weight: 700;
  color: #f1f5f9;
  margin-bottom: 0.5rem;
}

.plan-price {
  margin-bottom: 1rem;
}

.price-amount {
  font-size: 2.5rem;
  font-weight: 800;
  color: #fbbf24;
  line-height: 1;
}

.price-period {
  font-size: 1rem;
  color: #94a3b8;
  font-weight: 500;
}

.plan-desc {
  font-size: 0.9375rem;
  color: #94a3b8;
  margin: 0 0 1.5rem;
  line-height: 1.5;
}

.plan-features {
  list-style: none;
  padding: 0;
  margin: 0 0 2rem;
}

.plan-features li {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  font-size: 0.9375rem;
  color: #cbd5e1;
  margin-bottom: 0.75rem;
}

.plan-features li svg {
  flex-shrink: 0;
  margin-top: 2px;
  color: #22c55e;
}

.plan-cta {
  display: block;
  width: 100%;
  padding: 0.875rem;
  background: rgba(255, 255, 255, 0.05);
  color: #f1f5f9;
  text-align: center;
  font-weight: 600;
  font-size: 1rem;
  text-decoration: none;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: all 0.3s ease;
}

.plan-cta:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.15);
}

.cta-active {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: #1a1a2e;
  border: none;
}

.cta-active:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(245, 158, 11, 0.3);
}

@media (max-width: 768px) {
  .section-pricing {
    padding: 60px 1.25rem;
  }

  .pricing-grid {
    grid-template-columns: 1fr;
    max-width: 400px;
  }
}
</style>

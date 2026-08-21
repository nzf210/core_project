<template>
  <section id="fitur" class="section-features reveal">
    <div class="section-inner">
      <div class="section-label">Fitur Unggulan</div>
      <h2 class="section-title">Semua yang Anda butuhkan,<br>dalam satu aplikasi</h2>
      <div class="features-grid">
        <div v-for="(f, i) in features" :key="i" class="feature-card" :style="{ '--delay': `${Number(i) * 80}ms` }">
          <div class="feature-icon">{{ f.icon }}</div>
          <h3>{{ f.title }}</h3>
          <p>{{ f.desc }}</p>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
defineProps<{
  features: Array<{
    icon: string
    title: string
    desc: string
  }>
}>()
</script>

<style scoped>
.section-features {
  padding: 100px 1.5rem;
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

.features-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 2rem;
}

.feature-card {
  padding: 2rem;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 12px;
  transition: all 0.3s ease;
  opacity: 0;
  transform: translateY(20px);
}

.feature-card.revealed {
  opacity: 1;
  transform: translateY(0);
  transition-delay: var(--delay, 0ms);
}

.feature-card:hover {
  transform: translateY(-4px);
  border-color: rgba(245, 158, 11, 0.2);
  background: rgba(255, 255, 255, 0.04);
}

.feature-icon {
  font-size: 2.5rem;
  margin-bottom: 1rem;
}

.feature-card h3 {
  font-size: 1.25rem;
  font-weight: 700;
  color: #f1f5f9;
  margin: 0 0 0.75rem;
}

.feature-card p {
  font-size: 0.9375rem;
  color: #94a3b8;
  line-height: 1.6;
  margin: 0;
}

@media (max-width: 768px) {
  .section-features {
    padding: 60px 1.25rem;
  }

  .features-grid {
    grid-template-columns: 1fr;
  }
}
</style>

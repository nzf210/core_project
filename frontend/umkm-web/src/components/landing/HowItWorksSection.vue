<template>
  <section class="section-how reveal">
    <div class="section-inner">
      <div class="section-label">Cara Kerja</div>
      <h2 class="section-title">Dari nol ke operasional<br>dalam 3 menit</h2>
      <div class="steps">
        <div v-for="(step, i) in steps" :key="i" class="step" :style="{ '--delay': `${Number(i) * 120}ms` }">
          <div class="step-num">{{ String(Number(i) + 1).padStart(2, '0') }}</div>
          <div class="step-body">
            <h3>{{ step.title }}</h3>
            <p>{{ step.desc }}</p>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
defineProps<{
  steps: Array<{
    title: string
    desc: string
  }>
}>()
</script>

<style scoped>
.section-how {
  padding: 100px 1.5rem;
  background: rgba(255, 255, 255, 0.01);
}

.section-inner {
  max-width: 800px;
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

.steps {
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

.step {
  display: flex;
  gap: 1.5rem;
  padding: 2rem;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 12px;
  opacity: 0;
  transform: translateX(-20px);
  transition: all 0.5s ease;
}

.step.revealed {
  opacity: 1;
  transform: translateX(0);
  transition-delay: var(--delay, 0ms);
}

.step-num {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: #1a1a2e;
  font-size: 1.25rem;
  font-weight: 800;
  border-radius: 50%;
}

.step-body h3 {
  font-size: 1.25rem;
  font-weight: 700;
  color: #f1f5f9;
  margin: 0 0 0.5rem;
}

.step-body p {
  font-size: 0.9375rem;
  color: #94a3b8;
  line-height: 1.6;
  margin: 0;
}

@media (max-width: 768px) {
  .section-how {
    padding: 60px 1.25rem;
  }

  .step {
    flex-direction: column;
    align-items: center;
    text-align: center;
  }
}
</style>

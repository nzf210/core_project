<template>
  <section class="hero">
    <div class="hero-bg">
      <div class="hero-mesh"></div>
      <div class="hero-glow"></div>
    </div>
    <div class="hero-inner reveal">
      <div class="hero-badge">
        <span class="badge-dot"></span>
        {{ hero.badge }}
      </div>
      <h1 class="hero-title">
        {{ hero.title_line1 }}<br>
        <span class="title-accent">{{ hero.title_line2 }}</span><br>
        {{ hero.title_line3 }}
      </h1>
      <p class="hero-sub">
        {{ hero.subtitle }}
      </p>
      <div class="hero-cta">
        <a href="/register" class="cta-main">
          <span>{{ hero.cta_primary }}</span>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
        </a>
        <a href="#harga" class="cta-secondary">{{ hero.cta_secondary }}</a>
      </div>
      <div class="hero-stats">
        <template v-for="(stat, i) in hero.stats" :key="i">
          <div v-if="Number(i) > 0" class="stat-divider"></div>
          <div class="stat">
            <strong>{{ stat.value }}</strong>
            <span>{{ stat.label }}</span>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
defineProps<{
  hero: {
    badge: string
    title_line1: string
    title_line2: string
    title_line3: string
    subtitle: string
    cta_primary: string
    cta_secondary: string
    stats: Array<{ value: string; label: string }>
  }
}>()
</script>

<style scoped>
.hero {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  padding: 80px 1.5rem 2rem;
}

.hero-bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.hero-mesh {
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse 80% 50% at 50% -20%, rgba(245, 158, 11, 0.15), transparent),
              radial-gradient(ellipse 60% 50% at 50% 120%, rgba(139, 92, 246, 0.08), transparent);
  filter: blur(60px);
}

.hero-glow {
  position: absolute;
  top: -100px;
  left: 50%;
  transform: translateX(-50%);
  width: 600px;
  height: 600px;
  background: radial-gradient(circle, rgba(245, 158, 11, 0.2), transparent 70%);
  filter: blur(80px);
  animation: pulse 8s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 0.6; transform: translateX(-50%) scale(1); }
  50% { opacity: 1; transform: translateX(-50%) scale(1.1); }
}

.hero-inner {
  position: relative;
  max-width: 720px;
  text-align: center;
  opacity: 0;
  transform: translateY(30px);
  transition: all 0.8s cubic-bezier(0.16, 1, 0.3, 1);
}

.hero-inner.revealed {
  opacity: 1;
  transform: translateY(0);
}

.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.4rem 1rem;
  background: rgba(245, 158, 11, 0.1);
  border: 1px solid rgba(245, 158, 11, 0.2);
  border-radius: 999px;
  font-size: 0.875rem;
  font-weight: 500;
  color: #fbbf24;
  margin-bottom: 1.5rem;
}

.badge-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #fbbf24;
  animation: pulse-dot 2s ease-in-out infinite;
}

@keyframes pulse-dot {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.hero-title {
  font-size: clamp(2.5rem, 6vw, 4rem);
  font-weight: 800;
  line-height: 1.1;
  letter-spacing: -0.03em;
  color: #f1f5f9;
  margin: 0 0 1.5rem;
}

.title-accent {
  background: linear-gradient(135deg, #fbbf24 0%, #f59e0b 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.hero-sub {
  font-size: clamp(1.125rem, 2vw, 1.375rem);
  line-height: 1.6;
  color: #94a3b8;
  margin: 0 0 2.5rem;
  max-width: 600px;
  margin-left: auto;
  margin-right: auto;
}

.hero-cta {
  display: flex;
  gap: 1rem;
  justify-content: center;
  flex-wrap: wrap;
  margin-bottom: 3rem;
}

.cta-main {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.875rem 1.75rem;
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: #1a1a2e;
  font-weight: 600;
  font-size: 1rem;
  text-decoration: none;
  border-radius: 8px;
  transition: all 0.3s ease;
  box-shadow: 0 4px 16px rgba(245, 158, 11, 0.3);
}

.cta-main:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(245, 158, 11, 0.4);
}

.cta-secondary {
  display: inline-flex;
  align-items: center;
  padding: 0.875rem 1.75rem;
  background: rgba(255, 255, 255, 0.05);
  color: #f1f5f9;
  font-weight: 600;
  font-size: 1rem;
  text-decoration: none;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: all 0.3s ease;
}

.cta-secondary:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.15);
}

.hero-stats {
  display: flex;
  gap: 2rem;
  justify-content: center;
  flex-wrap: wrap;
}

.stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
}

.stat strong {
  font-size: 1.875rem;
  font-weight: 700;
  color: #fbbf24;
  line-height: 1;
}

.stat span {
  font-size: 0.875rem;
  color: #94a3b8;
  font-weight: 500;
}

.stat-divider {
  width: 1px;
  height: 40px;
  background: rgba(255, 255, 255, 0.1);
}

@media (max-width: 768px) {
  .hero {
    min-height: auto;
    padding: 100px 1.25rem 3rem;
  }

  .hero-cta {
    flex-direction: column;
    align-items: stretch;
  }

  .cta-main, .cta-secondary {
    justify-content: center;
  }

  .hero-stats {
    gap: 1.5rem;
  }

  .stat-divider {
    display: none;
  }
}
</style>

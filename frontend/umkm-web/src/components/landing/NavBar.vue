<template>
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
      <button class="nav-mobile-toggle" @click="mobileOpen = !mobileOpen" aria-label="Menu">
        <span></span><span></span><span></span>
      </button>
    </div>
    <div v-if="mobileOpen" class="nav-mobile-menu">
      <a href="#fitur" @click="mobileOpen = false">Fitur</a>
      <a href="#harga" @click="mobileOpen = false">Harga</a>
      <a href="#testimoni" @click="mobileOpen = false">Testimoni</a>
      <hr>
      <a href="/login" @click="mobileOpen = false">Masuk</a>
      <a href="/register" class="cta-mobile" @click="mobileOpen = false">Daftar Gratis</a>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const mobileOpen = ref(false)
</script>

<style scoped>
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
  color: #cbd5e1;
  font-weight: 500;
  font-size: 0.9375rem;
  transition: color 0.2s ease;
}

.nav-links a:hover {
  color: #fbbf24;
}

.nav-actions {
  display: flex;
  gap: 0.75rem;
}

.btn-ghost {
  padding: 0.5rem 1rem;
  background: transparent;
  color: #f1f5f9;
  text-decoration: none;
  font-weight: 600;
  font-size: 0.9375rem;
  border-radius: 6px;
  transition: background 0.2s ease;
}

.btn-ghost:hover {
  background: rgba(255, 255, 255, 0.05);
}

.btn-primary {
  padding: 0.5rem 1rem;
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: #1a1a2e;
  text-decoration: none;
  font-weight: 600;
  font-size: 0.9375rem;
  border-radius: 6px;
  transition: transform 0.2s ease;
}

.btn-primary:hover {
  transform: translateY(-1px);
}

.nav-mobile-toggle {
  display: none;
  flex-direction: column;
  gap: 4px;
  background: none;
  border: none;
  cursor: pointer;
  padding: 0.5rem;
  margin-left: auto;
}

.nav-mobile-toggle span {
  width: 20px;
  height: 2px;
  background: #f1f5f9;
  border-radius: 2px;
  transition: all 0.3s ease;
}

.nav-mobile-menu {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem 1.5rem;
  background: rgba(13, 13, 20, 0.95);
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

.nav-mobile-menu a {
  text-decoration: none;
  color: #cbd5e1;
  font-weight: 500;
  padding: 0.5rem 0;
}

.nav-mobile-menu hr {
  border: none;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  margin: 0.5rem 0;
}

.cta-mobile {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: #1a1a2e !important;
  font-weight: 600;
  padding: 0.75rem 1rem;
  border-radius: 6px;
  text-align: center;
}

@media (max-width: 768px) {
  .nav-links,
  .nav-actions {
    display: none;
  }

  .nav-mobile-toggle {
    display: flex;
  }
}

@media (min-width: 769px) {
  .nav-mobile-menu {
    display: none !important;
  }
}
</style>

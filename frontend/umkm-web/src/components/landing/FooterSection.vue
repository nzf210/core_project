<template>
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
</template>

<style scoped>
.footer {
  background: #0a0a10;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  padding: 60px 1.5rem 2rem;
}

.footer-inner {
  max-width: 1120px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: 1.5fr 2fr;
  gap: 3rem;
}

.footer-brand {
  max-width: 320px;
}

.footer-logo {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1.1rem;
  font-weight: 700;
  color: #f1f5f9;
  margin-bottom: 1rem;
}

.footer-brand p {
  font-size: 0.9375rem;
  color: #94a3b8;
  line-height: 1.6;
  margin: 0;
}

.footer-links {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 2rem;
}

.footer-col h4 {
  font-size: 0.875rem;
  font-weight: 700;
  color: #f1f5f9;
  margin: 0 0 1rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.footer-col a {
  display: block;
  font-size: 0.9375rem;
  color: #94a3b8;
  text-decoration: none;
  margin-bottom: 0.75rem;
  transition: color 0.2s ease;
}

.footer-col a:hover {
  color: #fbbf24;
}

.footer-bottom {
  max-width: 1120px;
  margin: 3rem auto 0;
  padding-top: 2rem;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  text-align: center;
}

.footer-bottom p {
  font-size: 0.875rem;
  color: #64748b;
  margin: 0;
}

@media (max-width: 768px) {
  .footer {
    padding: 40px 1.25rem 1.5rem;
  }

  .footer-inner {
    grid-template-columns: 1fr;
    gap: 2rem;
  }

  .footer-links {
    grid-template-columns: 1fr;
    gap: 1.5rem;
  }

  .footer-bottom {
    margin-top: 2rem;
    padding-top: 1.5rem;
  }
}
</style>

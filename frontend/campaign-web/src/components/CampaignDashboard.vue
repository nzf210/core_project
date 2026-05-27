<template>
  <div class="dashboard">
    <div class="summary-cards">
      <div class="card">
        <div class="label">Total Relawan</div>
        <div class="value">1,250</div>
        <div class="change">Tersebar di 15 Kecamatan</div>
      </div>
      <div class="card">
        <div class="label">Target Dukungan</div>
        <div class="value" style="color: var(--accent-primary)">100,000</div>
        <div class="change">Pemilih Potensial</div>
      </div>
      <div class="card">
        <div class="label">Dukungan Terkumpul</div>
        <div class="value text-gradient">45,000</div>
        <div class="change" style="color: var(--success)">45% dari target</div>
      </div>
    </div>

    <div class="grid-layout">
      <div class="card map-card">
        <h2 style="margin-bottom: 1rem; color: var(--text-secondary)">Peta Sebaran Dukungan</h2>
        <div id="map" class="map-container"></div>
      </div>

      <div class="card">
        <h2 style="margin-bottom: 1rem; color: var(--text-secondary)">Relawan Terbaik</h2>
        <ul class="volunteer-list">
          <li class="volunteer-item">
            <div class="flex items-center gap-4">
              <div class="avatar">1</div>
              <div>
                <div class="name">Budi Santoso</div>
                <div class="area">Kecamatan A</div>
              </div>
            </div>
            <div class="points"><span class="badge badge-primary">850 Dukungan</span></div>
          </li>
          <li class="volunteer-item">
            <div class="flex items-center gap-4">
              <div class="avatar">2</div>
              <div>
                <div class="name">Siti Aminah</div>
                <div class="area">Kecamatan B</div>
              </div>
            </div>
            <div class="points"><span class="badge badge-primary">620 Dukungan</span></div>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'

onMounted(() => {
  const map = L.map('map').setView([-6.200000, 106.816666], 11) // Jakarta coordinates

  L.tileLayer('https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png', {
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>',
    subdomains: 'abcd',
    maxZoom: 19
  }).addTo(map)

  // Add dummy markers
  L.marker([-6.2, 106.8]).addTo(map).bindPopup('<b>Kecamatan A</b><br>850 Dukungan')
  L.marker([-6.25, 106.85]).addTo(map).bindPopup('<b>Kecamatan B</b><br>620 Dukungan')
  L.marker([-6.15, 106.75]).addTo(map).bindPopup('<b>Kecamatan C</b><br>430 Dukungan')
})
</script>

<style scoped>
.summary-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}
.label {
  color: var(--text-muted);
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.5rem;
  font-weight: 600;
}
.value {
  font-size: 2.5rem;
  font-weight: 800;
  margin-bottom: 0.25rem;
}
.change {
  font-size: 0.875rem;
  color: var(--text-secondary);
}
.grid-layout {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 1.5rem;
}
@media (max-width: 768px) {
  .grid-layout { grid-template-columns: 1fr; }
}
.map-container {
  height: 350px;
  width: 100%;
  border-radius: var(--radius-sm);
  z-index: 1;
}
.volunteer-list {
  list-style: none;
}
.volunteer-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 0;
  border-bottom: 1px solid var(--border-color);
}
.volunteer-item:last-child { border-bottom: none; }
.avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--bg-tertiary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  color: var(--text-secondary);
}
.name { font-weight: 600; }
.area { font-size: 0.875rem; color: var(--text-muted); }
</style>

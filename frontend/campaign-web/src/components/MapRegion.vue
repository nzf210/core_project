<template>
  <div class="map-region">
    <h2>GIS & Map</h2>
    <div class="map-placeholder">
      <p>Interactive Map will be displayed here using Leaflet/Mapbox.</p>
      <ul>
        <li v-for="p in provinces" :key="p.id">{{ p.name }}</li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { apiClient } from '../api'

import { ref, onMounted } from 'vue'

const provinces = ref<any[]>([])

const fetchProvinces = async () => {
  try {
    const res = await apiClient('/regions/provinces')
    const data = await res.json()
    if (data.success) {
      provinces.value = data.data
    }
  } catch (err) {
    console.error(err)
  }
}

onMounted(fetchProvinces)
</script>

<style scoped>
.map-region { padding: 20px; }
.map-placeholder {
  height: 400px;
  background: #444;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
}
</style>

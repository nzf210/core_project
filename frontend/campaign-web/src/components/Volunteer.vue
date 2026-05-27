<template>
  <div class="volunteers">
    <h3 style="margin-bottom: 1rem; color: var(--text-secondary)">Manajemen Relawan</h3>
    <form @submit.prevent="addVolunteer" class="volunteer-form">
      <input v-model="form.name" placeholder="Nama Relawan" required class="input-field" />
      <input v-model="form.phone" placeholder="Nomor Telepon" required class="input-field" />
      <button type="submit" class="btn-primary">Tambah Relawan</button>
    </form>
    
    <div v-if="volunteers.length === 0" style="color: var(--text-muted)">
      Belum ada data relawan.
    </div>
    <ul class="volunteer-list">
      <li v-for="v in volunteers" :key="v.id" class="volunteer-item">
        <div class="flex items-center gap-4">
          <div class="avatar">{{ v.name.charAt(0).toUpperCase() }}</div>
          <div>
            <div class="name">{{ v.name }}</div>
            <div class="area">{{ v.phone }}</div>
          </div>
        </div>
        <div class="points"><span class="badge badge-primary">Rank: {{ v.rank || 0 }}</span></div>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { apiClient } from '../api'

import { ref, onMounted } from 'vue'

const volunteers = ref<any[]>([])
const form = ref({ name: '', phone: '' })

const fetchVolunteers = async () => {
  try {
    const res = await apiClient('/volunteers', {
      headers: { 'X-Tenant-ID': 'default' }
    })
    const data = await res.json()
    if (data.success) {
      volunteers.value = data.data
    }
  } catch (err) {
    console.error(err)
  }
}

const addVolunteer = async () => {
  try {
    const res = await apiClient('/volunteers', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'default' },
      body: JSON.stringify(form.value)
    })
    const data = await res.json()
    if (data.success) {
      form.value = { name: '', phone: '' }
      fetchVolunteers()
    }
  } catch (err) {
    console.error(err)
  }
}

onMounted(fetchVolunteers)
</script>

<style scoped>
.volunteer-form {
  display: flex;
  gap: 1rem;
  margin-bottom: 2rem;
}
@media (max-width: 768px) {
  .volunteer-form {
    flex-direction: column;
  }
}
.input-field {
  padding: 0.5rem 1rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  outline: none;
  font-family: inherit;
}
.input-field:focus {
  border-color: var(--accent-primary);
}
.btn-primary {
  background: var(--accent-gradient);
  color: white;
  border: none;
  padding: 0.5rem 1.5rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-weight: 600;
}
.volunteer-list { list-style: none; }
.volunteer-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 0;
  border-bottom: 1px solid var(--border-color);
}
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
.name { font-weight: 600; color: var(--text-primary); }
.area { font-size: 0.875rem; color: var(--text-muted); }
</style>

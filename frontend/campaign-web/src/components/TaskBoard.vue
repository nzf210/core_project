<template>
  <div class="task-board">
    <h3 style="margin-bottom: 1rem; color: var(--text-secondary)">Task & Operations</h3>
    <form @submit.prevent="addTask" class="flex flex-col gap-4" style="max-width: 500px; margin-bottom: 2rem;">
      <div>
        <label for="task-title" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">Task Title</label>
        <input id="task-title" v-model="form.title" placeholder="Task Title" required class="input-field" />
      </div>
      <div>
        <label for="task-desc" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">Description</label>
        <textarea id="task-desc" v-model="form.description" placeholder="Description" rows="3" class="input-field"></textarea>
      </div>

      <div>
        <label for="task-assign" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">Assign to</label>
        <select id="task-assign" v-model="form.assigned_to" class="input-field">
          <option value="">-- Assign to (Opsional) --</option>
          <option v-for="u in users" :key="u.id" :value="u.id">{{ u.username }} ({{ u.role }})</option>
        </select>
      </div>

      <div>
        <label for="task-verif" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">Verification Type</label>
        <select id="task-verif" v-model="form.verification_type" class="input-field">
          <option value="auto">Verifikasi Otomatis (GPS & AI)</option>
          <option value="manual">Verifikasi Manual (Admin)</option>
        </select>
      </div>

      <button type="submit" class="btn-primary" style="align-self: flex-start;">Create Task</button>
    </form>

    <div class="kanban-board">
      <div class="kanban-column" v-for="status in ['todo', 'in_progress', 'done']" :key="status">
        <h4>{{ status.replace('_', ' ').toUpperCase() }}</h4>
        <div class="task-card" v-for="t in tasks.filter(x => x.status === status)" :key="t.id">
          <h5>{{ t.title }}</h5>
          <p>{{ t.description }}</p>
          <div class="meta">
            <small v-if="t.assigned_to">Assignee: {{ getUserName(t.assigned_to) }}</small>
            <span class="badge" :class="t.is_verified ? 'badge-primary' : 'badge-secondary'">
              {{ t.is_verified ? 'Verified' : 'Unverified' }}
            </span>
          </div>
          
          <div v-if="status !== 'done'" style="margin-top: 10px;">
            <button class="btn-secondary" @click="updateTaskStatus(t.id, 'done')">Mark Done</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { apiClient } from '../api'
import { ref, onMounted } from 'vue'

const tasks = ref<any[]>([])
const users = ref<any[]>([])
const form = ref({ title: '', description: '', campaign_id: '', assigned_to: '', verification_type: 'auto' })

const fetchTasks = async () => {
  try {
    const res = await apiClient('/tasks')
    const data = await res.json()
    if (data.success) { tasks.value = data.data }
  } catch { /* ignore fetch errors */ }
}

const fetchUsers = async () => {
  try {
    const res = await apiClient('/users')
    const data = await res.json()
    if (data.success) { users.value = data.data }
  } catch { /* ignore fetch errors */ }
}

const fetchCampaigns = async () => {
  try {
    const res = await apiClient('/campaigns')
    const data = await res.json()
    if (data.success && data.data.length > 0) {
      form.value.campaign_id = data.data[0].id
    }
  } catch { /* ignore fetch errors */ }
}

const getUserName = (id: string) => {
  const u = users.value.find(user => user.id === id)
  return u ? u.username : 'Unknown'
}

const addTask = async () => {
  try {
    const res = await apiClient('/tasks', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form.value)
    })
    const data = await res.json()
    if (data.success) {
      form.value.title = ''
      form.value.description = ''
      form.value.assigned_to = ''
      fetchTasks()
    } else {
      console.error(data.message)
    }
  } catch { /* ignore fetch errors */ }
}

const updateTaskStatus = async (id: string, status: string) => {
  try {
    const res = await apiClient('/tasks', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id, status, is_verified: false, proof_image: '/mock-proof.png', gps_location: '-6.2088,106.8456' })
    })
    const data = await res.json()
    if (data.success) {
      fetchTasks()
    }
  } catch (err) { console.error(err) }
}

onMounted(() => {
  fetchTasks()
  fetchUsers()
  fetchCampaigns()
})
</script>

<style scoped>
.input-field {
  padding: 0.5rem 1rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  outline: none;
  font-family: inherit;
  width: 100%;
}
.input-field:focus { border-color: var(--accent-primary); }
.btn-primary {
  background: var(--accent-gradient); color: white; border: none;
  padding: 0.5rem 1.5rem; border-radius: var(--radius-sm);
  cursor: pointer; font-weight: 600;
}
.btn-secondary {
  background: var(--surface-0); color: var(--text-primary); border: 1px solid var(--border-color);
  padding: 0.25rem 0.75rem; border-radius: var(--radius-sm);
  cursor: pointer; font-size: 0.8rem;
}
.kanban-board {
  display: flex; gap: 1rem; overflow-x: auto;
}
.kanban-column {
  flex: 1; min-width: 250px; background: var(--bg-tertiary);
  padding: 1rem; border-radius: var(--radius-md);
}
.kanban-column h4 { margin-bottom: 1rem; color: var(--text-secondary); }
.task-card {
  background: var(--bg-secondary); padding: 1rem; border-radius: var(--radius-sm);
  box-shadow: var(--shadow-sm); margin-bottom: 0.5rem;
}
.task-card h5 { margin: 0 0 0.5rem; color: var(--text-primary); }
.task-card p { margin: 0; font-size: 0.875rem; color: var(--text-muted); }
.meta { display: flex; justify-content: space-between; align-items: center; margin-top: 10px; color: var(--text-muted); font-size: 0.8rem; }
.badge { padding: 0.2rem 0.5rem; border-radius: 4px; font-size: 0.7rem; font-weight: 600; }
.badge-primary { background: rgba(59, 130, 246, 0.1); color: var(--accent-primary); }
.badge-secondary { background: rgba(100, 116, 139, 0.1); color: var(--text-secondary); }
</style>

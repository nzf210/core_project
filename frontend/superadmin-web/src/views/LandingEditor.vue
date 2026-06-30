<script setup lang="ts">
import { ref, watch } from 'vue'
import { request } from '../api/client'

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref('')

const configs = ref<any[]>([])
const activeSection = ref('')
const editorContent = ref('')

const sectionLabel = (id: string) => ({
  hero: '🏠 Hero',
  features: '⭐ Features',
  steps: '🪜 Steps',
  testimonials: '💬 Testimonials',
  trust: '🏙️ Trust Bar',
  cta: '📢 CTA Banner',
  footer: '📄 Footer',
}[id] || id)

const loadConfigs = async () => {
  loading.value = true
  try {
    const res = await request('/landing-configs')
    if (res?.data && Array.isArray(res.data)) {
      configs.value = res.data.sort((a: any, b: any) => a.id.localeCompare(b.id))
    }
  } catch (e) {
    error.value = 'Gagal memuat konfigurasi'
  } finally {
    loading.value = false
  }
}

const selectSection = (id: string) => {
  activeSection.value = id
  const cfg = configs.value.find((c: any) => c.id === id)
  editorContent.value = cfg ? JSON.stringify(cfg.content, null, 2) : ''
  error.value = ''
  success.value = ''
}

const formatJSON = () => {
  try {
    const parsed = JSON.parse(editorContent.value)
    editorContent.value = JSON.stringify(parsed, null, 2)
    error.value = ''
  } catch (e: any) {
    error.value = 'JSON tidak valid: ' + e.message
  }
}

const saveContent = async () => {
  error.value = ''
  success.value = ''

  let content: any
  try {
    content = JSON.parse(editorContent.value)
  } catch (e: any) {
    error.value = 'JSON tidak valid: ' + e.message
    return
  }

  saving.value = true
  try {
    const res = await request(`/admin/landing-configs/?id=${encodeURIComponent(activeSection.value)}`, {
      method: 'PUT',
      body: JSON.stringify({ content }),
    })
    if (res?.success || res?.status === 200) {
      success.value = '✅ Tersimpan! Perubahan langsung tampil di landing page.'
      // reload to get updated data
      await loadConfigs()
    } else {
      error.value = res?.message || 'Gagal menyimpan'
    }
  } catch (e: any) {
    error.value = 'Network error: ' + e.message
  } finally {
    saving.value = false
  }
}

watch(() => activeSection.value, (val) => {
  if (val) {
    selectSection(val)
  }
})
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1>Landing Page Editor</h1>
        <p class="subtitle">Edit konten halaman landing secara real-time via JSON</p>
      </div>
    </div>

    <div v-if="loading" class="loading">Memuat...</div>

    <div v-else class="editor-wrap">
      <!-- Section selector -->
      <div class="section-tabs">
        <button
          v-for="cfg in configs"
          :key="cfg.id"
          :class="['section-tab', activeSection === cfg.id ? 'active' : '']"
          @click="selectSection(cfg.id)"
        >
          {{ sectionLabel(cfg.id) }}
        </button>
      </div>

      <!-- JSON Editor -->
      <div v-if="activeSection" class="editor-area">
        <div class="editor-toolbar">
          <span class="editor-label">{{ sectionLabel(activeSection) }}</span>
          <div class="editor-actions">
            <button class="btn btn-secondary btn-sm" @click="formatJSON" :disabled="saving">Format</button>
            <button class="btn btn-primary btn-sm" @click="saveContent" :disabled="saving">
              {{ saving ? 'Menyimpan...' : '💾 Simpan' }}
            </button>
          </div>
        </div>
        <textarea
          v-model="editorContent"
          class="json-editor"
          rows="22"
          spellcheck="false"
          :disabled="saving"
        ></textarea>
        <p v-if="error" class="error-text">{{ error }}</p>
        <p v-if="success" class="success-text">{{ success }}</p>

        <!-- Preview hint -->
        <div class="preview-hint">
          <a :href="`/landing#${activeSection === 'trust' || activeSection === 'cta' ? '' : activeSection}`" target="_blank" class="preview-link">
            👁 Lihat preview di landing page →
          </a>
          <span class="cache-note">Cache: 6 jam (auto-invalidated setelah save)</span>
        </div>
      </div>

      <div v-else class="empty-state">
        Pilih section di atas untuk mengedit konten.
      </div>
    </div>
  </div>
</template>

<style scoped>
.page-header { margin-bottom: 24px; }
.page-header h1 { font-size: 22px; margin-bottom: 2px; }
.subtitle { color: var(--muted); font-size: 13px; }
.loading { text-align: center; color: var(--muted); padding: 60px; }
.empty-state { text-align: center; color: var(--muted); padding: 60px; }

.editor-wrap {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 24px;
}

.section-tabs {
  display: flex;
  gap: 0.4rem;
  flex-wrap: wrap;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
}

.section-tab {
  padding: 0.35rem 0.8rem;
  border: 1px solid var(--border);
  border-radius: 100px;
  background: none;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--muted);
  cursor: pointer;
  transition: all 0.15s;
}
.section-tab:hover { border-color: var(--accent); color: var(--accent); }
.section-tab.active { background: var(--accent); border-color: var(--accent); color: white; }

.editor-area { display: flex; flex-direction: column; gap: 10px; }
.editor-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 1rem; }
.editor-label { font-weight: 700; font-size: 0.95rem; }
.editor-actions { display: flex; gap: 6px; }

.json-editor {
  width: 100%;
  min-height: 360px;
  font-family: 'SF Mono', 'Fira Code', 'Fira Mono', 'Roboto Mono', monospace;
  font-size: 13px;
  line-height: 1.6;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
  color: var(--text);
  resize: vertical;
  tab-size: 2;
  box-sizing: border-box;
}
.json-editor:focus { outline: none; border-color: var(--accent); box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15); }

.error-text { color: var(--danger); font-size: 13px; margin: 0; }
.success-text { color: var(--success); font-size: 13px; margin: 0; }

.preview-hint {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 8px;
  border-top: 1px solid var(--border);
  font-size: 12px;
}
.preview-link { color: var(--accent); text-decoration: none; font-weight: 600; }
.preview-link:hover { text-decoration: underline; }
.cache-note { color: var(--muted); }

.btn-sm { padding: 5px 12px; font-size: 12px; border-radius: 6px; cursor: pointer; }
.btn-secondary { background: var(--bg); border: 1px solid var(--border); color: var(--text); }
.btn-primary { background: var(--accent); border: 1px solid var(--accent); color: white; }
</style>

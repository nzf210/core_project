<template>
  <div v-if="show" class="modal-overlay">
    <div class="modal-content landing-editor-modal">
      <div class="modal-header">
        <h3>📝 Edit Landing Page Content</h3>
        <button class="modal-close" @click="$emit('close')">✕</button>
      </div>

      <div class="modal-body">
        <div v-if="loading" class="loading-block">Memuat...</div>

        <template v-else>
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
                <button class="btn btn-cancel btn-sm" @click="$emit('close')" :disabled="saving" v-if="!saving">Batal</button>
                <button class="btn btn-primary btn-sm" @click="saveContent" :disabled="saving">
                  {{ saving ? 'Menyimpan...' : '💾 Simpan' }}
                </button>
              </div>
            </div>
            <textarea
              v-model="editorContent"
              class="json-editor"
              rows="20"
              spellcheck="false"
              :disabled="saving"
            ></textarea>
            <p v-if="editorError" class="error-text">{{ editorError }}</p>
            <p v-if="saveSuccess" class="success-text">{{ saveSuccess }}</p>

            <!-- Live preview hint -->
            <div class="preview-hint">
              <a :href="`/landing#${activeSection === 'trust' || activeSection === 'cta' ? '' : activeSection}`" target="_blank" class="preview-link">
                👁 Lihat preview di landing page →
              </a>
              <span class="cache-note">Cache: 6 jam (auto-invalidated setelah save)</span>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { superadminApi } from '../superadminApi'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits(['close', 'saved'])

const configs = ref<any[]>([])
const loading = ref(false)
const activeSection = ref('')
const editorContent = ref('')
const editorError = ref('')
const saveSuccess = ref('')
const saving = ref(false)

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
    const res = await superadminApi.getLandingConfigs()
    if (res?.data && Array.isArray(res.data)) {
      configs.value = res.data.sort((a: any, b: any) => a.id.localeCompare(b.id))
    }
  } catch (e) {
    console.error('Failed to load landing configs', e)
  } finally {
    loading.value = false
  }
}

const selectSection = (id: string) => {
  activeSection.value = id
  const cfg = configs.value.find((c: any) => c.id === id)
  editorContent.value = cfg ? JSON.stringify(cfg.content, null, 2) : ''
  editorError.value = ''
  saveSuccess.value = ''
}

const formatJSON = () => {
  try {
    const parsed = JSON.parse(editorContent.value)
    editorContent.value = JSON.stringify(parsed, null, 2)
    editorError.value = ''
  } catch (e: any) {
    editorError.value = 'JSON tidak valid: ' + e.message
  }
}

const saveContent = async () => {
  editorError.value = ''
  saveSuccess.value = ''

  let content: any
  try {
    content = JSON.parse(editorContent.value)
  } catch (e: any) {
    editorError.value = 'JSON tidak valid: ' + e.message
    return
  }

  saving.value = true
  try {
    const res = await superadminApi.updateLandingConfig(activeSection.value, { content })
    if (res?.status === 200) {
      saveSuccess.value = '✅ Tersimpan! Perubahan langsung tampil di landing page.'
    } else {
      editorError.value = res?.message || 'Gagal menyimpan'
    }
  } catch (e: any) {
    editorError.value = 'Network error: ' + e.message
  } finally {
    saving.value = false
  }
}

watch(() => props.show, (val) => {
  if (val) {
    loadConfigs()
    activeSection.value = ''
    editorContent.value = ''
    editorError.value = ''
    saveSuccess.value = ''
  }
})
</script>

<style scoped>
.landing-editor-modal {
  max-width: 720px;
  width: 95%;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
}

.section-tabs {
  display: flex;
  gap: 0.4rem;
  flex-wrap: wrap;
  margin-bottom: 1rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-color, #e5e7eb);
}

.section-tab {
  padding: 0.35rem 0.75rem;
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 100px;
  background: none;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-muted, #6b7280);
  cursor: pointer;
  transition: all 0.15s;
}

.section-tab:hover { border-color: var(--accent-primary, #f59e0b); color: var(--accent-primary, #f59e0b); }
.section-tab.active { background: var(--accent-primary, #f59e0b); border-color: var(--accent-primary, #f59e0b); color: #fff; }

.editor-area { display: flex; flex-direction: column; gap: 0.5rem; flex: 1; min-height: 0; }

.editor-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.editor-label { font-weight: 700; font-size: 0.95rem; }

.editor-actions { display: flex; gap: 0.5rem; }

.btn-sm { padding: 0.3rem 0.75rem; font-size: 0.8rem; }
.btn-cancel { background: transparent; border: 1px solid var(--border-color, #e5e7eb); color: var(--text-muted, #6b7280); }
.btn-cancel:hover { border-color: #ef4444; color: #ef4444; }

.json-editor {
  flex: 1;
  min-height: 320px;
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 0.8rem;
  line-height: 1.5;
  padding: 0.75rem;
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 8px;
  background: var(--surface-1, #f8fafc);
  color: var(--text-primary, #1e293b);
  resize: vertical;
  tab-size: 2;
}

.json-editor:focus { outline: none; border-color: var(--accent-primary, #f59e0b); box-shadow: 0 0 0 2px rgba(245, 158, 11, 0.15); }

.error-text { color: #ef4444; font-size: 0.8rem; margin: 0; }
.success-text { color: #22c55e; font-size: 0.8rem; margin: 0; }

.preview-hint {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 0.5rem;
  border-top: 1px solid var(--border-color, #e5e7eb);
  font-size: 0.78rem;
}

.preview-link { color: var(--accent-primary, #f59e0b); text-decoration: none; font-weight: 600; }
.preview-link:hover { text-decoration: underline; }

.cache-note { color: var(--text-muted, #94a3b8); }
</style>

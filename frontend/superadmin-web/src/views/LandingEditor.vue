<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { request } from '../api/client'

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref('')

const sections = ['hero', 'features', 'steps', 'testimonials', 'cta', 'footer']

const configs = ref<Record<string, any>>({})
const activeSection = ref('hero')

onMounted(async () => {
  loading.value = true
  try {
    const data = await request('/landing-configs')
    if (data.success && data.data) {
      configs.value = data.data
    }
  } catch {
    error.value = 'Gagal memuat konfigurasi'
  } finally {
    loading.value = false
  }
})

function getSectionData(id: string): any {
  return configs.value[id] || {}
}

function updateSection(id: string, key: string, value: any) {
  if (!configs.value[id]) configs.value[id] = {}
  configs.value[id][key] = value
}

async function saveSection(id: string) {
  saving.value = true
  error.value = ''
  success.value = ''
  try {
    const data = await request(`/admin/landing-configs/?id=${id}`, {
      method: 'PUT',
      body: JSON.stringify({ content: configs.value[id] || {} }),
    })
    if (data.success) {
      success.value = 'Berhasil disimpan!'
      setTimeout(() => { success.value = '' }, 3000)
    } else {
      error.value = data.message || 'Gagal menyimpan'
    }
  } catch {
    error.value = 'Kesalahan jaringan'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1>Landing Page Editor</h1>
        <p class="subtitle">Edit konten halaman landing secara real-time</p>
      </div>
    </div>

    <div v-if="loading" class="loading">Memuat...</div>
    <div v-else>
      <!-- Section tabs -->
      <div class="section-tabs">
        <button
          v-for="s in sections"
          :key="s"
          :class="['tab', { active: activeSection === s }]"
          @click="activeSection = s"
        >{{ s.charAt(0).toUpperCase() + s.slice(1) }}</button>
      </div>

      <!-- Section editor -->
      <div class="editor-card" v-if="configs[activeSection] !== undefined || activeSection">
        <div v-if="activeSection === 'hero'" class="section-fields">
          <h3>Hero Section</h3>
          <label>Judul Utama <input class="form-control" :value="getSectionData('hero').title || ''" @input="updateSection('hero', 'title', ($event.target as HTMLInputElement).value)" /></label>
          <label>Subtitle <textarea class="form-control" :value="getSectionData('hero').subtitle || ''" @input="updateSection('hero', 'subtitle', ($event.target as HTMLTextAreaElement).value)" rows="2" /></label>
          <label>CTA Text <input class="form-control" :value="getSectionData('hero').cta_text || ''" @input="updateSection('hero', 'cta_text', ($event.target as HTMLInputElement).value)" /></label>
          <label>CTA Link <input class="form-control" :value="getSectionData('hero').cta_link || ''" @input="updateSection('hero', 'cta_link', ($event.target as HTMLInputElement).value)" /></label>
        </div>

        <div v-else-if="activeSection === 'features'" class="section-fields">
          <h3>Features</h3>
          <p class="hint">Edit array features di JSON:</p>
          <textarea
            class="form-control code"
            rows="12"
            :value="JSON.stringify(getSectionData('features') || [], null, 2)"
            @change="updateSection('features', '', JSON.parse(($event.target as HTMLTextAreaElement).value))"
          />
        </div>

        <div v-else-if="activeSection === 'steps'" class="section-fields">
          <h3>Steps</h3>
          <p class="hint">Edit array steps:</p>
          <textarea
            class="form-control code"
            rows="10"
            :value="JSON.stringify(getSectionData('steps') || [], null, 2)"
            @change="updateSection('steps', '', JSON.parse(($event.target as HTMLTextAreaElement).value))"
          />
        </div>

        <div v-else-if="activeSection === 'cta'" class="section-fields">
          <h3>CTA Section</h3>
          <label>Judul <input class="form-control" :value="getSectionData('cta').title || ''" @input="updateSection('cta', 'title', ($event.target as HTMLInputElement).value)" /></label>
          <label>Deskripsi <textarea class="form-control" :value="getSectionData('cta').description || ''" @input="updateSection('cta', 'description', ($event.target as HTMLTextAreaElement).value)" rows="2"></textarea></label>
          <label>Button Text <input class="form-control" :value="getSectionData('cta').button_text || ''" @input="updateSection('cta', 'button_text', ($event.target as HTMLInputElement).value)" /></label>
        </div>

        <div v-else-if="activeSection === 'footer'" class="section-fields">
          <h3>Footer</h3>
          <label>Copyright Text <input class="form-control" :value="getSectionData('footer').copyright || ''" @input="updateSection('footer', 'copyright', ($event.target as HTMLInputElement).value)" /></label>
          <label>Links (JSON) <textarea class="form-control code" rows="4" :value="JSON.stringify(getSectionData('footer').links || [], null, 2)" @change="updateSection('footer', 'links', JSON.parse(($event.target as HTMLTextAreaElement).value))" /></label>
        </div>

        <div v-else class="section-fields">
          <h3>{{ activeSection }}</h3>
          <textarea
            class="form-control code"
            rows="8"
            :value="JSON.stringify(getSectionData(activeSection) || {}, null, 2)"
            @change="updateSection(activeSection, '', JSON.parse(($event.target as HTMLTextAreaElement).value))"
          />
        </div>

        <p v-if="error" class="error">{{ error }}</p>
        <p v-if="success" class="success">{{ success }}</p>
        <button class="btn btn-accent" @click="saveSection(activeSection)" :disabled="saving">
          {{ saving ? 'Menyimpan...' : '💾 Simpan Perubahan' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page-header { margin-bottom: 24px; }
.page-header h1 { font-size: 22px; margin-bottom: 2px; }
.subtitle { color: var(--muted); font-size: 13px; }
.loading { text-align: center; color: var(--muted); padding: 60px; }

.section-tabs { display: flex; gap: 6px; margin-bottom: 20px; flex-wrap: wrap; }
.tab { background: var(--card); border: 1px solid var(--border); color: var(--muted); padding: 7px 16px; border-radius: 6px; font-size: 13px; cursor: pointer; }
.tab.active { background: var(--accent); border-color: var(--accent); color: white; }

.editor-card { background: var(--card); border: 1px solid var(--border); border-radius: 10px; padding: 24px; }
.section-fields { display: flex; flex-direction: column; gap: 14px; }
.section-fields h3 { margin: 0; font-size: 16px; color: var(--text); }
.hint { font-size: 12px; color: var(--muted); margin: 0; }
.form-control { background: var(--bg); border: 1px solid var(--border); color: var(--text); padding: 8px 12px; border-radius: 6px; font-size: 14px; width: 100%; box-sizing: border-box; font-family: inherit; }
.form-control.code { font-family: monospace; font-size: 12px; }
textarea.form-control { resize: vertical; }

.error { color: var(--danger); font-size: 13px; }
.success { color: var(--success); font-size: 13px; }
.btn { padding: 9px 18px; border-radius: 6px; font-size: 13px; cursor: pointer; border: none; }
.btn-accent { background: var(--accent); color: white; }
</style>

<template>
  <div class="chatbot-config-page">
    <div class="header-actions" style="margin-bottom: 1.5rem;">
      <h2>🤖 Setup AI Customer Service</h2>
      <p>Atur kepribadian, bahasa, jam operasional, dan kapan bot eskalasi ke admin.</p>
    </div>

    <!-- First-run banner -->
    <div v-if="isFirstRun" class="glass-card first-run-banner"
      style="margin-bottom: 1.5rem; padding: 1.25rem; border-left: 4px solid #4f46e5;">
      <strong>🎉 Selamat!</strong> Langganan Anda aktif. Lengkapi 3 langkah mudah ini agar AI CS toko Anda bisa langsung
      melayani pelanggan.
    </div>

    <!-- Stepper -->
    <div class="stepper" style="display: flex; gap: 0.5rem; margin-bottom: 2rem;">
      <div v-for="(label, i) in steps" :key="i" class="step-pill"
        :class="{ active: currentStep === i, done: currentStep > i }" @click="goToStep(i)">
        <span class="step-num">{{ currentStep > i ? '✓' : i + 1 }}</span>
        <span class="step-label">{{ label }}</span>
      </div>
    </div>

    <div v-if="loading" class="glass-card" style="padding: 2rem; text-align: center;">
      <p>Memuat konfigurasi...</p>
    </div>

    <div v-else class="config-layout" style="display: grid; grid-template-columns: 2fr 1fr; gap: 1.5rem;">
      <!-- FORM COLUMN -->
      <div class="glass-card" style="padding: 1.5rem;">
        <!-- STEP 1 — Identitas Bot -->
        <div v-show="currentStep === 0">
          <h3 style="margin-bottom: 1rem;">Step 1 — Identitas Bot</h3>
          <div style="display: flex; flex-direction: column; gap: 1rem;">
            <label>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Nama Bot (opsional)</span>
              <input v-model="form.bot_name" type="text" class="form-control" placeholder="Contoh: CS Toko Barokah" />
            </label>
            <div>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.5rem;">Bahasa</span>
              <div style="display: flex; gap: 1rem;">
                <label class="radio-pill">
                  <input type="radio" value="id" v-model="form.language" /> 🇮🇩 Indonesia
                </label>
                <label class="radio-pill">
                  <input type="radio" value="en" v-model="form.language" /> 🇬🇧 English
                </label>
              </div>
            </div>
            <div>
              <label for="tone-select" style="display: block; font-size: 0.85rem; margin-bottom: 0.5rem;">Tone / Gaya Bicara</label>
              <select id="tone-select" v-model="form.tone" class="form-control">
                <option value="friendly">Ramah & Hangat</option>
                <option value="formal">Formal</option>
                <option value="casual">Santai & Akrab</option>
                <option value="professional">Profesional & Solutif</option>
              </select>
            </div>
            <label>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">System Prompt Custom (opsional,
                advanced)</span>
              <textarea v-model="form.system_prompt" class="form-control" rows="3"
                placeholder="Biarkan kosong untuk pakai default. Isi jika ingin instruksi spesifik, misal 'Kamu selalu jawab pakai emoji ✨'."></textarea>
            </label>
          </div>
        </div>

        <!-- STEP 2 — Jam Operasional & Escalation -->
        <div v-show="currentStep === 1">
          <h3 style="margin-bottom: 1rem;">Step 2 — Jam Operasional & Auto-Eskalasi</h3>
          <div style="display: flex; flex-direction: column; gap: 1rem;">
            <div style="display: flex; gap: 1rem;">
              <label style="flex: 1;">
                <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Jam Buka</span>
                <input v-model="form.business_hours_start" type="time" class="form-control" />
              </label>
              <label style="flex: 1;">
                <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Jam Tutup</span>
                <input v-model="form.business_hours_end" type="time" class="form-control" />
              </label>
            </div>
            <div>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.5rem;">Hari Operasional</span>
              <div style="display: flex; gap: 0.5rem; flex-wrap: wrap;">
                <label v-for="d in dayList" :key="d.value" class="day-pill"
                  :class="{ active: form.business_days.includes(d.value) }">
                  <input type="checkbox" :value="d.value" v-model="form.business_days" hidden />
                  {{ d.short }}
                </label>
              </div>
            </div>
            <hr style="border: 0; border-top: 1px solid var(--border-color); margin: 0.5rem 0;" />
            <label class="toggle-row">
              <input type="checkbox" v-model="form.escalation_enabled" />
              <span>Aktifkan auto-eskalasi ke admin</span>
            </label>
            <div v-if="form.escalation_enabled">
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Kata Kunci Eskalasi (pisahkan
                dengan Enter)</span>
              <div class="keyword-input">
                <span v-for="(kw, i) in form.escalation_keywords" :key="i" class="kw-tag">
                  {{ kw }}
                  <button type="button" @click="form.escalation_keywords.splice(i, 1)">×</button>
                </span>
                <input id="keyword-input" type="text" v-model="newKeyword" @keydown.enter.prevent="addKeyword"
                  @keydown.,.prevent="addKeyword" placeholder="Tekan Enter untuk tambah"
                  class="form-control kw-input" aria-label="Kata kunci baru" />
              </div>
            </div>
            <label>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Auto-eskalasi setelah berapa
                menit tanpa jawaban?</span>
              <input v-model.number="form.auto_escalate_after_minutes" type="number" min="0" max="60"
                class="form-control" style="max-width: 120px;" />
            </label>
          </div>
        </div>

        <!-- STEP 3 — Kalimat & Channel -->
        <div v-show="currentStep === 2">
          <h3 style="margin-bottom: 1rem;">Step 3 — Kalimat & Channel</h3>
          <div style="display: flex; flex-direction: column; gap: 1rem;">
            <label>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Pesan Sambutan (welcome)</span>
              <textarea v-model="form.welcome_message" class="form-control" rows="2" />
            </label>
            <label>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Pesan Fallback (kalau bot
                bingung)</span>
              <textarea v-model="form.fallback_message" class="form-control" rows="2" />
            </label>
            <label>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Pesan di Luar Jam
                Operasional</span>
              <textarea v-model="form.outside_hours_message" class="form-control" rows="2" />
            </label>
            <hr style="border: 0; border-top: 1px solid var(--border-color); margin: 0.5rem 0;" />
            <div>
              <span style="display: block; font-weight: 600; font-size: 0.9rem; margin-bottom: 0.5rem;">AI
                Modality</span>
              <label class="toggle-row" style="margin-bottom: 0.25rem;">
                <input type="checkbox" v-model="form.enable_vision" />
                <span><strong>Enable Vision</strong> (Process image messages)</span>
              </label>
              <label class="toggle-row" style="margin-bottom: 0.25rem;">
                <input type="checkbox" v-model="form.enable_voice_reply" />
                <span><strong>Enable Voice Reply</strong> (Reply with voice notes)</span>
              </label>
              <label v-if="form.enable_voice_reply" style="display: block; margin-top: 0.5rem;">
                <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Voice Model</span>
                <select v-model="form.voice_model" class="form-control">
                  <option value="id-ID-ArdiNeural">id-ID-ArdiNeural (Laki-laki)</option>
                  <option value="id-ID-GadisNeural">id-ID-GadisNeural (Perempuan)</option>
                </select>
              </label>
            </div>
            <hr style="border: 0; border-top: 1px solid var(--border-color); margin: 0.5rem 0;" />

            <div>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.5rem;">Channel Aktif</span>
              <div style="display: flex; gap: 1rem; flex-wrap: wrap;">
                <label class="channel-pill"
                  :class="{ active: form.channels_enabled.includes('whatsapp'), locked: true }">
                  <input type="checkbox" value="whatsapp" v-model="form.channels_enabled" disabled /> 📱 WhatsApp
                </label>
                <label class="channel-pill" :class="{ active: form.channels_enabled.includes('telegram') }">
                  <input type="checkbox" value="telegram" v-model="form.channels_enabled" /> ✈️ Telegram
                </label>
                <label class="channel-pill" :class="{ active: form.channels_enabled.includes('webchat') }">
                  <input type="checkbox" value="webchat" v-model="form.channels_enabled" /> 💬 Web Chat
                </label>
              </div>
              <p style="font-size: 0.75rem; color: var(--text-secondary); margin-top: 0.5rem;">
                WhatsApp selalu aktif. Channel lain aktif jika Anda centang.
              </p>
            </div>
            <hr style="border: 0; border-top: 1px solid var(--border-color); margin: 0.5rem 0;" />

            <div>
              <label for="wa-provider-select" style="display: block; font-weight: 600; font-size: 0.9rem; margin-bottom: 0.5rem;">📡 WA
                Provider</label>
              <select id="wa-provider-select" v-model="form.wa_provider_preference" class="form-control">
                <option value="auto">⚡ Auto (Rekomendasi)</option>
                <option value="whatsmeow">📱 Whatsmeow Only</option>
                <option value="cloud_api" :disabled="!hasWaPremium">☁️ Cloud API (Meta) — butuh add-on</option>
              </select>
              <p style="font-size: 0.75rem; color: var(--text-secondary); margin-top: 0.25rem;">
                <template v-if="form.wa_provider_preference === 'auto'">Sistem otomatis pilih provider terbaik untuk
                  setiap pesan.</template>
                <template v-else-if="form.wa_provider_preference === 'whatsmeow'">Semua pesan dipaksa lewat whatsmeow.
                  Cloud API tidak dipakai.</template>
                <template v-else-if="form.wa_provider_preference === 'cloud_api'">Semua pesan dipaksa lewat Cloud API
                  Meta. Jika gagal, pesan tidak akan terkirim.</template>
              </p>
            </div>

            <hr style="border: 0; border-top: 1px solid var(--border-color); margin: 0.5rem 0;" />
            <label class="toggle-row">
              <input type="checkbox" v-model="form.is_active" />
              <span><strong>Aktifkan AI CS</strong> (jika nonaktif, customer akan dapat pesan di luar jam)</span>
            </label>
            <button class="btn btn-secondary" @click="openTestModal" :disabled="testing || !form.is_active">
              {{ testing ? 'Mengirim...' : '🧪 Test Bot' }}
            </button>
          </div>
        </div>

        <!-- Navigation buttons -->
        <div class="nav-row" style="display: flex; justify-content: space-between; margin-top: 1.5rem;">
          <button class="btn btn-secondary" @click="prev" :disabled="currentStep === 0">← Kembali</button>
          <div>
            <button v-if="currentStep < steps.length - 1" class="btn btn-primary" @click="next">Lanjut →</button>
            <button v-else class="btn btn-primary" @click="save" :disabled="saving">
              {{ saving ? 'Menyimpan...' : 'Simpan & Aktifkan' }}
            </button>
          </div>
        </div>
        <p v-if="errorMsg" style="color: #dc2626; margin-top: 0.5rem;">{{ errorMsg }}</p>
      </div>

      <!-- PREVIEW COLUMN -->
      <div class="glass-card preview-card" style="padding: 1.25rem;">
        <h4 style="margin-bottom: 0.75rem;">Preview</h4>
        <div class="preview-row"><span>Bot</span><b>{{ form.bot_name || 'CS Toko Anda' }}</b></div>
        <div class="preview-row"><span>Bahasa</span><b>{{ form.language === 'en' ? '🇬🇧 English' : '🇮🇩 Indonesia'
            }}</b>
        </div>
        <div class="preview-row"><span>Tone</span><b>{{ toneLabel }}</b></div>
        <div class="preview-row"><span>Aktif</span><b>{{ form.is_active ? '✅ Ya' : '❌ Tidak' }}</b></div>
        <hr style="border: 0; border-top: 1px solid var(--border-color); margin: 0.75rem 0;" />
        <div class="preview-row"><span>Jam</span><b>{{ form.business_hours_start }}–{{ form.business_hours_end }}</b>
        </div>
        <div class="preview-row"><span>Hari</span><b>{{ dayShortList || 'Setiap hari' }}</b></div>
        <div class="preview-row"><span>Escalation</span><b>{{ form.escalation_enabled ? '✅ On' : '❌ Off' }}</b></div>
        <div class="preview-row"><span>Channel</span><b>{{ form.channels_enabled.join(', ') || '-' }}</b></div>
      </div>
    </div>

    <!-- Test modal -->
    <div v-if="testOpen" class="modal-backdrop" @click.self="testOpen = false">
      <div class="modal-content" style="max-width: 480px; padding: 1.5rem;">
        <h3 style="margin-bottom: 0.75rem;">🧪 Test Bot</h3>
        <p style="font-size: 0.85rem; color: var(--text-secondary);">Coba kirim pesan ke bot untuk lihat bagaimana dia
          akan
          menjawab dengan konfigurasi saat ini.</p>
        <input id="test-input" v-model="testInput" type="text" class="form-control" placeholder='Misal: "Halo, ada diskon?"'
          @keydown.enter="runTest" aria-label="Pesan test" style="margin-top: 0.5rem;" />
        <div v-if="testReply" class="test-reply"
          style="margin-top: 1rem; padding: 0.75rem; background: var(--bg-tertiary); border-radius: 0.5rem;">
          <p style="margin: 0 0 0.5rem; font-size: 0.9rem;">{{ testReply }}</p>
          <p v-if="testWouldEscalate" style="margin: 0; font-size: 0.75rem; color: #f59e0b;">⚠️ Pesan ini akan
            di-eskalasi
            ke admin (kata kunci cocok).</p>
        </div>
        <p v-if="testError" style="color: #dc2626; font-size: 0.85rem; margin-top: 0.5rem;">{{ testError }}</p>
        <div style="display: flex; justify-content: flex-end; gap: 0.5rem; margin-top: 1rem;">
          <button class="btn btn-secondary" @click="testOpen = false">Tutup</button>
          <button class="btn btn-primary" @click="runTest" :disabled="testing || !testInput.trim()">{{ testing ?
            'Mengirim...' : 'Kirim' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'
import { useModalState } from '../utils/modalState'

const { openModal, closeModal } = useModalState()

const route = useRoute()
const isFirstRun = computed(() => route.query.first_run === '1')

const steps = ['Identitas', 'Jam & Eskalasi', 'Kalimat & Channel']
const currentStep = ref(0)
const loading = ref(true)
const saving = ref(false)
const testing = ref(false)
const errorMsg = ref('')
const hasWaPremium = ref(false) // F048: fetched from /chatbot/permissions


async function loadPermissions() {
  try {
    const res = await api.getChatbotPermissions()
    if (res && res.success && res.data) {
      hasWaPremium.value = !!res.data.has_wa_cloud_api
    }
  } catch (e: any) {
    console.warn('Failed to fetch chatbot permissions', e)
    hasWaPremium.value = false
  }
}

const testOpen = ref(false)
watch(testOpen, (v) => { if (v) openModal(); else closeModal(); })
const testInput = ref('')
const testReply = ref('')
const testWouldEscalate = ref(false)
const testError = ref('')
const newKeyword = ref('')

const form = reactive<any>({
  bot_name: '',
  language: 'id',
  tone: 'friendly',
  system_prompt: '',
  business_hours_start: '08:00',
  business_hours_end: '22:00',
  business_days: [1, 2, 3, 4, 5, 6],
  escalation_enabled: true,
  escalation_keywords: ['bicara cs', 'hubungi admin', 'operator', 'manusia', 'human'],
  auto_escalate_after_minutes: 5,
  welcome_message: 'Halo! Ada yang bisa saya bantu?',
  fallback_message: 'Maaf, saya belum bisa menjawab pertanyaan tersebut. Apakah Anda ingin dihubungkan dengan CS kami?',
  outside_hours_message: 'Terima kasih telah menghubungi kami. Saat ini di luar jam operasional. Pesan Anda akan dibalas saat jam kerja.',
  channels_enabled: ['whatsapp'],
  wa_provider_preference: 'auto',
  is_active: true,
  enable_vision: false,
  enable_voice_reply: false,
  voice_model: 'id-ID-GadisNeural',
})

const dayList = [
  { value: 0, short: 'Min' },
  { value: 1, short: 'Sen' },
  { value: 2, short: 'Sel' },
  { value: 3, short: 'Rab' },
  { value: 4, short: 'Kam' },
  { value: 5, short: 'Jum' },
  { value: 6, short: 'Sab' },
]

const toneLabel = computed(() => {
  const map: Record<string, string> = {
    friendly: 'Ramah & Hangat',
    formal: 'Formal',
    casual: 'Santai & Akrab',
    professional: 'Profesional & Solutif',
  }
  return map[form.tone] || form.tone
})

const dayShortList = computed(() => {
  return dayList
    .filter((d) => form.business_days.includes(d.value))
    .map((d) => d.short)
    .join(', ')
})

async function loadConfig() {
  loading.value = true
  try {
    const res = await api.getChatbotConfig()
    if (res.success && res.data) applyConfig(res.data)
  } catch {
    errorMsg.value = 'Gagal memuat konfigurasi'
  } finally {
    loading.value = false
  }
}

function applyConfig(d: any) {
  const fields = ['language','tone','system_prompt','welcome_message','fallback_message',
    'outside_hours_message','business_hours_start','business_hours_end',
    'business_days','escalation_keywords','auto_escalate_after_minutes',
    'channels_enabled','wa_provider_preference','bot_name']
  fields.forEach(k => { if (d[k]) (form as any)[k] = d[k] })
  form.escalation_enabled = !!d.escalation_enabled
  form.is_active = d.is_active !== false
}

function goToStep(i: number) {
  if (i <= currentStep.value) currentStep.value = i
}

function next() {
  if (currentStep.value < steps.length - 1) {
    currentStep.value++
    saveDraft()
  }
}
function prev() {
  if (currentStep.value > 0) currentStep.value--
}

function addKeyword() {
  const k = newKeyword.value.trim()
  if (k && !form.escalation_keywords.includes(k)) {
    form.escalation_keywords.push(k)
  }
  newKeyword.value = ''
}

function saveDraft() {
  try {
    sessionStorage.setItem('chatbot_config_draft', JSON.stringify(form))
  } catch {
    // sessionStorage not available — silently ignore
  }
}

function loadDraft() {
  try {
    const raw = sessionStorage.getItem('chatbot_config_draft')
    if (raw) Object.assign(form, JSON.parse(raw))
  } catch {
    // sessionStorage not available — silently ignore
  }
}

async function save() {
  errorMsg.value = ''
  // Frontend validation for nicer UX (backend also validates)
  if (form.business_hours_start >= form.business_hours_end) {
    errorMsg.value = 'Jam buka harus lebih awal dari jam tutup.'
    return
  }
  if (form.escalation_enabled && form.escalation_keywords.length === 0) {
    errorMsg.value = 'Minimal 1 kata kunci eskalasi jika escalation aktif.'
    return
  }
  if (form.channels_enabled.length === 0) {
    errorMsg.value = 'Minimal 1 channel harus aktif.'
    return
  }
  saving.value = true
  try {
    const payload = {
      language: form.language,
      tone: form.tone,
      system_prompt: form.system_prompt,
      welcome_message: form.welcome_message,
      fallback_message: form.fallback_message,
      outside_hours_message: form.outside_hours_message,
      business_hours_start: form.business_hours_start,
      business_hours_end: form.business_hours_end,
      business_days: form.business_days,
      escalation_enabled: form.escalation_enabled,
      escalation_keywords: form.escalation_keywords,
      auto_escalate_after_minutes: form.auto_escalate_after_minutes,
      channels_enabled: form.channels_enabled,
      is_active: form.is_active,
    }
    const res = await api.updateChatbotConfig(payload)
    if (res.success) {
      sessionStorage.removeItem('chatbot_config_draft')
      // Toast sederhana
      const toast = document.createElement('div')
      toast.textContent = '✅ Konfigurasi tersimpan & AI CS aktif'
      toast.style.cssText = 'position:fixed;top:16px;right:16px;background:#10b981;color:white;padding:12px 20px;border-radius:8px;z-index:2147483647;box-shadow:0 4px 12px rgba(0,0,0,.15);'
      document.body.appendChild(toast)
      setTimeout(() => toast.remove(), 2500)
    } else {
      errorMsg.value = res.message || 'Gagal menyimpan'
    }
  } catch (e: any) {
    errorMsg.value = 'Error: ' + (e?.message || e)
  } finally {
    saving.value = false
  }
}

function openTestModal() {
  testOpen.value = true
  testReply.value = ''
  testError.value = ''
  testInput.value = ''
}

async function runTest() {
  if (!testInput.value.trim()) return
  testing.value = true
  testError.value = ''
  try {
    const res = await api.testChatbotConfig(testInput.value)
    if (res.success && res.data) {
      testReply.value = res.data.reply
      testWouldEscalate.value = !!res.data.would_escalate
    } else {
      testError.value = res.message || 'Gagal menjalankan test'
    }
  } catch (e: any) {
    testError.value = 'Error: ' + (e?.message || e)
  } finally {
    testing.value = false
  }
}

onMounted(() => {
  loadDraft()
  loadConfig()
  loadPermissions()
})
</script>

<style scoped>
.step-pill {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border-radius: 999px;
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s;
}

.step-pill.active {
  background: #4f46e5;
  color: white;
}

.step-pill.done {
  background: #047857;
  color: #f0fdf4;
}

.step-num {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.4);
  font-size: 0.75rem;
  font-weight: 700;
}

.radio-pill,
.day-pill,
.channel-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.5rem 0.9rem;
  border-radius: 0.5rem;
  border: 1px solid var(--border-color);
  background: var(--bg-tertiary);
  cursor: pointer;
  font-size: 0.85rem;
  user-select: none;
  transition: all 0.15s;
}

.radio-pill input,
.channel-pill input {
  margin: 0;
}

.day-pill.active,
.channel-pill.active {
  background: #4f46e5;
  color: white;
  border-color: #4f46e5;
}

.channel-pill.locked {
  opacity: 0.6;
  cursor: not-allowed;
}

.toggle-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.9rem;
}

.preview-row {
  display: flex;
  justify-content: space-between;
  padding: 0.35rem 0;
  font-size: 0.85rem;
  border-bottom: 1px dashed var(--border-color);
}

.preview-row:last-child {
  border-bottom: 0;
}

.preview-row span {
  color: var(--text-secondary);
}

.keyword-input {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  align-items: center;
  padding: 0.4rem;
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  background: var(--bg-primary);
}

.kw-tag {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  padding: 0.2rem 0.6rem;
  background: #4f46e5;
  color: white;
  border-radius: 999px;
  font-size: 0.8rem;
}

.kw-tag button {
  background: rgba(0, 0, 0, 0.4);
  border: 0;
  color: white;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  cursor: pointer;
  font-size: 0.9rem;
  line-height: 1;
}

.kw-input {
  flex: 1;
  min-width: 120px;
  border: 0 !important;
  background: transparent !important;
  padding: 0.3rem !important;
}

.modal-content {
  width: 90%;
  max-width: 480px;
}

@media (max-width: 768px) {
  .config-layout {
    grid-template-columns: 1fr !important;
  }

  .stepper {
    flex-direction: column;
  }
}
</style>

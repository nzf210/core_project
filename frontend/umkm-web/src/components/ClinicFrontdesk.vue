<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'

type TabKey = 'queue' | 'records' | 'doctors'
const route = useRoute()
const activeTab = ref<TabKey>((route.query.tab as TabKey) || 'queue')

// ===== Antrean (existing F045) =====
const queueItems = ref<any[]>([])
const settings = ref({
  queue_type: 'sequential',
  slot_duration_minutes: 30,
  is_active: true
})
const loadingQueue = ref(true)

const fetchQueue = async () => {
  try {
    const res = await api.getClinicQueue()
    if (res && res.success) queueItems.value = res.data || []
  } catch (err) {
    console.error('Failed to fetch queue', err)
  } finally {
    loadingQueue.value = false
  }
}

const fetchSettings = async () => {
  try {
    const res = await api.getClinicSettings()
    if (res && res.success) settings.value = res.data
  } catch (err) {
    console.error('Failed to fetch settings', err)
  }
}

const saveSettings = async () => {
  try {
    const res = await api.updateClinicSettings(settings.value)
    if (res && res.success) alert('Pengaturan tersimpan!')
  } catch (err) {
    alert('Gagal menyimpan pengaturan')
  }
}

const callNext = async (id: string) => {
  try {
    const res = await api.callClinicAppointment(id)
    if (res && res.success) await fetchQueue()
  } catch (err) {
    console.error('Failed to call patient', err)
  }
}

const cancelAppointment = async (id: string) => {
  try {
    const res = await api.cancelClinicAppointment(id, 'frontdesk')
    if (res && res.success) await fetchQueue()
  } catch (err) {
    console.error('Failed to cancel appointment', err)
  }
}

// ===== Rekam Medis (F047) =====
const medicalRecords = ref<any[]>([])
const loadingRecords = ref(false)
const showRecordForm = ref(false)
const recordForm = ref({
  patient_name: '',
  patient_phone: '',
  complaint: '',
  diagnosis: '',
  prescription: '',
  notes: '',
})

const fetchMedicalRecords = async () => {
  loadingRecords.value = true
  try {
    const res = await api.getClinicMedicalRecords()
    if (res && res.success) {
      medicalRecords.value = res.data || res.records || res || []
    }
  } catch (err) {
    console.error('Failed to fetch medical records', err)
  } finally {
    loadingRecords.value = false
  }
}

const saveMedicalRecord = async () => {
  if (!recordForm.value.patient_name.trim()) {
    alert('Nama pasien wajib diisi')
    return
  }
  try {
    const res = await api.createClinicMedicalRecord(recordForm.value)
    if (res && res.success) {
      recordForm.value = { patient_name: '', patient_phone: '', complaint: '', diagnosis: '', prescription: '', notes: '' }
      showRecordForm.value = false
      await fetchMedicalRecords()
    } else {
      alert(res?.message || 'Gagal menyimpan rekam medis')
    }
  } catch (err) {
    alert('Kesalahan jaringan saat menyimpan')
  }
}

// ===== Jadwal Dokter (F047) =====
const doctors = ref<any[]>([])
const loadingDoctors = ref(false)
const showDoctorForm = ref(false)
const doctorForm = ref({
  doctor_name: '',
  specialization: '',
  day_of_week: 'monday',
  time_start: '08:00',
  time_end: '12:00',
  max_patients: 20,
  is_active: true,
})

const days = [
  { value: 'monday', label: 'Senin' },
  { value: 'tuesday', label: 'Selasa' },
  { value: 'wednesday', label: 'Rabu' },
  { value: 'thursday', label: 'Kamis' },
  { value: 'friday', label: 'Jumat' },
  { value: 'saturday', label: 'Sabtu' },
  { value: 'sunday', label: 'Minggu' },
]

const fetchDoctors = async () => {
  loadingDoctors.value = true
  try {
    const res = await api.getClinicDoctors()
    if (res && res.success) {
      doctors.value = res.data || res.doctors || res || []
    }
  } catch (err) {
    console.error('Failed to fetch doctors', err)
  } finally {
    loadingDoctors.value = false
  }
}

const saveDoctor = async () => {
  if (!doctorForm.value.doctor_name.trim()) {
    alert('Nama dokter wajib diisi')
    return
  }
  if (doctorForm.value.time_start >= doctorForm.value.time_end) {
    alert('Jam mulai harus sebelum jam selesai')
    return
  }
  try {
    const res = await api.createClinicDoctor(doctorForm.value)
    if (res && res.success) {
      doctorForm.value = {
        doctor_name: '', specialization: '', day_of_week: 'monday',
        time_start: '08:00', time_end: '12:00', max_patients: 20, is_active: true,
      }
      showDoctorForm.value = false
      await fetchDoctors()
    } else {
      alert(res?.message || 'Gagal menyimpan jadwal dokter')
    }
  } catch (err) {
    alert('Kesalahan jaringan saat menyimpan')
  }
}

const switchTab = (tab: TabKey) => {
  activeTab.value = tab
  if (tab === 'records' && medicalRecords.value.length === 0) fetchMedicalRecords()
  if (tab === 'doctors' && doctors.value.length === 0) fetchDoctors()
}

const dayLabel = (day: string) => days.find(d => d.value === day)?.label || day

onMounted(() => {
  fetchQueue()
  fetchSettings()
})
</script>

<template>
  <div class="p-6" style="min-height: 100vh;">
    <!-- Header -->
    <div class="flex justify-between items-center mb-8">
      <div>
        <h1 class="text-3xl font-bold">Dashboard Klinik</h1>
        <p style="color: var(--text-secondary); font-size: 0.9rem; margin-top: 0.25rem;">
          Modul khusus tenant dengan jenis usaha: <strong>Klinik / Praktik Dokter</strong>
        </p>
      </div>
      <button v-if="activeTab === 'queue'" @click="saveSettings" class="btn btn-primary">
        💾 Simpan Pengaturan
      </button>
    </div>

    <!-- Tab Bar -->
    <div class="clinic-tabs">
      <button :class="['tab-btn', { active: activeTab === 'queue' }]" @click="switchTab('queue')">
        🏥 Antrean Pasien
      </button>
      <button :class="['tab-btn', { active: activeTab === 'records' }]" @click="switchTab('records')">
        📋 Rekam Medis
      </button>
      <button :class="['tab-btn', { active: activeTab === 'doctors' }]" @click="switchTab('doctors')">
        📅 Jadwal Dokter
      </button>
    </div>

    <!-- ===== Tab: Antrean ===== -->
    <div v-if="activeTab === 'queue'">
      <div class="glass-card animate-fade-in mb-8" style="padding: 1.5rem;">
        <h2 class="text-xl font-bold mb-4">⚙️ Pengaturan Antrean</h2>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div>
            <label class="block text-sm font-semibold mb-2">Jenis Antrean</label>
            <select v-model="settings.queue_type" class="form-control">
              <option value="sequential">🔢 Nomor Urut (A-001, A-002, ...)</option>
              <option value="timeslot">🕐 Slot Waktu (09:00, 10:30, ...)</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-semibold mb-2">Durasi Slot (menit)</label>
            <input v-model="settings.slot_duration_minutes" type="number" min="15" max="120" class="form-control" />
          </div>
          <div class="flex items-end">
            <label class="flex items-center gap-3 cursor-pointer">
              <input v-model="settings.is_active" type="checkbox" class="w-5 h-5" />
              <span class="font-semibold">Klinik Aktif</span>
            </label>
          </div>
        </div>
      </div>

      <div class="glass-card animate-fade-in" style="padding: 0; overflow: hidden;">
        <div style="padding: 1.25rem; border-bottom: 1px solid var(--glass-border);">
          <h2 class="text-xl font-bold">📋 Daftar Antrean ({{ queueItems.length }} orang)</h2>
        </div>
        <div v-if="loadingQueue" class="text-center" style="padding: 2.5rem;">⏳ Memuat...</div>
        <div v-else-if="queueItems.length === 0" class="text-center" style="padding: 2.5rem; opacity: 0.6;">
          ✨ Tidak ada antrean hari ini.
        </div>
        <table v-else style="width: 100%; text-align: left;">
          <thead style="background: rgba(255,255,255,0.05);">
            <tr>
              <th class="font-semibold" style="padding: 1rem;">No. Antrian</th>
              <th class="font-semibold" style="padding: 1rem;">Nama Pasien</th>
              <th class="font-semibold" style="padding: 1rem;">No. WA</th>
              <th class="font-semibold" style="padding: 1rem;">Status</th>
              <th class="font-semibold" style="padding: 1rem;">Aksi</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in queueItems" :key="item.id" style="border-top: 1px solid var(--glass-border);">
              <td style="padding: 1rem; font-weight: bold; font-size: 1.1rem;">{{ item.queue_number }}</td>
              <td style="padding: 1rem;">{{ item.patient_name }}</td>
              <td style="padding: 1rem; opacity: 0.7;">{{ item.patient_phone }}</td>
              <td style="padding: 1rem;">
                <span :class="item.status === 'waiting' ? 'badge-waiting' : 'badge-called'" class="status-badge">
                  {{ item.status === 'waiting' ? '🟡 Menunggu' : '🟢 Dipanggil' }}
                </span>
              </td>
              <td style="padding: 1rem;">
                <div class="flex gap-2">
                  <button v-if="item.status === 'waiting'" @click="callNext(item.id)"
                          class="btn btn-primary" style="font-size: 0.85rem; padding: 0.4rem 0.8rem;">
                    📞 Panggil
                  </button>
                  <button @click="cancelAppointment(item.id)"
                          class="btn btn-secondary" style="font-size: 0.85rem; padding: 0.4rem 0.8rem; color: #f87171; border-color: rgba(248,113,113,0.3);">
                    ❌ Batal
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- ===== Tab: Rekam Medis ===== -->
    <div v-if="activeTab === 'records'">
      <div class="flex justify-between items-center mb-4">
        <p style="color: var(--text-secondary);">Total: {{ medicalRecords.length }} rekam medis</p>
        <button @click="showRecordForm = !showRecordForm" class="btn btn-primary">
          {{ showRecordForm ? '✕ Tutup' : '➕ Tambah Rekam Medis' }}
        </button>
      </div>

      <div v-if="showRecordForm" class="glass-card animate-fade-in mb-6" style="padding: 1.5rem;">
        <h3 class="text-lg font-bold mb-4">📝 Rekam Medis Baru</h3>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-semibold mb-2">Nama Pasien *</label>
            <input v-model="recordForm.patient_name" type="text" class="form-control" placeholder="cth: Budi Santoso" />
          </div>
          <div>
            <label class="block text-sm font-semibold mb-2">No. HP / WA</label>
            <input v-model="recordForm.patient_phone" type="text" class="form-control" placeholder="cth: 081234567890" />
          </div>
          <div class="md:col-span-2">
            <label class="block text-sm font-semibold mb-2">Keluhan</label>
            <textarea v-model="recordForm.complaint" class="form-control" rows="2" placeholder="cth: Demam 3 hari, sakit kepala"></textarea>
          </div>
          <div class="md:col-span-2">
            <label class="block text-sm font-semibold mb-2">Diagnosis</label>
            <textarea v-model="recordForm.diagnosis" class="form-control" rows="2" placeholder="cth: ISPA ringan"></textarea>
          </div>
          <div class="md:col-span-2">
            <label class="block text-sm font-semibold mb-2">Resep / Tindakan</label>
            <textarea v-model="recordForm.prescription" class="form-control" rows="2" placeholder="cth: Paracetamol 3x1, istirahat"></textarea>
          </div>
          <div class="md:col-span-2">
            <label class="block text-sm font-semibold mb-2">Catatan</label>
            <textarea v-model="recordForm.notes" class="form-control" rows="2" placeholder="Follow-up, alergi, dll"></textarea>
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-4">
          <button @click="showRecordForm = false" class="btn btn-secondary">Batal</button>
          <button @click="saveMedicalRecord" class="btn btn-primary">💾 Simpan</button>
        </div>
      </div>

      <div class="glass-card animate-fade-in" style="padding: 0; overflow: hidden;">
        <div v-if="loadingRecords" class="text-center" style="padding: 2.5rem;">⏳ Memuat...</div>
        <div v-else-if="medicalRecords.length === 0" class="text-center" style="padding: 2.5rem; opacity: 0.6;">
          📭 Belum ada rekam medis. Klik "Tambah Rekam Medis" untuk mulai.
        </div>
        <div v-else style="padding: 1rem; display: grid; gap: 1rem;">
          <div v-for="rec in medicalRecords" :key="rec.id" style="border: 1px solid var(--glass-border); border-radius: 12px; padding: 1rem;">
            <div class="flex justify-between items-start">
              <div>
                <strong style="font-size: 1.05rem;">{{ rec.patient_name }}</strong>
                <span style="opacity: 0.6; margin-left: 0.5rem; font-size: 0.85rem;">{{ rec.patient_phone || '—' }}</span>
              </div>
              <span style="font-size: 0.8rem; opacity: 0.6;">{{ new Date(rec.created_at).toLocaleString('id-ID') }}</span>
            </div>
            <div v-if="rec.complaint" style="margin-top: 0.5rem;"><strong>Keluhan:</strong> {{ rec.complaint }}</div>
            <div v-if="rec.diagnosis" style="margin-top: 0.25rem;"><strong>Diagnosis:</strong> {{ rec.diagnosis }}</div>
            <div v-if="rec.prescription" style="margin-top: 0.25rem;"><strong>Resep:</strong> {{ rec.prescription }}</div>
            <div v-if="rec.notes" style="margin-top: 0.25rem; opacity: 0.8; font-style: italic;">{{ rec.notes }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== Tab: Jadwal Dokter ===== -->
    <div v-if="activeTab === 'doctors'">
      <div class="flex justify-between items-center mb-4">
        <p style="color: var(--text-secondary);">Total: {{ doctors.length }} jadwal aktif</p>
        <button @click="showDoctorForm = !showDoctorForm" class="btn btn-primary">
          {{ showDoctorForm ? '✕ Tutup' : '➕ Tambah Jadwal Dokter' }}
        </button>
      </div>

      <div v-if="showDoctorForm" class="glass-card animate-fade-in mb-6" style="padding: 1.5rem;">
        <h3 class="text-lg font-bold mb-4">👨‍⚕️ Jadwal Dokter Baru</h3>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-semibold mb-2">Nama Dokter *</label>
            <input v-model="doctorForm.doctor_name" type="text" class="form-control" placeholder="cth: dr. Anisa Putri" />
          </div>
          <div>
            <label class="block text-sm font-semibold mb-2">Spesialisasi</label>
            <input v-model="doctorForm.specialization" type="text" class="form-control" placeholder="cth: Umum, Gigi, Anak" />
          </div>
          <div>
            <label class="block text-sm font-semibold mb-2">Hari</label>
            <select v-model="doctorForm.day_of_week" class="form-control">
              <option v-for="d in days" :key="d.value" :value="d.value">{{ d.label }}</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-semibold mb-2">Max Pasien / Hari</label>
            <input v-model="doctorForm.max_patients" type="number" min="1" max="200" class="form-control" />
          </div>
          <div>
            <label class="block text-sm font-semibold mb-2">Jam Mulai</label>
            <input v-model="doctorForm.time_start" type="time" class="form-control" />
          </div>
          <div>
            <label class="block text-sm font-semibold mb-2">Jam Selesai</label>
            <input v-model="doctorForm.time_end" type="time" class="form-control" />
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-4">
          <button @click="showDoctorForm = false" class="btn btn-secondary">Batal</button>
          <button @click="saveDoctor" class="btn btn-primary">💾 Simpan</button>
        </div>
      </div>

      <div class="glass-card animate-fade-in" style="padding: 0; overflow: hidden;">
        <div v-if="loadingDoctors" class="text-center" style="padding: 2.5rem;">⏳ Memuat...</div>
        <div v-else-if="doctors.length === 0" class="text-center" style="padding: 2.5rem; opacity: 0.6;">
          📭 Belum ada jadwal dokter. Tambahkan jadwal praktek pertama.
        </div>
        <table v-else style="width: 100%; text-align: left;">
          <thead style="background: rgba(255,255,255,0.05);">
            <tr>
              <th class="font-semibold" style="padding: 1rem;">Dokter</th>
              <th class="font-semibold" style="padding: 1rem;">Spesialisasi</th>
              <th class="font-semibold" style="padding: 1rem;">Hari</th>
              <th class="font-semibold" style="padding: 1rem;">Jam</th>
              <th class="font-semibold" style="padding: 1rem;">Max</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="d in doctors" :key="d.id" style="border-top: 1px solid var(--glass-border);">
              <td style="padding: 1rem; font-weight: 600;">{{ d.doctor_name }}</td>
              <td style="padding: 1rem;">{{ d.specialization || '—' }}</td>
              <td style="padding: 1rem;">{{ dayLabel(d.day_of_week) }}</td>
              <td style="padding: 1rem;">{{ d.time_start?.slice(0, 5) }} – {{ d.time_end?.slice(0, 5) }}</td>
              <td style="padding: 1rem;">{{ d.max_patients }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.status-badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.85rem;
  font-weight: 600;
}
.badge-waiting {
  background: rgba(251, 191, 36, 0.15);
  color: #fbbf24;
  border: 1px solid rgba(251, 191, 36, 0.3);
}
.badge-called {
  background: rgba(34, 197, 94, 0.15);
  color: #22c55e;
  border: 1px solid rgba(34, 197, 94, 0.3);
}

.clinic-tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
  background: var(--bg-tertiary, rgba(255,255,255,0.03));
  border-radius: var(--radius-sm);
  padding: 0.4rem;
  border: 1px solid var(--glass-border, rgba(255,255,255,0.08));
}
.clinic-tabs .tab-btn {
  flex: 1;
  padding: 0.75rem 1rem;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font-family: inherit;
  font-size: 0.95rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}
.clinic-tabs .tab-btn:hover:not(.active) {
  color: var(--text-primary);
  background: rgba(255,255,255,0.04);
}
.clinic-tabs .tab-btn.active {
  background: var(--surface-0);
  color: var(--accent-primary);
  box-shadow: 0 2px 6px rgba(0,0,0,0.1);
}
</style>

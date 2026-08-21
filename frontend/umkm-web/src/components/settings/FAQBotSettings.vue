<template>
  <div class="glass-card animate-fade-in" style="max-width: 600px; padding: 2rem;">
    <div class="flex justify-between items-center" style="margin-bottom: 1.5rem;">
      <h3 style="margin-bottom: 0;">Pengaturan FAQ Bot AI</h3>
      <button class="btn btn-secondary btn-sm" @click="$emit('generate')" :disabled="loading">
        ✨ Generate Otomatis
      </button>
    </div>
    <p style="color: var(--text-secondary); margin-bottom: 1rem; font-size: 0.9rem;">
      Tambahkan daftar pertanyaan umum agar AI bisa langsung menjawab pelanggan yang menanyakan hal ini.
    </p>

    <div style="display: flex; flex-direction: column; gap: 1rem; margin-bottom: 1rem;">
      <div v-for="faq in faqs" :key="faq.id"
        style="background: rgba(255,255,255,0.05); padding: 1rem; border-radius: 8px;">
        <template v-if="editingFaqId === faq.id">
          <input type="text" aria-label="Edit pertanyaan FAQ" v-model="editForm.question" class="form-control"
            style="margin-bottom: 0.5rem;" />
          <textarea aria-label="Edit jawaban FAQ" v-model="editForm.answer" class="form-control" rows="2"
            style="margin-bottom: 0.5rem;"></textarea>
          <div style="display: flex; gap: 0.5rem;">
            <button class="btn btn-primary btn-sm" @click="saveEdit(faq.id)"
              :disabled="!editForm.question || !editForm.answer">Simpan</button>
            <button class="btn btn-secondary btn-sm" @click="cancelEdit">Batal</button>
          </div>
        </template>
        <template v-else>
          <div style="font-weight: bold; margin-bottom: 0.3rem;">Q: {{ faq.question }}</div>
          <div style="color: var(--text-secondary); font-size: 0.9rem; margin-bottom: 0.5rem;">A: {{ faq.answer }}
          </div>
          <div style="display: flex; gap: 0.5rem;">
            <button class="btn btn-secondary btn-sm" @click="startEdit(faq)">✏️ Edit</button>
            <button class="btn btn-secondary btn-sm" style="color: #ef4444; border-color: #ef4444;"
              @click="$emit('delete', faq.id)">Hapus</button>
          </div>
        </template>
      </div>
    </div>

    <div
      style="display: flex; flex-direction: column; gap: 0.5rem; border-top: 1px solid var(--border-color); padding-top: 1rem;">
      <input type="text" aria-label="Pertanyaan baru FAQ" placeholder="Pertanyaan (Contoh: Jam Buka?)"
        v-model="newFaq.question" class="form-control" />
      <textarea aria-label="Jawaban baru FAQ" placeholder="Jawaban (Contoh: Buka dari jam 8 pagi...)"
        v-model="newFaq.answer" class="form-control" rows="2"></textarea>
      <button class="btn btn-primary" @click="addNew" :disabled="!newFaq.question || !newFaq.answer">Tambah
        FAQ</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

interface FAQ {
  id: string
  question: string
  answer: string
}

const props = defineProps<{
  faqs: FAQ[]
  loading?: boolean
}>()

const emit = defineEmits<{
  generate: []
  delete: [id: string]
  add: [faq: { question: string; answer: string }]
  update: [id: string, faq: { question: string; answer: string }]
}>()

const editingFaqId = ref<string | null>(null)
const editForm = ref({ question: '', answer: '' })
const newFaq = ref({ question: '', answer: '' })

function startEdit(faq: FAQ) {
  editingFaqId.value = faq.id
  editForm.value = { question: faq.question, answer: faq.answer }
}

function saveEdit(id: string) {
  emit('update', id, { ...editForm.value })
  editingFaqId.value = null
}

function cancelEdit() {
  editingFaqId.value = null
  editForm.value = { question: '', answer: '' }
}

function addNew() {
  emit('add', { ...newFaq.value })
  newFaq.value = { question: '', answer: '' }
}
</script>

<style scoped>
.glass-card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 12px;
  backdrop-filter: blur(10px);
}

.animate-fade-in {
  animation: fadeIn 0.3s ease-in;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>

<template>
  <div>
    <!-- Chat Container -->
    <div v-if="isOpen" class="chatbot-container surface-card">
      <div class="chat-header">
        <div style="display: flex; align-items: center; gap: 10px;">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" stroke-width="2">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
          </svg>
          <h3 style="margin: 0; font-size: 1rem;">AI Business Assistant</h3>
          <span class="status-indicator"></span>
        </div>
        <!-- Proper Close Button in Header -->
        <button @click="toggleChat" class="close-btn" title="Tutup">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>

      <div class="chat-window" ref="chatWindow">
        <div v-for="(msg, index) in messages" :key="index" :class="['chat-bubble', msg.role]">
          {{ msg.content }}
        </div>
        <div v-if="isLoading" class="chat-bubble assistant loading">
          Mengetik...
        </div>
      </div>

      <div class="chat-input-area">
        <input id="chat-input" v-model="inputMsg" @keyup.enter="sendMessage" type="text"
          placeholder="Tanya soal keuangan atau strategi bisnis..." class="form-control" aria-label="Pesan chat" />
        <button @click="sendMessage" class="btn btn-primary" :disabled="isLoading || !inputMsg.trim()">Kirim</button>
      </div>
    </div>

    <!-- Floating Button Icon — hidden when chat is open -->
    <button v-show="!isOpen" class="chatbot-toggle-btn" @click="toggleChat">
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
      </svg>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { api } from '../api'

const isOpen = ref(false)

const toggleChat = () => {
  isOpen.value = !isOpen.value
}

const messages = ref([
  { role: 'assistant', content: 'Halo! Saya AI Assistant Anda. Ada yang bisa saya bantu terkait pembukuan atau strategi penjualan hari ini?' }
])
const inputMsg = ref('')
const isLoading = ref(false)
const chatWindow = ref<HTMLElement | null>(null)

const scrollToBottom = () => {
  nextTick(() => {
    if (chatWindow.value) {
      chatWindow.value.scrollTop = chatWindow.value.scrollHeight
    }
  })
}

const sendMessage = async () => {
  if (!inputMsg.value.trim() || isLoading.value) return

  const userText = inputMsg.value
  messages.value.push({ role: 'user', content: userText })
  inputMsg.value = ''
  isLoading.value = true
  scrollToBottom()

  try {
    const data = await api.post('/api/umkm/chat', { message: userText })

    if (data.success) {
      messages.value.push({ role: 'assistant', content: data.data.reply || data.data })
    } else {
      throw new Error(data.message || 'API Error')
    }
  } catch (e) {
    console.warn('Chat send failed:', e)
    messages.value.push({
      role: 'assistant',
      content: 'Maaf, sistem AI Chatbot sedang bermasalah atau belum terhubung.'
    })
  }

  isLoading.value = false
  scrollToBottom()
}
</script>

<style scoped>
.chatbot-container {
  display: flex;
  flex-direction: column;
  height: 500px;
  max-width: 400px;
  position: fixed;
  bottom: 20px;
  right: 20px;
  z-index: 1000;
}

.chat-header {
  padding: 1rem;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--surface-2);
  border-radius: var(--radius-lg) var(--radius-lg) 0 0;
}

.status-indicator {
  width: 8px;
  height: 8px;
  background: var(--success);
  border-radius: 50%;
  box-shadow: 0 0 6px var(--success);
}

.chat-window {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.chat-bubble {
  max-width: 80%;
  padding: 0.625rem 0.875rem;
  border-radius: var(--radius-md);
  font-size: 0.875rem;
  line-height: 1.5;
}

.chat-bubble.assistant {
  background: var(--surface-2);
  color: var(--text-primary);
  align-self: flex-start;
  border-bottom-left-radius: var(--radius-sm);
}

.chat-bubble.user {
  background: var(--accent);
  color: #fff;
  align-self: flex-end;
  border-bottom-right-radius: var(--radius-sm);
}

.chat-bubble.loading {
  opacity: 0.6;
  font-style: italic;
}

.chat-input-area {
  padding: 0.75rem;
  border-top: 1px solid var(--border-color);
  display: flex;
  gap: 0.5rem;
  background: var(--surface-1);
  border-radius: 0 0 var(--radius-lg) var(--radius-lg);
}

.chatbot-toggle-btn {
  position: fixed;
  bottom: 20px;
  right: 20px;
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: var(--accent);
  border: none;
  color: white;
  box-shadow: 0 4px 12px var(--shadow-glow);
  cursor: pointer;
  z-index: 1001;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.15s ease;
  pointer-events: auto;
}

.chatbot-toggle-btn:hover {
  transform: scale(1.08);
  box-shadow: 0 6px 20px var(--shadow-glow);
}

.chatbot-toggle-btn:active {
  transform: scale(0.97);
}

.close-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
}

.close-btn:hover {
  color: var(--danger);
  background: rgba(255, 255, 255, 0.05);
}
</style>

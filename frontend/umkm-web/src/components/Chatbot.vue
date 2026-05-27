<template>
  <div>
    <!-- Floating Button Icon -->
    <button v-if="!isOpen" class="chatbot-toggle-btn" @click="isOpen = true">
      💬
    </button>

    <!-- Chat Container -->
    <div v-if="isOpen" class="chatbot-container glass-card">
      <div class="chat-header">
        <div style="display: flex; align-items: center; gap: 10px;">
          <h3>AI Business Assistant</h3>
          <span class="status-indicator"></span>
        </div>
        <button class="close-btn" @click="isOpen = false">✖</button>
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
        <input 
          v-model="inputMsg" 
          @keyup.enter="sendMessage" 
          type="text" 
          placeholder="Tanya soal keuangan atau strategi bisnis..."
          class="form-control"
        />
        <button @click="sendMessage" class="btn btn-primary" :disabled="isLoading || !inputMsg.trim()">Kirim</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { api } from '../api'

const isOpen = ref(false)

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
  } catch (e: any) {
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
  border: 1px solid var(--border-color);
  box-shadow: 0 10px 25px rgba(0,0,0,0.5);
}

.chat-header {
  padding: 1rem;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: rgba(0, 0, 0, 0.2);
}

.status-indicator {
  width: 10px;
  height: 10px;
  background: var(--success);
  border-radius: 50%;
  box-shadow: 0 0 8px var(--success);
}

.chat-window {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.chat-bubble {
  max-width: 80%;
  padding: 0.75rem 1rem;
  border-radius: 12px;
  font-size: 0.9rem;
  line-height: 1.4;
}

.chat-bubble.assistant {
  background: var(--bg-tertiary);
  align-self: flex-start;
  border-bottom-left-radius: 2px;
}

.chat-bubble.user {
  background: var(--accent-primary);
  color: white;
  align-self: flex-end;
  border-bottom-right-radius: 2px;
}

.chat-bubble.loading {
  opacity: 0.6;
  font-style: italic;
}

.chat-input-area {
  padding: 1rem;
  border-top: 1px solid var(--border-color);
  display: flex;
  gap: 0.5rem;
}

.form-control {
  flex: 1;
  padding: 0.5rem;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
}

.chatbot-toggle-btn {
  position: fixed;
  bottom: 20px;
  right: 20px;
  width: 60px;
  height: 60px;
  border-radius: 50%;
  background: var(--accent-gradient);
  border: none;
  font-size: 24px;
  color: white;
  box-shadow: 0 4px 12px rgba(0,0,0,0.3);
  cursor: pointer;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.2s ease;
}
.chatbot-toggle-btn:hover {
  transform: scale(1.1);
}

.close-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 16px;
  cursor: pointer;
  padding: 4px;
  border-radius: 50%;
}
.close-btn:hover {
  color: var(--danger);
  background: rgba(255, 255, 255, 0.1);
}
</style>

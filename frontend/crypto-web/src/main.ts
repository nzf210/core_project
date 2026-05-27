import { createApp } from 'vue'
import Vue3Toastify, { type ToastContainerOptions } from 'vue3-toastify'
import 'vue3-toastify/dist/index.css'
import './style.css'
import App from './App.vue'

const app = createApp(App)

app.use(Vue3Toastify, {
  autoClose: 3000,
  theme: 'dark'
} as ToastContainerOptions)

app.mount('#app')

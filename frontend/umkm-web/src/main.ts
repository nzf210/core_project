import { createApp } from 'vue'
import { createHead } from '@unhead/vue/client'
import './assets/main.css'
import App from './App.vue'
import router from './router'
import { initDomain } from './api'

const head = createHead()

initDomain().then(() => {
  createApp(App).use(router).use(head).mount('#app')
})

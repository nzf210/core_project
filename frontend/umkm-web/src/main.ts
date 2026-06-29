import { createApp } from 'vue'
import { createUnhead } from '@unhead/vue'
import './assets/main.css'
import App from './App.vue'
import router from './router'
import { initDomain } from './api'

const head = createUnhead()

initDomain().then(() => {
  createApp(App).use(router).use(head).mount('#app')
})

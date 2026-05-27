import { createApp } from 'vue'
import './assets/main.css'
import App from './App.vue'
import router from './router'
import { initDomain } from './api'

initDomain().then(() => {
  createApp(App).use(router).mount('#app')
})

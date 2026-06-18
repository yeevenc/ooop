import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import App from '@/App.vue'
import router from '@/router'
import pinia from '@/stores'
import { applyTheme } from '@/stores/theme'
import './style.css'

const app = createApp(App)

app.use(ElementPlus)
app.use(pinia)
app.use(router)

// Apply initial theme before mount
applyTheme()

app.mount('#app')

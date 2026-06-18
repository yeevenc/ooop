import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'

// 统一创建 Pinia 实例，避免在入口和其他模块重复创建
const pinia = createPinia()

pinia.use(piniaPluginPersistedstate)

export default pinia

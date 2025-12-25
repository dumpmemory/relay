import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import './styles/variables.css'
import './styles/main.scss'
import App from './App.vue'
import router, { initApp } from './router'

// 初始化应用
async function bootstrap() {
  // 先检查健康状态
  await initApp()

  const app = createApp(App)

  // 注册所有图标
  for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
    app.component(key, component)
  }

  app.use(ElementPlus)
  app.use(router)
  app.mount('#app')
}

bootstrap()

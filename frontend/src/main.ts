import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import router from './router'
import { loadPlugins } from './plugins/plugins'
import './plugins/monaco' // Initialize Monaco Editor workers

async function bootstrap() {
  if (import.meta.env.DEV && new URLSearchParams(location.search).get('mock') === 'node-quality') {
    const { installNodeQualityPreview } = await import('./lab/nodeQualityPreview')
    installNodeQualityPreview()
  }
  const app = createApp(App)

  app.use(router)
  loadPlugins(app)

  app.mount('#app')
}
void bootstrap()

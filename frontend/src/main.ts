import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import router from './router'
import { loadPlugins } from './plugins/plugins'

const app = createApp(App)

app.use(router)
loadPlugins(app)

app.mount('#app')

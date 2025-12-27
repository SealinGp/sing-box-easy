import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import router from './router'
import { loadPlugins } from './plugins/plugins'
import './plugins/monaco' // Initialize Monaco Editor workers

const app = createApp(App)

app.use(router)
loadPlugins(app)

app.mount('#app')

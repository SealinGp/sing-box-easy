import { createApp } from 'vue'
import { loadPlugins } from '../plugins/plugins'
import '../style.css'
import '../plugins/monaco'
import LocalRuleSetPreview from './LocalRuleSetPreview.vue'

const app = createApp(LocalRuleSetPreview)
loadPlugins(app)
app.mount('#app')

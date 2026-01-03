import type { App } from "vue";
import { loadPrimeVue } from "./primevue";
import { loadDayjs } from "./dayjs";
import { loadPinia } from "./pinia";

export function loadPlugins(app: App<Element>) {
  loadPrimeVue(app)
  loadDayjs(app)
  loadPinia(app)
}
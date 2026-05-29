import type { App } from "vue";
import { loadPrimeVue } from "./primevue";
import { loadDayjs } from "./dayjs";
import { loadPinia } from "./pinia";
import { loadI18n } from "../i18n";

export function loadPlugins(app: App<Element>) {
  loadPrimeVue(app)
  loadDayjs(app)
  loadI18n(app)
  loadPinia(app)
}
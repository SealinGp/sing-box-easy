import type { App } from "vue";
import { loadPrimeVue } from "./primevue";
import { loadDayjs } from "./dayjs";

export function loadPlugins(app: App<Element>) {
  loadPrimeVue(app)
  loadDayjs(app)
}
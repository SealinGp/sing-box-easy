import type { App } from "vue";
import { loadPrimeVue } from "./primevue";

export function loadPlugins(app: App<Element>) {
  loadPrimeVue(app)
}
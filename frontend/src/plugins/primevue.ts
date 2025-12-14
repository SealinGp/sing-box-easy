
import PrimeVue from 'primevue/config';
import type { App } from 'vue';
import ToastService from 'primevue/toastservice';

export function loadPrimeVue(app: App<Element>) {
  app.use(PrimeVue, {
    unstyled: true
  });
  app.use(ToastService);
}
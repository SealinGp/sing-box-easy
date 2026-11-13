<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { userService } from '../services'
import { EyeIcon, EyeSlashIcon } from '@heroicons/vue/24/outline'
import Button from '../components/Button.vue'
import LanguageSwitcher from '../components/LanguageSwitcher.vue'

const props = defineProps<{ preview?: boolean }>()
const router = useRouter()
const { t } = useI18n()
const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const showPassword = ref(false)
async function handleLogin() {
  if (loading.value) return
  if (!username.value.trim() || !password.value) { error.value = t('workspace.login.required'); return }
  if (props.preview) { error.value = t('workspace.login.preview'); return }
  error.value = ''
  loading.value = true
  try {
    await userService.login(username.value.trim(), password.value)
    await router.push('/dashboard')
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('workspace.login.failed')
  } finally { loading.value = false }
}
</script>

<template>
  <main class="login-page">
    <header class="login-brand-row">
      <a href="/" class="inline-flex items-center gap-3 font-semibold text-text-primary"><img src="/logo.svg" alt="" width="36" height="36" />Sing Box Easy</a>
      <LanguageSwitcher variant="compact" />
    </header>
    <section class="login-card" aria-labelledby="login-title">
      <img src="/logo.svg" alt="" width="48" height="48" class="mb-6" />
      <h1 id="login-title" class="text-2xl font-semibold text-text-primary">{{ t('workspace.login.title') }}</h1>
      <p class="mt-2 text-sm text-text-secondary">{{ t('workspace.login.description') }}</p>
      <form class="mt-7 space-y-5" @submit.prevent="handleLogin" :aria-busy="loading">
        <div>
          <label for="login-username" class="mb-2 block text-sm font-medium text-text-primary">{{ t('workspace.login.username') }}</label>
          <input id="login-username" v-model="username" name="username" autocomplete="username" autofocus required class="login-input" :disabled="loading" />
        </div>
        <div>
          <label for="login-password" class="mb-2 block text-sm font-medium text-text-primary">{{ t('workspace.login.password') }}</label>
          <div class="relative">
            <input id="login-password" v-model="password" name="password" :type="showPassword ? 'text' : 'password'" autocomplete="current-password" required class="login-input login-password" :disabled="loading" />
            <button type="button" class="absolute inset-y-0 right-2 my-auto flex h-8 w-8 items-center justify-center rounded-control text-text-secondary hover:bg-bg-tertiary focus-visible:outline-2 focus-visible:outline-primary"
              :aria-label="t(showPassword ? 'workspace.login.hidePassword' : 'workspace.login.showPassword')" :aria-pressed="showPassword" @click="showPassword = !showPassword">
              <EyeSlashIcon v-if="showPassword" class="h-4 w-4" /><EyeIcon v-else class="h-4 w-4" />
            </button>
          </div>
        </div>
        <p v-if="error" :role="preview ? 'status' : 'alert'" class="rounded-control border px-3 py-2 text-sm" :class="preview ? 'border-border bg-bg-tertiary text-text-secondary' : 'border-danger/25 bg-danger/5 text-danger dark:text-red-300'">{{ error }}</p>
        <Button type="submit" full-width :loading="loading" class="min-h-11">{{ t('workspace.login.submit') }}</Button>
      </form>
      <p class="mt-6 border-t border-border pt-5 text-xs leading-relaxed text-text-secondary">{{ t('workspace.login.help') }}</p>
    </section>
    <p class="login-footer text-xs text-text-secondary"><RouterLink v-if="preview" to="/lab" class="hover:underline">{{ t('workspace.login.backToLab') }}</RouterLink><span v-else>sing-box-easy · Network workspace</span></p>
  </main>
</template>

<style scoped>
.login-page { min-height: 100dvh; display: grid; grid-template-rows: auto 1fr auto; align-items: center; gap: 32px; padding: 28px 40px; background: var(--color-bg-primary); }
.login-brand-row { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.login-card { width: min(100%, 408px); margin: 24px auto; padding: 32px; border: 1px solid var(--color-border); border-radius: 16px; background: var(--color-bg-secondary); box-shadow: 0 4px 24px rgb(23 40 62 / 6%); }
.login-input { width: 100%; min-height: 44px; padding: 10px 12px; border: 1px solid var(--control-border); border-radius: var(--radius-control); background: var(--control-bg); color: var(--color-text-primary); transition: border-color 140ms, box-shadow 140ms; }
.login-input:focus { outline: none; border-color: var(--color-primary); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-primary) 12%, transparent); }
.login-password { padding-right: 48px; }
.login-input:disabled { opacity: .6; }
.login-footer { text-align: center; }
@media (max-width: 600px) { .login-page { padding: 20px; gap: 24px; } .login-card { padding: 24px; } }
</style>

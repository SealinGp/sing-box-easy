<script setup lang="ts">
/**
 * GitHub sign-in card (OAuth device flow).
 *
 * Anonymous GitHub API calls are capped at 60/hour per IP, which the update
 * checker exhausts behind a shared egress. Signing in raises the cap to
 * 5000/hour using the user's own account.
 *
 * Device flow is used because this app is self-hosted at arbitrary addresses:
 * there is no stable callback URL to register, and no client secret to ship.
 */
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useGitHubAuth } from '../composables/useGitHubAuth'
import { useNotify } from '../composables/useNotify'
import { settingsService } from '../services'

const {
  status,
  session,
  loading,
  starting,
  signingOut,
  error,
  isPending,
  expiresLabel,
  refresh,
  signIn,
  cancel,
  signOut,
  dismiss,
} = useGitHubAuth()

const { t } = useI18n()
const notify = useNotify()

/**
 * OAuth App client ID, editable here and stored in the database.
 *
 * It used to require an app.yml edit plus a restart, which is a poor fit for a
 * router appliance. Saving it re-checks sign-in availability immediately
 * because the backend resolves the ID per call rather than at startup.
 */
const clientIdInput = ref('')
const savingClientId = ref(false)

onMounted(async () => {
  await refresh()
  try {
    const { data } = await settingsService.getSettings()
    clientIdInput.value = data.github_oauth_client_id ?? ''
  } catch {
    // Non-fatal: the field simply starts empty. Sign-in state is unaffected.
  }
})

const saveClientId = async () => {
  const value = clientIdInput.value.trim()
  if (!value) return

  savingClientId.value = true
  try {
    await settingsService.updateSettings({ github_oauth_client_id: value })
    // The manager reads the ID per call, so sign-in becomes available at once.
    await refresh()
    notify.success(t('settings.githubAuth.clientIdSaved'))
  } catch (err) {
    notify.apiError(err, t('settings.githubAuth.clientIdSaveFailed'))
  } finally {
    savingClientId.value = false
  }
}

/** Copy the device code so the user need not retype it. */
const copied = ref(false)

async function copyCode() {
  if (!session.value?.user_code) return
  try {
    await navigator.clipboard.writeText(session.value.user_code)
    copied.value = true
    setTimeout(() => (copied.value = false), 2000)
  } catch {
    // Clipboard is unavailable over plain HTTP on some browsers — the code is
    // displayed in full, so the user can still type it.
  }
}
</script>

<template>
  <div class="bg-white dark:bg-gray-800 rounded-surface shadow p-5">
    <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-1">
      {{ $t('settings.githubAuth.title') }}
    </h3>
    <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
      {{ $t('settings.githubAuth.desc') }}
    </p>

    <div v-if="loading" class="h-10 flex items-center">
      <div class="animate-spin rounded-pill h-5 w-5 border-b-2 border-primary-600"></div>
    </div>

    <!-- No OAuth App configured: sign-in cannot work, so explain instead of
         offering a button that would always fail. -->
    <div
      v-else-if="!status.configured"
      class="rounded-control bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 p-3"
    >
      <p class="text-sm text-amber-800 dark:text-amber-300">
        {{ $t('settings.githubAuth.notConfigured') }}
      </p>

      <!-- Saved to the database, not app.yml: no file edit, no restart. The
           client ID is public by design (device flow has no client secret),
           so a plain text input is appropriate here. -->
      <div class="mt-3 flex items-stretch gap-2">
        <input
          v-model="clientIdInput"
          type="text"
          spellcheck="false"
          autocomplete="off"
          placeholder="Ov23li..."
          class="flex-1 min-w-0 rounded-control border border-amber-300 dark:border-amber-700 bg-white dark:bg-gray-900 px-3 py-2 font-mono text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
          @keyup.enter="saveClientId"
        />
        <button
          @click="saveClientId"
          :disabled="savingClientId || !clientIdInput.trim()"
          class="flex-shrink-0 px-4 text-sm font-medium text-white bg-primary-600 rounded-control hover:bg-primary-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
        >
          {{ savingClientId ? $t('common.saving') : $t('common.save') }}
        </button>
      </div>
      <p class="mt-2 text-xs text-amber-700 dark:text-amber-400">
        {{ $t('settings.githubAuth.clientIdHelp') }}
      </p>
    </div>

    <template v-else>
      <!-- Pending: show the code to enter on github.com -->
      <div
        v-if="isPending"
        class="rounded-surface border border-primary-200 dark:border-primary-800 bg-primary-50 dark:bg-primary-900/20 p-4"
      >
        <p class="text-sm text-gray-700 dark:text-gray-200 mb-3">
          {{ $t('settings.githubAuth.step1') }}
        </p>

        <div class="flex items-center gap-3 mb-3">
          <code
            class="flex-1 text-center text-2xl font-mono font-bold tracking-widest text-gray-900 dark:text-white bg-white dark:bg-gray-900 rounded-surface py-3 border border-gray-200 dark:border-gray-700 select-all"
          >
            {{ session?.user_code }}
          </code>
          <button
            class="px-3 py-2 text-sm font-medium text-primary-700 dark:text-primary-300 bg-primary-100 dark:bg-primary-900/40 rounded-control hover:bg-primary-200 dark:hover:bg-primary-900/60 transition-colors"
            @click="copyCode"
          >
            {{ copied ? $t('settings.githubAuth.copied') : $t('settings.githubAuth.copy') }}
          </button>
        </div>

        <a
          :href="session?.verification_uri"
          target="_blank"
          rel="noopener noreferrer"
          class="block w-full text-center px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-control hover:bg-primary-700 transition-colors"
        >
          {{ $t('settings.githubAuth.openGitHub') }} →
        </a>

        <div class="flex items-center justify-between mt-3">
          <span class="inline-flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
            <span class="animate-spin rounded-pill h-3 w-3 border-b-2 border-primary-600"></span>
            {{ $t('settings.githubAuth.waiting') }}
            <template v-if="expiresLabel">({{ expiresLabel }})</template>
          </span>
          <button
            class="text-xs font-medium text-gray-500 dark:text-gray-400 hover:underline"
            @click="cancel"
          >
            {{ $t('common.cancel') }}
          </button>
        </div>
      </div>

      <!-- Finished but not authorized: denied / expired / failed -->
      <div
        v-else-if="session && session.status !== 'authorized'"
        class="rounded-control border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-3"
      >
        <p class="text-sm text-red-700 dark:text-red-300">
          {{ session.error || $t('settings.githubAuth.failed') }}
        </p>
        <div class="flex gap-3 mt-2">
          <button
            class="text-sm font-medium text-primary-600 dark:text-primary-400 hover:underline"
            @click="signIn"
          >
            {{ $t('settings.githubAuth.retry') }}
          </button>
          <button
            class="text-sm font-medium text-gray-500 dark:text-gray-400 hover:underline"
            @click="dismiss"
          >
            {{ $t('common.close') }}
          </button>
        </div>
      </div>

      <!-- Connected -->
      <div v-else-if="status.connected" class="flex flex-wrap items-center gap-3">
        <span
          class="inline-flex items-center gap-2 text-sm text-emerald-700 dark:text-emerald-400"
        >
          <span class="h-2 w-2 rounded-pill bg-emerald-500"></span>
          <template v-if="status.login">
            {{ $t('settings.githubAuth.connectedAs', { login: status.login }) }}
          </template>
          <template v-else>
            {{ $t('settings.githubAuth.connected') }}
          </template>
        </span>
        <span class="text-xs text-gray-500 dark:text-gray-400">
          {{ $t('settings.githubAuth.rateLimitOk') }}
        </span>
        <button
          :disabled="signingOut"
          class="ml-auto px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-700 rounded-control hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors disabled:opacity-50"
          @click="signOut"
        >
          {{ $t('settings.githubAuth.signOut') }}
        </button>
      </div>

      <!-- Disconnected -->
      <div v-else class="flex flex-wrap items-center gap-3">
        <span class="text-sm text-amber-600 dark:text-amber-400">
          {{ $t('settings.githubAuth.disconnected') }}
        </span>
        <button
          :disabled="starting"
          class="ml-auto inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-gray-900 dark:bg-gray-700 rounded-control hover:bg-gray-800 dark:hover:bg-gray-600 transition-colors disabled:opacity-50"
          @click="signIn"
        >
          <svg class="h-4 w-4" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
            <path
              d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0016 8c0-4.42-3.58-8-8-8z"
            />
          </svg>
          {{ starting ? $t('settings.githubAuth.starting') : $t('settings.githubAuth.signIn') }}
        </button>
      </div>

      <p v-if="error" class="text-sm text-red-600 dark:text-red-400 mt-3">{{ error }}</p>

      <p class="text-xs text-gray-500 dark:text-gray-400 mt-3">
        {{ $t('settings.githubAuth.scopeHint') }}
      </p>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter, useRoute } from 'vue-router'
import type { InitState } from '../types/api'
import InstallSingBox from './init-steps/InstallSingBox.vue'
import ConfigureLog from './init-steps/ConfigureLog.vue'
import ConfigureExperimental from './init-steps/ConfigureExperimental.vue'
import DownloadDashboard from './init-steps/DownloadDashboard.vue'
import ConfigureOutbounds from './init-steps/ConfigureOutbounds.vue'
import ConfigureRuleSets from './init-steps/ConfigureRuleSets.vue'
import ConfigureDNS from './init-steps/ConfigureDNS.vue'
import ConfigureInbounds from './init-steps/ConfigureInbounds.vue'
import ConfigureRoutes from './init-steps/ConfigureRoutes.vue'
import Complete from './init-steps/Complete.vue'
import LanguageSwitcher from '../components/LanguageSwitcher.vue'
import { serviceControlService } from '../services'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const currentStep = ref(0)
const initState = ref<InitState | null>(null)
const loading = ref(false)
const error = ref('')

// Computed so step titles re-translate when the locale changes. Components are
// static and unaffected.
const steps = computed(() => [
  { title: t('wizard.steps.install'), component: InstallSingBox },
  { title: t('wizard.steps.log'), component: ConfigureLog },
  { title: t('wizard.steps.experimental'), component: ConfigureExperimental },
  { title: t('wizard.steps.dashboard'), component: DownloadDashboard },
  { title: t('wizard.steps.outbounds'), component: ConfigureOutbounds },
  { title: t('wizard.steps.ruleSets'), component: ConfigureRuleSets },
  { title: t('wizard.steps.dns'), component: ConfigureDNS },
  { title: t('wizard.steps.inbounds'), component: ConfigureInbounds },
  { title: t('wizard.steps.routes'), component: ConfigureRoutes },
  { title: t('wizard.steps.complete'), component: Complete },
])

// Initialize step from query param
const initializeStep = () => {
  const stepParam = route.query.step
  if (stepParam) {
    const stepIndex = parseInt(stepParam as string, 10)
    if (!isNaN(stepIndex) && stepIndex >= 0 && stepIndex < steps.value.length) {
      currentStep.value = stepIndex
      return
    }
  }
  // Default to step 0 if no valid query param
  if (currentStep.value === 0) {
    updateQueryParam(0)
  }
}

// Update query param when step changes
const updateQueryParam = (step: number) => {
  router.replace({ query: { step: step.toString() } })
}

// Watch for query param changes (browser back/forward)
watch(() => route.query.step, (newStep) => {
  if (newStep) {
    const stepIndex = parseInt(newStep as string, 10)
    if (!isNaN(stepIndex) && stepIndex >= 0 && stepIndex < steps.value.length) {
      currentStep.value = stepIndex
    }
  }
})

// Watch currentStep and update query param
watch(currentStep, (newStep) => {
  updateQueryParam(newStep)
})

onMounted(async () => {
  // Initialize step from query param first
  initializeStep()

  try {
    loading.value = true
    const {data} = await serviceControlService.getInitStatus()
    initState.value = data

    // Skip the wizard entirely when:
    //   - the user explicitly completed it (initialized=true), OR
    //   - the backend's live detection found a meaningful config.json
    //     (config_generated=true). The latter handles "deployed
    //     sing-box-easy over an existing sing-box" — the operator never
    //     needs to walk the wizard.
    const skipWizard =
      initState.value?.initialized ||
      initState.value?.steps?.config_generated
    if (skipWizard) {
      router.push('/dashboard')
    }
  } catch (err: any) {
    error.value = err.message || t('wizard.failedStatus')
  } finally {
    loading.value = false
  }
})

const nextStep = () => {
  if (currentStep.value < steps.value.length - 1) {
    currentStep.value++
  }
}

const prevStep = () => {
  if (currentStep.value > 0) {
    currentStep.value--
  }
}
</script>

<template>
  <div class="min-h-screen bg-gradient-to-br from-primary-50 to-indigo-100 py-12 px-4 flex justify-center items-center">
    <div class="mx-auto w-full 2xl:w-3/4 xl:w-2/3 p-3 grid grid-cols-1 gap-y-2">
      <!-- Header -->
      <div class="text-center relative">
        <div class="absolute right-0 top-0">
          <LanguageSwitcher variant="compact" />
        </div>
        <h1 class="text-4xl font-bold text-gray-900 mb-1">{{ $t('wizard.title') }}</h1>
        <p class="text-gray-600">{{ $t('wizard.welcome') }}</p>
      </div>

      <!-- Progress Steps -->
      <div class="">
        <div class="flex items-center justify-between">
          <div
            v-for="(step, index) in steps"
            :key="index"
            class="flex-1 relative"
          >
            <div class="flex items-center">
              <div
                :class="[
                  'w-10 h-10 rounded-pill flex items-center justify-center font-semibold transition-all',
                  index < currentStep
                    ? 'bg-green-500 text-white'
                    : index === currentStep
                    ? 'bg-primary-600 text-white ring-4 ring-primary-200'
                    : 'bg-gray-300 text-gray-600',
                ]"
              >
                <span v-if="index < currentStep">✓</span>
                <span v-else>{{ index + 1 }}</span>
              </div>
              <div
                v-if="index < steps.length - 1"
                :class="[
                  'flex-1 h-1 mx-2',
                  index < currentStep ? 'bg-green-500' : 'bg-gray-300',
                ]"
              ></div>
            </div>
            <div class="absolute top-12 left-0 right-0 ">
              <p
                :class="[
                  'text-xs font-medium',
                  index === currentStep ? 'text-primary-600' : 'text-gray-500',
                ]"
              >
                {{ step.title }}
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="bg-white dark:bg-slate-800 rounded-surface shadow-float dark:shadow-float dark:shadow-slate-700/50 p-6 text-center">
        <div class="inline-block animate-spin rounded-pill h-12 w-12 border-b-2 border-primary-600"></div>
        <p class="mt-4 text-gray-600">{{ $t('wizard.loadingStatus') }}</p>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-surface p-4">
        <h3 class="text-red-800 font-semibold mb-1">{{ $t('common.error') }}</h3>
        <p class="text-red-600">{{ error }}</p>
      </div>

      <!-- Main Content -->
      <div v-else class="bg-white dark:bg-slate-800 rounded-surface shadow-float dark:shadow-float dark:shadow-slate-700/50 mt-9 p-3">
        <div class="mb-4">
          <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-1">
            {{ steps[currentStep]?.title }}
          </h2>
          <div class="w-full bg-gray-200 rounded-pill h-2">
            <div
              class="bg-primary-600 h-2 rounded-pill transition-all"
              :style="{ width: `${((currentStep + 1) / steps.length) * 100}%` }"
            ></div>
          </div>
        </div>

        <!-- Step Content -->
        <div class="min-h-[400px]">
          <component
            :is="steps[currentStep]?.component"
            v-if="steps[currentStep]?.component"
            @next="nextStep"
            @prev="prevStep"
          />
          <div v-else class="text-center py-12">
            <p class="text-gray-600">{{ $t('wizard.stepN', { n: currentStep + 1, title: steps[currentStep]?.title }) }}</p>
            <p class="text-gray-500 mt-2">{{ $t('wizard.comingSoon') }}</p>
            <div class=" flex justify-center gap-3">
              <button
                @click="prevStep"
                :disabled="currentStep === 0"
                class="px-6 py-2 border border-gray-300 rounded-control text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {{ $t('common.previous') }}
              </button>
              <button
                v-if="currentStep < steps.length - 1"
                @click="nextStep"
                class="px-6 py-2 bg-primary-600 text-white rounded-control hover:bg-primary-700"
              >
                {{ $t('common.next') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

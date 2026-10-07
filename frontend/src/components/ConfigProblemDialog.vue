<script setup lang="ts">
/**
 * App-wide dialog for a config the panel's pre-check refused: a setting names
 * an outbound that does not exist, or a group with no members.
 *
 * `sing-box check` passes such a config and the core then fails to start on
 * it, so the panel checks first (backend: reference_check.go) and this is
 * where the operator is told. Each problem is listed in their terms, grouped
 * by the page that fixes it, with a button that goes there — the fix is never
 * on the page that triggered the error.
 *
 * Mounted once in App.vue; opened by the API client through useConfigProblem.
 */
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { ExclamationTriangleIcon } from '@heroicons/vue/24/outline'
import { Dialog } from '../volt'
import Button from './Button.vue'
import { useConfigProblem } from '../composables/useConfigProblem'
import { groupProblems } from '../utils/configProblems'

const router = useRouter()
const { problems, visible, dismiss } = useConfigProblem()

const groups = computed(() => groupProblems(problems.value))

const goFix = async (to: string) => {
  dismiss()
  await router.push(to)
}
</script>

<template>
  <Dialog v-model:visible="visible" modal :header="$t('configProblem.title')" class="w-full max-w-lg">
    <div class="space-y-4 text-sm">
      <div class="flex gap-2.5 rounded-control border border-amber-300 bg-amber-50 px-3 py-2.5 text-amber-800 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200">
        <ExclamationTriangleIcon class="mt-0.5 h-4 w-4 shrink-0" />
        <p>{{ $t('configProblem.intro') }}</p>
      </div>

      <section v-for="group in groups" :key="group.field" class="space-y-2">
        <div class="flex items-center justify-between gap-3">
          <h4 class="font-semibold text-gray-900 dark:text-gray-100">{{ $t(`configProblem.fields.${group.field}`) }}</h4>
          <Button size="sm" @click="goFix(group.to)">{{ $t(`configProblem.fix.${group.field}`) }}</Button>
        </div>
        <ul class="space-y-1.5">
          <li
            v-for="problem in group.problems"
            :key="problem.where + problem.tag"
            class="rounded-control border border-gray-200 px-2.5 py-1.5 dark:border-gray-700"
          >
            <p class="text-gray-900 dark:text-gray-100">
              {{ $t(`configProblem.kinds.${problem.kind}`, { tag: problem.tag }) }}
            </p>
            <p class="font-mono text-xs text-gray-500 dark:text-gray-400">{{ problem.where }}</p>
          </li>
        </ul>
      </section>

      <p class="text-xs text-gray-500 dark:text-gray-400">{{ $t('configProblem.unchanged') }}</p>
    </div>

    <template #footer>
      <Button variant="secondary" @click="dismiss">{{ $t('common.close') }}</Button>
    </template>
  </Dialog>
</template>

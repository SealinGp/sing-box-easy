<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t, locale } = useI18n()

import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowTopRightOnSquareIcon, MoonIcon, SunIcon } from '@heroicons/vue/24/outline'
import CopyIcon from './CopyIcon.vue'
import Skeleton from './Skeleton.vue'
import { contrastRatio, coolorsUrl, getLabPalette, labPalettes, LOGO_COLORS } from '../lab/palettes'

const route = useRoute()
const router = useRouter()
const chosen = ref(getLabPalette(route.query.palette).id)
const palette = computed(() => getLabPalette(chosen.value))
const iframe = ref<HTMLIFrameElement | null>(null)
const ready = ref(false)
// Keep one document mounted: changing color must not discard an edited field.
const previewUrl = `/lab/palette-preview?palette=${encodeURIComponent(chosen.value)}`
const colorList = computed(() => palette.value.colors.join(', '))
const roles = computed(() => [
  { get name() { return t('lab.canvas') }, color: palette.value.canvas }, { get name() { return t('lab.card') }, color: palette.value.surface },
  { get name() { return t('lab.action') }, color: palette.value.primary }, { get name() { return t('lab.accent') }, color: palette.value.accent },
  { get name() { return t('lab.text') }, color: palette.value.ink },
])
const contrast = computed(() => [
  { get name() { return t('lab.bodyCard') }, value: contrastRatio(palette.value.ink, palette.value.surface) },
  { get name() { return t('lab.mutedCard') }, value: contrastRatio(palette.value.muted, palette.value.surface) },
  { get name() { return t('lab.buttonLabel') }, value: contrastRatio(palette.value.onPrimary, palette.value.primary) },
])
function sendPalette() {
  iframe.value?.contentWindow?.postMessage({ type: 'sbe:palette', id: chosen.value, locale: locale.value }, window.location.origin)
}
function onMessage(event: MessageEvent) {
  if (event.origin !== window.location.origin || event.source !== iframe.value?.contentWindow) return
  if (event.data?.type === 'sbe:palette-ready') { ready.value = true; sendPalette() }
}
watch(locale, sendPalette)
watch(chosen, () => {
  sendPalette()
  if (route.query.palette !== chosen.value) void router.replace({ query: { ...route.query, palette: chosen.value }, hash: route.hash })
})
watch(() => route.query.palette, id => { chosen.value = getLabPalette(id).id })
onMounted(() => window.addEventListener('message', onMessage))
onBeforeUnmount(() => window.removeEventListener('message', onMessage))

</script>

<template>
  <section id="palettes" aria-labelledby="palette-title" class="palette-studio scroll-mt-4 space-y-4">
    <header class="palette-section-heading">
      <div><h2 id="palette-title" class="text-lg font-semibold">{{ t('lab.colorDirections') }}</h2><p class="mt-1 text-sm text-text-secondary">{{ t('lab.oneBrandFourWaysToWorkChooseADirectionAnd') }}</p></div>
      <div class="brand-anchors" :aria-label="t('lab.fixedLogoColors')"><span v-for="color in LOGO_COLORS" :key="color"><i :style="{ background: color }" />{{ color }}</span></div>
    </header>
    <fieldset class="direction-grid">
      <legend class="sr-only">{{ t('lab.chooseAPalette') }}</legend>
      <label v-for="option in labPalettes" :key="option.id" class="direction-card" :class="{ 'is-current': option.id === chosen }"
        :style="{ background: option.surface, color: option.ink, borderColor: option.border, colorScheme: option.mode, '--direction-focus': option.primary }">
        <div class="direction-heading"><span class="flex items-center gap-2"><input v-model="chosen" type="radio" name="lab-palette" :value="option.id" :aria-label="`${option.name} ${option.mode}`" :style="{ accentColor: option.primary }" /><strong>{{ option.name }}</strong></span><span class="direction-mode" :style="{ color: option.muted }"><SunIcon v-if="option.mode === 'light'" class="h-3.5 w-3.5" aria-hidden="true" /><MoonIcon v-else class="h-3.5 w-3.5" aria-hidden="true" />{{ t('lab.' + option.mode) }}</span></div>
        <div class="direction-swatches" aria-hidden="true"><span v-for="color in option.colors" :key="color" :style="{ background: color }" /></div>
        <p :style="{ color: option.muted }">{{ t('lab.paletteDescription.' + option.id) }}</p>
        <span v-if="option.recommended" class="direction-recommendation" :style="{ color: option.muted }">{{ t('lab.selectedForTheDashboard') }}</span>
      </label>
    </fieldset>
    <div class="palette-review-bar">
      <div aria-live="polite"><strong>{{ palette.name }}</strong><span class="ml-2 text-xs text-text-secondary">{{ t('lab.' + palette.mode) }} {{ t('lab.preview') }}</span></div>
      <div class="flex items-center gap-3"><span class="inline-flex items-center gap-1 text-xs text-text-secondary"><CopyIcon :text="colorList" :label="t('lab.copyPaletteHexColors')" />{{ t('lab.copyColors') }}</span><a :href="coolorsUrl(palette)" target="_blank" rel="noopener noreferrer" class="coolors-link">{{ t('lab.openInCoolors') }}<ArrowTopRightOnSquareIcon class="h-3.5 w-3.5" aria-hidden="true" /><span class="sr-only">{{ t('lab.opensInANewTab') }}</span></a></div>
    </div>
    <div class="palette-frame-wrap">
      <iframe ref="iframe" :src="previewUrl" :title="`${palette.name} ${palette.mode} — interactive sample workspace`"
        class="palette-frame" :class="{ 'is-ready': ready }" :style="{ colorScheme: palette.mode, background: palette.canvas }" @load="sendPalette" />
      <div v-if="!ready" class="palette-frame-placeholder" role="status" :aria-label="t('lab.loadingPalettePreview')"><Skeleton class="h-8 w-48" /><div class="grid grid-cols-3 gap-3"><Skeleton v-for="n in 3" :key="n" class="h-28 w-full" /></div><Skeleton class="h-40 w-full" /><Skeleton class="h-48 w-full" /></div>
    </div>
    <div class="palette-detail-grid">
      <div>
        <h3 class="text-sm font-semibold">{{ t('lab.appliedColorRoles') }}</h3>
        <div class="palette-role-grid"><div v-for="role in roles" :key="role.name" class="palette-role"><span class="palette-role-chip" :style="{ background: role.color }" /><div><span>{{ role.name }}</span><code>{{ role.color }}</code></div></div></div>
      </div>
      <div><h3 class="text-sm font-semibold">{{ t('lab.textContrast') }}</h3><dl class="contrast-list"><div v-for="item in contrast" :key="item.name"><dt>{{ item.name }}</dt><dd>{{ item.value.toFixed(1) }}:1 <span>AA</span></dd></div></dl></div>
    </div>
    <p class="text-xs leading-relaxed text-text-secondary">{{ t('lab.sandstoneIsTheSelectedDashboardPaletteWarmMineralSurfacesInk') }}</p>
  </section>
</template>

<style scoped>
.palette-section-heading { display: flex; align-items: flex-start; justify-content: space-between; flex-wrap: wrap; gap: 16px; }
.brand-anchors { display: flex; flex-wrap: wrap; gap: 12px; font: 11px ui-monospace, monospace; color: var(--color-text-secondary); }
.brand-anchors span { display: inline-flex; align-items: center; gap: 6px; }
.brand-anchors i { width: 16px; height: 16px; border-radius: 4px; border: 1px solid var(--color-border); }
.direction-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; }
.direction-card { display: flex; flex-direction: column; gap: 12px; min-width: 0; padding: 14px; border: 1px solid; border-radius: 10px; cursor: pointer; }
.direction-card.is-current { outline: 2px solid var(--direction-focus); outline-offset: 2px; }
.direction-card:has(input:focus-visible) { outline: 3px solid var(--direction-focus); outline-offset: 3px; }
.direction-heading { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px; font-size: 13px; }
.direction-heading input { width: 16px; height: 16px; margin: 0; }
.direction-heading strong { font-weight: 600; }
.direction-mode { display: inline-flex; align-items: center; gap: 4px; font-size: 11px; }
.direction-swatches { display: flex; height: 28px; overflow: hidden; border-radius: 5px; }
.direction-swatches span { flex: 1; }
.direction-card p { font-size: 11px; line-height: 1.5; }
.direction-recommendation { margin-top: auto; font-size: 10px; }
.palette-review-bar { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; padding-top: 8px; }
.coolors-link { display: inline-flex; align-items: center; gap: 6px; font-size: 12px; font-weight: 500; color: var(--color-primary); }
.coolors-link:hover { text-decoration: underline; }
.coolors-link:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 3px; }
.palette-frame-wrap { position: relative; overflow: hidden; border: 1px solid var(--color-border); border-radius: 12px; background: var(--color-bg-secondary); }
.palette-frame { display: block; width: 100%; height: 780px; border: 0; opacity: 0; }
.palette-frame.is-ready { opacity: 1; }
.palette-frame-placeholder { position: absolute; inset: 0; padding: 24px; display: flex; flex-direction: column; gap: 20px; pointer-events: none; }
.palette-detail-grid { display: grid; grid-template-columns: 1.5fr 1fr; gap: 24px; border-bottom: 1px solid var(--color-border); padding: 4px 0 16px; }
.palette-role-grid { display: flex; flex-wrap: wrap; gap: 12px 20px; margin-top: 12px; }
.palette-role { display: flex; align-items: center; gap: 8px; font-size: 11px; }
.palette-role-chip { width: 32px; height: 32px; border: 1px solid var(--color-border); border-radius: 6px; }
.palette-role code { display: block; margin-top: 3px; color: var(--color-text-secondary); font-size: 10px; }
.contrast-list { display: grid; gap: 6px; margin-top: 12px; font-size: 11px; color: var(--color-text-secondary); }
.contrast-list > div { display: flex; justify-content: space-between; gap: 12px; }
.contrast-list dd { font-variant-numeric: tabular-nums; }
.contrast-list dd span { margin-left: 6px; font-size: 10px; }
@media (max-width: 1100px) { .direction-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 640px) { .direction-grid { gap: 10px; } .direction-card { padding: 12px; } .direction-heading { font-size: 12px; } .direction-recommendation { font-size: 9px; } .palette-detail-grid { grid-template-columns: 1fr; gap: 20px; } .palette-frame { height: 860px; } }
</style>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RuleSet } from '../../types/api'
import { Button, Alert, Card, Badge } from '../../components'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'
import { routeService } from '../../services'

const { t } = useI18n()

const emit = defineEmits<{
  next: []
  prev: []
}>()

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref(false)

const existingRuleSets = ref<RuleSet[]>([])
const selectedPresets = ref<Set<string>>(new Set())

// 预设规则集
interface PresetRuleSet {
  id: string
  tag: string
  // i18n preset key under setup.ruleSets.presets.*
  presetKey: string
  name: string
  description: string
  type: 'remote'
  format: 'binary'
  url: string
  download_detour?: string
}

// Display label/description re-translate on locale switch; technical fields
// (tag/type/format/url) sent to the backend stay unchanged.
const presetRuleSets = computed<PresetRuleSet[]>(() => [
  {
    id: 'geosite-cn',
    tag: 'geosite-cn',
    presetKey: 'geositeCn',
    name: t('setup.ruleSets.presets.geositeCn.name'),
    description: t('setup.ruleSets.presets.geositeCn.description'),
    type: 'remote',
    format: 'binary',
    url: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs',
  },
  {
    id: 'geosite-geolocation-!cn',
    tag: 'geosite-geolocation-!cn',
    presetKey: 'geositeNonCn',
    name: t('setup.ruleSets.presets.geositeNonCn.name'),
    description: t('setup.ruleSets.presets.geositeNonCn.description'),
    type: 'remote',
    format: 'binary',
    url: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-geolocation-!cn.srs',
  },
  {
    id: 'geosite-category-ads-all',
    tag: 'geosite-category-ads-all',
    presetKey: 'adsAll',
    name: t('setup.ruleSets.presets.adsAll.name'),
    description: t('setup.ruleSets.presets.adsAll.description'),
    type: 'remote',
    format: 'binary',
    url: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs',
  },
  {
    id: 'geoip-cn',
    tag: 'geoip-cn',
    presetKey: 'geoipCn',
    name: t('setup.ruleSets.presets.geoipCn.name'),
    description: t('setup.ruleSets.presets.geoipCn.description'),
    type: 'remote',
    format: 'binary',
    url: 'https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs',
  },
  {
    id: 'geosite-google',
    tag: 'geosite-google',
    presetKey: 'google',
    name: t('setup.ruleSets.presets.google.name'),
    description: t('setup.ruleSets.presets.google.description'),
    type: 'remote',
    format: 'binary',
    url: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-google.srs',
  },
  {
    id: 'geosite-github',
    tag: 'geosite-github',
    presetKey: 'github',
    name: t('setup.ruleSets.presets.github.name'),
    description: t('setup.ruleSets.presets.github.description'),
    type: 'remote',
    format: 'binary',
    url: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-github.srs',
  },
])

onMounted(async () => {
  await loadRuleSets()
})

const loadRuleSets = async () => {
  loading.value = true
  error.value = ''

  try {
    const {data} = await routeService.getRuleSets()
    const rule_sets = data.rule_sets
    existingRuleSets.value = rule_sets || []

    // 预选已存在的规则集
    rule_sets?.forEach((ruleSet) => {
      const preset = presetRuleSets.value.find(p => p.tag === ruleSet.tag)
      if (preset) {
        selectedPresets.value.add(preset.id)
      }
    })
  } catch (err: any) {
    console.log('No existing rule sets found')
  } finally {
    loading.value = false
  }
}

const togglePreset = (presetId: string) => {
  const next = new Set(selectedPresets.value)
  if (next.has(presetId)) next.delete(presetId); else next.add(presetId)
  selectedPresets.value = next
}

const selectAll = () => {
  selectedPresets.value = new Set(presetRuleSets.value.map(p => p.id))
}

const deselectAll = () => {
  selectedPresets.value = new Set()
}

const saveRuleSets = async () => {
  saving.value = true
  error.value = ''
  success.value = false

  try {
    // 获取当前已存在的规则集标签
    const existingTags = new Set(existingRuleSets.value.map(rs => rs.tag))

    // 添加选中的预设规则集
    for (const preset of presetRuleSets.value) {

      if (selectedPresets.value.has(preset.id)) {
        // 如果不存在，则添加
        if (!existingTags.has(preset.tag)) {
          const ruleSet: RuleSet = {
            tag: preset.tag,
            type: preset.type,
            format: preset.format,
            url: preset.url,
          }
          await routeService.addRuleSet(ruleSet)
        }
      } else {
        // 如果已存在但未选中，则删除
        if (existingTags.has(preset.tag)) {
          await routeService.deleteRuleSet(preset.tag)
        }
      }
    }

    success.value = true

    // 2秒后自动进入下一步
    setTimeout(() => {
      emit('next')
    }, 2000)
  } catch (err: any) {
    error.value = err.message || t('setup.ruleSets.saveFailed')
  } finally {
    saving.value = false
  }
}

const handleNext = () => {
  if (!success.value) {
    saveRuleSets()
  } else {
    emit('next')
  }
}

const handleSkip = () => {
  emit('next')
}
</script>

<template>
  <div class="space-y-6">
    <!-- 成功提示 -->
    <Alert v-if="success" type="success" :title="$t('setup.ruleSets.successTitle')">
      {{ $t('setup.ruleSets.successDesc') }}
    </Alert>

    <!-- 错误提示 -->
    <Alert v-if="error" type="error" closable @close="error = ''">
      {{ error }}
    </Alert>

    <!-- 预设规则集 -->
    <Card>
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 mb-2">{{ $t('setup.ruleSets.selectHeading') }}</h3>
            <p class="text-sm text-gray-600">
              {{ $t('setup.ruleSets.selectDesc') }}
            </p>
          </div>
          <div class="flex gap-2">
            <Button
              variant="secondary"
              size="sm"
              :disabled="loading || saving || success"
              @click="selectAll"
            >
              {{ $t('setup.ruleSets.selectAll') }}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              :disabled="loading || saving || success"
              @click="deselectAll"
            >
              {{ $t('setup.ruleSets.clear') }}
            </Button>
          </div>
        </div>

        <!-- 规则集列表 -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div
            v-for="preset in presetRuleSets"
            :key="preset.id"
            class="flex items-start gap-3 p-4 border border-gray-200 rounded-lg hover:border-violet-300 hover:bg-violet-50 cursor-pointer transition-colors"
            :class="{ 'border-violet-500 bg-violet-50': selectedPresets.has(preset.id) }"
            @click="togglePreset(preset.id)"
          >
            <input
              type="checkbox"
              :checked="selectedPresets.has(preset.id)"
              class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500 mt-0.5"
              :disabled="loading || saving || success"
              @click.stop="togglePreset(preset.id)"
            />
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <p class="text-sm font-medium text-gray-900">{{ preset.name }}</p>
                <Badge variant="gray" size="sm">{{ preset.format }}</Badge>
              </div>
              <p class="text-xs text-gray-600 mt-1">{{ preset.description }}</p>
              <p class="text-xs text-gray-400 mt-1 truncate font-mono">{{ preset.tag }}</p>
            </div>
          </div>
        </div>

        <!-- 选择统计 -->
        <div class="bg-gray-50 rounded-lg p-3 text-center">
          <p class="text-sm text-gray-600">
            <span class="font-semibold text-gray-900">{{ selectedPresets.size }}</span> {{ $t('setup.ruleSets.selectedCount') }}
          </p>
        </div>
      </div>
    </Card>

    <!-- 操作按钮 -->
    <div class="flex gap-3">
      <Button
        variant="primary"
        :loading="saving"
        :disabled="saving || success"
        @click="handleNext"
      >
        {{ success ? $t('setup.ruleSets.saved') : saving ? $t('common.saving') : selectedPresets.size > 0 ? $t('setup.ruleSets.addRuleSets', { count: selectedPresets.size }) : $t('setup.ruleSets.continue') }}
      </Button>

      <Button
        variant="ghost"
        :disabled="saving || success"
        @click="handleSkip"
      >
        {{ $t('setup.ruleSets.skip') }}
      </Button>
    </div>

    <!-- 说明信息 -->
    <Card padding="sm" class="bg-violet-50 border-violet-200">
      <div class="flex items-start space-x-3">
        <InformationCircleIcon class="h-5 w-5 text-violet-600 flex-shrink-0 mt-0.5" />
        <div class="text-sm text-violet-900 space-y-2">
          <p class="font-medium">{{ $t('setup.ruleSets.aboutHeading') }}</p>
          <ul class="list-disc list-inside space-y-1 ml-2 text-violet-800">
            <li><strong>{{ $t('setup.ruleSets.aboutGeositeLabel') }}</strong> {{ $t('setup.ruleSets.aboutGeosite') }}</li>
            <li><strong>{{ $t('setup.ruleSets.aboutGeoipLabel') }}</strong> {{ $t('setup.ruleSets.aboutGeoip') }}</li>
            <li><strong>{{ $t('setup.ruleSets.aboutFormatLabel') }}</strong> {{ $t('setup.ruleSets.aboutFormat') }}</li>
            <li>{{ $t('setup.ruleSets.aboutDownloaded') }}</li>
            <li>{{ $t('setup.ruleSets.aboutManage') }}</li>
          </ul>
          <p class="mt-2 text-xs text-violet-700">
            {{ $t('setup.ruleSets.tip') }}
          </p>
        </div>
      </div>
    </Card>

    <!-- 使用建议 -->
    <Card padding="sm" class="bg-gray-50">
      <div class="text-sm text-gray-600 space-y-2">
        <p class="font-medium text-gray-900">{{ $t('setup.ruleSets.recommendedHeading') }}</p>
        <ul class="list-disc list-inside space-y-1 ml-2">
          <li><strong>{{ $t('setup.ruleSets.recBasicLabel') }}</strong> {{ $t('setup.ruleSets.recBasic') }}</li>
          <li><strong>{{ $t('setup.ruleSets.recAdsLabel') }}</strong> {{ $t('setup.ruleSets.recAds') }}</li>
          <li><strong>{{ $t('setup.ruleSets.recServicesLabel') }}</strong> {{ $t('setup.ruleSets.recServices') }}</li>
        </ul>
      </div>
    </Card>
  </div>
</template>

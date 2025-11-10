<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { apiService } from '../../services/api'
import type { RuleSet } from '../../types/api'
import { Button, Alert, Card, Badge } from '../../components'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'

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
  name: string
  description: string
  type: 'remote'
  format: 'binary'
  url: string
  download_detour?: string
}

const presetRuleSets: PresetRuleSet[] = [
  {
    id: 'geosite-cn',
    tag: 'geosite-cn',
    name: 'GeoSite CN',
    description: 'Chinese domains (direct connection recommended)',
    type: 'remote',
    format: 'binary',
    url: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs',
  },
  {
    id: 'geosite-geolocation-!cn',
    tag: 'geosite-geolocation-!cn',
    name: 'GeoSite Non-CN',
    description: 'Non-Chinese domains (proxy recommended)',
    type: 'remote',
    format: 'binary',
    url: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-geolocation-!cn.srs',
  },
  {
    id: 'geosite-category-ads-all',
    tag: 'geosite-category-ads-all',
    name: 'Ad Blocking',
    description: 'Advertisement and tracking domains (block recommended)',
    type: 'remote',
    format: 'binary',
    url: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs',
  },
  {
    id: 'geoip-cn',
    tag: 'geoip-cn',
    name: 'GeoIP CN',
    description: 'Chinese IP addresses (direct connection recommended)',
    type: 'remote',
    format: 'binary',
    url: 'https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs',
  },
  {
    id: 'geosite-google',
    tag: 'geosite-google',
    name: 'Google Services',
    description: 'Google domains (proxy recommended)',
    type: 'remote',
    format: 'binary',
    url: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-google.srs',
  },
  {
    id: 'geosite-github',
    tag: 'geosite-github',
    name: 'GitHub',
    description: 'GitHub domains (proxy recommended)',
    type: 'remote',
    format: 'binary',
    url: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-github.srs',
  },
]

onMounted(async () => {
  await loadRuleSets()
})

const loadRuleSets = async () => {
  loading.value = true
  error.value = ''

  try {
    const ruleSets = await apiService.getRuleSets()
    existingRuleSets.value = ruleSets || []

    // 预选已存在的规则集
    ruleSets?.forEach((ruleSet) => {
      const preset = presetRuleSets.find(p => p.tag === ruleSet.tag)
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
  if (selectedPresets.value.has(presetId)) {
    selectedPresets.value.delete(presetId)
  } else {
    selectedPresets.value.add(presetId)
  }
}

const selectAll = () => {
  presetRuleSets.forEach(preset => {
    selectedPresets.value.add(preset.id)
  })
}

const deselectAll = () => {
  selectedPresets.value.clear()
}

const saveRuleSets = async () => {
  saving.value = true
  error.value = ''
  success.value = false

  try {
    // 获取当前已存在的规则集标签
    const existingTags = new Set(existingRuleSets.value.map(rs => rs.tag))

    // 添加选中的预设规则集
    for (const preset of presetRuleSets) {
      if (selectedPresets.value.has(preset.id)) {
        // 如果不存在，则添加
        if (!existingTags.has(preset.tag)) {
          const ruleSet: RuleSet = {
            tag: preset.tag,
            type: preset.type,
            format: preset.format,
            url: preset.url,
          }
          await apiService.addRuleSet(ruleSet)
        }
      } else {
        // 如果已存在但未选中，则删除
        if (existingTags.has(preset.tag)) {
          await apiService.deleteRuleSet(preset.tag)
        }
      }
    }

    success.value = true

    // 2秒后自动进入下一步
    setTimeout(() => {
      emit('next')
    }, 2000)
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || 'Failed to save rule sets'
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
    <Alert v-if="success" type="success" title="Rule Sets Configured">
      Rule sets have been configured successfully. Proceeding to next step...
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
            <h3 class="text-lg font-semibold text-gray-900 mb-2">Select Rule Sets</h3>
            <p class="text-sm text-gray-600">
              Choose commonly used rule sets for routing. You can add custom rule sets later.
            </p>
          </div>
          <div class="flex gap-2">
            <Button
              variant="secondary"
              size="sm"
              :disabled="loading || saving || success"
              @click="selectAll"
            >
              Select All
            </Button>
            <Button
              variant="ghost"
              size="sm"
              :disabled="loading || saving || success"
              @click="deselectAll"
            >
              Clear
            </Button>
          </div>
        </div>

        <!-- 规则集列表 -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div
            v-for="preset in presetRuleSets"
            :key="preset.id"
            class="flex items-start gap-3 p-4 border border-gray-200 rounded-lg hover:border-blue-300 hover:bg-blue-50 cursor-pointer transition-colors"
            :class="{ 'border-blue-500 bg-blue-50': selectedPresets.has(preset.id) }"
            @click="togglePreset(preset.id)"
          >
            <input
              type="checkbox"
              :checked="selectedPresets.has(preset.id)"
              class="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500 mt-0.5"
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
            <span class="font-semibold text-gray-900">{{ selectedPresets.size }}</span> rule sets selected
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
        {{ success ? 'Saved' : saving ? 'Saving...' : selectedPresets.size > 0 ? `Add ${selectedPresets.size} Rule Sets` : 'Continue' }}
      </Button>

      <Button
        variant="ghost"
        :disabled="saving || success"
        @click="handleSkip"
      >
        Skip this step
      </Button>
    </div>

    <!-- 说明信息 -->
    <Card padding="sm" class="bg-blue-50 border-blue-200">
      <div class="flex items-start space-x-3">
        <InformationCircleIcon class="h-5 w-5 text-blue-600 flex-shrink-0 mt-0.5" />
        <div class="text-sm text-blue-900 space-y-2">
          <p class="font-medium">About Rule Sets:</p>
          <ul class="list-disc list-inside space-y-1 ml-2 text-blue-800">
            <li><strong>GeoSite:</strong> Domain-based rules (e.g., cn = Chinese domains)</li>
            <li><strong>GeoIP:</strong> IP address-based rules (e.g., cn = Chinese IPs)</li>
            <li><strong>Format:</strong> Binary (.srs) format for better performance</li>
            <li>Rule sets are downloaded from official repositories and cached locally</li>
            <li>You can add custom rule sets or manage them in the dashboard later</li>
          </ul>
          <p class="mt-2 text-xs text-blue-700">
            💡 Tip: Select "GeoSite CN" + "GeoIP CN" for basic routing (direct for CN, proxy for others)
          </p>
        </div>
      </div>
    </Card>

    <!-- 使用建议 -->
    <Card padding="sm" class="bg-gray-50">
      <div class="text-sm text-gray-600 space-y-2">
        <p class="font-medium text-gray-900">Recommended Combinations:</p>
        <ul class="list-disc list-inside space-y-1 ml-2">
          <li><strong>Basic Setup:</strong> GeoSite CN + GeoIP CN (direct CN traffic)</li>
          <li><strong>Ad Blocking:</strong> Add "Ad Blocking" rule set (block ads)</li>
          <li><strong>Specific Services:</strong> Add Google/GitHub rule sets for better routing</li>
        </ul>
      </div>
    </Card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Inbound } from '../../types/api'
import { Button, Alert, Card, Badge } from '../../components'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'
import { inboundService } from '../../services'
import { RECOMMENDED_TUN_ADDRESS, RECOMMENDED_TUN_ADDRESS_V6 } from '../../composables/useSetupDefaults'

const RECOMMENDED_TUN_ADDRESS_ALL = [RECOMMENDED_TUN_ADDRESS, RECOMMENDED_TUN_ADDRESS_V6]

const { t } = useI18n()

const emit = defineEmits<{
  next: []
  prev: []
}>()

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref(false)

const selectedInbounds = ref<Set<string>>(new Set())

// 预设入站配置
interface InboundPreset {
  id: string
  tag: string
  name: string
  description: string
  type: string
  listen: string
  listen_port: number
  config: Partial<Inbound>
}

// Display name/description re-translate on locale switch; config/type/listen
// values sent to the backend stay unchanged.
const inboundPresets = computed<InboundPreset[]>(() => [
  {
    id: 'mixed',
    tag: 'mixed-in',
    name: t('setup.inbounds.presets.mixed.name'),
    description: t('setup.inbounds.presets.mixed.description'),
    type: 'mixed',
    listen: '127.0.0.1',
    listen_port: 7890,
    config: {
      tag: 'mixed-in',
      type: 'mixed',
      listen: '127.0.0.1',
      listen_port: 7890,
      sniff: true,
      sniff_override_destination: true,
      domain_strategy: 'prefer_ipv4',
    },
  },
  {
    id: 'http',
    tag: 'http-in',
    name: t('setup.inbounds.presets.http.name'),
    description: t('setup.inbounds.presets.http.description'),
    type: 'http',
    listen: '127.0.0.1',
    listen_port: 7891,
    config: {
      tag: 'http-in',
      type: 'http',
      listen: '127.0.0.1',
      listen_port: 7891,
      sniff: true,
      sniff_override_destination: true,
    },
  },
  {
    id: 'socks',
    tag: 'socks-in',
    name: t('setup.inbounds.presets.socks.name'),
    description: t('setup.inbounds.presets.socks.description'),
    type: 'socks',
    listen: '127.0.0.1',
    listen_port: 7892,
    config: {
      tag: 'socks-in',
      type: 'socks',
      listen: '127.0.0.1',
      listen_port: 7892,
      sniff: true,
      sniff_override_destination: true,
    },
  },
  {
    id: 'tun',
    tag: 'tun-in',
    name: t('setup.inbounds.presets.tun.name'),
    description: t('setup.inbounds.presets.tun.description'),
    type: 'tun',
    listen: '',
    listen_port: 0,
    config: {
      tag: 'tun-in',
      type: 'tun',
      interface_name: 'tun0',
      // Not sing-box's own 172.19.0.1/30 default: Docker allocates bridges
      // from 172.17.0.0/16 up, so on a router running Docker that default
      // collides with a live bridge. See useSetupDefaults.
      address: [...RECOMMENDED_TUN_ADDRESS_ALL],
      mtu: 9000,
      auto_route: true,
      strict_route: true,
      sniff: true,
      sniff_override_destination: true,
    },
  },
])

onMounted(async () => {
  await loadInbounds()
})

const loadInbounds = async () => {
  loading.value = true
  error.value = ''

  try {
    const {data} = await inboundService.getInbounds()
    let inbounds = data.inbounds
    if (!inbounds) {
      inbounds = []
    }


    // 预选已存在的入站
    inbounds?.forEach((inbound) => {
      const preset = inboundPresets.value.find(p => p.tag === inbound.tag)
      if (preset) {
        selectedInbounds.value.add(preset.id)
      }
    })
  } catch (err: any) {
    console.log('No existing inbounds found')
  } finally {
    loading.value = false
  }
}

const toggleInbound = (presetId: string) => {
  const next = new Set(selectedInbounds.value)
  if (next.has(presetId)) next.delete(presetId); else next.add(presetId)
  selectedInbounds.value = next
}

const selectRecommended = () => {
  // 推荐选择 mixed
  selectedInbounds.value = new Set(['mixed'])
}

// Track inbounds we have created so far in this batch. If a later op fails,
// we undo by deleting them. Pre-existing inbounds that we deleted are
// re-added from the snapshot. Best-effort: if any restore call fails we
// surface a clear message pointing the user at /config/rollback.
const saveInbounds = async () => {
  if (selectedInbounds.value.size === 0) {
    error.value = t('setup.inbounds.selectAtLeastOneError')
    return
  }

  saving.value = true
  error.value = ''
  success.value = false

  let inboundsSnapshot: Inbound[] = []
  try {
    const { data } = await inboundService.getInbounds()
    inboundsSnapshot = data.inbounds || []
  } catch (err: any) {
    error.value = err.message || t('setup.inbounds.snapshotFailed')
    saving.value = false
    return
  }

  const existingByTag = new Map(inboundsSnapshot.map((ib) => [ib.tag, ib]))
  const addedTags: string[] = []
  const removedSnapshots: Inbound[] = []

  try {
    for (const preset of inboundPresets.value) {
      const wanted = selectedInbounds.value.has(preset.id)
      const present = existingByTag.has(preset.tag)
      if (wanted && !present) {
        await inboundService.addInbound(preset.config as Inbound)
        addedTags.push(preset.tag)
      } else if (!wanted && present) {
        removedSnapshots.push(existingByTag.get(preset.tag)!)
        await inboundService.deleteInbound(preset.tag)
      }
    }

    success.value = true
    setTimeout(() => emit('next'), 2000)
  } catch (err: any) {
    let restoreOk = true
    // Undo additions
    for (const tag of addedTags) {
      try { await inboundService.deleteInbound(tag) } catch { restoreOk = false }
    }
    // Re-add deletions
    for (const ib of removedSnapshots) {
      try { await inboundService.addInbound(ib) } catch { restoreOk = false }
    }
    const baseMsg = err.message || t('setup.inbounds.saveFailed')
    error.value = restoreOk
      ? t('setup.inbounds.restored', { message: baseMsg })
      : t('setup.inbounds.restoreFailed', { message: baseMsg })
  } finally {
    saving.value = false
  }
}

const handleNext = () => {
  if (!success.value) {
    saveInbounds()
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
    <Alert v-if="success" type="success" :title="$t('setup.inbounds.successTitle')">
      {{ $t('setup.inbounds.successDesc') }}
    </Alert>

    <!-- 错误提示 -->
    <Alert v-if="error" type="error" closable @close="error = ''">
      {{ error }}
    </Alert>

    <!-- 入站配置选择 -->
    <Card>
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 mb-2">{{ $t('setup.inbounds.selectHeading') }}</h3>
            <p class="text-sm text-gray-600">
              {{ $t('setup.inbounds.selectDesc') }}
            </p>
          </div>
          <Button
            variant="secondary"
            size="sm"
            :disabled="loading || saving || success"
            @click="selectRecommended"
          >
            {{ $t('setup.inbounds.useRecommended') }}
          </Button>
        </div>

        <!-- 入站列表 -->
        <div class="space-y-3">
          <div
            v-for="preset in inboundPresets"
            :key="preset.id"
            class="flex items-start gap-3 p-4 border border-gray-200 rounded-control hover:border-primary-300 hover:bg-primary-50 cursor-pointer transition-colors"
            :class="{ 'border-primary-500 bg-primary-50': selectedInbounds.has(preset.id) }"
            @click="toggleInbound(preset.id)"
          >
            <input
              type="checkbox"
              :checked="selectedInbounds.has(preset.id)"
              class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500 mt-0.5"
              :disabled="loading || saving || success"
              @click.stop="toggleInbound(preset.id)"
            />
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <p class="text-sm font-medium text-gray-900">{{ preset.name }}</p>
                <Badge v-if="preset.id === 'mixed'" variant="primary" size="sm">{{ $t('setup.inbounds.recommended') }}</Badge>
                <Badge variant="gray" size="sm">{{ preset.type }}</Badge>
              </div>
              <p class="text-xs text-gray-600 mt-1">{{ preset.description }}</p>
              <div class="flex items-center gap-4 mt-2 text-xs text-gray-500">
                <span v-if="preset.listen">{{ $t('setup.inbounds.listen', { address: preset.listen, port: preset.listen_port }) }}</span>
                <span v-if="preset.type === 'tun'">{{ $t('setup.inbounds.interface', { name: (preset.config as any).interface_name }) }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 选择统计 -->
        <div class="bg-gray-50 rounded-control p-3 text-center">
          <p class="text-sm text-gray-600">
            <span class="font-semibold text-gray-900">{{ selectedInbounds.size }}</span> {{ $t('setup.inbounds.selectedCount') }}
          </p>
        </div>
      </div>
    </Card>

    <!-- 操作按钮 -->
    <div class="flex gap-3">
      <Button
        variant="primary"
        :loading="saving"
        :disabled="saving || success || selectedInbounds.size === 0"
        @click="handleNext"
      >
        {{ success ? $t('setup.inbounds.saved') : saving ? $t('common.saving') : selectedInbounds.size > 0 ? $t('setup.inbounds.addInbounds', { count: selectedInbounds.size }) : $t('setup.inbounds.selectAtLeastOne') }}
      </Button>

      <Button
        variant="ghost"
        :disabled="saving || success"
        @click="handleSkip"
      >
        {{ $t('setup.inbounds.skip') }}
      </Button>
    </div>

    <!-- 说明信息 -->
    <Card padding="sm" class="bg-primary-50 border-primary-200">
      <div class="flex items-start space-x-3">
        <InformationCircleIcon class="h-5 w-5 text-primary-600 flex-shrink-0 mt-0.5" />
        <div class="text-sm text-primary-900 space-y-2">
          <p class="font-medium">{{ $t('setup.inbounds.aboutHeading') }}</p>
          <ul class="list-disc list-inside space-y-1 ml-2 text-primary-800">
            <li><strong>{{ $t('setup.inbounds.aboutMixedLabel') }}</strong> {{ $t('setup.inbounds.aboutMixed') }}</li>
            <li><strong>{{ $t('setup.inbounds.aboutHttpLabel') }}</strong> {{ $t('setup.inbounds.aboutHttp') }}</li>
            <li><strong>{{ $t('setup.inbounds.aboutSocksLabel') }}</strong> {{ $t('setup.inbounds.aboutSocks') }}</li>
            <li><strong>{{ $t('setup.inbounds.aboutTunLabel') }}</strong> {{ $t('setup.inbounds.aboutTun') }}</li>
            <li>{{ $t('setup.inbounds.aboutSniff') }}</li>
          </ul>
          <p class="mt-2 text-xs text-primary-700">
            {{ $t('setup.inbounds.tip') }}
          </p>
        </div>
      </div>
    </Card>

    <!-- 端口说明 -->
    <Card padding="sm" class="bg-gray-50">
      <div class="text-sm text-gray-600 space-y-2">
        <p class="font-medium text-gray-900">{{ $t('setup.inbounds.portsHeading') }}</p>
        <ul class="list-disc list-inside space-y-1 ml-2">
          <li><strong>{{ $t('setup.inbounds.portMixedLabel') }}</strong> {{ $t('setup.inbounds.portMixed') }}</li>
          <li><strong>{{ $t('setup.inbounds.portHttpLabel') }}</strong> {{ $t('setup.inbounds.portHttp') }}</li>
          <li><strong>{{ $t('setup.inbounds.portSocksLabel') }}</strong> {{ $t('setup.inbounds.portSocks') }}</li>
          <li><strong>{{ $t('setup.inbounds.portTunLabel') }}</strong> {{ $t('setup.inbounds.portTun') }}</li>
        </ul>
        <p class="text-xs text-gray-500 mt-2">
          {{ $t('setup.inbounds.portsHint') }}
        </p>
      </div>
    </Card>
  </div>
</template>

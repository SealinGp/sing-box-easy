<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { Inbound } from '../../types/api'
import { Button, Alert, Card, Badge } from '../../components'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'
import { inboundService } from '../../services'

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

const inboundPresets: InboundPreset[] = [
  {
    id: 'mixed',
    tag: 'mixed-in',
    name: 'Mixed Proxy (HTTP + SOCKS5)',
    description: 'Best for most use cases, supports both HTTP and SOCKS5',
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
    name: 'HTTP Proxy',
    description: 'HTTP proxy protocol only',
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
    name: 'SOCKS5 Proxy',
    description: 'SOCKS5 proxy protocol only',
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
    name: 'TUN (Virtual Network)',
    description: 'System-wide proxy via virtual network interface (requires root/admin)',
    type: 'tun',
    listen: '',
    listen_port: 0,
    config: {
      tag: 'tun-in',
      type: 'tun',
      interface_name: 'tun0',
      address: [
        "172.19.0.1/30",
        "fdfe:dcba:9876::1/126"
      ],
      mtu: 9000,
      auto_route: true,
      strict_route: true,
      sniff: true,
      sniff_override_destination: true,
    },
  },
]

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
      const preset = inboundPresets.find(p => p.tag === inbound.tag)
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
    error.value = 'Please select at least one inbound'
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
    error.value = err.message || 'Failed to snapshot existing inbounds'
    saving.value = false
    return
  }

  const existingByTag = new Map(inboundsSnapshot.map((ib) => [ib.tag, ib]))
  const addedTags: string[] = []
  const removedSnapshots: Inbound[] = []

  try {
    for (const preset of inboundPresets) {
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
    error.value = restoreOk
      ? `${err.message || 'Failed to save inbounds'} (previous inbounds restored)`
      : `${err.message || 'Failed to save inbounds'} (restore also failed — use /config/rollback to recover)`
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
    <Alert v-if="success" type="success" title="Inbounds Configured">
      Inbound configurations have been saved successfully. Proceeding to next step...
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
            <h3 class="text-lg font-semibold text-gray-900 mb-2">Select Inbound Protocols</h3>
            <p class="text-sm text-gray-600">
              Choose which proxy protocols to accept incoming connections.
            </p>
          </div>
          <Button
            variant="secondary"
            size="sm"
            :disabled="loading || saving || success"
            @click="selectRecommended"
          >
            Use Recommended
          </Button>
        </div>

        <!-- 入站列表 -->
        <div class="space-y-3">
          <div
            v-for="preset in inboundPresets"
            :key="preset.id"
            class="flex items-start gap-3 p-4 border border-gray-200 rounded-lg hover:border-violet-300 hover:bg-violet-50 cursor-pointer transition-colors"
            :class="{ 'border-violet-500 bg-violet-50': selectedInbounds.has(preset.id) }"
            @click="toggleInbound(preset.id)"
          >
            <input
              type="checkbox"
              :checked="selectedInbounds.has(preset.id)"
              class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500 mt-0.5"
              :disabled="loading || saving || success"
              @click.stop="toggleInbound(preset.id)"
            />
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <p class="text-sm font-medium text-gray-900">{{ preset.name }}</p>
                <Badge v-if="preset.id === 'mixed'" variant="primary" size="sm">Recommended</Badge>
                <Badge variant="gray" size="sm">{{ preset.type }}</Badge>
              </div>
              <p class="text-xs text-gray-600 mt-1">{{ preset.description }}</p>
              <div class="flex items-center gap-4 mt-2 text-xs text-gray-500">
                <span v-if="preset.listen">Listen: {{ preset.listen }}:{{ preset.listen_port }}</span>
                <span v-if="preset.type === 'tun'">Interface: {{ (preset.config as any).interface_name }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 选择统计 -->
        <div class="bg-gray-50 rounded-lg p-3 text-center">
          <p class="text-sm text-gray-600">
            <span class="font-semibold text-gray-900">{{ selectedInbounds.size }}</span> inbound(s) selected
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
        {{ success ? 'Saved' : saving ? 'Saving...' : selectedInbounds.size > 0 ? `Add ${selectedInbounds.size} Inbound(s)` : 'Select at least one' }}
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
    <Card padding="sm" class="bg-violet-50 border-violet-200">
      <div class="flex items-start space-x-3">
        <InformationCircleIcon class="h-5 w-5 text-violet-600 flex-shrink-0 mt-0.5" />
        <div class="text-sm text-violet-900 space-y-2">
          <p class="font-medium">About Inbound Protocols:</p>
          <ul class="list-disc list-inside space-y-1 ml-2 text-violet-800">
            <li><strong>Mixed:</strong> Accepts both HTTP and SOCKS5 connections (most versatile)</li>
            <li><strong>HTTP:</strong> Standard HTTP proxy protocol (widely supported)</li>
            <li><strong>SOCKS5:</strong> SOCKS5 proxy protocol (supports UDP)</li>
            <li><strong>TUN:</strong> Virtual network interface for system-wide proxy (requires elevated privileges)</li>
            <li>All inbounds enable traffic sniffing for better routing</li>
          </ul>
          <p class="mt-2 text-xs text-violet-700">
            💡 Tip: Start with "Mixed" for the best compatibility. You can add more later.
          </p>
        </div>
      </div>
    </Card>

    <!-- 端口说明 -->
    <Card padding="sm" class="bg-gray-50">
      <div class="text-sm text-gray-600 space-y-2">
        <p class="font-medium text-gray-900">Default Ports:</p>
        <ul class="list-disc list-inside space-y-1 ml-2">
          <li><strong>Mixed:</strong> 127.0.0.1:7890 (HTTP + SOCKS5)</li>
          <li><strong>HTTP:</strong> 127.0.0.1:7891</li>
          <li><strong>SOCKS5:</strong> 127.0.0.1:7892</li>
          <li><strong>TUN:</strong> Virtual interface tun0 (no port)</li>
        </ul>
        <p class="text-xs text-gray-500 mt-2">
          Configure your applications to use these addresses as proxy settings.
        </p>
      </div>
    </Card>
  </div>
</template>

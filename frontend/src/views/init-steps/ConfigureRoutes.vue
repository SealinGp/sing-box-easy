<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { apiService } from '../../services/api'
import type { RouteRule, Outbound } from '../../types/api'
import { Button, Alert, Card, Badge, Select } from '../../components'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'

const emit = defineEmits<{
  next: []
  prev: []
}>()

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref(false)

const selectedPreset = ref<string>('smart')
const availableOutbounds = ref<Outbound[]>([])
const selectedProxyOutbound = ref<string>('')

// 路由策略预设
interface RoutePreset {
  id: string
  name: string
  description: string
  rules: Partial<RouteRule>[]
  final: string
}

const routePresets = computed<RoutePreset[]>(() => {
  const proxyOutbound = selectedProxyOutbound.value || 'proxy-group'

  return [
    {
      id: 'smart',
      name: 'Smart Routing (Recommended)',
      description: 'Route traffic based on rules: Block ads, direct CN traffic, proxy others',
      rules: [
        {
          rule_set: ['geosite-category-ads-all'],
          outbound: 'block',
        },
        {
          rule_set: ['geosite-cn', 'geoip-cn'],
          outbound: 'direct',
        },
      ],
      final: proxyOutbound,
    },
    {
      id: 'proxy-all',
      name: 'Global Proxy',
      description: 'Route all traffic through proxy (except private IPs)',
      rules: [
        {
          ip_cidr: ['10.0.0.0/8', '172.16.0.0/12', '192.168.0.0/16', '127.0.0.0/8'],
          outbound: 'direct',
        },
      ],
      final: proxyOutbound,
    },
    {
      id: 'direct-all',
      name: 'Direct Connection',
      description: 'Route all traffic directly without proxy',
      rules: [],
      final: 'direct',
    },
    {
      id: 'gfwlist',
      name: 'GFWList Mode',
      description: 'Proxy non-CN domains, direct CN traffic',
      rules: [
        {
          rule_set: ['geosite-category-ads-all'],
          outbound: 'block',
        },
        {
          rule_set: ['geosite-cn', 'geoip-cn'],
          outbound: 'direct',
        },
        {
          rule_set: ['geosite-geolocation-!cn'],
          outbound: proxyOutbound,
        },
      ],
      final: 'direct',
    },
  ]
})

onMounted(async () => {
  await loadConfiguration()
})

const loadConfiguration = async () => {
  loading.value = true
  error.value = ''

  try {
    // 加载可用的出站节点
    const outbounds = await apiService.getOutbounds()
    availableOutbounds.value = outbounds || []

    // 查找第一个代理节点或选择器组
    const proxyNode = outbounds.find(ob =>
      ob.type === 'selector' ||
      ob.type === 'urltest' ||
      ['shadowsocks', 'vmess', 'trojan', 'vless', 'hysteria', 'hysteria2'].includes(ob.type)
    )

    if (proxyNode) {
      selectedProxyOutbound.value = proxyNode.tag
    }

    // 尝试识别当前配置
    const existingRules = await apiService.getRouteRules()
    const finalOutbound = await apiService.getRouteFinal()

    if (existingRules.length > 0) {
      // 尝试匹配预设
      const preset = routePresets.value.find(p => {
        if (p.rules.length !== existingRules.length) return false
        return p.final === finalOutbound.final
      })

      if (preset) {
        selectedPreset.value = preset.id
      }
    }
  } catch (err: any) {
    console.log('No existing route config found')
  } finally {
    loading.value = false
  }
}

const proxyOutboundOptions = computed(() => {
  return availableOutbounds.value
    .filter(ob => ob.type !== 'direct' && ob.type !== 'block')
    .map(ob => ({
      label: `${ob.tag} (${ob.type})`,
      value: ob.tag,
    }))
})

const saveRouteConfig = async () => {
  saving.value = true
  error.value = ''
  success.value = false

  try {
    const preset = routePresets.value.find(p => p.id === selectedPreset.value)
    if (!preset) {
      error.value = 'Invalid route preset selected'
      return
    }

    // 删除所有现有规则
    const existingRules = await apiService.getRouteRules()
    for (let i = existingRules.length - 1; i >= 0; i--) {
      await apiService.deleteRouteRule(i)
    }

    // 添加新规则
    for (const rule of preset.rules) {
      await apiService.addRouteRule(rule as RouteRule)
    }

    // 设置最终出站
    await apiService.updateRouteFinal(preset.final)

    success.value = true

    // 2秒后自动进入下一步
    setTimeout(() => {
      emit('next')
    }, 2000)
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || 'Failed to save route configuration'
  } finally {
    saving.value = false
  }
}

const handleNext = () => {
  if (!success.value) {
    saveRouteConfig()
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
    <Alert v-if="success" type="success" title="Routes Configured">
      Route configuration has been saved successfully. Proceeding to next step...
    </Alert>

    <!-- 错误提示 -->
    <Alert v-if="error" type="error" closable @close="error = ''">
      {{ error }}
    </Alert>

    <!-- 代理出站选择 -->
    <Card v-if="proxyOutboundOptions.length > 0">
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 mb-2">Select Proxy Outbound</h3>
          <p class="text-sm text-gray-600">
            Choose which outbound to use for proxied traffic.
          </p>
        </div>

        <Select
          v-model="selectedProxyOutbound"
          :options="proxyOutboundOptions"
          label="Proxy Outbound"
          :disabled="loading || saving || success"
        />
      </div>
    </Card>

    <!-- 路由策略选择 -->
    <Card>
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 mb-2">Select Routing Strategy</h3>
          <p class="text-sm text-gray-600">
            Choose how traffic should be routed through different outbounds.
          </p>
        </div>

        <!-- 预设策略列表 -->
        <div class="space-y-3">
          <div
            v-for="preset in routePresets"
            :key="preset.id"
            class="flex items-start gap-3 p-4 border border-gray-200 rounded-lg hover:border-blue-300 hover:bg-blue-50 cursor-pointer transition-colors"
            :class="{ 'border-blue-500 bg-blue-50': selectedPreset === preset.id }"
            @click="selectedPreset = preset.id"
          >
            <input
              type="radio"
              :checked="selectedPreset === preset.id"
              class="w-4 h-4 text-blue-600 border-gray-300 mt-0.5"
              :disabled="loading || saving || success"
              @click.stop="selectedPreset = preset.id"
            />
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <p class="text-sm font-medium text-gray-900">{{ preset.name }}</p>
                <Badge v-if="preset.id === 'smart'" variant="primary" size="sm">Recommended</Badge>
              </div>
              <p class="text-xs text-gray-600 mt-1">{{ preset.description }}</p>
              <div class="flex items-center gap-2 mt-2">
                <Badge variant="gray" size="sm">{{ preset.rules.length }} rules</Badge>
                <Badge variant="gray" size="sm">Final: {{ preset.final }}</Badge>
              </div>
            </div>
          </div>
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
        {{ success ? 'Saved' : saving ? 'Saving...' : 'Save & Continue' }}
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
          <p class="font-medium">About Routing Strategies:</p>
          <ul class="list-disc list-inside space-y-1 ml-2 text-blue-800">
            <li><strong>Smart Routing:</strong> Blocks ads, directs CN traffic, proxies everything else (best for China users)</li>
            <li><strong>Global Proxy:</strong> Routes all traffic through proxy except private IPs</li>
            <li><strong>Direct Connection:</strong> No proxy, all traffic goes directly</li>
            <li><strong>GFWList Mode:</strong> Only proxy non-CN domains, direct for CN traffic</li>
            <li>Rules are evaluated in order, first match wins</li>
            <li>Final outbound is used when no rules match</li>
          </ul>
          <p class="mt-2 text-xs text-blue-700">
            💡 Tip: Use "Smart Routing" for balanced performance and privacy.
          </p>
        </div>
      </div>
    </Card>

    <!-- 规则详情 -->
    <Card padding="sm" class="bg-gray-50">
      <div class="text-sm text-gray-600 space-y-2">
        <p class="font-medium text-gray-900">Selected Strategy Rules:</p>
        <div v-if="routePresets.find(p => p.id === selectedPreset)">
          <ul class="list-disc list-inside space-y-1 ml-2">
            <li
              v-for="(rule, index) in routePresets.find(p => p.id === selectedPreset)!.rules"
              :key="index"
            >
              <strong v-if="rule.rule_set">Rule Set {{ rule.rule_set.join(', ') }}:</strong>
              <strong v-else-if="rule.ip_cidr">Private IPs:</strong>
              → {{ rule.outbound }}
            </li>
            <li class="text-gray-700">
              <strong>All other traffic:</strong> → {{ routePresets.find(p => p.id === selectedPreset)!.final }}
            </li>
          </ul>
        </div>
      </div>
    </Card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RouteRule, Outbound } from '../../types/api'
import { Button, Alert, Card, Badge } from '../../components'
import { Select } from '../../volt'
import { FILTER_THRESHOLD } from '../../utils/selectFilter'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'
import { outboundService, routeService } from '../../services';

const { t } = useI18n()

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
      name: t('setup.routes.presets.smart.name'),
      description: t('setup.routes.presets.smart.description'),
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
      name: t('setup.routes.presets.proxyAll.name'),
      description: t('setup.routes.presets.proxyAll.description'),
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
      name: t('setup.routes.presets.directAll.name'),
      description: t('setup.routes.presets.directAll.description'),
      rules: [],
      final: 'direct',
    },
    {
      id: 'gfwlist',
      name: t('setup.routes.presets.gfwlist.name'),
      description: t('setup.routes.presets.gfwlist.description'),
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
    const {data} = await outboundService.getOutbounds()
    const outbounds = data.outbounds
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
    const resp = await routeService.getRouteRules()
    const existingRules = resp.data.rules    
    const resp1 = await routeService.getRouteFinal()
    const finalOutbound = resp1.data

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

// Best-effort restore of the route table to a snapshot. Used when a multi-op
// batch fails partway through and we want to leave the user where they
// started. Returns true on full restore, false on partial.
const restoreRouteSnapshot = async (
  rulesSnapshot: RouteRule[],
  finalSnapshot: string,
): Promise<boolean> => {
  try {
    const { data: cur } = await routeService.getRouteRules()
    for (let i = (cur.rules || []).length - 1; i >= 0; i--) {
      await routeService.deleteRouteRule(i)
    }
    for (const rule of rulesSnapshot) {
      await routeService.addRouteRule(rule)
    }
    if (finalSnapshot) await routeService.updateRouteFinal(finalSnapshot)
    return true
  } catch {
    return false
  }
}

const saveRouteConfig = async () => {
  saving.value = true
  error.value = ''
  success.value = false

  const preset = routePresets.value.find(p => p.id === selectedPreset.value)
  if (!preset) {
    error.value = t('setup.routes.invalidPreset')
    saving.value = false
    return
  }

  // Snapshot the existing config so we can restore on partial failure. The
  // server's /config/rollback only undoes one step, which is not enough for a
  // multi-rule batch — keep the snapshot in JS and restore explicitly.
  let rulesSnapshot: RouteRule[] = []
  let finalSnapshot = ''
  try {
    const [rulesResp, finalResp] = await Promise.all([
      routeService.getRouteRules(),
      routeService.getRouteFinal(),
    ])
    rulesSnapshot = rulesResp.data.rules || []
    finalSnapshot = finalResp.data.final || ''
  } catch (err: any) {
    error.value = err.message || t('setup.routes.snapshotFailed')
    saving.value = false
    return
  }

  try {
    for (let i = rulesSnapshot.length - 1; i >= 0; i--) {
      await routeService.deleteRouteRule(i)
    }
    for (const rule of preset.rules) {
      await routeService.addRouteRule(rule as RouteRule)
    }
    await routeService.updateRouteFinal(preset.final)

    success.value = true
    setTimeout(() => emit('next'), 2000)
  } catch (err: any) {
    const restored = await restoreRouteSnapshot(rulesSnapshot, finalSnapshot)
    const baseMsg = err.message || t('setup.routes.saveFailed')
    error.value = restored
      ? t('setup.routes.restored', { message: baseMsg })
      : t('setup.routes.restoreFailed', { message: baseMsg })
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
  <div class="space-y-4">
    <!-- 成功提示 -->
    <Alert v-if="success" type="success" :title="$t('setup.routes.successTitle')">
      {{ $t('setup.routes.successDesc') }}
    </Alert>

    <!-- 错误提示 -->
    <Alert v-if="error" type="error" closable @close="error = ''">
      {{ error }}
    </Alert>

    <!-- 代理出站选择 -->
    <Card v-if="proxyOutboundOptions.length > 0">
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 mb-2">{{ $t('setup.routes.proxyHeading') }}</h3>
          <p class="text-sm text-gray-600">
            {{ $t('setup.routes.proxyDesc') }}
          </p>
        </div>

        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('setup.routes.proxyOutbound') }}</label>
        <Select
          class="w-full"
          v-model="selectedProxyOutbound"
          :options="proxyOutboundOptions"
          optionLabel="label"
          optionValue="value"
          :filter="proxyOutboundOptions.length >= FILTER_THRESHOLD"
          :filterPlaceholder="$t('common.search')"
          :emptyFilterMessage="$t('common.noMatch')"
          :disabled="loading || saving || success"
        />
      </div>
    </Card>

    <!-- 路由策略选择 -->
    <Card>
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 mb-2">{{ $t('setup.routes.strategyHeading') }}</h3>
          <p class="text-sm text-gray-600">
            {{ $t('setup.routes.strategyDesc') }}
          </p>
        </div>

        <!-- 预设策略列表 -->
        <div class="space-y-3">
          <div
            v-for="preset in routePresets"
            :key="preset.id"
            class="flex items-start gap-3 p-4 border border-gray-200 rounded-control hover:border-primary-300 hover:bg-primary-50 cursor-pointer transition-colors"
            :class="{ 'border-primary-500 bg-primary-50': selectedPreset === preset.id }"
            @click="selectedPreset = preset.id"
          >
            <input
              type="radio"
              :checked="selectedPreset === preset.id"
              class="w-4 h-4 text-primary-600 border-gray-300 mt-0.5"
              :disabled="loading || saving || success"
              @click.stop="selectedPreset = preset.id"
            />
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <p class="text-sm font-medium text-gray-900">{{ preset.name }}</p>
                <Badge v-if="preset.id === 'smart'" variant="primary" size="sm">{{ $t('setup.routes.recommended') }}</Badge>
              </div>
              <p class="text-xs text-gray-600 mt-1">{{ preset.description }}</p>
              <div class="flex items-center gap-2 mt-2">
                <Badge variant="gray" size="sm">{{ $t('setup.routes.rulesCount', { count: preset.rules.length }) }}</Badge>
                <Badge variant="gray" size="sm">{{ $t('setup.routes.final', { outbound: preset.final }) }}</Badge>
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
        {{ success ? $t('setup.routes.saved') : saving ? $t('common.saving') : $t('setup.routes.saveContinue') }}
      </Button>

      <Button
        variant="ghost"
        :disabled="saving || success"
        @click="handleSkip"
      >
        {{ $t('setup.routes.skip') }}
      </Button>
    </div>

    <!-- 说明信息 -->
    <Card padding="sm" class="bg-primary-50 border-primary-200">
      <div class="flex items-start space-x-3">
        <InformationCircleIcon class="h-5 w-5 text-primary-600 flex-shrink-0 mt-0.5" />
        <div class="text-sm text-primary-900 space-y-2">
          <p class="font-medium">{{ $t('setup.routes.aboutHeading') }}</p>
          <ul class="list-disc list-inside space-y-1 ml-2 text-primary-800">
            <li><strong>{{ $t('setup.routes.aboutSmartLabel') }}</strong> {{ $t('setup.routes.aboutSmart') }}</li>
            <li><strong>{{ $t('setup.routes.aboutGlobalLabel') }}</strong> {{ $t('setup.routes.aboutGlobal') }}</li>
            <li><strong>{{ $t('setup.routes.aboutDirectLabel') }}</strong> {{ $t('setup.routes.aboutDirect') }}</li>
            <li><strong>{{ $t('setup.routes.aboutGfwlistLabel') }}</strong> {{ $t('setup.routes.aboutGfwlist') }}</li>
            <li>{{ $t('setup.routes.aboutOrder') }}</li>
            <li>{{ $t('setup.routes.aboutFinal') }}</li>
          </ul>
          <p class="mt-2 text-xs text-primary-700">
            {{ $t('setup.routes.tip') }}
          </p>
        </div>
      </div>
    </Card>

    <!-- 规则详情 -->
    <Card padding="sm" class="bg-gray-50">
      <div class="text-sm text-gray-600 space-y-2">
        <p class="font-medium text-gray-900">{{ $t('setup.routes.rulesHeading') }}</p>
        <div v-if="routePresets.find(p => p.id === selectedPreset)">
          <ul class="list-disc list-inside space-y-1 ml-2">
            <li
              v-for="(rule, index) in routePresets.find(p => p.id === selectedPreset)!.rules"
              :key="index"
            >
              <strong v-if="rule.rule_set">{{ $t('setup.routes.ruleSet', { value: Array.isArray(rule.rule_set) ? rule.rule_set.join(', ') : rule.rule_set }) }}</strong>
              <strong v-else-if="rule.ip_cidr">{{ $t('setup.routes.privateIps') }}</strong>
              → {{ rule.outbound }}
            </li>
            <li class="text-gray-700">
              <strong>{{ $t('setup.routes.allOther') }}</strong> → {{ routePresets.find(p => p.id === selectedPreset)!.final }}
            </li>
          </ul>
        </div>
      </div>
    </Card>
  </div>
</template>

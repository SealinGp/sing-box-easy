<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { apiService } from '../../services/api'
import type { DNS, DNSServer } from '../../types/api'
import { Button, Alert, Card, Select, Badge } from '../../components'
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
const enableFakeIP = ref(false)

// DNS 策略选项
const strategyOptions = [
  { label: 'Prefer IPv4', value: 'prefer_ipv4' },
  { label: 'Prefer IPv6', value: 'prefer_ipv6' },
  { label: 'IPv4 Only', value: 'ipv4_only' },
  { label: 'IPv6 Only', value: 'ipv6_only' },
]

const selectedStrategy = ref('prefer_ipv4')

// 预设 DNS 配置方案
interface DNSPreset {
  id: string
  name: string
  description: string
  servers: DNSServer[]
}

const dnsPresets: DNSPreset[] = [
  {
    id: 'smart',
    name: 'Smart DNS (Recommended)',
    description: 'Uses different DNS servers for domestic and foreign domains',
    servers: [
      {
        tag: 'dns-remote',
        address: 'https://1.1.1.1/dns-query',
        address_resolver: 'dns-local',
        detour: 'proxy-group',
      },
      {
        tag: 'dns-local',
        address: '223.5.5.5',
        detour: 'direct',
      },
      {
        tag: 'dns-block',
        address: 'rcode://success',
      },
    ],
  },
  {
    id: 'cloudflare',
    name: 'Cloudflare DNS',
    description: 'Uses Cloudflare DNS (1.1.1.1) for all queries',
    servers: [
      {
        tag: 'dns-cloudflare',
        address: 'https://1.1.1.1/dns-query',
      },
      {
        tag: 'dns-block',
        address: 'rcode://success',
      },
    ],
  },
  {
    id: 'google',
    name: 'Google DNS',
    description: 'Uses Google DNS (8.8.8.8) for all queries',
    servers: [
      {
        tag: 'dns-google',
        address: 'https://dns.google/dns-query',
      },
      {
        tag: 'dns-block',
        address: 'rcode://success',
      },
    ],
  },
  {
    id: 'china',
    name: 'China DNS Only',
    description: 'Uses domestic DNS servers (Alibaba 223.5.5.5)',
    servers: [
      {
        tag: 'dns-china',
        address: '223.5.5.5',
      },
      {
        tag: 'dns-block',
        address: 'rcode://success',
      },
    ],
  },
]

onMounted(async () => {
  await loadDNSConfig()
})

const loadDNSConfig = async () => {
  loading.value = true
  error.value = ''

  try {
    const dns = await apiService.getDNS()
    if (dns && dns.servers && dns.servers.length > 0) {
      // 尝试识别预设配置
      const dnsServers = dns.servers
      const matchedPreset = dnsPresets.find(preset => {
        return preset.servers.length === dnsServers.length &&
          preset.servers.every((ps, i) => {
            const dnsServer = dnsServers[i]
            return dnsServer && ps.tag === dnsServer.tag
          })
      })

      if (matchedPreset) {
        selectedPreset.value = matchedPreset.id
      }

      // 检查是否启用了 FakeIP
      if (dns.fakeip) {
        enableFakeIP.value = true
      }

      // 检查策略
      if (dns.strategy) {
        selectedStrategy.value = dns.strategy
      }
    }
  } catch (err: any) {
    console.log('No existing DNS config found')
  } finally {
    loading.value = false
  }
}

const saveDNSConfig = async () => {
  saving.value = true
  error.value = ''
  success.value = false

  try {
    const preset = dnsPresets.find(p => p.id === selectedPreset.value)
    if (!preset) {
      error.value = 'Invalid DNS preset selected'
      return
    }

    const dnsConfig: DNS = {
      servers: preset.servers,
      strategy: selectedStrategy.value,
    }

    // 如果启用 FakeIP
    if (enableFakeIP.value) {
      dnsConfig.fakeip = {
        enabled: true,
        inet4_range: '198.18.0.0/15',
        inet6_range: 'fc00::/18',
      }
    }

    // 添加规则（如果是智能 DNS）
    if (selectedPreset.value === 'smart') {
      dnsConfig.rules = [
        {
          rule_set: ['geosite-cn'],
          server: 'dns-local',
        },
        {
          rule_set: ['geosite-category-ads-all'],
          server: 'dns-block',
          disable_cache: true,
        },
      ]
      dnsConfig.final = 'dns-remote'
    }

    await apiService.updateDNS(dnsConfig)
    success.value = true

    // 2秒后自动进入下一步
    setTimeout(() => {
      emit('next')
    }, 2000)
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || 'Failed to save DNS configuration'
  } finally {
    saving.value = false
  }
}

const handleNext = () => {
  if (!success.value) {
    saveDNSConfig()
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
    <Alert v-if="success" type="success" title="DNS Configured">
      DNS configuration has been saved successfully. Proceeding to next step...
    </Alert>

    <!-- 错误提示 -->
    <Alert v-if="error" type="error" closable @close="error = ''">
      {{ error }}
    </Alert>

    <!-- DNS 预设方案选择 -->
    <Card>
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 mb-2">Select DNS Configuration</h3>
          <p class="text-sm text-gray-600">
            Choose a DNS configuration preset that best suits your needs.
          </p>
        </div>

        <!-- 预设方案列表 -->
        <div class="space-y-3">
          <div
            v-for="preset in dnsPresets"
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
              <div class="flex flex-wrap gap-2 mt-2">
                <Badge
                  v-for="server in preset.servers"
                  :key="server.tag"
                  variant="gray"
                  size="sm"
                >
                  {{ server.tag }}
                </Badge>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Card>

    <!-- DNS 策略 -->
    <Card>
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 mb-2">DNS Strategy</h3>
          <p class="text-sm text-gray-600">
            Choose the IP version preference for DNS queries.
          </p>
        </div>

        <Select
          v-model="selectedStrategy"
          :options="strategyOptions"
          label="Query Strategy"
          :disabled="loading || saving || success"
        />
      </div>
    </Card>

    <!-- FakeIP 配置 -->
    <Card>
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 mb-2">FakeIP</h3>
          <p class="text-sm text-gray-600">
            Enable FakeIP for faster connection establishment and better privacy.
          </p>
        </div>

        <div class="flex items-center space-x-3 p-4 bg-gray-50 rounded-lg">
          <input
            v-model="enableFakeIP"
            type="checkbox"
            id="enable-fakeip"
            class="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            :disabled="loading || saving || success"
          />
          <label for="enable-fakeip" class="flex-1 cursor-pointer">
            <span class="text-sm font-medium text-gray-900">Enable FakeIP</span>
            <p class="text-xs text-gray-500 mt-0.5">
              Return fake IP addresses for DNS queries to speed up connections
            </p>
          </label>
        </div>

        <Alert v-if="enableFakeIP" type="info" title="FakeIP Enabled">
          FakeIP will use ranges: 198.18.0.0/15 (IPv4) and fc00::/18 (IPv6)
        </Alert>
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
          <p class="font-medium">About DNS Configuration:</p>
          <ul class="list-disc list-inside space-y-1 ml-2 text-blue-800">
            <li><strong>Smart DNS:</strong> Uses domestic DNS for CN domains, foreign DNS for others (best for China users)</li>
            <li><strong>Cloudflare/Google:</strong> Uses single DNS provider for all queries (simple, reliable)</li>
            <li><strong>China DNS:</strong> Uses only domestic DNS (fastest for China-only access)</li>
            <li><strong>FakeIP:</strong> Returns fake IPs to speed up connections, recommended for most users</li>
            <li><strong>Strategy:</strong> Prefer IPv4 is recommended for better compatibility</li>
          </ul>
          <p class="mt-2 text-xs text-blue-700">
            💡 Tip: Use "Smart DNS" for the best routing performance with split DNS.
          </p>
        </div>
      </div>
    </Card>

    <!-- DNS 服务器详情 -->
    <Card padding="sm" class="bg-gray-50">
      <div class="text-sm text-gray-600 space-y-2">
        <p class="font-medium text-gray-900">Selected DNS Servers:</p>
        <ul class="list-disc list-inside space-y-1 ml-2">
          <li v-for="server in dnsPresets.find(p => p.id === selectedPreset)?.servers" :key="server.tag">
            <strong>{{ server.tag }}:</strong> {{ server.address }}
            <span v-if="server.detour" class="text-gray-500"> (via {{ server.detour }})</span>
          </li>
        </ul>
      </div>
    </Card>
  </div>
</template>

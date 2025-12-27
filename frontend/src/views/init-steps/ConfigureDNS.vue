<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { DNS } from '../../types/api'
import { Button, Alert, Card, Badge } from '../../components'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'
import type { DNSServerOptions, HostsDNSServerOptions, DomainStrategy } from '../../types/dns'
import { dnsService } from '../../services'

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
const enableHosts = ref(false)
const hostEntries = ref<Record<string, string | string[]>>({})
const newHostDomain = ref('')
const newHostIP = ref('')

// DNS 策略选项
const strategyOptions: Array<{ label: string; value: DomainStrategy }> = [
  { label: 'Prefer IPv4', value: 'prefer_ipv4' },
  { label: 'Prefer IPv6', value: 'prefer_ipv6' },
  { label: 'IPv4 Only', value: 'ipv4_only' },
  { label: 'IPv6 Only', value: 'ipv6_only' },
]

const selectedStrategy = ref<DomainStrategy>('prefer_ipv4')

// 预设 DNS 配置方案
interface DNSPreset {
  id: string
  name: string
  description: string
  servers: DNSServerOptions[]
}

const dnsPresets: DNSPreset[] = [
  {
    id: 'smart',
    name: 'Smart DNS (Recommended)',
    description: 'Uses different DNS servers for domestic and foreign domains',
    servers: [
      {
        type: 'udp',
        tag: 'dns-china',
        server: '223.5.5.5',
        server_port: 53,
      },
      {
        type: 'tls',
        tag: 'dns-google-tls',
        server: '8.8.8.8',
        server_port: 853,
        tls: {}
      }
    ],
  },
  {
    id: 'cloudflare',
    name: 'Cloudflare DNS',
    description: 'Uses Cloudflare DNS (1.1.1.1) for all queries',
    servers: [
      {
        type: 'https',
        tag: 'dns-cloudflare',
        server: '1.1.1.1',
        server_port: 443,
        tls: {},
      }
    ],
  },
  {
    id: 'google',
    name: 'Google DNS',
    description: 'Uses Google DNS (8.8.8.8) for all queries',
    servers: [
      {
        type: 'https',
        tag: 'dns-google',
        server: 'dns.google',
        server_port: 443,
        tls: {},
      }
    ],
  },
  {
    id: 'china',
    name: 'China DNS Only',
    description: 'Uses domestic DNS servers (Alibaba 223.5.5.5)',
    servers: [
      {
        type: 'udp',
        tag: 'dns-china',
        server: '223.5.5.5',
        server_port: 53,
      }
    ],
  },
]

// Hosts management functions
const addHostEntry = () => {
  if (!newHostDomain.value || !newHostIP.value) {
    return
  }

  // Check if IP contains comma (multiple IPs)
  if (newHostIP.value.includes(',')) {
    const ips = newHostIP.value.split(',').map(ip => ip.trim()).filter(ip => ip)
    hostEntries.value[newHostDomain.value] = ips
  } else {
    hostEntries.value[newHostDomain.value] = newHostIP.value.trim()
  }

  newHostDomain.value = ''
  newHostIP.value = ''
}

const removeHostEntry = (domain: string) => {
  delete hostEntries.value[domain]
}

const getHostIPDisplay = (value: string | string[]): string => {
  if (Array.isArray(value)) {
    return value.join(', ')
  }
  return value
}

onMounted(async () => {
  await loadDNSConfig()
})

const loadDNSConfig = async () => {
  loading.value = true
  error.value = ''

  try {
    const {data} = await dnsService.getDNS()
    const dns = data
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

      // 检查 hosts DNS 服务器
      const hostsServer = dnsServers.find(s => s.type === 'hosts')
      if (hostsServer && 'predefined' in hostsServer && hostsServer.predefined) {
        enableHosts.value = true
        hostEntries.value = { ...hostsServer.predefined }
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

    const servers: DNSServerOptions[] = [...preset.servers]

    // 添加 hosts DNS 服务器（如果启用）
    if (enableHosts.value && Object.keys(hostEntries.value).length > 0) {
      const hostsServer: HostsDNSServerOptions = {
        type: 'hosts',
        tag: 'local-hosts',
        path: [],
        predefined: hostEntries.value,
      }
      servers.push(hostsServer)
    }

    const dnsConfig: DNS = {
      servers,
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

    // 添加规则和 final 服务器
    switch (selectedPreset.value) {
      case 'smart':
        dnsConfig.rules = [
          {
            rule_set: ['geosite-cn'],
            server: 'dns-china',
          },
          {
            rule_set: ['geosite-category-ads-all'],
            server: 'dns-block',
            disable_cache: true,
          },
        ]
        dnsConfig.final = 'dns-google-tls'
        break
      case 'cloudflare':
        dnsConfig.rules = [
          {
            rule_set: ['geosite-category-ads-all'],
            server: 'dns-block',
            disable_cache: true,
          },
        ]
        dnsConfig.final = 'dns-cloudflare'
        break
      case 'google':
        dnsConfig.rules = [
          {
            rule_set: ['geosite-category-ads-all'],
            server: 'dns-block',
            disable_cache: true,
          },
        ]
        dnsConfig.final = 'dns-google'
        break
      case 'china':
        dnsConfig.rules = [
          {
            rule_set: ['geosite-category-ads-all'],
            server: 'dns-block',
            disable_cache: true,
          },
        ]
        dnsConfig.final = 'dns-china'
        break
    }

    await dnsService.updateDNS(dnsConfig)
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
            class="flex items-start gap-3 p-4 border border-gray-200 rounded-lg hover:border-violet-300 hover:bg-violet-50 cursor-pointer transition-colors"
            :class="{ 'border-violet-500 bg-violet-50': selectedPreset === preset.id }"
            @click="selectedPreset = preset.id"
          >
            <input
              type="radio"
              :checked="selectedPreset === preset.id"
              class="w-4 h-4 text-violet-600 border-gray-300 mt-0.5"
              :disabled="loading || saving || success"
              @click.stop="selectedPreset = preset.id"
            />
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <p class="text-sm font-medium text-gray-900">{{ preset.name }}</p>
                <Badge v-if="preset.id === 'smart'" variant="primary" size="sm">Recommended</Badge>
              </div>
              <p class="text-xs text-gray-600 mt-1">{{ preset.description }}</p>
              <div class="space-y-1 mt-2">
                <div
                  v-for="server in preset.servers"
                  :key="server.tag"
                  class="text-xs text-gray-700"
                >
                  <span class="font-medium">{{ server.tag }}</span>
                  <span class="text-gray-500"> ({{ server.type || 'legacy' }})</span>
                  <span v-if="'server' in server" class="text-gray-600">
                    : {{ server.server }}<span v-if="'server_port' in server && server.server_port !== 53 && server.server_port !== 853">:{{ server.server_port }}</span>
                  </span>
                  <span v-else-if="'address' in server" class="text-gray-600">
                    : {{ server.address }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Card>

    <!-- Hosts DNS 配置 -->
    <Card>
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 mb-2">Hosts DNS (Optional)</h3>
          <p class="text-sm text-gray-600">
            Add custom host entries to override DNS resolution for specific domains.
          </p>
        </div>

        <div class="flex items-center space-x-3 p-4 bg-gray-50 rounded-lg">
          <input
            v-model="enableHosts"
            type="checkbox"
            id="enable-hosts"
            class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500"
            :disabled="loading || saving || success"
          />
          <label for="enable-hosts" class="flex-1 cursor-pointer">
            <span class="text-sm font-medium text-gray-900">Enable Hosts DNS</span>
            <p class="text-xs text-gray-500 mt-0.5">
              Map domain names to specific IP addresses
            </p>
          </label>
        </div>

        <div v-if="enableHosts" class="space-y-3">
          <!-- Add new host entry -->
          <div class="p-4 bg-violet-50 border border-violet-200 rounded-lg">
            <p class="text-sm font-medium text-gray-900 mb-3">Add Host Entry</p>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
              <div>
                <label class="block text-xs font-medium text-gray-700 mb-1">Domain</label>
                <input
                  v-model="newHostDomain"
                  type="text"
                  placeholder="e.g., www.example.com"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-violet-500 focus:border-violet-500"
                  :disabled="saving || success"
                  @keyup.enter="addHostEntry"
                />
              </div>
              <div>
                <label class="block text-xs font-medium text-gray-700 mb-1">
                  IP Address(es)
                  <span class="text-gray-500 font-normal">(comma-separated for multiple)</span>
                </label>
                <div class="flex gap-2">
                  <input
                    v-model="newHostIP"
                    type="text"
                    placeholder="e.g., 127.0.0.1 or 127.0.0.1, ::1"
                    class="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-violet-500 focus:border-violet-500"
                    :disabled="saving || success"
                    @keyup.enter="addHostEntry"
                  />
                  <Button
                    variant="primary"
                    size="sm"
                    :disabled="!newHostDomain || !newHostIP || saving || success"
                    @click="addHostEntry"
                  >
                    Add
                  </Button>
                </div>
              </div>
            </div>
          </div>

          <!-- Host entries list -->
          <div v-if="Object.keys(hostEntries).length > 0" class="space-y-2">
            <p class="text-sm font-medium text-gray-900">Host Entries:</p>
            <div class="space-y-2">
              <div
                v-for="(value, domain) in hostEntries"
                :key="domain"
                class="flex items-center justify-between p-3 bg-white border border-gray-200 rounded-lg"
              >
                <div class="flex-1 min-w-0">
                  <p class="text-sm font-medium text-gray-900 truncate">{{ domain }}</p>
                  <p class="text-xs text-gray-600 truncate">{{ getHostIPDisplay(value) }}</p>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  :disabled="saving || success"
                  @click="removeHostEntry(domain)"
                >
                  Remove
                </Button>
              </div>
            </div>
          </div>

          <Alert v-else type="info" title="No host entries">
            Add custom host entries above to map domains to specific IP addresses.
          </Alert>
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
            class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500"
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
    <Card padding="sm" class="bg-violet-50 border-violet-200">
      <div class="flex items-start space-x-3">
        <InformationCircleIcon class="h-5 w-5 text-violet-600 flex-shrink-0 mt-0.5" />
        <div class="text-sm text-violet-900 space-y-2">
          <p class="font-medium">About DNS Configuration:</p>
          <ul class="list-disc list-inside space-y-1 ml-2 text-violet-800">
            <li><strong>Smart DNS:</strong> Uses domestic DNS for CN domains, foreign DNS for others (best for China users)</li>
            <li><strong>Cloudflare/Google:</strong> Uses single DNS provider for all queries (simple, reliable)</li>
            <li><strong>China DNS:</strong> Uses only domestic DNS (fastest for China-only access)</li>
            <li><strong>FakeIP:</strong> Returns fake IPs to speed up connections, recommended for most users</li>
            <li><strong>Strategy:</strong> Prefer IPv4 is recommended for better compatibility</li>
          </ul>
          <p class="mt-2 text-xs text-violet-700">
            💡 Tip: Use "Smart DNS" for the best routing performance with split DNS.
          </p>
        </div>
      </div>
    </Card>

    <!-- DNS 服务器详情 -->
    <Card padding="sm" class="bg-gray-50">
      <div class="text-sm text-gray-600 space-y-2">
        <p class="font-medium text-gray-900">Selected DNS Configuration:</p>
        <ul class="list-disc list-inside space-y-1 ml-2">
          <li v-for="server in dnsPresets.find(p => p.id === selectedPreset)?.servers" :key="server.tag">
            <strong>{{ server.tag }}</strong>
            <span class="text-gray-500"> ({{ server.type || 'legacy' }})</span>:
            <span v-if="'server' in server">
              {{ server.server }}<span v-if="'server_port' in server">:{{ server.server_port }}</span>
            </span>
            <span v-else-if="'address' in server">
              {{ server.address }}
            </span>
            <span v-if="'detour' in server && server.detour" class="text-gray-500"> (via {{ server.detour }})</span>
          </li>
          <li v-if="enableHosts && Object.keys(hostEntries).length > 0">
            <strong>local-hosts</strong>
            <span class="text-gray-500"> (hosts)</span>:
            {{ Object.keys(hostEntries).length }} custom host{{ Object.keys(hostEntries).length > 1 ? 's' : '' }}
          </li>
        </ul>
      </div>
    </Card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DNS } from '../../types/api'
import { Button, Alert, Card, Badge } from '../../components'
import { Select } from '../../volt'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'
import type { DNSServerOptions, HostsDNSServerOptions, DomainStrategy } from '../../types/dns'
import { dnsService } from '../../services'

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
const enableFakeIP = ref(false)
const enableHosts = ref(false)
const hostEntries = ref<Record<string, string | string[]>>({})
const newHostDomain = ref('')
const newHostIP = ref('')

// DNS 策略选项 — value is the backend identifier, label re-translates.
const strategyOptions = computed<Array<{ label: string; value: DomainStrategy }>>(() => [
  { label: t('setup.dns.strategyOptions.preferIpv4'), value: 'prefer_ipv4' },
  { label: t('setup.dns.strategyOptions.preferIpv6'), value: 'prefer_ipv6' },
  { label: t('setup.dns.strategyOptions.ipv4Only'), value: 'ipv4_only' },
  { label: t('setup.dns.strategyOptions.ipv6Only'), value: 'ipv6_only' },
])

const selectedStrategy = ref<DomainStrategy>('prefer_ipv4')

// 预设 DNS 配置方案
interface DNSPreset {
  id: string
  name: string
  description: string
  servers: DNSServerOptions[]
}

// Display name/description re-translate on locale switch; server configs
// (tag/server/port/type) sent to the backend stay unchanged.
const dnsPresets = computed<DNSPreset[]>(() => [
  {
    id: 'smart',
    name: t('setup.dns.presets.smart.name'),
    description: t('setup.dns.presets.smart.description'),
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
    name: t('setup.dns.presets.cloudflare.name'),
    description: t('setup.dns.presets.cloudflare.description'),
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
    name: t('setup.dns.presets.google.name'),
    description: t('setup.dns.presets.google.description'),
    servers: [
      {
        type: 'https',
        tag: 'dns-google',
        server: '8.8.8.8',
        server_port: 443,
        tls: {},
      }
    ],
  },
  {
    id: 'china',
    name: t('setup.dns.presets.china.name'),
    description: t('setup.dns.presets.china.description'),
    servers: [
      {
        type: 'udp',
        tag: 'dns-china',
        server: '223.5.5.5',
        server_port: 53,
      }
    ],
  },
])

// Hosts management functions
const addHostEntry = () => {
  if (!newHostDomain.value || !newHostIP.value) {
    return
  }

  // Immutable host-entries update — never mutate the existing object.
  const value: string | string[] = newHostIP.value.includes(',')
    ? newHostIP.value.split(',').map(ip => ip.trim()).filter(ip => ip)
    : newHostIP.value.trim()
  hostEntries.value = { ...hostEntries.value, [newHostDomain.value]: value }

  newHostDomain.value = ''
  newHostIP.value = ''
}

const removeHostEntry = (domain: string) => {
  const { [domain]: _removed, ...rest } = hostEntries.value
  hostEntries.value = rest
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
      const matchedPreset = dnsPresets.value.find(preset => {
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
    const preset = dnsPresets.value.find(p => p.id === selectedPreset.value)
    if (!preset) {
      error.value = t('setup.dns.invalidPreset')
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
    error.value = err.message || t('setup.dns.saveFailed')
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
    <Alert v-if="success" type="success" :title="$t('setup.dns.successTitle')">
      {{ $t('setup.dns.successDesc') }}
    </Alert>

    <!-- 错误提示 -->
    <Alert v-if="error" type="error" closable @close="error = ''">
      {{ error }}
    </Alert>

    <!-- DNS 预设方案选择 -->
    <Card>
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 mb-2">{{ $t('setup.dns.selectHeading') }}</h3>
          <p class="text-sm text-gray-600">
            {{ $t('setup.dns.selectDesc') }}
          </p>
        </div>

        <!-- 预设方案列表 -->
        <div class="space-y-3">
          <div
            v-for="preset in dnsPresets"
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
                <Badge v-if="preset.id === 'smart'" variant="primary" size="sm">{{ $t('setup.dns.recommended') }}</Badge>
              </div>
              <p class="text-xs text-gray-600 mt-1">{{ preset.description }}</p>
              <div class="space-y-1 mt-2">
                <div
                  v-for="server in preset.servers"
                  :key="server.tag"
                  class="text-xs text-gray-700"
                >
                  <span class="font-medium">{{ server.tag }}</span>
                  <span class="text-gray-500"> ({{ server.type || $t('setup.dns.legacy') }})</span>
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
          <h3 class="text-lg font-semibold text-gray-900 mb-2">{{ $t('setup.dns.hostsHeading') }}</h3>
          <p class="text-sm text-gray-600">
            {{ $t('setup.dns.hostsDesc') }}
          </p>
        </div>

        <div class="flex items-center space-x-3 p-4 bg-gray-50 rounded-surface">
          <input
            v-model="enableHosts"
            type="checkbox"
            id="enable-hosts"
            class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
            :disabled="loading || saving || success"
          />
          <label for="enable-hosts" class="flex-1 cursor-pointer">
            <span class="text-sm font-medium text-gray-900">{{ $t('setup.dns.enableHosts') }}</span>
            <p class="text-xs text-gray-500 mt-0.5">
              {{ $t('setup.dns.enableHostsDesc') }}
            </p>
          </label>
        </div>

        <div v-if="enableHosts" class="space-y-3">
          <!-- Add new host entry -->
          <div class="p-4 bg-primary-50 border border-primary-200 rounded-surface">
            <p class="text-sm font-medium text-gray-900 mb-3">{{ $t('setup.dns.addHostEntry') }}</p>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
              <div>
                <label class="block text-xs font-medium text-gray-700 mb-1">{{ $t('setup.dns.domain') }}</label>
                <input
                  v-model="newHostDomain"
                  type="text"
                  :placeholder="$t('setup.dns.domainPlaceholder')"
                  class="w-full px-3 py-2 border border-gray-300 rounded-control text-sm focus:ring-primary-500 focus:border-primary-500"
                  :disabled="saving || success"
                  @keyup.enter="addHostEntry"
                />
              </div>
              <div>
                <label class="block text-xs font-medium text-gray-700 mb-1">
                  {{ $t('setup.dns.ipAddresses') }}
                  <span class="text-gray-500 font-normal">{{ $t('setup.dns.ipAddressesHint') }}</span>
                </label>
                <div class="flex gap-2">
                  <input
                    v-model="newHostIP"
                    type="text"
                    :placeholder="$t('setup.dns.ipPlaceholder')"
                    class="flex-1 px-3 py-2 border border-gray-300 rounded-control text-sm focus:ring-primary-500 focus:border-primary-500"
                    :disabled="saving || success"
                    @keyup.enter="addHostEntry"
                  />
                  <Button
                    variant="primary"
                    size="sm"
                    :disabled="!newHostDomain || !newHostIP || saving || success"
                    @click="addHostEntry"
                  >
                    {{ $t('setup.dns.add') }}
                  </Button>
                </div>
              </div>
            </div>
          </div>

          <!-- Host entries list -->
          <div v-if="Object.keys(hostEntries).length > 0" class="space-y-2">
            <p class="text-sm font-medium text-gray-900">{{ $t('setup.dns.hostEntries') }}</p>
            <div class="space-y-2">
              <div
                v-for="(value, domain) in hostEntries"
                :key="domain"
                class="flex items-center justify-between p-3 bg-white border border-gray-200 rounded-surface"
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
                  {{ $t('setup.dns.remove') }}
                </Button>
              </div>
            </div>
          </div>

          <Alert v-else type="info" :title="$t('setup.dns.noEntriesTitle')">
            {{ $t('setup.dns.noEntriesDesc') }}
          </Alert>
        </div>
      </div>
    </Card>

    <!-- DNS 策略 -->
    <Card>
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 mb-2">{{ $t('setup.dns.strategyHeading') }}</h3>
          <p class="text-sm text-gray-600">
            {{ $t('setup.dns.strategyDesc') }}
          </p>
        </div>

        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('setup.dns.queryStrategy') }}</label>
        <Select
          class="w-full"
          v-model="selectedStrategy"
          :options="strategyOptions"
          optionLabel="label"
          optionValue="value"
          :disabled="loading || saving || success"
        />
      </div>
    </Card>

    <!-- FakeIP 配置 -->
    <Card>
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 mb-2">{{ $t('setup.dns.fakeipHeading') }}</h3>
          <p class="text-sm text-gray-600">
            {{ $t('setup.dns.fakeipDesc') }}
          </p>
        </div>

        <div class="flex items-center space-x-3 p-4 bg-gray-50 rounded-surface">
          <input
            v-model="enableFakeIP"
            type="checkbox"
            id="enable-fakeip"
            class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
            :disabled="loading || saving || success"
          />
          <label for="enable-fakeip" class="flex-1 cursor-pointer">
            <span class="text-sm font-medium text-gray-900">{{ $t('setup.dns.enableFakeip') }}</span>
            <p class="text-xs text-gray-500 mt-0.5">
              {{ $t('setup.dns.enableFakeipDesc') }}
            </p>
          </label>
        </div>

        <Alert v-if="enableFakeIP" type="info" :title="$t('setup.dns.fakeipEnabledTitle')">
          {{ $t('setup.dns.fakeipEnabledDesc') }}
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
        {{ success ? $t('setup.dns.saved') : saving ? $t('common.saving') : $t('setup.dns.saveContinue') }}
      </Button>

      <Button
        variant="ghost"
        :disabled="saving || success"
        @click="handleSkip"
      >
        {{ $t('setup.dns.skip') }}
      </Button>
    </div>

    <!-- 说明信息 -->
    <Card padding="sm" class="bg-primary-50 border-primary-200">
      <div class="flex items-start space-x-3">
        <InformationCircleIcon class="h-5 w-5 text-primary-600 flex-shrink-0 mt-0.5" />
        <div class="text-sm text-primary-900 space-y-2">
          <p class="font-medium">{{ $t('setup.dns.aboutHeading') }}</p>
          <ul class="list-disc list-inside space-y-1 ml-2 text-primary-800">
            <li><strong>{{ $t('setup.dns.aboutSmartLabel') }}</strong> {{ $t('setup.dns.aboutSmart') }}</li>
            <li><strong>{{ $t('setup.dns.aboutSingleLabel') }}</strong> {{ $t('setup.dns.aboutSingle') }}</li>
            <li><strong>{{ $t('setup.dns.aboutChinaLabel') }}</strong> {{ $t('setup.dns.aboutChina') }}</li>
            <li><strong>{{ $t('setup.dns.aboutFakeipLabel') }}</strong> {{ $t('setup.dns.aboutFakeip') }}</li>
            <li><strong>{{ $t('setup.dns.aboutStrategyLabel') }}</strong> {{ $t('setup.dns.aboutStrategy') }}</li>
          </ul>
          <p class="mt-2 text-xs text-primary-700">
            {{ $t('setup.dns.tip') }}
          </p>
        </div>
      </div>
    </Card>

    <!-- DNS 服务器详情 -->
    <Card padding="sm" class="bg-gray-50">
      <div class="text-sm text-gray-600 space-y-2">
        <p class="font-medium text-gray-900">{{ $t('setup.dns.selectedHeading') }}</p>
        <ul class="list-disc list-inside space-y-1 ml-2">
          <li v-for="server in dnsPresets.find(p => p.id === selectedPreset)?.servers" :key="server.tag">
            <strong>{{ server.tag }}</strong>
            <span class="text-gray-500"> ({{ server.type || $t('setup.dns.legacy') }})</span>:
            <span v-if="'server' in server">
              {{ server.server }}<span v-if="'server_port' in server">:{{ server.server_port }}</span>
            </span>
            <span v-else-if="'address' in server">
              {{ server.address }}
            </span>
            <span v-if="'detour' in server && server.detour" class="text-gray-500"> ({{ $t('setup.dns.via', { detour: server.detour }) }})</span>
          </li>
          <li v-if="enableHosts && Object.keys(hostEntries).length > 0">
            <strong>local-hosts</strong>
            <span class="text-gray-500"> (hosts)</span>:
            {{ Object.keys(hostEntries).length > 1
              ? $t('setup.dns.customHosts', { count: Object.keys(hostEntries).length })
              : $t('setup.dns.customHost', { count: Object.keys(hostEntries).length }) }}
          </li>
        </ul>
      </div>
    </Card>
  </div>
</template>

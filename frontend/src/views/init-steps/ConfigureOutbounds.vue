<script setup lang="ts">
import { ref } from 'vue'
import { apiService } from '../../services/api'
import type { ParsedNode, Outbound } from '../../types/api'
import { Button, Input, Alert, Card, Badge } from '../../components'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'

const emit = defineEmits<{
  next: []
  prev: []
}>()

const subscriptionUrl = ref('')
const parsing = ref(false)
const parsedNodes = ref<ParsedNode[]>([])
const selectedNodes = ref<Set<string>>(new Set())
const error = ref('')
const parseError = ref('')

const saving = ref(false)
const success = ref(false)

// 解析订阅
const parseSubscription = async () => {
  if (!subscriptionUrl.value.trim()) {
    parseError.value = 'Please enter a subscription URL'
    return
  }

  parsing.value = true
  parseError.value = ''
  parsedNodes.value = []
  selectedNodes.value.clear()

  try {
    const nodes = await apiService.parseNodes(subscriptionUrl.value.trim())
    parsedNodes.value = nodes

    // 默认全选
    nodes.forEach((node) => {
      selectedNodes.value.add(node.outbound.tag)
    })

    if (nodes.length === 0) {
      parseError.value = 'No nodes found in subscription'
    }
  } catch (err: any) {
    parseError.value = err.response?.data?.error || err.message || 'Failed to parse subscription'
  } finally {
    parsing.value = false
  }
}

// 切换节点选择
const toggleNode = (tag: string) => {
  if (selectedNodes.value.has(tag)) {
    selectedNodes.value.delete(tag)
  } else {
    selectedNodes.value.add(tag)
  }
}

// 全选/取消全选
const toggleSelectAll = () => {
  if (selectedNodes.value.size === parsedNodes.value.length) {
    selectedNodes.value.clear()
  } else {
    parsedNodes.value.forEach((node) => {
      selectedNodes.value.add(node.outbound.tag)
    })
  }
}

// 保存出站配置
const saveOutbounds = async () => {
  saving.value = true
  error.value = ''
  success.value = false

  try {
    // 准备要添加的节点
    const outboundsToAdd: Outbound[] = []

    // 添加选中的代理节点
    parsedNodes.value.forEach((node) => {
      if (selectedNodes.value.has(node.outbound.tag)) {
        outboundsToAdd.push(node.outbound)
      }
    })

    // 添加基础出站：direct 和 block
    outboundsToAdd.push({
      tag: 'direct',
      type: 'direct',
    })

    outboundsToAdd.push({
      tag: 'block',
      type: 'block',
    })

    // 批量添加
    if (outboundsToAdd.length > 0) {
      await apiService.addOutboundsBatch(outboundsToAdd)
    }

    success.value = true

    // 2秒后自动进入下一步
    setTimeout(() => {
      emit('next')
    }, 2000)
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || 'Failed to save outbounds'
  } finally {
    saving.value = false
  }
}

const handleNext = () => {
  if (!success.value) {
    saveOutbounds()
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
    <Alert v-if="success" type="success" title="Outbounds Saved">
      Outbound nodes have been added successfully. Proceeding to next step...
    </Alert>

    <!-- 错误提示 -->
    <Alert v-if="error" type="error" closable @close="error = ''">
      {{ error }}
    </Alert>

    <!-- 订阅解析 -->
    <Card>
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 mb-2">Parse Subscription</h3>
          <p class="text-sm text-gray-600">
            Enter your subscription URL to parse and import proxy nodes.
          </p>
        </div>

        <div class="flex gap-3">
          <Input
            v-model="subscriptionUrl"
            placeholder="https://example.com/subscribe?token=xxx"
            :disabled="parsing"
            :error="parseError"
            full-width
          />
          <Button
            variant="primary"
            :loading="parsing"
            :disabled="parsing"
            @click="parseSubscription"
          >
            {{ parsing ? 'Parsing...' : 'Parse' }}
          </Button>
        </div>
      </div>
    </Card>

    <!-- 节点列表 -->
    <Card v-if="parsedNodes.length > 0">
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="text-lg font-semibold text-gray-900">
              Parsed Nodes
              <Badge variant="primary" class="ml-2">{{ parsedNodes.length }}</Badge>
            </h3>
            <p class="text-sm text-gray-600 mt-1">
              Select nodes to add ({{ selectedNodes.size }} selected)
            </p>
          </div>
          <Button
            variant="secondary"
            size="sm"
            @click="toggleSelectAll"
          >
            {{ selectedNodes.size === parsedNodes.length ? 'Deselect All' : 'Select All' }}
          </Button>
        </div>

        <!-- 节点列表 -->
        <div class="max-h-96 overflow-y-auto border border-gray-200 rounded-lg">
          <div
            v-for="node in parsedNodes"
            :key="node.outbound.tag"
            class="flex items-center gap-3 p-3 hover:bg-gray-50 border-b border-gray-100 last:border-0 cursor-pointer"
            @click="toggleNode(node.outbound.tag)"
          >
            <input
              type="checkbox"
              :checked="selectedNodes.has(node.outbound.tag)"
              class="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
              @click.stop="toggleNode(node.outbound.tag)"
            />
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <p class="text-sm font-medium text-gray-900 truncate">
                  {{ node.outbound.tag }}
                </p>
                <Badge variant="gray" size="sm">{{ node.protocol }}</Badge>
              </div>
              <p class="text-xs text-gray-500 truncate">
                {{ node.outbound.server }}:{{ node.outbound.server_port }}
              </p>
            </div>
          </div>
        </div>

        <!-- 保存按钮 -->
        <div class="flex gap-3 pt-4 border-t">
          <Button
            variant="primary"
            :loading="saving"
            :disabled="saving || success || selectedNodes.size === 0"
            @click="saveOutbounds"
          >
            {{ success ? 'Saved' : saving ? 'Saving...' : `Add ${selectedNodes.size} Nodes` }}
          </Button>
          <Button
            v-if="success"
            variant="primary"
            @click="handleNext"
          >
            Continue to Next Step
          </Button>
        </div>
      </div>
    </Card>

    <!-- 手动添加基础节点 -->
    <Card v-if="parsedNodes.length === 0 && !parsing">
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 mb-2">Manual Setup</h3>
          <p class="text-sm text-gray-600">
            Skip subscription parsing and add basic outbounds (direct, block) only.
          </p>
        </div>

        <div class="bg-blue-50 rounded-lg p-4">
          <div class="flex items-start space-x-3">
            <InformationCircleIcon class="h-5 w-5 text-blue-600 flex-shrink-0 mt-0.5" />
            <div class="text-sm text-blue-900">
              <p class="font-medium mb-1">Basic outbounds will be added:</p>
              <ul class="list-disc list-inside ml-2 text-blue-800">
                <li><strong>direct</strong>: Direct connection (no proxy)</li>
                <li><strong>block</strong>: Block connection</li>
              </ul>
              <p class="mt-2 text-xs text-blue-700">
                You can add proxy nodes later from the dashboard.
              </p>
            </div>
          </div>
        </div>

        <div class="flex gap-3">
          <Button
            variant="primary"
            :loading="saving"
            :disabled="saving || success"
            @click="saveOutbounds"
          >
            {{ success ? 'Saved' : saving ? 'Saving...' : 'Add Basic Outbounds' }}
          </Button>
          <Button
            variant="ghost"
            :disabled="saving || success"
            @click="handleSkip"
          >
            Skip this step
          </Button>
        </div>
      </div>
    </Card>

    <!-- 说明信息 -->
    <Card padding="sm" class="bg-gray-50">
      <div class="text-sm text-gray-600 space-y-2">
        <p class="font-medium text-gray-900">About Outbounds:</p>
        <ul class="list-disc list-inside space-y-1 ml-2">
          <li><strong>Proxy Nodes:</strong> Servers that relay your traffic (shadowsocks, vmess, etc.)</li>
          <li><strong>Direct:</strong> Connect directly without proxy</li>
          <li><strong>Block:</strong> Block the connection entirely</li>
          <li>You can manage nodes and create groups in the dashboard later</li>
        </ul>
      </div>
    </Card>
  </div>
</template>

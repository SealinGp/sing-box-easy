<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Outbound } from '../../types/api'
import { Button, Textarea, Alert, Card, Badge, NodeList } from '../../components'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'
import { nodesService, outboundService } from '../../services'

const { t } = useI18n()

const emit = defineEmits<{
  next: []
  prev: []
}>()

// Current outbounds from config
const currentOutbounds = ref<Outbound[]>([])
const loadingOutbounds = ref(false)

const subscriptionInput = ref('')
const parsing = ref(false)
const parsedNodes = ref<Outbound[]>([])
const selectedNodes = ref<Set<string>>(new Set())
const error = ref('')
const parseError = ref('')

const saving = ref(false)
const success = ref(false)

// 解析订阅或节点链接
const parseSubscription = async () => {
  if (!subscriptionInput.value.trim()) {
    parseError.value = t('setup.outbounds.enterInput')
    return
  }

  parsing.value = true
  parseError.value = ''
  parsedNodes.value = []
  selectedNodes.value.clear()

  try {
    // 将多行输入转换为单行，用换行符分隔
    const lines = subscriptionInput.value
      .split('\n')
      .map(line => line.trim())
      .filter(line => line.length > 0)

    // 拼接成换行符分隔的字符串
    const linesToParse = lines.join('\n')

    const {data} = await nodesService.parseNodes(linesToParse)    
    const nodes = data.nodes
    parsedNodes.value = nodes

    // 默认全选
    nodes.forEach((node) => {
      selectedNodes.value.add(node.tag)
    })

    if (nodes.length === 0) {
      parseError.value = t('setup.outbounds.noNodesFound')
    }
  } catch (err: any) {
    parseError.value = err.message || t('setup.outbounds.parseFailed')
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
      selectedNodes.value.add(node.tag)
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
      if (selectedNodes.value.has(node.tag)) {
        outboundsToAdd.push(node)
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
      await outboundService.addOutboundsBatch(outboundsToAdd)
    }

    success.value = true

    // Refresh current outbounds list
    await loadCurrentOutbounds()

    // 2秒后自动进入下一步
    setTimeout(() => {
      emit('next')
    }, 2000)
  } catch (err: any) {
    error.value = err.message || t('setup.outbounds.saveFailed')
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

// Load current outbounds on mount
const loadCurrentOutbounds = async () => {
  loadingOutbounds.value = true
  try {
    const {data} = await outboundService.getOutbounds()
    currentOutbounds.value = data.outbounds || []
  } catch (err) {
    // Silently fail - outbounds may not exist yet
    currentOutbounds.value = []
  } finally {
    loadingOutbounds.value = false
  }
}

onMounted(() => {
  loadCurrentOutbounds()
})
</script>

<template>
  <div class="2xl:grid 2xl:grid-cols-[1fr_400px] 2xl:gap-6">
    <!-- Left column: Main content -->
    <div class="space-y-6">
      <!-- 成功提示 -->
      <Alert v-if="success" type="success" :title="$t('setup.outbounds.successTitle')">
        {{ $t('setup.outbounds.successDesc') }}
      </Alert>

      <!-- 错误提示 -->
      <Alert v-if="error" type="error" closable @close="error = ''">
        {{ error }}
      </Alert>

      <!-- 订阅解析 -->
      <Card>
        <div class="space-y-4">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 mb-2">{{ $t('setup.outbounds.parseHeading') }}</h3>
            <p class="text-sm text-gray-600">
              {{ $t('setup.outbounds.parseDesc') }}
            </p>
          </div>

          <div class="space-y-3">
            <Textarea
              v-model="subscriptionInput"
              :placeholder="$t('setup.outbounds.inputPlaceholder')"
              :disabled="parsing"
              :error="parseError"
              :rows="6"
              full-width
            />
            <div class="flex justify-end">
              <Button
                variant="primary"
                :loading="parsing"
                :disabled="parsing"
                @click="parseSubscription"
              >
                {{ parsing ? $t('setup.outbounds.parsing') : $t('setup.outbounds.parse') }}
              </Button>
            </div>
          </div>
        </div>
      </Card>

      <!-- 节点列表 -->
      <Card v-if="parsedNodes.length > 0">
        <div class="space-y-4">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-lg font-semibold text-gray-900">
                {{ $t('setup.outbounds.parsedNodes') }}
                <Badge variant="primary" class="ml-2">{{ parsedNodes.length }}</Badge>
              </h3>
              <p class="text-sm text-gray-600 mt-1">
                {{ $t('setup.outbounds.selectToAdd', { count: selectedNodes.size }) }}
              </p>
            </div>
            <Button
              variant="secondary"
              size="sm"
              @click="toggleSelectAll"
            >
              {{ selectedNodes.size === parsedNodes.length ? $t('setup.outbounds.deselectAll') : $t('setup.outbounds.selectAll') }}
            </Button>
          </div>

          <!-- 节点列表 -->
          <NodeList
            :nodes="parsedNodes"
            max-height="max-h-96"
            selectable
            :selected-tags="selectedNodes"
            @select="toggleNode"
          />

          <!-- 保存按钮 -->
          <div class="flex gap-3 pt-4 border-t">
            <Button
              variant="primary"
              :loading="saving"
              :disabled="saving || success || selectedNodes.size === 0"
              @click="saveOutbounds"
            >
              {{ success ? $t('setup.outbounds.saved') : saving ? $t('common.saving') : $t('setup.outbounds.addNodes', { count: selectedNodes.size }) }}
            </Button>
            <Button
              v-if="success"
              variant="primary"
              @click="handleNext"
            >
              {{ $t('setup.outbounds.continueNext') }}
            </Button>
          </div>
        </div>
      </Card>

      <!-- 手动添加基础节点 -->
      <Card v-if="parsedNodes.length === 0 && !parsing">
        <div class="space-y-4">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 mb-2">{{ $t('setup.outbounds.manualHeading') }}</h3>
            <p class="text-sm text-gray-600">
              {{ $t('setup.outbounds.manualDesc') }}
            </p>
          </div>

          <div class="bg-primary-50 rounded-surface p-4">
            <div class="flex items-start space-x-3">
              <InformationCircleIcon class="h-5 w-5 text-primary-600 flex-shrink-0 mt-0.5" />
              <div class="text-sm text-primary-900">
                <p class="font-medium mb-1">{{ $t('setup.outbounds.basicWillBeAdded') }}</p>
                <ul class="list-disc list-inside ml-2 text-primary-800">
                  <li><strong>direct</strong>: {{ $t('setup.outbounds.basicDirect') }}</li>
                  <li><strong>block</strong>: {{ $t('setup.outbounds.basicBlock') }}</li>
                </ul>
                <p class="mt-2 text-xs text-primary-700">
                  {{ $t('setup.outbounds.addProxyLater') }}
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
              {{ success ? $t('setup.outbounds.saved') : saving ? $t('common.saving') : $t('setup.outbounds.addBasic') }}
            </Button>
            <Button
              variant="ghost"
              :disabled="saving || success"
              @click="handleSkip"
            >
              {{ $t('setup.outbounds.skip') }}
            </Button>
          </div>
        </div>
      </Card>

      <!-- 当前节点列表 - shown inline on smaller screens -->
      <Card v-if="!loadingOutbounds && currentOutbounds.length > 0" class="2xl:hidden">
        <div class="space-y-4">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-lg font-semibold text-gray-900">
                {{ $t('setup.outbounds.currentNodes') }}
                <Badge variant="gray" class="ml-2">{{ currentOutbounds.length }}</Badge>
              </h3>
              <p class="text-sm text-gray-600 mt-1">
                {{ $t('setup.outbounds.currentNodesDesc') }}
              </p>
            </div>
            <Button
              variant="secondary"
              size="sm"
              @click="loadCurrentOutbounds"
            >
              {{ $t('common.refresh') }}
            </Button>
          </div>

          <!-- 节点列表 -->
          <NodeList :nodes="currentOutbounds" max-height="max-h-64" />
        </div>
      </Card>

      <!-- Loading state for current outbounds - shown inline on smaller screens -->
      <Card v-if="loadingOutbounds" class="2xl:hidden">
        <div class="flex items-center justify-center py-4">
          <div class="animate-spin rounded-pill h-6 w-6 border-b-2 border-primary-600"></div>
          <span class="ml-2 text-sm text-gray-600">{{ $t('setup.outbounds.loadingCurrent') }}</span>
        </div>
      </Card>

      <!-- 说明信息 -->
      <Card padding="sm" class="bg-gray-50">
        <div class="text-sm text-gray-600 space-y-2">
          <p class="font-medium text-gray-900">{{ $t('setup.outbounds.aboutHeading') }}</p>
          <ul class="list-disc list-inside space-y-1 ml-2">
            <li><strong>{{ $t('setup.outbounds.aboutProxyLabel') }}</strong> {{ $t('setup.outbounds.aboutProxy') }}</li>
            <li><strong>{{ $t('setup.outbounds.aboutDirectLabel') }}</strong> {{ $t('setup.outbounds.aboutDirect') }}</li>
            <li><strong>{{ $t('setup.outbounds.aboutBlockLabel') }}</strong> {{ $t('setup.outbounds.aboutBlock') }}</li>
            <li>{{ $t('setup.outbounds.aboutManage') }}</li>
          </ul>
        </div>
      </Card>
    </div>

    <!-- Right column: Current Nodes (2xl screens only) -->
    <div class="hidden 2xl:block">
      <div class="sticky top-6 space-y-4">
        <!-- 当前节点列表 -->
        <Card v-if="!loadingOutbounds && currentOutbounds.length > 0">
          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="text-lg font-semibold text-gray-900">
                  {{ $t('setup.outbounds.currentNodes') }}
                  <Badge variant="gray" class="ml-2">{{ currentOutbounds.length }}</Badge>
                </h3>
                <p class="text-sm text-gray-600 mt-1">
                  {{ $t('setup.outbounds.currentNodesDesc') }}
                </p>
              </div>
              <Button
                variant="secondary"
                size="sm"
                @click="loadCurrentOutbounds"
              >
                {{ $t('common.refresh') }}
              </Button>
            </div>

            <!-- 节点列表 -->
            <NodeList :nodes="currentOutbounds" max-height="max-h-[calc(100vh-200px)]" />
          </div>
        </Card>

        <!-- Loading state for current outbounds -->
        <Card v-if="loadingOutbounds">
          <div class="flex items-center justify-center py-4">
            <div class="animate-spin rounded-pill h-6 w-6 border-b-2 border-primary-600"></div>
            <span class="ml-2 text-sm text-gray-600">{{ $t('setup.outbounds.loadingCurrent') }}</span>
          </div>
        </Card>

        <!-- Empty state -->
        <Card v-if="!loadingOutbounds && currentOutbounds.length === 0">
          <div class="text-center py-8">
            <p class="text-sm text-gray-500">{{ $t('setup.outbounds.noNodes') }}</p>
          </div>
        </Card>
      </div>
    </div>
  </div>
</template>

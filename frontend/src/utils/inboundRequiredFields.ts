import type { Inbound } from '../types/api'

type MutableInbound = Partial<Inbound> & Record<string, any>

export interface InboundValidationError {
  key: string
}

interface InboundRequiredFieldConfig {
  applyDefaults?: (inbound: MutableInbound) => void
  validate?: (inbound: MutableInbound) => InboundValidationError | null
}

const generateUUID = () => {
  const cryptoSource = globalThis.crypto
  if (cryptoSource && typeof cryptoSource.randomUUID === 'function') {
    return cryptoSource.randomUUID()
  }

  return '10000000-1000-4000-8000-100000000000'.replace(/[018]/g, c =>
    (Number(c) ^ ((cryptoSource?.getRandomValues(new Uint8Array(1))[0] ?? Math.random() * 16) & (15 >> (Number(c) / 4)))).toString(16)
  )
}

const inboundRequiredFieldConfigs: Record<string, InboundRequiredFieldConfig> = {
  shadowsocks: {
    applyDefaults: (inbound) => {
      if (!inbound.method) {
        inbound.method = '2022-blake3-aes-128-gcm'
      }
    },
    validate: (inbound) => {
      if (!inbound.method) {
        return { key: 'inbounds.validation.ssMethodRequired' }
      }
      if (inbound.method !== 'none' && !inbound.password) {
        return { key: 'inbounds.validation.ssPasswordRequired' }
      }
      return null
    },
  },
  vmess: {
    applyDefaults: (inbound) => {
      if (!Array.isArray(inbound.users) || inbound.users.length === 0) {
        inbound.users = [
          {
            name: 'sekai',
            uuid: generateUUID(),
            alterId: 0,
          },
        ]
      } else if (inbound.users[0]) {
        inbound.users[0].alterId ??= 0
        if (!inbound.users[0].uuid) {
          inbound.users[0].uuid = generateUUID()
        }
      }
    },
    validate: (inbound) => {
      if (!Array.isArray(inbound.users) || inbound.users.length === 0) {
        return { key: 'inbounds.validation.vmessUsersRequired' }
      }
      if (!inbound.users[0]?.uuid) {
        return { key: 'inbounds.validation.vmessUUIDRequired' }
      }
      return null
    },
  },
}

export const applyInboundTypeDefaults = (inbound: MutableInbound) => {
  if (!inbound.type) return
  inboundRequiredFieldConfigs[inbound.type]?.applyDefaults?.(inbound)
}

export const validateInboundRequiredFields = (inbound: MutableInbound): InboundValidationError | null => {
  if (!inbound.tag?.trim()) {
    return { key: 'inbounds.validation.tagRequired' }
  }

  if (!inbound.type) {
    return { key: 'inbounds.validation.typeRequired' }
  }

  if (inbound.type !== 'tun' && !inbound.listen_port) {
    return { key: 'inbounds.validation.listenPortRequired' }
  }

  return inboundRequiredFieldConfigs[inbound.type]?.validate?.(inbound) ?? null
}

export const generateVmessUUID = generateUUID

// Lab fixtures only. Nothing here reads or writes configuration or storage.
export interface PreviewDNSGroup {
  id: string
  name: string
  servers: string[]
  timeout: number
  fallback: 'strict' | 'general'
  source?: string
  drift?: boolean
}

export const previewDNSServers = ['GoogleDoT', 'dns_google', 'dns_router', 'aliyun', 'tencent']

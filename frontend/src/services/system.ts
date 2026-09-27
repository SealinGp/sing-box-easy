import { ApiService } from './api'
import type { SystemInfo, BasicResponse, LanClients } from '../types/api'

/** Host and version details shown in the Settings "About" card. */
export class SystemService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getInfo(): Promise<SystemInfo> {
    const { data } = await this.api.get<BasicResponse<SystemInfo>>('/system/info')
    return data.data
  }

  /**
   * LAN devices from the router's DHCP leases, static hosts and neighbour
   * table — the vocabulary for a route rule's source_mac_address /
   * source_hostname. `supported` is false off OpenWrt.
   */
  async getLanClients(): Promise<LanClients> {
    const { data } = await this.api.get<BasicResponse<LanClients>>('/system/lan-clients')
    return data.data
  }
}

import { computed } from 'vue'
import { useDeployment } from './useDeployment'

/**
 * Recommended values for the setup wizard, per platform.
 *
 * The wizard used to ship one set of defaults for every host, which left an
 * OpenWrt operator guessing at paths and — worse — accepting defaults that are
 * actively wrong on a router. Everything here is a *recommendation*: fields
 * stay editable, the values are just pre-filled with what actually works.
 *
 * Kept in one place so the values quoted in openwrt_install.md and the ones the
 * UI pre-fills cannot drift apart.
 */

/**
 * Default TUN address.
 *
 * NOT sing-box's own default of 172.19.0.1/30. Docker allocates bridge
 * networks from 172.17.0.0/16 upwards, so on any router also running Docker
 * — which describes most x86 OpenWrt boxes — 172.19.x collides with a live
 * bridge and traffic for that subnet disappears into the tunnel. 172.16.250/30
 * sits below Docker's range and outside the usual 192.168.x LAN space.
 */
export const RECOMMENDED_TUN_ADDRESS = '172.16.250.1/30'
export const RECOMMENDED_TUN_ADDRESS_V6 = 'fdfe:dcba:9876::1/126'

/** Where OpenWrt keeps sing-box state. Matches the ipk's layout. */
const OPENWRT_SINGBOX_DIR = '/etc/sing-box'

export interface SetupDefaults {
  /** sing-box log level. */
  logLevel: string
  /**
   * sing-box `log.output`.
   *
   * Empty on OpenWrt is deliberate and load-bearing: with no file configured
   * sing-box writes to stdout/stderr, procd forwards that to syslog, and the
   * panel's Log page reads it back with `logread`. Point it at a file and the
   * Log page shows nothing unless that exact file exists and is readable.
   */
  logOutput: string
  /** Clash API listen address. */
  clashController: string
  /** Directory the Clash dashboard is served from. */
  clashExternalUI: string
  /** sing-box cache database — persistent, so rule-sets survive a reboot. */
  cachePath: string
  tunAddress: string[]
}

const openwrtDefaults: SetupDefaults = {
  logLevel: 'info',
  logOutput: '',
  clashController: '0.0.0.0:9090',
  clashExternalUI: `${OPENWRT_SINGBOX_DIR}/ui`,
  cachePath: `${OPENWRT_SINGBOX_DIR}/cache.db`,
  tunAddress: [RECOMMENDED_TUN_ADDRESS, RECOMMENDED_TUN_ADDRESS_V6],
}

// Relative paths resolve against the working directory, which for a tarball
// install is the install directory — keeping everything self-contained.
const genericDefaults: SetupDefaults = {
  logLevel: 'info',
  logOutput: '',
  clashController: '127.0.0.1:9090',
  clashExternalUI: 'ui',
  cachePath: 'cache.db',
  tunAddress: [RECOMMENDED_TUN_ADDRESS, RECOMMENDED_TUN_ADDRESS_V6],
}

export function useSetupDefaults() {
  const { isOpenWrt } = useDeployment()

  const defaults = computed<SetupDefaults>(() =>
    isOpenWrt.value ? openwrtDefaults : genericDefaults,
  )

  return { defaults, isOpenWrt }
}

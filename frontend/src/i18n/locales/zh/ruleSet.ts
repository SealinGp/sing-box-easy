// DNS 与路由探测共用：两者对规则集的报告方式一致。
export default {
  unavailable: '部分规则集无法读取，使用它们的规则因此无法判定：',
  reason: {
    unknown_tag: '配置中不存在该标签的规则集。',
    not_cached: '尚未下载。请启动 sing-box 让它抓取，或在「规则集」标签页更新。',
    cache_unavailable:
      '无法读取 sing-box 缓存文件。远程规则集保存在该文件中，而它只存在于真正以此配置运行过 sing-box 的机器上。',
    cache_disabled: 'experimental.cache_file 未开启，远程规则集仅存在于运行进程的内存中。',
    file_missing: '配置的路径下不存在该本地规则集文件。',
    unsupported_srs_version: '由更新版本的 sing-box 生成，本面板无法解析，且已安装的二进制也无法代为判定。',
    parse_error: '已找到内容，但无法解码。',
  },
}

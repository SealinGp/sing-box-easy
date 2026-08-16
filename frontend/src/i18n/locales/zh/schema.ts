// 基于 schema 的表单（入站 / DNS 服务器 / 出站）共用的文案。
// 版本相关措辞放在这里而非各自的域，因为判断逻辑到处都一样：生成的字段清单
// 描述的是固定依赖的 sing-box 库，而这些文案说明的是实际安装的二进制会接受什么。
export default {
  field: {
    retired: '已在 sing-box {removed} 中移除；本机运行的是 {version}，将拒绝该字段。',
    deprecated: '自 sing-box {since} 起已弃用。',
    deprecatedUnversioned: '上游已弃用 —— 仍可使用，但新配置不应再使用。',
    retiredHidden: '另有 {count} 个字段已隐藏：它们在不高于当前 sing-box {version} 的版本中已被移除。',
    typeRetired: '已在 sing-box {removed} 中移除 —— 本机运行的是 {version}，将拒绝该类型。',
  },
}

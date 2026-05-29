// 初始化向导外壳（InitWizard.vue）+ 各步骤标题。
export default {
  title: 'Sing-box Easy 安装向导',
  welcome: '欢迎！让我们开始配置 sing-box',
  loadingStatus: '正在加载初始化状态...',
  comingSoon: '功能即将推出...',
  stepN: '步骤 {n}：{title}',
  failedStatus: '获取初始化状态失败',
  steps: {
    install: '安装 sing-box',
    log: '配置日志',
    experimental: '配置实验性功能',
    dashboard: '下载控制面板',
    outbounds: '配置出站',
    ruleSets: '配置路由 / 规则集',
    dns: '配置 DNS',
    inbounds: '配置入站',
    routes: '配置路由',
    complete: '完成',
  },
}

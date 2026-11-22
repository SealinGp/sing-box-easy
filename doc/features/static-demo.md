# GitHub Pages 静态演示

线上入口：https://sealingp.github.io/sing-box-easy/

演示复用真实产品组件，包含概览、代理节点、订阅、路由规则、DNS 与节点质量标准。
桌面默认侧边栏，可在页头切换顶部导航；选择保存在浏览器中。窄屏自动使用顶部导航。

所有数据由 `frontend/src/demo/data.ts` 生成，不包含线上数据库、订阅密钥或设备配置。

- 独立入口 `frontend/demo/index.html`，不进入正常应用或 release 安装包。
- `frontend/src/demo/api.ts` 在挂载前替换 API 方法；未覆盖的请求明确拒绝，不回退到网络。
- 概览流量 SSE 来自内存模拟流，离页时取消；浏览器测速和 IP 查询通过专用 Vite alias 替换，避免访问外部探测站点。
- 节点选择、模拟测速、质量标准只改变页面内存，刷新恢复。
- 其他配置提交和订阅更新提示安装后使用，不执行真实操作。
- Hash 路由支持 Pages 子目录、深链接和刷新，无需服务器重写规则。

## 本地验证

```sh
cd frontend
bun run test
bun run build:demo
bun run preview:demo
```

访问 http://localhost:5180/sing-box-easy/ 。可用 `PAGES_BASE_PATH=/` 调整部署路径。
生产管理界面仍使用 `bun run build`，输出目录为 `dist/`；演示单独输出到忽略的 `pages-dist/`。

## 发布

仓库 Settings → Pages → Source 选择 GitHub Actions。
`.github/workflows/pages.yml` 在 main 的前端或工作流更新后自动部署，也支持手动触发。
使用官方 configure-pages / upload-pages-artifact / deploy-pages 工作流：
https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages

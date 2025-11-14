以下步骤均需要在sing-box-easy的前端(待开发)引导用户一步步操作即可

步骤一 : sing-box节点配置生成(docs/config.json为模板)
1. outbound节点信息解析: 根据输入的 base64encode 节点/节点订阅(每行一个/v1/parse-nodes 接口已实现) 去解析singbox 的outbound节点信息
1.1 outbound节点逻辑:
1.1.1 解析并获取节点首先放入全部节点,并分一个统一的组: GLOBAL
1.1.1 解析并获取节点后根据国家分组: 
1.1.2 自动选择默认配置有
1.1.3 直连inbounds节点默认有
2. dns服务器配置:
2.1 dns服务器
  - 直连配置
  - 代理配置
  - 静态DNS配置(可选)

2.2 dns规则(根据docs/config.json模板调整)
2.2.1 dns servers
  dns_direct默认有
  dns_proxy默认有
  hosts 前端需要提示可选,是否增加
2.2.2 dns rules:
  假设增加了hosts配置,则dns_lan第一个,其他默认有

3. inbounds配置: 默认tun + mixed-in
4. route配置:
4.1 auto_detect_interface默认true
4.2 final默认直连 (白名单策略)
4.3 route.rule_set:
模板内的默认都有,除了SealinGp的，这个是需要用户手动输入自定义的规则集
4.4 route.rules:
ip_cidr: ["172.17.0.0/16"] -> 直连
sniff默认开
dns默认hijack-dns
clash_mode direct,global默认有
其他的规则集参考模板,rule_set默认用数组，不要用字符串,因为这样用户可以自行添加


步骤二: sing-box内核下载与配置


步骤三: sing-box ui zdashboard下载与放置到指定位置,


以上步骤配置完毕后，在sing-box-easy web前端均可以修改，目前前端支持的功能:
1. 启动/重启/暂停 sing-box内核服务,所有启动类的命令均需要先用sing-box命令(因此sing-box命令需要在sing-box-easy后端对接,用命令行调用)测试配置文件是否可用
2. 修改sing-box配置(重点)
注意: sing box配置的修改需要: 1.假设现在的配置文件为config.json,先copy一个现在的配置,config_new.json,当config_new.json配置测试没问题的时候，将config.json -> config.old.json,config_new.json -> config.json,这样做的好处是: 可以随时回滚到上一个正确的配置(如果配置有问题的话)
2.1 链式代理配置功能(前端一步步引导配置,配置完毕后提示用户重启sing-box)
2.2 
3. 订阅链接定时更新


请你根据以上我的需求，设计一个v1.12.12版api,api的设计应该用层层递进的关系，因为这个配置就是这样的
比如 添加inbounds -> /1.12.12/inbound/add
比如 inbounds编辑更新 -> /1.12.12/inbound/$inbound_tag/edit
比如 inbounds编辑更新 -> /1.12.12/inbound/$inbound_tag/edit

目前来说，模板配置文件里面没有的配置，可以暂时不支持接口修改，等我后期发现了再加，另外singbox内核是有版本的，我们的版本支持也应该对应其版本，目前版本就支持 1.12.12，因此接口应该这样设计: /1.12.12/xxx，
目前来说你暂时别做前端，只做接口设计以及接口实现,接口设计跟实现分为两步，第一步设计完了先给我审核，并告知每个接口设计的作用以及修改的是哪部分配置内容
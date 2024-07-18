## 接口文档

目前这模块集成了4个主要接口，post数据主要以json进行传输

- hermit
- 登录 [swift获取方式](https://gist.github.com/zhyc9de/842f2157743d33e2c02c0720a8a33635)
- 更新fcm_token 接入firebase
- 校验支付receipt

### header字段

- request中需要添加的字段
    - Accept-Language (当中文的时候，只有台湾地区写zh_TW，其他为zh_CN)
    - bundle-id
    - version
- response需要处理的字段
    - set-token 更新用户token

## 旧版后台迁移指南

- response header新增set-token，用于更新前端token
- user表新增ip_addr
- 既然数据上了RDS，那么注意表引擎改用x-engine

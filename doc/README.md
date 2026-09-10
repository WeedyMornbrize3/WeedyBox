# WeedyBox 源码文档索引

Wails v3 (Go) + Vue 3 + Pinia + UnoCSS 桌面应用。本目录只记录**常用函数及其位置**，不含教程与实现细节。

| 文档 | 内容 |
| --- | --- |
| [backend.md](./backend.md) | Go 后端：数据库层、服务层、数据模型 |
| [frontend.md](./frontend.md) | 前端：路由、Pinia store、组件、API 封装 |
| [styles.md](./styles.md) | UnoCSS 自定义规则、shortcuts、CSS 变量主题 |
| [conventions.md](./conventions.md) | 目录约定、构建命令、注意事项 |

## 目录速查

```
main.go                        应用入口 / 依赖注入 / 窗口创建
internal/model/                数据结构 (Todo, DTO, Stats)
internal/database/             SQLite 通用执行层
internal/service/              业务服务 (Todo, Window)
frontend/src/router/           路由表
frontend/src/stores/           Pinia 状态 (todo / sidebar / Theme)
frontend/src/services/         Wails 绑定封装
frontend/src/layouts/          页面骨架
frontend/src/views/            页面
frontend/src/components/       组件 (todo / settings)
frontend/src/styles/           CSS 变量主题
frontend/bindings/             自动生成，勿手改
frontend/uno.config.ts         UnoCSS 规则与 shortcuts
```

## 数据流

```
Vue 组件 → stores/*.ts → services/todoService.ts → bindings/…/todoservice.ts
        → Go service.TodoService → database.SQLiteService → SQLite
```

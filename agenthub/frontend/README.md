# AgentHub Frontend

基于React + TypeScript + Tailwind CSS的现代化AgentHub管理界面。

## 功能特性

### 🎯 核心功能

- **Dashboard概览** - 系统状态、Agent统计、快速操作
- **Agent管理** - 列表查看、详情展示、状态控制
- **Agent注册** - 表单化注册新Agent，支持技能配置
- **能力发现** - 根据技能ID查找Agent服务地址
- **智能分析** - AI驱动的需求分析和Agent匹配

### 🎨 界面特性

- **响应式设计** - 适配桌面端和移动端
- **现代化UI** - 基于Tailwind CSS的美观界面
- **实时状态** - 实时显示Agent在线状态
- **操作反馈** - 完善的成功/错误提示
- **快速操作** - 复制、链接跳转等便捷功能

### 🔧 技术特性

- **TypeScript** - 完整的类型定义和检查
- **模块化设计** - 清晰的组件和服务分离
- **API集成** - 完整的后端API调用封装
- **错误处理** - 统一的错误处理机制

## 项目结构

```
frontend/
├── public/
│   └── index.html          # HTML模板
├── src/
│   ├── components/         # 通用组件
│   │   └── Layout.tsx      # 布局组件
│   ├── pages/              # 页面组件
│   │   ├── Dashboard.tsx   # 概览页面
│   │   ├── AgentList.tsx   # Agent列表
│   │   ├── AgentDetail.tsx # Agent详情
│   │   ├── AgentRegister.tsx # Agent注册
│   │   ├── AgentDiscover.tsx # 能力发现
│   │   └── ContextAnalysis.tsx # 智能分析
│   ├── services/           # API服务
│   │   └── api.ts          # API调用封装
│   ├── types/              # 类型定义
│   │   └── index.ts        # TypeScript类型
│   ├── utils/              # 工具函数
│   │   └── index.ts        # 通用工具
│   ├── App.tsx             # 主应用组件
│   ├── index.tsx           # 入口文件
│   └── index.css           # 全局样式
├── package.json            # 依赖配置
├── tailwind.config.js      # Tailwind配置
├── tsconfig.json           # TypeScript配置
└── postcss.config.js       # PostCSS配置
```

## 快速开始

### 1. 安装依赖

```bash
cd frontend
npm install
```

### 2. 启动开发服务器

```bash
npm start
```

访问 http://localhost:3000 查看应用

### 3. 构建生产版本

```bash
npm run build
```

## API配置

默认API地址为 `http://localhost:8080`，如需修改可以：

1. 在根目录创建 `.env` 文件
2. 添加环境变量：
   ```
   REACT_APP_API_BASE_URL=http://your-api-server:port
   ```

## 使用说明

### 1. 概览页面 (/)

- 查看系统整体状态
- 显示Agent统计信息
- 提供快速操作入口

### 2. Agent管理 (/agents)

- 列表显示所有注册的Agent
- 支持搜索和状态筛选
- 快速操作：心跳、启停、删除

### 3. Agent注册 (/register)

- 表单化注册新Agent
- 支持技能配置和能力设置
- 提供注册模板

### 4. 能力发现 (/discover)

- 通过技能ID查找Agent
- 显示匹配的Agent信息
- 提供URL复制和跳转

### 5. 智能分析 (/analyze)

- 自然语言需求输入
- AI智能匹配相关技能
- 推荐最佳Agent服务

## 开发指南

### 添加新页面

1. 在 `src/pages/` 创建新组件
2. 在 `App.tsx` 中添加路由
3. 在 `Layout.tsx` 中添加导航菜单

### 添加新API

1. 在 `src/types/` 定义相关类型
2. 在 `src/services/api.ts` 添加API方法
3. 在组件中使用API调用

### 样式定制

- 修改 `tailwind.config.js` 中的主题配置
- 在 `src/index.css` 中添加自定义样式类
- 使用Tailwind CSS的工具类进行样式设计

## 部署说明

### 1. 构建应用

```bash
npm run build
```

### 2. 部署静态文件

将 `build/` 目录中的文件部署到Web服务器

### 3. 配置代理

如果前后端部署在不同域名，需要配置CORS或代理

## 故障排除

### 常见问题

1. **API连接失败**
    - 检查AgentHub后端服务是否启动
    - 确认API地址配置是否正确
    - 检查网络连接和防火墙设置

2. **页面空白**
    - 检查浏览器控制台的错误信息
    - 确认所有依赖是否正确安装
    - 尝试清除浏览器缓存

3. **样式问题**
    - 确认Tailwind CSS是否正确编译
    - 检查PostCSS配置是否正确
    - 重新启动开发服务器

### 调试技巧

1. 开启浏览器开发者工具
2. 查看Network标签页的API请求
3. 检查Console的错误信息
4. 使用React DevTools扩展

## 技术栈

- **React 18** - 前端框架
- **TypeScript** - 类型系统
- **Tailwind CSS** - CSS框架
- **React Router** - 路由管理
- **Axios** - HTTP客户端
- **Lucide React** - 图标库

## 许可证

此项目遵循MIT许可证。
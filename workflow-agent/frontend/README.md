# Workflow Agent Frontend

<div align="center">

![Workflow Agent](https://img.shields.io/badge/Workflow-Agent-blue)
![React](https://img.shields.io/badge/React-19.1.1-61dafb)
![TypeScript](https://img.shields.io/badge/TypeScript-5.9.3-3178c6)
![Tailwind CSS](https://img.shields.io/badge/Tailwind-4.1.16-38bdf8)
![License](https://img.shields.io/badge/License-Apache%202.0-green)

一个优雅、功能强大的AI驱动工作流编排平台前端

</div>

---

## ✨ 特性亮点

- 🎨 **优雅界面** - 现代化设计,精心打磨的视觉效果
- 📡 **实时追踪** - SSE实时进度流,即时反馈编排状态
- 🎯 **智能编排** - AI驱动的工作流自动生成
- 📊 **可视化画布** - React Flow驱动的交互式工作流图
- 💬 **自然交互** - 用自然语言描述需求即可
- 🗂️ **多会话管理** - 并发处理多个工作流编排
- 📱 **响应式设计** - 完美适配各种屏幕尺寸
- 🚀 **高性能** - Vite构建,快速加载

## 🚀 快速开始

### 前置要求

- Node.js 18+
- npm 或 yarn
- Workflow Agent 后端运行在 `http://localhost:8081`

### 安装

```bash
# 进入frontend目录
cd workflow-agent/frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

访问 http://localhost:3000

### 构建生产版本

```bash
npm run build
npm run preview
```

## 📚 使用指南

### 1. 创建工作流

1. 点击"创建新工作流"按钮
2. 输入详细的工作流描述,例如:
   ```
   创建一个多Agent工作流来生成综合研究报告。工作流应该:
   1) 使用网络搜索Agent从多个来源收集信息
   2) 使用数据分析Agent处理和总结收集的数据
   3) 使用文档生成Agent创建格式良好的报告
   ```
3. (可选)添加JSON格式的上下文信息
4. 点击"开始编排"
5. 实时查看编排进度
6. 工作流生成完成后查看可视化画布

### 2. 查看进度

- **实时状态更新** - 分析中 → 发现Agent → 生成工作流 → 完成
- **详细步骤展示** - 每一步都有时间戳和描述
- **能力发现结果** - 查看匹配的Agent和能力

### 3. 工作流可视化

- **交互操作**
  - 鼠标滚轮缩放
  - 拖动平移画布
  - 点击节点查看详情
- **导出功能** - 下载完整的工作流JSON定义

### 4. 会话管理

- 查看所有历史会话
- 支持多个并发会话
- 快速切换和查看

## 🏗️ 技术架构

### 技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| React | 19.1.1 | UI框架 |
| TypeScript | 5.9.3 | 类型系统 |
| Tailwind CSS | 4.1.16 | 样式框架 |
| React Flow | 11.11.4 | 工作流可视化 |
| Vite | 7.1.7 | 构建工具 |
| Lucide React | 0.548.0 | 图标库 |

### 项目结构

```
src/
├── components/
│   ├── Layout/          # 布局组件(Header, Sidebar)
│   ├── Session/         # 会话组件(表单,列表,详情,进度)
│   └── Workflow/        # 工作流组件(画布)
├── services/
│   └── api.ts           # API客户端封装
├── types/
│   └── api.ts           # TypeScript类型定义
├── utils/
│   ├── cn.ts            # 类名工具
│   └── formatters.ts    # 格式化工具
├── App.tsx              # 主应用
├── main.tsx             # 入口文件
└── index.css            # 全局样式
```

## 🔌 API集成

### 后端接口

```typescript
// 创建会话
POST /api/v1/sessions
{
  "description": "工作流描述",
  "context": { "key": "value" }
}

// 获取会话列表
GET /api/v1/sessions

// 实时进度流(SSE)
GET /api/v1/sessions/{id}/stream

// 获取最终结果
GET /api/v1/sessions/{id}/result
```

### SSE事件流

```javascript
const eventSource = new EventSource('/api/v1/sessions/{id}/stream');

eventSource.addEventListener('progress', (e) => {
  const event = JSON.parse(e.data);
  // event.status: pending|analyzing|discovering|generating|completed|failed
  // event.message: 进度描述
  // event.capabilities: 发现的能力
  // event.workflow: 生成的工作流
});
```

## 🎨 设计系统

### 颜色方案

| 用途 | 颜色 | Hex |
|------|------|-----|
| 主色 | Primary | #0284c7 |
| 成功 | Success | #10b981 |
| 警告 | Warning | #f59e0b |
| 错误 | Error | #ef4444 |
| 信息 | Info | #3b82f6 |

### 状态标识

- 🔵 **等待中** - 灰色
- 🔵 **分析中** - 蓝色
- 🟣 **发现Agent** - 紫色
- 🟡 **生成工作流** - 黄色
- 🟢 **已完成** - 绿色
- 🔴 **失败** - 红色

## 📖 文档

- [ARCHITECTURE.md](./ARCHITECTURE.md) - 详细的技术架构文档
- [USAGE.md](./USAGE.md) - 完整的使用指南

## 🔧 开发

### 脚本命令

```bash
npm run dev       # 启动开发服务器
npm run build     # 构建生产版本
npm run preview   # 预览生产版本
npm run lint      # 代码检查
```

### 环境变量

```env
# .env.development
VITE_API_BASE_URL=http://localhost:8081
```

## 🐛 故障排查

### 构建失败

```bash
# 清除缓存重新安装
rm -rf node_modules package-lock.json
npm install
npm run build
```

### SSE连接失败

1. 检查后端是否运行
2. 确认CORS配置正确
3. 检查浏览器控制台错误

### 样式不生效

1. 清除浏览器缓存
2. 检查Tailwind配置
3. 重新构建项目

## 📄 许可证

Apache License 2.0

## 🙏 致谢

- [React](https://react.dev/) - UI框架
- [Tailwind CSS](https://tailwindcss.com/) - CSS框架
- [React Flow](https://reactflow.dev/) - 工作流可视化
- [Lucide](https://lucide.dev/) - 图标库
- [Vite](https://vitejs.dev/) - 构建工具

---

<div align="center">

**用❤️打造 | 为workflow-agent提供优雅的前端体验**

</div>

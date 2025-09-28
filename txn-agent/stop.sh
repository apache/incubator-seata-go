#!/bin/bash

# 停止 Seata Workflow Agent 服务脚本

echo "🛑 停止 Seata Workflow Agent 服务..."

# 默认端口
BACKEND_PORT=8080
FRONTEND_PORT=5173

# 停止后端服务
echo "🔴 停止后端服务..."
if [ -f "backend.pid" ]; then
    BACKEND_PID=$(cat backend.pid)
    if ps -p $BACKEND_PID > /dev/null 2>&1; then
        kill -TERM $BACKEND_PID
        echo "   后端服务已停止 (PID: $BACKEND_PID)"
    fi
    rm -f backend.pid
else
    # 通过端口查找并停止
    BACKEND_PID=$(lsof -ti:$BACKEND_PORT 2>/dev/null)
    if [ ! -z "$BACKEND_PID" ]; then
        kill -TERM $BACKEND_PID
        echo "   后端服务已停止 (PID: $BACKEND_PID)"
    fi
fi

# 停止前端服务
echo "🔴 停止前端服务..."
if [ -f "frontend.pid" ]; then
    FRONTEND_PID=$(cat frontend.pid)
    if ps -p $FRONTEND_PID > /dev/null 2>&1; then
        kill -TERM $FRONTEND_PID
        echo "   前端服务已停止 (PID: $FRONTEND_PID)"
    fi
    rm -f frontend.pid
else
    # 通过端口查找并停止
    FRONTEND_PID=$(lsof -ti:$FRONTEND_PORT 2>/dev/null)
    if [ ! -z "$FRONTEND_PID" ]; then
        kill -TERM $FRONTEND_PID
        echo "   前端服务已停止 (PID: $FRONTEND_PID)"
    fi
fi

# 停止所有相关的Node.js进程 (Vite开发服务器)
NODE_PIDS=$(ps aux | grep 'vite\|npm run dev' | grep -v grep | awk '{print $2}')
if [ ! -z "$NODE_PIDS" ]; then
    echo "🔴 停止 Node.js 开发服务器..."
    for pid in $NODE_PIDS; do
        kill -TERM $pid 2>/dev/null
        echo "   停止进程 PID: $pid"
    done
fi

# 停止所有相关的Go进程
GO_PIDS=$(ps aux | grep 'websocket-server\|workflow-agent' | grep -v grep | awk '{print $2}')
if [ ! -z "$GO_PIDS" ]; then
    echo "🔴 停止 Go WebSocket 服务器..."
    for pid in $GO_PIDS; do
        kill -TERM $pid 2>/dev/null
        echo "   停止进程 PID: $pid"
    done
fi

# 等待进程结束
sleep 2

# 强制停止未响应的进程
echo "🔍 检查残留进程..."
REMAINING=$(lsof -ti:$BACKEND_PORT,$FRONTEND_PORT 2>/dev/null)
if [ ! -z "$REMAINING" ]; then
    echo "🚫 强制停止残留进程..."
    for pid in $REMAINING; do
        kill -KILL $pid 2>/dev/null
        echo "   强制停止进程 PID: $pid"
    done
fi

echo
echo "✅ 所有服务已停止"

# 显示当前端口状态
echo "📊 端口状态检查:"
if lsof -Pi :$BACKEND_PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "   ❌ 后端端口 $BACKEND_PORT 仍被占用"
else
    echo "   ✅ 后端端口 $BACKEND_PORT 已释放"
fi

if lsof -Pi :$FRONTEND_PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "   ❌ 前端端口 $FRONTEND_PORT 仍被占用"
else
    echo "   ✅ 前端端口 $FRONTEND_PORT 已释放"
fi

echo
echo "💡 如需重新启动，请运行:"
echo "   ./start.sh          # 完整启动脚本"
echo "   ./quick-start.sh    # 快速启动脚本"
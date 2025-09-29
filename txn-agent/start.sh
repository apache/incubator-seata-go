#!/bin/bash
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# Seata Workflow Agent 启动脚本
# 联合启动前端和后端服务

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# 配置变量 - 使用用户提供的默认值
BACKEND_PORT=${PORT:-8080}
FRONTEND_PORT=5173
API_KEY=${API_KEY:-"sk-1cd306a727af41c29267b17fd1ba750b"}
LOG_LEVEL=${LOG_LEVEL:-"info"}
MODEL=${MODEL:-"qwen-max"}
BASE_URL=${BASE_URL:-"https://dashscope.aliyuncs.com/compatible-mode/v1"}

# PID文件
BACKEND_PID_FILE="backend.pid"
FRONTEND_PID_FILE="frontend.pid"

# 日志文件
BACKEND_LOG="logs/backend.log"
FRONTEND_LOG="logs/frontend.log"

# 打印带颜色的消息
print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# 显示Logo
show_logo() {
    echo -e "${CYAN}"
    cat << 'EOF'
    ╔═══════════════════════════════════════════════════════════════════╗
    ║                                                                   ║
    ║               🚀 Seata Workflow Agent 启动器                      ║
    ║                                                                   ║
    ║         分布式事务工作流设计与可视化平台                             ║
    ║                                                                   ║
    ╚═══════════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
}

# 检查依赖
check_dependencies() {
    print_message $BLUE "📋 检查系统依赖..."
    
    # 检查Go
    if ! command -v go &> /dev/null; then
        print_message $RED "❌ 错误：未找到 Go 环境"
        echo "   请安装 Go 1.20 或更高版本"
        exit 1
    fi
    
    # 检查Node.js
    if ! command -v node &> /dev/null; then
        print_message $RED "❌ 错误：未找到 Node.js 环境"
        echo "   请安装 Node.js 18 或更高版本"
        exit 1
    fi
    
    # 检查npm
    if ! command -v npm &> /dev/null; then
        print_message $RED "❌ 错误：未找到 npm"
        echo "   请安装 npm 包管理器"
        exit 1
    fi
    
    print_message $GREEN "✅ 依赖检查通过"
}

# 检查端口是否被占用
check_port() {
    local port=$1
    local service=$2
    
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        print_message $YELLOW "⚠️  警告：端口 $port 已被占用 ($service)"
        read -p "是否要杀死占用端口的进程? (y/n): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_message $BLUE "🔧 正在释放端口 $port..."
            sudo lsof -ti:$port | xargs sudo kill -9 2>/dev/null || true
            sleep 2
        else
            print_message $RED "❌ 无法启动 $service - 端口冲突"
            exit 1
        fi
    fi
}

# 创建必要的目录
setup_directories() {
    print_message $BLUE "📁 创建必要目录..."
    mkdir -p logs
    mkdir -p bin
}

# 编译后端
build_backend() {
    print_message $BLUE "🔨 编译后端服务..."
    
    if [ ! -f "go.mod" ]; then
        print_message $RED "❌ 错误：未找到 go.mod 文件"
        echo "   请确保在项目根目录执行此脚本"
        exit 1
    fi
    
    print_message $BLUE "   正在编译简化版 WebSocket 服务器..."
    go build -ldflags="-s -w" -o bin/websocket-server ./cmd/simple-agent/
    
    if [ $? -eq 0 ]; then
        print_message $GREEN "✅ 后端编译成功"
    else
        print_message $RED "❌ 后端编译失败"
        exit 1
    fi
}

# 安装前端依赖
install_frontend_deps() {
    print_message $BLUE "📦 安装前端依赖..."
    
    if [ ! -d "frontend" ]; then
        print_message $RED "❌ 错误：未找到 frontend 目录"
        exit 1
    fi
    
    cd frontend
    
    if [ ! -f "package.json" ]; then
        print_message $RED "❌ 错误：未找到 package.json 文件"
        exit 1
    fi
    
    # 检查是否需要安装依赖
    if [ ! -d "node_modules" ] || [ "package.json" -nt "node_modules" ]; then
        print_message $BLUE "   正在安装 npm 依赖..."
        npm install --silent
        if [ $? -eq 0 ]; then
            print_message $GREEN "✅ 前端依赖安装成功"
        else
            print_message $RED "❌ 前端依赖安装失败"
            exit 1
        fi
    else
        print_message $GREEN "✅ 前端依赖已是最新"
    fi
    
    cd ..
}

# 启动后端服务
start_backend() {
    print_message $BLUE "🚀 启动后端服务..."
    
    check_port $BACKEND_PORT "后端服务"
    
    # 启动后端
    nohup ./bin/websocket-server \
        -port=$BACKEND_PORT \
        -api-key="$API_KEY" \
        -base-url="$BASE_URL" \
        -model="$MODEL" \
        -log-level="$LOG_LEVEL" \
        -timeout=120 \
        -max-retries=3 \
        > $BACKEND_LOG 2>&1 &
    
    BACKEND_PID=$!
    echo $BACKEND_PID > $BACKEND_PID_FILE
    
    # 等待后端启动
    print_message $BLUE "   等待后端服务启动..."
    for i in {1..10}; do
        if curl -s http://localhost:$BACKEND_PORT/health >/dev/null 2>&1; then
            print_message $GREEN "✅ 后端服务启动成功 (PID: $BACKEND_PID)"
            break
        fi
        sleep 1
        if [ $i -eq 10 ]; then
            print_message $RED "❌ 后端服务启动超时"
            cat $BACKEND_LOG
            exit 1
        fi
    done
}

# 启动前端服务
start_frontend() {
    print_message $BLUE "🌐 启动前端服务..."
    
    check_port $FRONTEND_PORT "前端服务"
    
    cd frontend
    
    # 启动前端开发服务器
    nohup npm run dev > ../$FRONTEND_LOG 2>&1 &
    
    FRONTEND_PID=$!
    echo $FRONTEND_PID > ../$FRONTEND_PID_FILE
    
    cd ..
    
    # 等待前端启动
    print_message $BLUE "   等待前端服务启动..."
    for i in {1..15}; do
        if curl -s http://localhost:$FRONTEND_PORT >/dev/null 2>&1; then
            print_message $GREEN "✅ 前端服务启动成功 (PID: $FRONTEND_PID)"
            break
        fi
        sleep 1
        if [ $i -eq 15 ]; then
            print_message $RED "❌ 前端服务启动超时"
            cat $FRONTEND_LOG
            exit 1
        fi
    done
}

# 显示服务状态
show_status() {
    echo
    print_message $WHITE "📊 服务状态信息"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # 后端状态
    if curl -s http://localhost:$BACKEND_PORT/health >/dev/null 2>&1; then
        print_message $GREEN "🟢 后端服务：运行中"
        echo "   📍 WebSocket: ws://localhost:$BACKEND_PORT/ws"
        echo "   📍 健康检查: http://localhost:$BACKEND_PORT/health"
        echo "   📍 状态接口: http://localhost:$BACKEND_PORT/api/v1/status"
        echo "   📊 模型: $MODEL"
        echo "   📝 日志级别: $LOG_LEVEL"
    else
        print_message $RED "🔴 后端服务：未运行"
    fi
    
    echo
    
    # 前端状态
    if curl -s http://localhost:$FRONTEND_PORT >/dev/null 2>&1; then
        print_message $GREEN "🟢 前端服务：运行中"
        echo "   📍 应用地址: http://localhost:$FRONTEND_PORT"
        echo "   🎨 支持主题: 深色/浅色/霓虹"
        echo "   📱 响应式设计"
    else
        print_message $RED "🔴 前端服务：未运行"
    fi
    
    echo
    print_message $CYAN "💡 使用说明："
    echo "   1. 打开浏览器访问 http://localhost:$FRONTEND_PORT"
    echo "   2. 在右侧聊天界面输入业务需求"
    echo "   3. 查看左侧任务规划和中间的工作流可视化"
    echo "   4. 可以切换不同主题和视图模式"
    echo "   5. 按 Ctrl+C 停止所有服务"
    
    echo
    print_message $YELLOW "📁 日志文件："
    echo "   后端: $BACKEND_LOG"
    echo "   前端: $FRONTEND_LOG"
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# 清理函数
cleanup() {
    echo
    print_message $YELLOW "🛑 正在停止服务..."
    
    # 停止后端
    if [ -f "$BACKEND_PID_FILE" ]; then
        BACKEND_PID=$(cat $BACKEND_PID_FILE)
        if ps -p $BACKEND_PID > /dev/null 2>&1; then
            print_message $BLUE "   停止后端服务 (PID: $BACKEND_PID)"
            kill -TERM $BACKEND_PID 2>/dev/null || true
            # 等待优雅停止
            for i in {1..5}; do
                if ! ps -p $BACKEND_PID > /dev/null 2>&1; then
                    break
                fi
                sleep 1
            done
            # 强制停止
            kill -KILL $BACKEND_PID 2>/dev/null || true
        fi
        rm -f $BACKEND_PID_FILE
    fi
    
    # 停止前端
    if [ -f "$FRONTEND_PID_FILE" ]; then
        FRONTEND_PID=$(cat $FRONTEND_PID_FILE)
        if ps -p $FRONTEND_PID > /dev/null 2>&1; then
            print_message $BLUE "   停止前端服务 (PID: $FRONTEND_PID)"
            kill -TERM $FRONTEND_PID 2>/dev/null || true
            # 等待优雅停止
            for i in {1..5}; do
                if ! ps -p $FRONTEND_PID > /dev/null 2>&1; then
                    break
                fi
                sleep 1
            done
            # 强制停止
            kill -KILL $FRONTEND_PID 2>/dev/null || true
        fi
        rm -f $FRONTEND_PID_FILE
    fi
    
    print_message $GREEN "✅ 所有服务已停止"
    exit 0
}

# 显示帮助信息
show_help() {
    echo "用法: $0 [选项]"
    echo
    echo "选项:"
    echo "  -h, --help           显示此帮助信息"
    echo "  -p, --port PORT      设置后端端口 (默认: 8080)"
    echo "  -k, --api-key KEY    设置 API 密钥"
    echo "  -m, --model MODEL    设置 LLM 模型 (默认: qwen-max)"
    echo "  -l, --log-level LVL  设置日志级别 (默认: info)"
    echo "  --build-only         仅编译后端，不启动服务"
    echo "  --frontend-only      仅启动前端服务"
    echo "  --backend-only       仅启动后端服务"
    echo
    echo "环境变量:"
    echo "  API_KEY             API 密钥"
    echo "  MODEL               LLM 模型名称"
    echo "  LOG_LEVEL           日志级别"
    echo "  BASE_URL            LLM API 基础URL"
    echo
    echo "示例:"
    echo "  $0                          # 启动完整服务"
    echo "  $0 -p 9090 -l debug        # 自定义端口和日志级别"
    echo "  $0 --backend-only           # 仅启动后端"
    echo
}

# 解析命令行参数
BUILD_ONLY=false
FRONTEND_ONLY=false
BACKEND_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -p|--port)
            BACKEND_PORT="$2"
            shift 2
            ;;
        -k|--api-key)
            API_KEY="$2"
            shift 2
            ;;
        -m|--model)
            MODEL="$2"
            shift 2
            ;;
        -l|--log-level)
            LOG_LEVEL="$2"
            shift 2
            ;;
        --build-only)
            BUILD_ONLY=true
            shift
            ;;
        --frontend-only)
            FRONTEND_ONLY=true
            shift
            ;;
        --backend-only)
            BACKEND_ONLY=true
            shift
            ;;
        *)
            print_message $RED "未知选项: $1"
            show_help
            exit 1
            ;;
    esac
done

# 注册信号处理
trap cleanup SIGINT SIGTERM

# 主执行流程
main() {
    show_logo
    
    check_dependencies
    setup_directories
    
    if [ "$BUILD_ONLY" = true ]; then
        build_backend
        print_message $GREEN "🎉 编译完成！"
        exit 0
    fi
    
    if [ "$FRONTEND_ONLY" = false ]; then
        build_backend
        start_backend
    fi
    
    if [ "$BACKEND_ONLY" = false ]; then
        install_frontend_deps
        start_frontend
    fi
    
    show_status
    
    print_message $GREEN "🎉 所有服务启动成功！"
    print_message $CYAN "💫 在浏览器中打开 http://localhost:$FRONTEND_PORT 开始使用"
    
    # 等待信号
    print_message $YELLOW "按 Ctrl+C 停止所有服务"
    while true; do
        sleep 1
    done
}

# 执行主函数
main "$@"
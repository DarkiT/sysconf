#!/bin/bash

# Sysconf 基准测试运行脚本
# 这个脚本用于运行配置管理基准测试并生成报告

set -e

echo "🚀 启动 Sysconf 配置管理基准测试"
echo "=================================="

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装或不在PATH中"
    exit 1
fi

echo "✅ Go 版本: $(go version)"

# 进入基准测试目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "📁 当前目录: $(pwd)"

# 确保依赖已下载
echo "📦 检查依赖..."
go mod tidy

# 创建测试数据目录
mkdir -p testdata
cat > testdata/config.yaml << EOF
# 测试配置文件
app:
  name: "benchmark_test"
  version: "1.0.0"

database:
  host: "localhost"
  port: 5432
  username: "testuser"
  
server:
  port: 8080
  debug: false
  timeout: 30

logging:
  level: "info"
  file: "/tmp/test.log"
EOF

echo "✅ 测试数据已创建"

# 设置测试环境变量
echo "🌍 设置测试环境变量..."
export BENCH_DATABASE_HOST="benchmark.db.com"
export BENCH_DATABASE_PORT="5432"
export BENCH_SERVER_DEBUG="true"
export BENCH_LOG_LEVEL="debug"

# 创建一些额外的环境变量用于测试
for i in {1..100}; do
    export "BENCH_TEST_VAR_$i"="test_value_$i"
done

echo "✅ 环境变量设置完成"

# 运行基准测试
echo ""
echo "🏃 开始运行基准测试..."
echo "=================================="

# 运行基准测试工具
if go run benchmark_tool.go; then
    echo ""
    echo "✅ 基准测试完成"
    
    # 检查生成的报告文件
    if [ -f "benchmark_report.md" ]; then
        echo "📊 基准测试报告已生成: benchmark_report.md"
        echo ""
        echo "📋 报告摘要:"
        head -20 benchmark_report.md
        echo "..."
        echo ""
        echo "🔗 查看完整报告: cat benchmark_report.md"
    fi
    
    if [ -f "benchmark_trends.json" ]; then
        echo "📈 趋势数据已生成: benchmark_trends.json"
        echo ""
        echo "📋 数据摘要:"
        head -10 benchmark_trends.json
        echo "..."
        echo ""
        echo "🔗 查看完整数据: cat benchmark_trends.json"
    fi
    
else
    echo "❌ 基准测试运行失败"
    exit 1
fi

# 清理环境变量
echo ""
echo "🧹 清理测试环境..."
unset BENCH_DATABASE_HOST
unset BENCH_DATABASE_PORT  
unset BENCH_SERVER_DEBUG
unset BENCH_LOG_LEVEL

for i in {1..100}; do
    unset "BENCH_TEST_VAR_$i"
done

echo "✅ 环境清理完成"

echo ""
echo "🎉 基准测试流程完成！"
echo "=================================="
echo "📁 生成的文件:"
echo "   - benchmark_report.md (详细报告)"
echo "   - benchmark_trends.json (性能数据)"
echo "   - testdata/config.yaml (测试配置)"
echo ""
echo "💡 使用建议:"
echo "   - 定期运行基准测试监控性能变化"
echo "   - 对比不同版本的性能数据"
echo "   - 根据报告中的建议优化配置处理逻辑" 
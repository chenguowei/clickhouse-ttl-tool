# 项目配置
BINARY_NAME=clickhouse-ttl-tool
VERSION?=1.0.0
BUILD_DIR=dist
LDFLAGS=-ldflags="-s -w -X main.Version=$(VERSION)"

# Go 命令
GO=go
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOTEST=$(GO) test
GOMOD=$(GO) mod

# 默认目标
.DEFAULT_GOAL := help

# 编译当前平台版本
.PHONY: build
build: ## 编译当前平台版本
	@echo "🔨 编译 $(BINARY_NAME)..."
	@$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)
	@echo "✓ 编译完成: $(BINARY_NAME)"

# 编译 Linux AMD64 版本
.PHONY: linux
linux: ## 编译 Linux AMD64 版本
	@echo "🔨 编译 Linux AMD64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64
	@echo "✓ 编译完成: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

# 编译 Linux ARM64 版本
.PHONY: linux-arm
linux-arm: ## 编译 Linux ARM64 版本
	@echo "🔨 编译 Linux ARM64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64
	@echo "✓ 编译完成: $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64"

# 编译 Mac Intel 版本
.PHONY: darwin
darwin: ## 编译 Mac Intel 版本
	@echo "🔨 编译 Mac AMD64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64
	@echo "✓ 编译完成: $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64"

# 编译 Mac Apple Silicon 版本
.PHONY: darwin-arm
darwin-arm: ## 编译 Mac ARM64 版本
	@echo "🔨 编译 Mac ARM64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64
	@echo "✓ 编译完成: $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64"

# 编译所有平台版本
.PHONY: build-all
build-all: linux linux-arm darwin darwin-arm ## 编译所有平台版本
	@echo "✓ 所有平台编译完成"
	@ls -lh $(BUILD_DIR)/

# 运行程序（开发模式）
.PHONY: run
run: ## 运行程序（需要提供参数: make run ARGS="--help"）
	@$(GO) run main.go $(ARGS)

# 运行测试
.PHONY: test
test: ## 运行测试
	@echo "🧪 运行测试..."
	@$(GOTEST) -v ./...

# 运行测试并输出覆盖率
.PHONY: test-coverage
test-coverage: ## 运行测试并生成覆盖率报告
	@echo "🧪 运行测试并生成覆盖率..."
	@$(GOTEST) -v -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ 覆盖率报告: coverage.html"

# 下载依赖
.PHONY: deps
deps: ## 下载并整理依赖
	@echo "📦 下载依赖..."
	@$(GOMOD) download
	@$(GOMOD) tidy
	@echo "✓ 依赖下载完成"

# 清理编译产物
.PHONY: clean
clean: ## 清理编译产物
	@echo "🧹 清理编译产物..."
	@$(GOCLEAN)
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "✓ 清理完成"

# 格式化代码
.PHONY: fmt
fmt: ## 格式化代码
	@echo "📝 格式化代码..."
	@$(GO) fmt ./...
	@echo "✓ 代码格式化完成"

# 代码检查
.PHONY: lint
lint: ## 运行代码检查（需要安装 golangci-lint）
	@echo "🔍 运行代码检查..."
	@golangci-lint run ./...

# 安装到系统
.PHONY: install
install: build ## 安装到 $GOPATH/bin
	@echo "📦 安装 $(BINARY_NAME)..."
	@$(GO) install
	@echo "✓ 安装完成"

# 显示版本信息
.PHONY: version
version: ## 显示版本信息
	@echo "版本: $(VERSION)"

# 帮助信息
.PHONY: help
help: ## 显示帮助信息
	@echo "可用命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36mmake %-15s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "示例:"
	@echo "  make linux          # 编译 Linux 版本"
	@echo "  make build-all      # 编译所有平台"
	@echo "  make run ARGS=\"--help\"  # 运行程序"

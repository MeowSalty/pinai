help: ## 显示帮助信息
	@grep -F -h "##" $(MAKEFILE_LIST) | grep -F -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

run-local: ## 本地运行应用
	make requirements
	go run app.go

requirements: ## 生成go.mod和go.sum文件
	go mod tidy

clean-packages: ## 清理包缓存
	go clean -modcache

gen-config: ## 生成配置文件 (删除旧的query目录并运行生成脚本)
	rm -rf database/query || true
	go run cmd/gen/configuration.go

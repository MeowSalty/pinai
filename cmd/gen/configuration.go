package main

import (
	"github.com/MeowSalty/pinai/database/types"
	"gorm.io/gen"
)

func main() {
	// 初始化生成器配置
	g := gen.NewGenerator(gen.Config{
		OutPath:       "./database/query", // 输出目录，默认是./query
		Mode:          gen.WithDefaultQuery | gen.WithQueryInterface,
		FieldNullable: true,
	})

	// 为指定的模型生成默认 DAO 接口
	// 导入包中所有注册的结构体
	for _, model := range types.Types {
		g.ApplyBasic(model)
	}

	// 执行生成器
	g.Execute()
}

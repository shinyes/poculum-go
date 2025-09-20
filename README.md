# Poculum Go 实现文档

## 概述
Poculum 一种支持int、uint、float32、float64、list（元素类型可以不同）、map（键为字符串类型，值类型可以为poculum-go支持的类型）的数据交换格式

## 特性

- **高性能**: 利用 Go 语言的编译优化和内存管理
- **零依赖**: 仅使用 Go 标准库
- **反射支持**: 自动处理接口类型
- **布尔值支持**: true/false 正确序列化，跨语言兼
- **接口友好**: 支持 interface{}，但具体类型局限在下面所说的数据类型中

## 支持的数据类型

### 基本类型
- **整数**: `int`, `int8/16/32/64`, `uint`, `uint8/16/32/64` - 自动选择最优编码
- **浮点数**: `float32`, `float64` - 高精度浮点数
- **布尔值**: `bool` - true/false
- **字符串**: `string` - UTF-8 编码
- **字节数组**: `[]byte` - 原始二进制数据
- **空值**: `nil` - 支持空值

### 复合类型
- **切片**: `[]T` - 任意类型的切片
- **数组**: `[N]T` - 固定长度数组
- **映射**: `map[string]T` - 字符串键的映射
- **接口**: `interface{}` - 任意类型，但具体类型局限在上面所说的数据类型中

## 快速开始

除了下面的例子之外，还可以使用 WithLimits 创建具有自定义限制的 Poculum 实例。

```go
package main

import (
	"fmt"
	"log"

	poculum "poculum-go/pkg"
)

func main() {
	fmt.Println("=== 基本类型示例 ===")

	list := make([]any, 3)
	list[0] = 1
	list[1] = "2"
	list[2] = nil
	// 基本数据类型
	basicData := map[string]any{
		"integer":       int32(42),
		"float":         float64(3.14159),
		"boolean_true":  true,
		"boolean_false": false,
		"string":        "Hello, 世界!",
		"unicode":       "🌟✨🚀💫",
		"bytes":         []byte("binary data"),
		"null":          nil,
		"list":          list,
	}

	// 序列化
	serialized, err := poculum.DumpPoculum(basicData)
	if err != nil {
		log.Fatal("序列化失败:", err)
	}

	fmt.Printf("序列化后大小: %d 字节\n", len(serialized))
	fmt.Printf("十六进制: %x\n", serialized)

	// 反序列化
	deserialized, err := poculum.LoadPoculum(serialized)
	if err != nil {
		log.Fatal("反序列化失败:", err)
	}

	fmt.Printf("反序列化成功: %+v\n", deserialized)
}
```

# BenchMark BenchmarkPoculumVsJSON
```bash
go test -benchmem -run=^$ -bench ^BenchmarkPoculumVsJSON$ poculum-go
```
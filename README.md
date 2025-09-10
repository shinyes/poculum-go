# poculum Go 实现文档

## 🐹 概述

poculum Go 实现是一个高性能的二进制序列化库，使用 Go 语言的反射机制和类型安全特性。它提供了强类型的 API，完全支持 Go 语言特性，并与其他语言版本保持完全兼容。

## ✨ 特性

- 🚀 **高性能**: 利用 Go 语言的编译优化和内存管理
- 📦 **反射支持**: 自动处理 Go 结构体和接口类型
- 🔄 **布尔值原生支持**: true/false 正确序列化，跨语言兼容
- 🛡️ **类型安全**: 强类型系统，编译时类型检查
- 🎯 **零依赖**: 仅使用 Go 标准库
- 📊 **接口友好**: 支持 interface{} 和自定义类型
- ⚡ **内存高效**: 优化的内存分配和复用
- 🔧 **并发安全**: 支持多协程并发使用

## 📦 安装

### 模块安装

```bash
# 初始化 Go 模块
go mod init your-project

# 复制文件到项目目录
cp main.go your-project/poculum.go

# 或者直接在项目中使用
```

### 直接使用

```go
// 复制 main.go 到项目中，重命名为合适的包名
package main // 或 package poculum

import (
    "encoding/hex"
    "fmt"
    "log"
)

// 使用 Dumppoculum 和 Loadpoculum 函数
```

## 🗂️ 支持的数据类型

### 基本类型
- **整数**: `int`, `int8/16/32/64`, `uint`, `uint8/16/32/64` - 自动选择最优编码
- **浮点数**: `float32`, `float64` - 高精度浮点数
- **布尔值**: `bool` - true/false，编码为 UInt8 格式
- **字符串**: `string` - UTF-8 编码
- **字节数组**: `[]byte` - 原始二进制数据

### 复合类型
- **切片**: `[]T` - 任意类型的切片
- **数组**: `[N]T` - 固定长度数组
- **映射**: `map[string]T` - 字符串键的映射
- **结构体**: `struct` - 自定义结构体 (通过反射)
- **接口**: `interface{}` - 任意类型

### 指针和特殊类型
- **指针**: `*T` - 指针类型 (自动解引用)
- **空值**: `nil` - 空指针/空接口

## 🚀 快速开始

```go
package main

import (
    "fmt"
    "log"
)

func main() {
    fmt.Println("=== 基本类型示例 ===")
    
    // 基本数据类型
    basicData := map[string]interface{}{
        "integer":       int32(42),
        "float":         float64(3.14159),
        "boolean_true":  true,
        "boolean_false": false,
        "string":        "Hello, 世界!",
        "unicode":       "🌟✨🚀💫",
        "bytes":         []byte("binary data"),
        "null":          nil,
    }
    
    // 序列化
    serialized, err := DumpPoculum(basicData)
    if err != nil {
        log.Fatal("序列化失败:", err)
    }
    
    fmt.Printf("序列化后大小: %d 字节\n", len(serialized))
    fmt.Printf("十六进制: %x\n", serialized)
    
    // 反序列化
    deserialized, err := LoadPoculum(serialized)
    if err != nil {
        log.Fatal("反序列化失败:", err)
    }
    
    fmt.Printf("反序列化成功: %+v\n", deserialized)
}
```
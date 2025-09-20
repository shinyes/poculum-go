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

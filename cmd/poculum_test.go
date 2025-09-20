package main

import (
	"encoding/json"
	poculum "poculum-go/pkg"
	"strings"
	"testing"
)

func BenchmarkPoculumVsJSON(b *testing.B) {
	// 推荐使用 any 切片来作为所有需要序列化的切片和字典的类型，这样就可以避免使用反射导致性能降低，避免使用 []int 这种切片，这样就会使用反射来实现
	numbers := make([]any, 1000)
	for i := 0; i < 1000; i++ {
		numbers[i] = i + 1
	}

	testData := []any{
		42,
		1000,
		100000,
		-50,
		-1000,
		-100000,
		3.14,
		1.23456789,
		numbers,
		nil,
		true,
		false,
		"hello",
		strings.Repeat("a", 1000),
		map[string]any{
			"key": "map",
			"a":   1,
			"b":   2,
			"c":   3,
		},
	}

	b.Run("Poculum", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			poc_bin, err := poculum.DumpPoculum(testData)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = poculum.LoadPoculum(poc_bin) // 忽略结果，只保证不被优化掉
		}
	})

	b.Run("JSON", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data, err := json.Marshal(testData)
			if err != nil {
				b.Fatal(err)
			}
			var decoded []any
			_ = json.Unmarshal(data, &decoded)
		}
	})
}

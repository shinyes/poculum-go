package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func BenchmarkPoculumVsJSON(b *testing.B) {
	numbers := make([]int, 1000)
	for i := 0; i < 1000; i++ {
		numbers[i] = i + 1
	}

	testData := []interface{}{
		42,
		1000,
		100000,
		-50,
		-1000,
		-100000,
		3.14,
		1.23456789,
		numbers,
		"hello",
		strings.Repeat("a", 1000),
		map[string]Value{
			"key": "map",
			"a":   1,
			"b":   2,
			"c":   3,
		},
	}

	b.Run("Poculum", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			poc_bin, err := DumpPoculum(testData)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = LoadPoculum(poc_bin) // 忽略结果，只保证不被优化掉
		}
	})

	b.Run("JSON", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data, err := json.Marshal(testData)
			if err != nil {
				b.Fatal(err)
			}
			var decoded []interface{}
			_ = json.Unmarshal(data, &decoded)
		}
	})
}

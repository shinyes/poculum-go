// Package messagebox 实现高效二进制序列化库 (Go 实现)
//
// 一个轻量级、高性能的 Go 二进制序列化库，
// 支持多种数据类型的紧凑存储和快速解析。
//
// 与 Python、JavaScript、Rust 版本完全兼容
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// 类型标识符常量
const (
	TypeUInt8   = 0x01
	TypeUInt16  = 0x02
	TypeUInt32  = 0x03
	TypeUInt64  = 0x04
	TypeUInt128 = 0x05

	TypeInt8   = 0x11
	TypeInt16  = 0x12
	TypeInt32  = 0x13
	TypeInt64  = 0x14
	TypeInt128 = 0x15

	TypeFloat32 = 0x21
	TypeFloat64 = 0x22

	TypeFixStringBase = 0x30
	TypeString16      = 0x41
	TypeString32      = 0x42

	TypeFixListBase = 0x50
	TypeList16      = 0x61
	TypeList32      = 0x62

	TypeFixMapBase = 0x70
	TypeMap16      = 0x81
	TypeMap32      = 0x82

	TypeBytes8  = 0x91
	TypeBytes16 = 0x92
	TypeBytes32 = 0x93
)

// 安全限制常量
const (
	MaxRecursionDepth = 100
	MaxStringSize     = 100 * 1024 * 1024 // 100MB
	MaxContainerItems = 1000000
)

// MessageBoxError 错误类型
type MessageBoxError struct {
	Type    string
	Message string
}

func (e *MessageBoxError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// 错误构造函数
func newError(errType, message string) *MessageBoxError {
	return &MessageBoxError{Type: errType, Message: message}
}

// Value 表示 MessageBox 支持的所有值类型
type Value interface{}

// MessageBox 编码器/解码器
type MessageBox struct {
	maxRecursionDepth int
	maxStringSize     int
	maxContainerItems int
}

// NewMessageBox 创建新的 MessageBox 实例
func NewMessageBox() *MessageBox {
	return &MessageBox{
		maxRecursionDepth: MaxRecursionDepth,
		maxStringSize:     MaxStringSize,
		maxContainerItems: MaxContainerItems,
	}
}

// WithLimits 创建具有自定义限制的 MessageBox 实例
func WithLimits(maxRecursion, maxStringSize, maxContainerItems int) *MessageBox {
	return &MessageBox{
		maxRecursionDepth: maxRecursion,
		maxStringSize:     maxStringSize,
		maxContainerItems: maxContainerItems,
	}
}

// Dump 序列化值为字节数组
func (mb *MessageBox) Dump(value Value) ([]byte, error) {
	var buf bytes.Buffer
	err := mb.encodeValue(value, &buf, 0)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Load 从字节数组反序列化值
func (mb *MessageBox) Load(data []byte) (Value, error) {
	if len(data) == 0 {
		return nil, nil
	}

	reader := bytes.NewReader(data)
	return mb.decodeValue(reader, 0)
}

// encodeValue 编码值到缓冲区
func (mb *MessageBox) encodeValue(value Value, buf *bytes.Buffer, depth int) error {
	if depth > mb.maxRecursionDepth {
		return newError("MaxRecursionDepth", "Maximum recursion depth exceeded")
	}

	switch v := value.(type) {
	case uint8:
		buf.WriteByte(TypeUInt8)
		buf.WriteByte(v)
	case uint16:
		buf.WriteByte(TypeUInt16)
		binary.Write(buf, binary.BigEndian, v)
	case uint32:
		buf.WriteByte(TypeUInt32)
		binary.Write(buf, binary.BigEndian, v)
	case uint64:
		buf.WriteByte(TypeUInt64)
		binary.Write(buf, binary.BigEndian, v)
	case int8:
		buf.WriteByte(TypeInt8)
		buf.WriteByte(byte(v))
	case int16:
		buf.WriteByte(TypeInt16)
		binary.Write(buf, binary.BigEndian, v)
	case int32:
		buf.WriteByte(TypeInt32)
		binary.Write(buf, binary.BigEndian, v)
	case int64:
		buf.WriteByte(TypeInt64)
		binary.Write(buf, binary.BigEndian, v)
	case int:
		// Go 的 int 类型，转换为适当的整数类型
		if v >= 0 {
			if v <= math.MaxUint32 {
				return mb.encodeValue(uint32(v), buf, depth)
			} else {
				return mb.encodeValue(uint64(v), buf, depth)
			}
		} else {
			if v >= math.MinInt32 {
				return mb.encodeValue(int32(v), buf, depth)
			} else {
				return mb.encodeValue(int64(v), buf, depth)
			}
		}
	case uint:
		// Go 的 uint 类型
		if v <= math.MaxUint32 {
			return mb.encodeValue(uint32(v), buf, depth)
		} else {
			return mb.encodeValue(uint64(v), buf, depth)
		}
	case float32:
		buf.WriteByte(TypeFloat32)
		binary.Write(buf, binary.BigEndian, v)
	case float64:
		buf.WriteByte(TypeFloat64)
		binary.Write(buf, binary.BigEndian, v)
	case string:
		return mb.encodeString(v, buf)
	case []Value:
		return mb.encodeArray(v, buf, depth)
	case []interface{}:
		// 将 []interface{} 转换为 []Value
		values := make([]Value, len(v))
		for i, item := range v {
			values[i] = item
		}
		return mb.encodeArray(values, buf, depth)
	case map[string]Value:
		return mb.encodeObject(v, buf, depth)
	case map[string]interface{}:
		// 将 map[string]interface{} 转换为 map[string]Value
		values := make(map[string]Value)
		for k, v := range v {
			values[k] = v
		}
		return mb.encodeObject(values, buf, depth)
	case []byte:
		return mb.encodeBytes(v, buf)
	case bool:
		// 布尔值转换为整数
		if v {
			return mb.encodeValue(uint8(1), buf, depth)
		} else {
			return mb.encodeValue(uint8(0), buf, depth)
		}
	case nil:
		// 空值不编码任何内容
		return nil
	default:
		// 使用反射处理其他类型
		return mb.encodeWithReflection(value, buf, depth)
	}

	return nil
}

// encodeWithReflection 使用反射编码未知类型
func (mb *MessageBox) encodeWithReflection(value Value, buf *bytes.Buffer, depth int) error {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Bool:
		// 处理布尔类型
		if rv.Bool() {
			return mb.encodeValue(uint8(1), buf, depth)
		} else {
			return mb.encodeValue(uint8(0), buf, depth)
		}
	case reflect.Slice:
		// 处理切片类型
		length := rv.Len()
		values := make([]Value, length)
		for i := 0; i < length; i++ {
			values[i] = rv.Index(i).Interface()
		}
		return mb.encodeArray(values, buf, depth)
	case reflect.Map:
		// 处理映射类型
		if rv.Type().Key().Kind() != reflect.String {
			return newError("UnsupportedType", "Map keys must be strings")
		}
		values := make(map[string]Value)
		for _, key := range rv.MapKeys() {
			keyStr := key.String()
			value := rv.MapIndex(key).Interface()
			values[keyStr] = value
		}
		return mb.encodeObject(values, buf, depth)
	default:
		return newError("UnsupportedType", fmt.Sprintf("Unsupported type: %T", value))
	}
}

// encodeString 编码字符串
func (mb *MessageBox) encodeString(s string, buf *bytes.Buffer) error {
	data := []byte(s)
	length := len(data)

	if length > mb.maxStringSize {
		return newError("DataTooLarge", fmt.Sprintf("String too long: %d bytes (max %d)", length, mb.maxStringSize))
	}

	if !utf8.Valid(data) {
		return newError("Utf8Error", "Invalid UTF-8 string")
	}

	if length <= 15 {
		// fixstring
		buf.WriteByte(TypeFixStringBase + byte(length))
		buf.Write(data)
	} else if length <= 0xFFFF {
		// string16
		buf.WriteByte(TypeString16)
		binary.Write(buf, binary.BigEndian, uint16(length))
		buf.Write(data)
	} else {
		// string32
		buf.WriteByte(TypeString32)
		binary.Write(buf, binary.BigEndian, uint32(length))
		buf.Write(data)
	}

	return nil
}

// encodeArray 编码数组
func (mb *MessageBox) encodeArray(arr []Value, buf *bytes.Buffer, depth int) error {
	length := len(arr)

	if length > mb.maxContainerItems {
		return newError("DataTooLarge", fmt.Sprintf("Array too long: %d items (max %d)", length, mb.maxContainerItems))
	}

	if length <= 15 {
		// fixlist
		buf.WriteByte(TypeFixListBase + byte(length))
	} else if length <= 0xFFFF {
		// list16
		buf.WriteByte(TypeList16)
		binary.Write(buf, binary.BigEndian, uint16(length))
	} else {
		// list32
		buf.WriteByte(TypeList32)
		binary.Write(buf, binary.BigEndian, uint32(length))
	}

	for _, item := range arr {
		err := mb.encodeValue(item, buf, depth+1)
		if err != nil {
			return err
		}
	}

	return nil
}

// encodeObject 编码对象
func (mb *MessageBox) encodeObject(obj map[string]Value, buf *bytes.Buffer, depth int) error {
	length := len(obj)

	if length > mb.maxContainerItems {
		return newError("DataTooLarge", fmt.Sprintf("Object too large: %d items (max %d)", length, mb.maxContainerItems))
	}

	if length <= 15 {
		// fixmap
		buf.WriteByte(TypeFixMapBase + byte(length))
	} else if length <= 0xFFFF {
		// map16
		buf.WriteByte(TypeMap16)
		binary.Write(buf, binary.BigEndian, uint16(length))
	} else {
		// map32
		buf.WriteByte(TypeMap32)
		binary.Write(buf, binary.BigEndian, uint32(length))
	}

	for key, value := range obj {
		err := mb.encodeString(key, buf)
		if err != nil {
			return err
		}
		err = mb.encodeValue(value, buf, depth+1)
		if err != nil {
			return err
		}
	}

	return nil
}

// encodeBytes 编码字节数据
func (mb *MessageBox) encodeBytes(data []byte, buf *bytes.Buffer) error {
	length := len(data)

	if length <= 0xFF {
		// bytes8
		buf.WriteByte(TypeBytes8)
		buf.WriteByte(byte(length))
		buf.Write(data)
	} else if length <= 0xFFFF {
		// bytes16
		buf.WriteByte(TypeBytes16)
		binary.Write(buf, binary.BigEndian, uint16(length))
		buf.Write(data)
	} else {
		// bytes32
		buf.WriteByte(TypeBytes32)
		binary.Write(buf, binary.BigEndian, uint32(length))
		buf.Write(data)
	}

	return nil
}

// decodeValue 从读取器解码值
func (mb *MessageBox) decodeValue(reader *bytes.Reader, depth int) (Value, error) {
	if depth > mb.maxRecursionDepth {
		return nil, newError("MaxRecursionDepth", "Maximum recursion depth exceeded while parsing nested structure")
	}

	typeByte, err := reader.ReadByte()
	if err != nil {
		return nil, newError("InsufficientData", "No type byte")
	}

	switch typeByte {
	case TypeUInt8:
		var value uint8
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "uint8")
		}
		return value, nil
	case TypeUInt16:
		var value uint16
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "uint16")
		}
		return value, nil
	case TypeUInt32:
		var value uint32
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "uint32")
		}
		return value, nil
	case TypeUInt64:
		var value uint64
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "uint64")
		}
		return value, nil
	case TypeInt8:
		var value int8
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "int8")
		}
		return value, nil
	case TypeInt16:
		var value int16
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "int16")
		}
		return value, nil
	case TypeInt32:
		var value int32
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "int32")
		}
		return value, nil
	case TypeInt64:
		var value int64
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "int64")
		}
		return value, nil
	case TypeFloat32:
		var value float32
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "float32")
		}
		return value, nil
	case TypeFloat64:
		var value float64
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "float64")
		}
		return value, nil
	default:
		// 处理字符串类型
		if typeByte >= TypeFixStringBase && typeByte <= TypeFixStringBase+15 {
			length := int(typeByte - TypeFixStringBase)
			return mb.decodeString(reader, length)
		}
		if typeByte == TypeString16 {
			var length uint16
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "string16 length")
			}
			return mb.decodeString(reader, int(length))
		}
		if typeByte == TypeString32 {
			var length uint32
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "string32 length")
			}
			if int(length) > mb.maxStringSize {
				return nil, newError("DataTooLarge", fmt.Sprintf("String32 length too large: %d", length))
			}
			return mb.decodeString(reader, int(length))
		}

		// 处理数组类型
		if typeByte >= TypeFixListBase && typeByte <= TypeFixListBase+15 {
			length := int(typeByte - TypeFixListBase)
			return mb.decodeArray(reader, length, depth)
		}
		if typeByte == TypeList16 {
			var length uint16
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "list16 length")
			}
			return mb.decodeArray(reader, int(length), depth)
		}
		if typeByte == TypeList32 {
			var length uint32
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "list32 length")
			}
			return mb.decodeArray(reader, int(length), depth)
		}

		// 处理对象类型
		if typeByte >= TypeFixMapBase && typeByte <= TypeFixMapBase+15 {
			length := int(typeByte - TypeFixMapBase)
			return mb.decodeObject(reader, length, depth)
		}
		if typeByte == TypeMap16 {
			var length uint16
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "map16 length")
			}
			return mb.decodeObject(reader, int(length), depth)
		}
		if typeByte == TypeMap32 {
			var length uint32
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "map32 length")
			}
			return mb.decodeObject(reader, int(length), depth)
		}

		// 处理字节数据类型
		if typeByte == TypeBytes8 {
			var length uint8
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "bytes8 length")
			}
			return mb.decodeBytes(reader, int(length))
		}
		if typeByte == TypeBytes16 {
			var length uint16
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "bytes16 length")
			}
			return mb.decodeBytes(reader, int(length))
		}
		if typeByte == TypeBytes32 {
			var length uint32
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "bytes32 length")
			}
			return mb.decodeBytes(reader, int(length))
		}

		return nil, newError("UnknownTypeId", fmt.Sprintf("Unknown type identifier: 0x%02x", typeByte))
	}
}

// decodeString 解码字符串
func (mb *MessageBox) decodeString(reader *bytes.Reader, length int) (string, error) {
	if length == 0 {
		return "", nil
	}

	data := make([]byte, length)
	n, err := reader.Read(data)
	if err != nil || n != length {
		return "", newError("InsufficientData", "string data")
	}

	if !utf8.Valid(data) {
		return "", newError("Utf8Error", "Invalid UTF-8 string")
	}

	return string(data), nil
}

// decodeArray 解码数组
func (mb *MessageBox) decodeArray(reader *bytes.Reader, length int, depth int) ([]Value, error) {
	if length > mb.maxContainerItems {
		return nil, newError("DataTooLarge", fmt.Sprintf("Array length too large: %d items (max %d)", length, mb.maxContainerItems))
	}

	arr := make([]Value, length)
	for i := 0; i < length; i++ {
		value, err := mb.decodeValue(reader, depth+1)
		if err != nil {
			return nil, err
		}
		arr[i] = value
	}

	return arr, nil
}

// decodeObject 解码对象
func (mb *MessageBox) decodeObject(reader *bytes.Reader, length int, depth int) (map[string]Value, error) {
	if length > mb.maxContainerItems {
		return nil, newError("DataTooLarge", fmt.Sprintf("Object length too large: %d items (max %d)", length, mb.maxContainerItems))
	}

	obj := make(map[string]Value)
	for i := 0; i < length; i++ {
		// 解码键
		keyValue, err := mb.decodeValue(reader, depth+1)
		if err != nil {
			return nil, err
		}
		key, ok := keyValue.(string)
		if !ok {
			return nil, newError("UnsupportedType", "Object key must be string")
		}

		// 解码值
		value, err := mb.decodeValue(reader, depth+1)
		if err != nil {
			return nil, err
		}
		obj[key] = value
	}

	return obj, nil
}

// decodeBytes 解码字节数据
func (mb *MessageBox) decodeBytes(reader *bytes.Reader, length int) ([]byte, error) {
	data := make([]byte, length)
	n, err := reader.Read(data)
	if err != nil || n != length {
		return nil, newError("InsufficientData", "bytes data")
	}

	return data, nil
}

// 便捷函数
func DumpMessageBox(value Value) ([]byte, error) {
	mb := NewMessageBox()
	return mb.Dump(value)
}

func LoadMessageBox(data []byte) (Value, error) {
	mb := NewMessageBox()
	return mb.Load(data)
}

// 主函数 - 测试和演示
func main() {
	fmt.Println("=== MessageBox Go 测试 ===")

	// 布尔值编码测试
	fmt.Println("\n--- 布尔值编码测试 ---")
	testBoolEncoding()

	// 基本类型测试
	testBasicTypes()

	// 跨平台兼容性测试
	if len(os.Args) > 1 {
		testCrossPlatform(os.Args[1])
	} else {
		testSelfCompatibility()
	}

	// 性能测试
	performanceTest()
}

func testBoolEncoding() {
	// 测试布尔值编码
	trueData, err := DumpMessageBox(true)
	if err != nil {
		fmt.Printf("编码 true 失败: %v\n", err)
		return
	}

	falseData, err := DumpMessageBox(false)
	if err != nil {
		fmt.Printf("编码 false 失败: %v\n", err)
		return
	}

	fmt.Printf("Go 编码 true: %x\n", trueData)
	fmt.Printf("Go 编码 false: %x\n", falseData)
}

func testBasicTypes() {
	fmt.Println("\n--- 基本类型测试 ---")

	testCases := []struct {
		name  string
		value Value
	}{
		{"整数", uint32(42)},
		{"负整数", int32(-123)},
		{"浮点数", 3.14159},
		{"字符串", "Hello, 世界! 🌍"},
		{"空字符串", ""},
		{"数组", []Value{int32(1), int32(2), int32(3), "四", 5.5}},
		{"空数组", []Value{}},
		{"字节数据", []byte{72, 101, 108, 108, 111}}, // "Hello"
	}

	mb := NewMessageBox()

	for _, tc := range testCases {
		serialized, err := mb.Dump(tc.value)
		if err != nil {
			fmt.Printf("❌ %s: 序列化失败 - %v\n", tc.name, err)
			continue
		}

		deserialized, err := mb.Load(serialized)
		if err != nil {
			fmt.Printf("❌ %s: 反序列化失败 - %v\n", tc.name, err)
			continue
		}

		if deepEqual(tc.value, deserialized) {
			fmt.Printf("✅ %s: 通过 (%d 字节)\n", tc.name, len(serialized))
		} else {
			fmt.Printf("❌ %s: 数据不匹配\n", tc.name)
		}
	}

	// 测试对象
	obj := map[string]Value{
		"name":   "Alice",
		"age":    uint32(30),
		"active": uint8(1), // 布尔值作为整数
	}

	serialized, err := mb.Dump(obj)
	if err != nil {
		fmt.Printf("❌ 对象: 序列化失败 - %v\n", err)
	} else {
		deserialized, err := mb.Load(serialized)
		if err != nil {
			fmt.Printf("❌ 对象: 反序列化失败 - %v\n", err)
		} else if deepEqual(obj, deserialized) {
			fmt.Printf("✅ 对象: 通过 (%d 字节)\n", len(serialized))
		} else {
			fmt.Printf("❌ 对象: 数据不匹配\n")
		}
	}
}

func testCrossPlatform(hexData string) {
	fmt.Println("\n--- 跨平台兼容性测试 ---")

	// 解析十六进制数据
	data, err := parseHexString(hexData)
	if err != nil {
		fmt.Printf("❌ 十六进制解析失败: %v\n", err)
		return
	}

	mb := NewMessageBox()
	value, err := mb.Load(data)
	if err != nil {
		fmt.Printf("❌ 反序列化失败: %v\n", err)
		return
	}

	fmt.Println("✅ 成功反序列化其他语言的数据:")
	printValue(value, 0)

	// 尝试重新序列化
	reSerialized, err := mb.Dump(value)
	if err != nil {
		fmt.Printf("❌ 重新序列化失败: %v\n", err)
	} else {
		reHex := bytesToHex(reSerialized)
		fmt.Printf("GO_SERIALIZED:%s\n", reHex)
	}
}

func testSelfCompatibility() {
	fmt.Println("\n--- 自兼容性测试 ---")

	// 创建复杂测试数据
	testData := map[string]Value{
		"users": []Value{
			map[string]Value{
				"id":   uint32(1),
				"name": "Alice",
			},
			map[string]Value{
				"id":   uint32(2),
				"name": "Bob",
			},
		},
		"metadata": map[string]Value{
			"version": "1.0",
			"stats":   []Value{uint32(10), uint32(20), uint32(30)},
		},
	}

	mb := NewMessageBox()
	serialized, err := mb.Dump(testData)
	if err != nil {
		fmt.Printf("❌ 序列化失败: %v\n", err)
		return
	}

	hex := bytesToHex(serialized)
	fmt.Printf("序列化数据 (%d 字节): %s\n", len(serialized), hex[:min(32, len(hex))])

	deserialized, err := mb.Load(serialized)
	if err != nil {
		fmt.Printf("❌ 反序列化失败: %v\n", err)
		return
	}

	if deepEqual(testData, deserialized) {
		fmt.Println("✅ 复杂数据结构序列化/反序列化成功")
	} else {
		fmt.Println("❌ 复杂数据结构验证失败")
	}
}

func performanceTest() {
	fmt.Println("\n--- 性能测试 ---")

	// 创建测试数据
	testData := createPerformanceTestData()
	mb := NewMessageBox()

	iterations := 1000

	// 序列化性能测试
	start := time.Now()
	var serialized []byte
	for i := 0; i < iterations; i++ {
		serialized, _ = mb.Dump(testData)
	}
	serializeTime := time.Since(start)

	// 反序列化性能测试
	start = time.Now()
	for i := 0; i < iterations; i++ {
		mb.Load(serialized)
	}
	deserializeTime := time.Since(start)

	fmt.Printf("序列化 %d 次: %.2fms (平均 %.3fms)\n",
		iterations,
		float64(serializeTime.Nanoseconds())/1e6,
		float64(serializeTime.Nanoseconds())/1e6/float64(iterations))
	fmt.Printf("反序列化 %d 次: %.2fms (平均 %.3fms)\n",
		iterations,
		float64(deserializeTime.Nanoseconds())/1e6,
		float64(deserializeTime.Nanoseconds())/1e6/float64(iterations))
	fmt.Printf("序列化后大小: %d 字节\n", len(serialized))
}

func createPerformanceTestData() map[string]Value {
	// 创建数字数组
	numbers := make([]Value, 1000)
	for i := 0; i < 1000; i++ {
		numbers[i] = uint32(i)
	}

	// 创建字符串数组
	strings := make([]Value, 100)
	for i := 0; i < 100; i++ {
		strings[i] = fmt.Sprintf("test_string_%d", i)
	}

	// 创建嵌套对象
	nested := map[string]Value{
		"level1": map[string]Value{
			"level2": map[string]Value{
				"level3": map[string]Value{
					"deep": "value",
				},
			},
		},
	}

	return map[string]Value{
		"numbers": numbers,
		"strings": strings,
		"nested":  nested,
	}
}

// 辅助函数
func deepEqual(a, b Value) bool {
	// 简化的深度比较，实际项目中可能需要更复杂的实现
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func parseHexString(hexStr string) ([]byte, error) {
	if len(hexStr)%2 != 0 {
		return nil, errors.New("hex string length must be even")
	}

	data := make([]byte, len(hexStr)/2)
	for i := 0; i < len(hexStr); i += 2 {
		b, err := strconv.ParseUint(hexStr[i:i+2], 16, 8)
		if err != nil {
			return nil, err
		}
		data[i/2] = byte(b)
	}

	return data, nil
}

func bytesToHex(data []byte) string {
	var sb strings.Builder
	for _, b := range data {
		sb.WriteString(fmt.Sprintf("%02x", b))
	}
	return sb.String()
}

func printValue(value Value, indent int) {
	prefix := strings.Repeat("  ", indent)
	switch v := value.(type) {
	case uint8:
		fmt.Printf("%sUInt8(%d)\n", prefix, v)
	case uint16:
		fmt.Printf("%sUInt16(%d)\n", prefix, v)
	case uint32:
		fmt.Printf("%sUInt32(%d)\n", prefix, v)
	case uint64:
		fmt.Printf("%sUInt64(%d)\n", prefix, v)
	case int8:
		fmt.Printf("%sInt8(%d)\n", prefix, v)
	case int16:
		fmt.Printf("%sInt16(%d)\n", prefix, v)
	case int32:
		fmt.Printf("%sInt32(%d)\n", prefix, v)
	case int64:
		fmt.Printf("%sInt64(%d)\n", prefix, v)
	case float32:
		fmt.Printf("%sFloat32(%g)\n", prefix, v)
	case float64:
		fmt.Printf("%sFloat64(%g)\n", prefix, v)
	case string:
		fmt.Printf("%sString(\"%s\")\n", prefix, v)
	case []Value:
		fmt.Printf("%sArray[%d]:\n", prefix, len(v))
		for i, item := range v {
			if i >= 3 {
				fmt.Printf("%s  ... (%d more items)\n", prefix, len(v)-3)
				break
			}
			printValue(item, indent+1)
		}
	case map[string]Value:
		fmt.Printf("%sObject{%d}:\n", prefix, len(v))
		count := 0
		for key, value := range v {
			if count >= 3 {
				fmt.Printf("%s  ... (%d more items)\n", prefix, len(v)-3)
				break
			}
			fmt.Printf("%s  \"%s\":\n", prefix, key)
			printValue(value, indent+2)
			count++
		}
	case []byte:
		fmt.Printf("%sBytes[%d]: %v\n", prefix, len(v), v[:min(10, len(v))])
	default:
		fmt.Printf("%sUnknown(%T): %v\n", prefix, v, v)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

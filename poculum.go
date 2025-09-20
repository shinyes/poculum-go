package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"reflect"
	"unicode/utf8"
)

// 类型标识符常量，长度都是一个字节

/*
	关于名称中带有 Fix 的类型

如果名称中带有 fix 则代表其中的元素个数<=15，专门设立这个类型是为了当元素个数比较小时，还要用去存储长度字段，
这种类型的长度字段是存储在字节的低位的，所以名称中才会带有Base，
例如如果某个值的类型字节为 0x3F，则代表这是一个字符串，则这个字符串占用的字节数为 15，
对于 fix 的 List 和 Map，类型字节的低位代表的是其中的元素个数
*/
const (
	TypeUInt8  = 0x01
	TypeUInt16 = 0x02
	TypeUInt32 = 0x03
	TypeUInt64 = 0x04
	// TypeUInt128 = 0x05 // 暂时不使用

	TypeInt8  = 0x11
	TypeInt16 = 0x12
	TypeInt32 = 0x13
	TypeInt64 = 0x14
	// TypeInt128 = 0x15 // 暂时不使用

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

	TypeTrue  = 0xA0
	TypeFalse = 0xA1
	// TypeUnkown = 0xA2 // 暂不使用
	TypeNil = 0xA3
)

// 安全限制常量
const (
	MaxRecursionDepth = 100               // list、map的最大嵌套深度
	MaxStringSize     = 100 * 1024 * 1024 // 100MB
	MaxContainerItems = 1000000           // list、map中的最多元素数量
)

// PoculumError 错误类型
type PoculumError struct {
	Type    string
	Message string
}

func (e *PoculumError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// 错误构造函数
func newError(errType, message string) *PoculumError {
	return &PoculumError{Type: errType, Message: message}
}

// Poculum 编码器/解码器
type Poculum struct {
	maxRecursionDepth int
	maxStringSize     int
	maxContainerItems int
}

// NewPoculum 创建新的 Poculum 实例
func NewPoculum() *Poculum {
	return &Poculum{
		maxRecursionDepth: MaxRecursionDepth,
		maxStringSize:     MaxStringSize,
		maxContainerItems: MaxContainerItems,
	}
}

// WithLimits 创建具有自定义限制的 Poculum 实例
func WithLimits(maxRecursion, maxStringSize, maxContainerItems int) *Poculum {
	return &Poculum{
		maxRecursionDepth: maxRecursion,
		maxStringSize:     maxStringSize,
		maxContainerItems: maxContainerItems,
	}
}

// 序列化值为字节数组
func (poc *Poculum) dump(value any) ([]byte, error) {
	var buf bytes.Buffer
	err := poc.encodeValue(value, &buf, 0)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// 从字节数组反序列化值
func (poc *Poculum) load(data []byte) (any, error) {
	if len(data) == 0 {
		return nil, nil
	}

	reader := bytes.NewReader(data)
	return poc.decodeValue(reader, 0)
}

// 编码值到缓冲区
func (poc *Poculum) encodeValue(value any, buf *bytes.Buffer, depth int) error {
	if depth > poc.maxRecursionDepth {
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
				return poc.encodeValue(uint32(v), buf, depth)
			} else {
				return poc.encodeValue(uint64(v), buf, depth)
			}
		} else {
			if v >= math.MinInt32 {
				return poc.encodeValue(int32(v), buf, depth)
			} else {
				return poc.encodeValue(int64(v), buf, depth)
			}
		}
	case uint:
		// Go 的 uint 类型
		if v <= math.MaxUint32 {
			return poc.encodeValue(uint32(v), buf, depth)
		} else {
			return poc.encodeValue(uint64(v), buf, depth)
		}
	case float32:
		buf.WriteByte(TypeFloat32)
		binary.Write(buf, binary.BigEndian, v)
	case float64:
		buf.WriteByte(TypeFloat64)
		binary.Write(buf, binary.BigEndian, v)
	case string:
		return poc.encodeString(v, buf)
	case []any: // 这里对应的是序列化数组的部分
		return poc.encodeArray(v, buf, depth)
	case map[string]any:
		return poc.encodeMap(v, buf, depth)
	case []byte:
		return poc.encodeBytes(v, buf)
	case bool:
		// 布尔值
		if v {
			buf.WriteByte(TypeTrue)
		} else {
			buf.WriteByte(TypeFalse)
		}
	case nil:
		return buf.WriteByte(TypeNil)
	default:
		// 使用反射处理其他类型
		return poc.encodeWithReflection(value, buf, depth)
	}

	return nil
}

// encodeWithReflection 使用反射编码未知类型
func (poc *Poculum) encodeWithReflection(value any, buf *bytes.Buffer, depth int) error {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Bool:
		// 处理布尔类型
		if rv.Bool() {
			return poc.encodeValue(uint8(1), buf, depth)
		} else {
			return poc.encodeValue(uint8(0), buf, depth)
		}
	case reflect.Slice:
		// 处理切片类型
		length := rv.Len()
		values := make([]any, length)
		for i := 0; i < length; i++ {
			values[i] = rv.Index(i).Interface()
		}
		return poc.encodeArray(values, buf, depth)
	case reflect.Map:
		// 处理映射类型
		if rv.Type().Key().Kind() != reflect.String {
			return newError("UnsupportedType", "Map keys must be strings")
		}
		values := make(map[string]any)
		for _, key := range rv.MapKeys() {
			keyStr := key.String()
			value := rv.MapIndex(key).Interface()
			values[keyStr] = value
		}
		return poc.encodeMap(values, buf, depth)
	default:
		return newError("UnsupportedType", fmt.Sprintf("Unsupported type: %T", value))
	}
}

// encodeString 编码字符串
func (poc *Poculum) encodeString(s string, buf *bytes.Buffer) error {
	data := []byte(s)
	length := len(data)

	if length > poc.maxStringSize {
		return newError("DataTooLarge", fmt.Sprintf("String too long: %d bytes (max %d)", length, poc.maxStringSize))
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
func (poc *Poculum) encodeArray(arr []any, buf *bytes.Buffer, depth int) error {
	length := len(arr)

	if length > poc.maxContainerItems {
		return newError("DataTooLarge", fmt.Sprintf("Array too long: %d items (max %d)", length, poc.maxContainerItems))
	}

	// 先把类型字节与长度写入到字节缓冲区
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

	// 再逐个序列化数组中的项
	for _, item := range arr {
		err := poc.encodeValue(item, buf, depth+1)
		if err != nil {
			return err
		}
	}

	return nil
}

// encodeMap 编码对象
func (poc *Poculum) encodeMap(obj map[string]any, buf *bytes.Buffer, depth int) error {
	length := len(obj)

	if length > poc.maxContainerItems {
		return newError("DataTooLarge", fmt.Sprintf("Object too large: %d items (max %d)", length, poc.maxContainerItems))
	}

	// 先把类型字节写入到字节缓冲区
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
	// 再逐个序列化键与值
	for key, value := range obj {
		err := poc.encodeString(key, buf)
		if err != nil {
			return err
		}
		err = poc.encodeValue(value, buf, depth+1)
		if err != nil {
			return err
		}
	}

	return nil
}

// encodeBytes 编码字节数据
func (poc *Poculum) encodeBytes(data []byte, buf *bytes.Buffer) error {
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
func (poc *Poculum) decodeValue(reader *bytes.Reader, depth int) (any, error) {
	if depth > poc.maxRecursionDepth {
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
	case TypeTrue:
		return true, nil
	case TypeFalse:
		return false, nil
	case TypeNil:
		return nil, nil
	default:
		// 处理字符串类型
		if typeByte >= TypeFixStringBase && typeByte <= TypeFixStringBase+15 {
			length := int(typeByte - TypeFixStringBase)
			return poc.decodeString(reader, length)
		}
		if typeByte == TypeString16 {
			var length uint16
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "string16 length")
			}
			return poc.decodeString(reader, int(length))
		}
		if typeByte == TypeString32 {
			var length uint32
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "string32 length")
			}
			if int(length) > poc.maxStringSize {
				return nil, newError("DataTooLarge", fmt.Sprintf("String32 length too large: %d", length))
			}
			return poc.decodeString(reader, int(length))
		}

		// 处理数组类型
		if typeByte >= TypeFixListBase && typeByte <= TypeFixListBase+15 {
			length := int(typeByte - TypeFixListBase)
			return poc.decodeArray(reader, length, depth)
		}
		if typeByte == TypeList16 {
			var length uint16
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "list16 length")
			}
			return poc.decodeArray(reader, int(length), depth)
		}
		if typeByte == TypeList32 {
			var length uint32
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "list32 length")
			}
			return poc.decodeArray(reader, int(length), depth)
		}

		// 处理对象类型
		if typeByte >= TypeFixMapBase && typeByte <= TypeFixMapBase+15 {
			length := int(typeByte - TypeFixMapBase)
			return poc.decodeMap(reader, length, depth)
		}
		if typeByte == TypeMap16 {
			var length uint16
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "map16 length")
			}
			return poc.decodeMap(reader, int(length), depth)
		}
		if typeByte == TypeMap32 {
			var length uint32
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "map32 length")
			}
			return poc.decodeMap(reader, int(length), depth)
		}

		// 处理字节数据类型
		if typeByte == TypeBytes8 {
			var length uint8
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "bytes8 length")
			}
			return poc.decodeBytes(reader, int(length))
		}
		if typeByte == TypeBytes16 {
			var length uint16
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "bytes16 length")
			}
			return poc.decodeBytes(reader, int(length))
		}
		if typeByte == TypeBytes32 {
			var length uint32
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "bytes32 length")
			}
			return poc.decodeBytes(reader, int(length))
		}

		return nil, newError("UnknownTypeId", fmt.Sprintf("Unknown type identifier: 0x%02x", typeByte))
	}
}

// decodeString 解码字符串
func (poc *Poculum) decodeString(reader *bytes.Reader, length int) (string, error) {
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
func (poc *Poculum) decodeArray(reader *bytes.Reader, length int, depth int) ([]any, error) {
	if length > poc.maxContainerItems {
		return nil, newError("DataTooLarge", fmt.Sprintf("Array length too large: %d items (max %d)", length, poc.maxContainerItems))
	}

	arr := make([]any, length)
	for i := 0; i < length; i++ {
		value, err := poc.decodeValue(reader, depth+1)
		if err != nil {
			return nil, err
		}
		arr[i] = value
	}

	return arr, nil
}

// decodeMap 解码对象
func (poc *Poculum) decodeMap(reader *bytes.Reader, length int, depth int) (map[string]any, error) {
	if length > poc.maxContainerItems {
		return nil, newError("DataTooLarge", fmt.Sprintf("Object length too large: %d items (max %d)", length, poc.maxContainerItems))
	}

	obj := make(map[string]any)
	for i := 0; i < length; i++ {
		// 解码键
		keyValue, err := poc.decodeValue(reader, depth+1)
		if err != nil {
			return nil, err
		}
		key, ok := keyValue.(string)
		if !ok {
			return nil, newError("UnsupportedType", "Object key must be string")
		}

		// 解码值
		value, err := poc.decodeValue(reader, depth+1)
		if err != nil {
			return nil, err
		}
		obj[key] = value
	}

	return obj, nil
}

// decodeBytes 解码字节数据
func (poc *Poculum) decodeBytes(reader *bytes.Reader, length int) ([]byte, error) {
	data := make([]byte, length)
	n, err := reader.Read(data)
	if err != nil || n != length {
		return nil, newError("InsufficientData", "bytes data")
	}

	return data, nil
}

// 便捷函数
func DumpPoculum(value any) ([]byte, error) {
	poc := NewPoculum()
	return poc.dump(value)
}
func LoadPoculum(data []byte) (any, error) {
	mb := NewPoculum()
	return mb.load(data)
}

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

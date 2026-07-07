package list

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"unicode/utf16"
	"unsafe"

	"github.com/caretdev/go-irisnative/src/iris"
)

type ListItemType byte

const (
	LISTITEM_STRING       ListItemType = 0x01
	LISTITEM_UNICODE      ListItemType = 0x02
	LISTITEM_POSINT       ListItemType = 0x04
	LISTITEM_NEGINT       ListItemType = 0x05
	LISTITEM_POSDECIMAL   ListItemType = 0x06
	LISTITEM_NEGDECIMAL   ListItemType = 0x07
	LISTITEM_COMPACTFLOAT ListItemType = 0x08
	LISTITEM_IEEEDOUBLE   ListItemType = 0x09
	LISTITEM_OREF         ListItemType = 0x19
)

type ListItem struct {
	size     uint16
	itemType ListItemType
	data     []byte
	isNull   bool
	byRef    bool
}

func (li *ListItem) IsNull() bool {
	return li.isNull
}

func (li *ListItem) IsString() bool {
	return li.itemType == LISTITEM_STRING || li.itemType == LISTITEM_UNICODE
}

func (li *ListItem) IsEmpty() bool {
	return li.itemType == LISTITEM_STRING && len(li.data) == 0
}

func (li *ListItem) Type() ListItemType {
	return li.itemType
}

var scale = []float64{
	1.0, 10.0, 100.0, 1000.0, 10000.0, 100000.0, 1000000.0, 1.0e7, 1.0e8, 1.0e9,
	1.0e10, 1.0e11, 1.0e12, 1.0e13, 1.0e14, 1.0e15, 1.0e16, 1.0e17, 1.0e18, 1.0e19,
	1.0e20, 1.0e21, 1.0e22, 9.999999999999999e22, 1.0e24, 1.0e25, 1.0e26, 1.0e27, 1.0e28, 1.0e29,
	1.0e30, 1.0e31, 1.0e32, 1.0e33, 1.0e34, 1.0e35, 1.0e36, 1.0e37, 1.0e38, 1.0e39,
	1.0e40, 1.0e41, 1.0e42, 1.0e43, 1.0e44, 1.0e45, 1.0e46, 1.0e47, 1.0e48, 1.0e49,
	1.0e50, 1.0e51, 1.0e52, 1.0e53, 1.0e54, 1.0e55, 1.0e56, 1.0e57, 1.0e58, 1.0e59,
	1.0e60, 1.0e61, 1.0e62, 1.0e63, 1.0e64, 1.0e65, 1.0e66, 1.0e67, 1.0e68, 1.0e69,
	1.0e70, 1.0e71, 1.0e72, 1.0e73, 1.0e74, 1.0e75, 1.0e76, 1.0e77, 1.0e78, 1.0e79,
	1.0e80, 1.0e81, 1.0e82, 1.0e83, 1.0e84, 1.0e85, 1.0e86, 1.0e87, 1.0e88, 1.0e89,
	1.0e90, 1.0e91, 1.0e92, 1.0e93, 1.0e94, 1.0e95, 1.0e96, 1.0e97, 1.0e98, 1.0e99,
	1.0e100, 1.0e101, 1.0e102, 1.0e103, 1.0e104, 1.0e105, 1.0e106, 1.0e107, 1.0e108, 1.0e109,
	1.0e110, 1.0e111, 1.0e112, 1.0e113, 1.0e114, 1.0e115, 1.0e116, 1.0e117, 1.0e118, 1.0e119,
	1.0e120, 1.0e121, 1.0e122, 1.0e123, 1.0e124, 1.0e125, 1.0e126, 1.0e127, 1.0e-128, 1.0e-127,
	1.0e-126, 1.0e-125, 1.0e-124, 1.0e-123, 1.0e-122, 1.0e-121, 1.0e-120, 1.0e-119, 1.0e-118, 1.0e-117,
	1.0e-116, 1.0e-115, 1.0e-114, 1.0e-113, 1.0e-112, 1.0e-111, 1.0e-110, 1.0e-109, 1.0e-108, 1.0e-107,
	1.0e-106, 1.0e-105, 1.0e-104, 1.0e-103, 1.0e-102, 1.0e-101, 1.0e-100, 1.0e-99, 1.0e-98, 1.0e-97,
	1.0e-96, 1.0e-95, 1.0e-94, 1.0e-93, 1.0e-92, 1.0e-91, 1.0e-90, 1.0e-89, 1.0e-88, 1.0e-87,
	1.0e-86, 1.0e-85, 1.0e-84, 1.0e-83, 1.0e-82, 1.0e-81, 1.0e-80, 1.0e-79, 1.0e-78, 1.0e-77,
	1.0e-76, 1.0e-75, 1.0e-74, 1.0e-73, 1.0e-72, 1.0e-71, 1.0e-70, 1.0e-69, 1.0e-68, 1.0e-67,
	1.0e-66, 1.0e-65, 1.0e-64, 1.0e-63, 1.0e-62, 1.0e-61, 1.0e-60, 1.0e-59, 1.0e-58, 1.0e-57,
	1.0e-56, 1.0e-55, 1.0e-54, 1.0e-53, 1.0e-52, 1.0e-51, 1.0e-50, 1.0e-49, 1.0e-48, 1.0e-47,
	1.0e-46, 1.0e-45, 1.0e-44, 1.0e-43, 1.0e-42, 1.0e-41, 1.0e-40, 1.0e-39, 1.0e-38, 1.0e-37,
	1.0e-36, 1.0e-35, 1.0e-34, 1.0e-33, 1.0e-32, 1.0e-31, 1.0e-30, 1.0e-29, 1.0e-28, 1.0e-27,
	1.0e-26, 1.0e-25, 1.0e-24, 1.0e-23, 1.0e-22, 1.0e-21, 1.0e-20, 1.0e-19, 1.0e-18, 1.0e-17,
	1.0e-16, 1.0e-15, 1.0e-14, 1.0e-13, 1.0e-12, 1.0e-11, 1.0e-10, 1.0e-9, 1.0e-8, 1.0e-7,
	1.0e-6, 1.0e-5, 1.0e-4, 0.001, 0.01, 0.1}

func (listItem *ListItem) Dump() []byte {
	if listItem.isNull {
		return []byte{1}
	}
	var dump = make([]byte, 0)
	if listItem.size > 253 {
		size := listItem.size + 1
		dump = append(dump, 0)
		dump = append(dump, byte((size)&0xff))
		dump = append(dump, byte((size>>8)&0xff))
	} else {
		dump = append(dump, byte(listItem.size+2))
	}
	dump = append(dump, byte(listItem.itemType))
	dump = append(dump, listItem.data...)
	return dump
}

func GetListItem(buffer []byte, ooffset *uint) ListItem {
	var byRef = false
	var isNull = false
	var size uint16 = 0
	var itemType byte = 0
	offset := *ooffset

	if offset >= uint(len(buffer)) {
		return ListItem{size, ListItemType(itemType), []byte{}, true, false}
	}

	switch buffer[offset] {
	case 0:
		if offset+2 >= uint(len(buffer)) {
			return ListItem{size, ListItemType(itemType), []byte{}, true, false}
		}
		size = uint16((buffer[offset+1] & 0xff))
		size |= ((uint16(buffer[offset+2]) & 0xff) << 8)
		size -= 1
		offset += 3
		if offset >= uint(len(buffer)) {
			return ListItem{size, ListItemType(itemType), []byte{}, true, false}
		}
		itemType = buffer[offset]
		offset += 1
	case 1:
		isNull = true
		offset += 1
	default:
		size = uint16(buffer[offset]) - 2
		offset += 1
		if offset >= uint(len(buffer)) {
			return ListItem{size, ListItemType(itemType), []byte{}, true, false}
		}
		itemType = buffer[offset]
		offset += 1
		if itemType >= 32 && itemType < 64 {
			itemType = itemType - 32
			byRef = true
		}
	}
	var data = []byte{}
	if size > 0 {
		if offset+uint(size) > uint(len(buffer)) {
			return ListItem{size, ListItemType(itemType), []byte{}, true, false}
		}
		data = buffer[offset : offset+uint(size)]
	}
	offset += uint(size)
	*ooffset = offset
	return ListItem{size, ListItemType(itemType), data, isNull, byRef}
}

func NewListItem(value interface{}) ListItem {
	var itemType ListItemType = 0
	var size uint16 = 0
	var data = make([]byte, 0)
	var isNull = false
	var byRef = false

	switch v := value.(type) {
	case *string:
		var listItem = NewListItem(*v)
		listItem.byRef = true
		return listItem
	case int, int8, int16, int32, int64:
		var ival int64
		switch i := v.(type) {
		case int:
			ival = int64(i)
		case int8:
			ival = int64(i)
		case int16:
			ival = int64(i)
		case int32:
			ival = int64(i)
		case int64:
			ival = i
		}
		itemType = 4
		var base = 0
		var temp = ival
		if ival < 0 {
			itemType = 5
			base = 0xff
			temp = ival*-1 - 1
		}
		for temp > 0 {
			data = append(data, byte((temp^int64(base))&0xff))
			temp = temp >> 8
		}
	case uint, uint8, uint16, uint32, uint64:
		var uval uint64
		switch u := v.(type) {
		case uint:
			uval = uint64(u)
		case uint8:
			uval = uint64(u)
		case uint16:
			uval = uint64(u)
		case uint32:
			uval = uint64(u)
		case uint64:
			uval = u
		}
		itemType = 4
		temp := uval
		for temp > 0 {
			data = append(data, byte(temp&0xff))
			temp = temp >> 8
		}
	case float64, float32:
		// Use binary float encoding (types 08/09) instead of decimal
		switch f := v.(type) {
		case float32:
			itemType = LISTITEM_COMPACTFLOAT
			// IEEE 754 single-precision, little-endian
			bits := binary.LittleEndian.Uint32((*[4]byte)(unsafe.Pointer(&f))[:])
			data = make([]byte, 4)
			binary.LittleEndian.PutUint32(data, bits)
			// Remove trailing zero bytes for compact encoding
			for len(data) > 0 && data[len(data)-1] == 0 {
				data = data[:len(data)-1]
			}
		case float64:
			// Check for special IEEE values
			if math.IsNaN(f) {
				// NaN uses type 09 with specific encoding
				itemType = LISTITEM_IEEEDOUBLE
				data = make([]byte, 2)
				if math.Signbit(f) {
					// Negative NaN: 04 09 F8 FF
					data[0] = 0xF8
					data[1] = 0xFF
				} else {
					// Positive NaN: 04 09 F8 7F
					data[0] = 0xF8
					data[1] = 0x7F
				}
			} else if math.IsInf(f, 1) {
				// Positive Infinity: 04 08 80 7F
				itemType = LISTITEM_COMPACTFLOAT
				data = []byte{0x80, 0x7F}
			} else if math.IsInf(f, -1) {
				// Negative Infinity: 04 08 80 FF
				itemType = LISTITEM_COMPACTFLOAT
				data = []byte{0x80, 0xFF}
			} else {
				itemType = LISTITEM_IEEEDOUBLE
				// IEEE 754 double-precision, fixed 8-byte payload, little-endian
				bits := binary.LittleEndian.Uint64((*[8]byte)(unsafe.Pointer(&f))[:])
				data = make([]byte, 8)
				binary.LittleEndian.PutUint64(data, bits)
			}
		}
	case bool:
		itemType = 4
		if v {
			data = []byte{0x1}
		} else {
			data = []byte{0x0}
		}
	case string:
		itemType = 1
		for _, r := range v {
			if r > 255 {
				itemType = 2
				break
			}
		}
		if itemType == 2 {
			// Encode as UTF-16LE
			runes := []rune(v)
			utf16Runes := utf16.Encode(runes)
			data = make([]byte, len(utf16Runes)*2)
			for i, r := range utf16Runes {
				data[i*2] = byte(r & 0xff)
				data[i*2+1] = byte((r >> 8) & 0xff)
			}
		} else {
			data = []byte(v)
		}
	case []byte:
		itemType = 1
		data = v
	case nil:
		isNull = true
		// itemType = 1
		// data = []byte("")
	case iris.Oref:
		itemType = 25
		byRef = true
		data = []byte(v)
	default:
		fmt.Printf("unknown: %#v %T\n", v, v)
		itemType = 1
		data = []byte(fmt.Sprintf("%v", v))
	}
	size = uint16(len(data))
	return ListItem{
		size,
		itemType,
		data,
		isNull,
		byRef,
	}
}

func (li *ListItem) getString() string {
	if li.itemType == LISTITEM_UNICODE {
		// Decode UTF-16LE
		if len(li.data)%2 != 0 {
			return string(li.data) // Fallback for invalid data
		}
		utf16Runes := make([]uint16, len(li.data)/2)
		for i := 0; i < len(utf16Runes); i++ {
			utf16Runes[i] = binary.LittleEndian.Uint16(li.data[i*2 : i*2+2])
		}
		runes := utf16.Decode(utf16Runes)
		return string(runes)
	} else {
		return string(li.data)
	}
}

func getPosInt(data []byte) int {
	temp := make([]byte, 8)
	copy(temp, data)
	return int(binary.LittleEndian.Uint64(temp[:8]))
}

func getNegInt(data []byte) int {
	temp := make([]byte, 8)
	copy(temp, data)
	for i := range data {
		temp[i] ^= 0xff
	}
	return -int(binary.LittleEndian.Uint64(temp[:8]) + 1)
}

func getPosFloat(data []byte) float64 {
	scaleIndex := int(data[0])
	if scaleIndex < 0 || scaleIndex >= len(scale) {
		scaleIndex = 128 // Default to middle of array if out of bounds
	}
	d := scale[scaleIndex]
	return float64(getPosInt(data[1:])) * d
}

func getNegFloat(data []byte) float64 {
	scaleIndex := int(data[0])
	if scaleIndex < 0 || scaleIndex >= len(scale) {
		scaleIndex = 128 // Default to middle of array if out of bounds
	}
	d := scale[scaleIndex]
	return float64(getNegInt(data[1:])) * d
}

func (li *ListItem) asString() (value string, err error) {
	if li.isNull {
		value = ""
		return
	}
	switch li.itemType {
	case 1, 2, 25:
		value = li.getString()
	case 4:
		value = fmt.Sprint(getPosInt(li.data))
	case 5:
		value = fmt.Sprint(getNegInt(li.data))
	case 6:
		value = fmt.Sprint(getPosFloat(li.data))
	case 7:
		value = fmt.Sprint(getNegFloat(li.data))
	case 8, 9:
		// Binary float types - convert to string via float64
		f, err := li.asFloat64()
		if err != nil {
			return "", err
		}
		value = fmt.Sprint(f)
	default:
		err = errors.New("not implemented")
	}
	return
}

func (li *ListItem) asInt() (value int, err error) {
	if li.isNull {
		value = 0
		return
	}
	switch li.itemType {
	case 1, 2:
		value, err = strconv.Atoi(li.getString())
	case 4:
		value = getPosInt(li.data)
	case 5:
		value = getNegInt(li.data)
	case 6:
		value = int(getPosFloat(li.data))
	case 7:
		value = int(getNegFloat(li.data))
	default:
		err = errors.New("not implemented")
	}
	return
}

func (li *ListItem) asFloat64() (value float64, err error) {
	if li.isNull {
		value = 0
		return
	}
	switch li.itemType {
	case 1, 2:
		value, err = strconv.ParseFloat(li.getString(), 64)
		if err != nil {
			return
		}
	case 4:
		value = float64(getPosInt(li.data))
	case 5:
		value = float64(getNegInt(li.data))
	case 6:
		value = getPosFloat(li.data)
	case 7:
		value = getNegFloat(li.data)
	case 8:
		if len(li.data) == 2 && li.data[0] == 0x80 {
			if li.data[1] == 0x7F {
				value = math.Inf(1) // Positive Infinity
			} else if li.data[1] == 0xFF {
				value = math.Inf(-1) // Negative Infinity
			} else {
				bits := uint32(binary.LittleEndian.Uint16(li.data)) << 16
				value = float64(math.Float32frombits(bits))
			}
		} else {
			switch len(li.data) {
			case 8:
				value = math.Float64frombits(binary.LittleEndian.Uint64(li.data))
			case 4:
				value = float64(math.Float32frombits(binary.LittleEndian.Uint32(li.data)))
			case 2:
				bits := uint32(binary.LittleEndian.Uint16(li.data)) << 16
				value = float64(math.Float32frombits(bits))
			default:
				err = errors.New("invalid compact float data length")
			}
		}
	case 9:
		// IEEE Double (IEEE 754 double-precision)
		// Check for special NaN encoding
		if len(li.data) == 2 && li.data[0] == 0xF8 {
			if li.data[1] == 0x7F {
				value = math.NaN()
			} else if li.data[1] == 0xFF {
				value = math.Copysign(math.NaN(), -1)
			} else {
				if len(li.data) != 8 {
					err = errors.New("invalid IEEE double data length")
					return
				}
				bits := binary.LittleEndian.Uint64(li.data)
				value = math.Float64frombits(bits)
			}
		} else {
			if len(li.data) != 8 {
				err = errors.New("invalid IEEE double data length")
				return
			}
			bits := binary.LittleEndian.Uint64(li.data)
			value = math.Float64frombits(bits)
		}
	default:
		err = errors.New("not implemented")
	}
	return
}

type AnyType ListItem

func (v *AnyType) Int() int {
	var value int
	(*ListItem)(v).Get(&value)
	return value
}

func (li *ListItem) GetAny() AnyType {
	return AnyType(*li)
}

func (li *ListItem) DataLength() int {
	return len(li.data)
}

func (li *ListItem) Get(value interface{}) (err error) {
	switch v := value.(type) {
	case *int:
		*v, err = li.asInt()
	case *bool:
		var temp int
		temp, err = li.asInt()
		*v = temp != 0
	case *int8:
		var temp int
		temp, err = li.asInt()
		*v = int8(temp)
	case *int16:
		var temp int
		temp, err = li.asInt()
		*v = int16(temp)
	case *int32:
		var temp int
		temp, err = li.asInt()
		*v = int32(temp)
	case *int64:
		var temp int
		temp, err = li.asInt()
		*v = int64(temp)
	case *uint:
		var temp int
		temp, err = li.asInt()
		*v = uint(temp)
	case *uint8:
		var temp int
		temp, err = li.asInt()
		*v = uint8(temp)
	case *uint16:
		var temp int
		temp, err = li.asInt()
		*v = uint16(temp)
	case *uint32:
		var temp int
		temp, err = li.asInt()
		*v = uint32(temp)
	case *uint64:
		var temp int
		temp, err = li.asInt()
		*v = uint64(temp)
	case *float64:
		*v, err = li.asFloat64()
	case *float32:
		var temp float64
		temp, err = li.asFloat64()
		*v = float32(temp)
	case *string:
		*v, err = li.asString()
	case *[]byte:
		*v = li.data
	case *iris.Oref:
		var temp string
		temp, err = li.asString()
		*v = iris.Oref(temp)
	default:
		err = errors.New("not implemented")
	}
	return
}

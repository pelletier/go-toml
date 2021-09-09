package toml

import (
	"io"
	"os"
	"strings"
	"time"
)

var cachedCnf map[string]interface{}

func ReadInString(s string) error {
	return ReadInBytes([]byte(s))
}

func ReadInFile(filepath string) error {
	file, err := os.OpenFile(filepath, os.O_RDONLY, 0444)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	bs, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	return ReadInBytes(bs)
}

func ReadInBytes(bs []byte) error {
	clearMap(cachedCnf)
	return Unmarshal(bs, &cachedCnf)
}

// clearMap is optimized by the go compiler
func clearMap(m map[string]interface{}) {
	for k := range m {
		delete(m, k)
	}
}

const keyDelimiter = "."

func GetInterface(key string, deft interface{}) interface{} {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	return value
}

func GetInterfaceSlice(key string, deft []interface{}) []interface{} {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.([]interface{})
	if !ok {
		return deft
	}
	return ret
}

func GetString(key string, deft string) string {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.(string)
	if !ok {
		return deft
	}
	return ret
}

func GetStringSlice(key string, deft []string) []string {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.([]string)
	if !ok {
		return deft
	}
	return ret
}

func GetInt(key string, deft int) int {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.(int)
	if !ok {
		return deft
	}
	return ret
}

func GetIntSlice(key string, deft []int) []int {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.([]int)
	if !ok {
		return deft
	}
	return ret
}

func GetInt8(key string, deft int8) int8 {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.(int8)
	if !ok {
		return deft
	}
	return ret
}

func GetInt8Slice(key string, deft []int8) []int8 {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.([]int8)
	if !ok {
		return deft
	}
	return ret
}

func GetInt16(key string, deft int16) int16 {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.(int16)
	if !ok {
		return deft
	}
	return ret
}

func GetInt16Slice(key string, deft []int16) []int16 {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.([]int16)
	if !ok {
		return deft
	}
	return ret
}

func GetInt32(key string, deft int32) int32 {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.(int32)
	if !ok {
		return deft
	}
	return ret
}

func GetInt32Slice(key string, deft []int32) []int32 {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.([]int32)
	if !ok {
		return deft
	}
	return ret
}

func GetInt64(key string, deft int64) int64 {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.(int64)
	if !ok {
		return deft
	}
	return ret
}

func GetInt64Slice(key string, deft []int64) []int64 {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.([]int64)
	if !ok {
		return deft
	}
	return ret
}

func GetFloat32(key string, deft float32) float32 {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.(float32)
	if !ok {
		return deft
	}
	return ret
}

func GetFloat32Slice(key string, deft []float32) []float32 {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.([]float32)
	if !ok {
		return deft
	}
	return ret
}

func GetFloat64(key string, deft float64) float64 {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.(float64)
	if !ok {
		return deft
	}
	return ret
}

func GetFloat64Slice(key string, deft []float64) []float64 {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.([]float64)
	if !ok {
		return deft
	}
	return ret
}

func GetBoolean(key string, deft bool) bool {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.(bool)
	if !ok {
		return deft
	}
	return ret
}

func GetMap(key string, deft map[string]interface{}) map[string]interface{} {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.(map[string]interface{})
	if !ok {
		return deft
	}
	return ret
}

func GetMapSlice(key string, deft []map[string]interface{}) []map[string]interface{} {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.([]map[string]interface{})
	if !ok {
		return deft
	}
	return ret
}

func GetOffsetDateTime(key string, deft time.Time) time.Time {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.(time.Time)
	if !ok {
		return deft
	}
	return ret
}

func GetLocalDateTime(key string, deft LocalDateTime) LocalDateTime {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.(LocalDateTime)
	if !ok {
		return deft
	}
	return ret
}

func GetLocalDate(key string, deft LocalDate) LocalDate {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.(LocalDate)
	if !ok {
		return deft
	}
	return ret
}

func GetLocalTime(key string, deft LocalTime) LocalTime {
	value, ok := findInCnf(key, cachedCnf)
	if !ok {
		return deft
	}

	ret, ok := value.(LocalTime)
	if !ok {
		return deft
	}
	return ret
}

func findInCnf(key string, from map[string]interface{}) (interface{}, bool) {
	if from == nil {
		return nil, false
	}

	n := strings.Index(key, keyDelimiter)
	if n == -1 {
		// if has no delimiter
		value, ok := from[key]
		return value, ok
	}

	ks := strings.Split(key, keyDelimiter)
	l := len(ks)

	for i := 0; i < l; i++ {
		v, ok := findInCnf(ks[i], from)
		if !ok {
			return nil, false
		}

		if i == l-1 {
			return v, true
		} else {
			// if has delimiter, the value must be map
			f, ok := v.(map[string]interface{})
			if !ok {
				return nil, false
			}
			from = f
		}
	}

	return nil, false
}

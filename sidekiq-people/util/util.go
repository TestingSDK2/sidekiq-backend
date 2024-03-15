package util

import (
	cr "crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func EncodeToString(max int) (string, error) {
	table := [...]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}
	b := make([]byte, max)
	n, err := io.ReadAtLeast(cr.Reader, b, max)
	if n != max {
		return "", err
	}
	for i := 0; i < len(b); i++ {
		b[i] = table[int(b[i])%len(table)]
	}
	return string(b), nil
}

func Get8DigitCode() (res string) {
	rand.Seed(time.Now().UnixNano())
	letters := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	for i := 0; i < 8; i++ {
		res += string(letters[rand.Intn(len(letters))])
	}
	return
}

func Round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

func ConvertBytesToHumanReadable(sizeInBytes int64) (sizeHumanReadable string) {
	suffixes := [5]string{"B", "KB", "MB", "GB", "TB"}
	base := math.Log(float64(sizeInBytes)) / math.Log(1024)
	getSize := Round(math.Pow(1024, base-math.Floor(base)), .5, 2)
	getSuffix := suffixes[int(math.Floor(base))]

	sizeHumanReadable = strconv.FormatFloat(getSize, 'f', -1, 64) + " " + string(getSuffix)
	return
}

func DetermineFileType(fileName string) (fileType string) {
	fileType = filepath.Ext(fileName)
	fmt.Println("filetype: ", fileType)
	return fileType
}

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func ToJson(boardIds map[string]string) string {
	json, _ := json.Marshal(boardIds)
	return string(json)
}

func SetResponse(data interface{}, status int, message string) map[string]interface{} {
	response := make(map[string]interface{})
	response["data"] = nil
	if data != nil {
		response["data"] = data
	}
	response["status"] = status
	response["message"] = message
	return response
}

func ParseDate(createDate interface{}) time.Time {
	if newDate, ok := createDate.(primitive.DateTime); ok {
		return newDate.Time()
	} else if newDate, ok := createDate.(string); ok {
		parsedTime, _ := time.Parse(time.RFC3339Nano, newDate)
		return parsedTime
	}

	// fmt.Printf("No match found. Missing createDate %T type\n", createDate)
	return time.Time{}
}
func GetTitle(data map[string]interface{}) string {
	title := ""
	if v, ok := data["title"]; ok {
		if v.(string) != "" {
			title = v.(string)
		}
	}
	return title
}

func SetPaginationResponse(data interface{}, total, status int, message string) map[string]interface{} {
	response := map[string]interface{}{
		"data": map[string]interface{}{
			"info":  []int{},
			"total": 0,
		},
		"status":  status,
		"message": message,
	}
	if data != nil {
		response["data"].(map[string]interface{})["info"] = data
		response["data"].(map[string]interface{})["total"] = total
	}
	return response
}

func RemoveArrayDuplicate(arr []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range arr {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func RemoveHtmlTag(in string) string {
	// regex to match html tag
	const pattern = `(<\/?[a-zA-A]+?[^>]*\/?>)*`
	r := regexp.MustCompile(pattern)
	groups := r.FindAllString(in, -1)
	// should replace long string first
	sort.Slice(groups, func(i, j int) bool {
		return len(groups[i]) > len(groups[j])
	})
	for _, group := range groups {
		if strings.TrimSpace(group) != "" {
			in = strings.ReplaceAll(in, group, "")
		}
	}
	return in
}

// Remove string element from a string array
func Remove(s []string, r string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func PrettyPrint(data ...interface{}) error {
	fmt.Println()
	byteData, err := json.MarshalIndent(data[len(data)-1], "", " ")
	if err != nil {
		return err
	}
	if len(data) == 1 {
		fmt.Print(data[:len(data)-1]...)
	} else {
		fmt.Println(data[:len(data)-1]...)
	}
	fmt.Println(string(byteData))
	fmt.Println()
	return nil
}

func Recover2(errChan chan<- error, line int) {
	if r := recover(); r != nil {
		// Handle the panic here
		fmt.Printf("Recovered from go routine panic: %v, line: %d\n", r, line)
		errChan <- errors.Wrap(nil, "error due to panic")
	}
}

func Recover3(errChan chan<- error, line int, ch chan bool) {
	<-ch
	if r := recover(); r != nil {
		// Handle the panic here
		fmt.Printf("Recovered from go routine panic: %v, line: %d\n", r, line)
		fmt.Println("Channel is emptied")
		errChan <- errors.Wrap(nil, "error due to panic")
	}
}
func RecoverGoroutinePanic(errChan chan<- error) {
	if r := recover(); r != nil {
		// Handle the panic here
		fmt.Println("Recovered from go routine panic:", r)
		errChan <- errors.Wrap(nil, "error due to panic")
	}
}

func Recover() {
	if r := recover(); r != nil {
		// Handle the panic here
		fmt.Println("Recovered from panic:", r)
	}
}

func PaginateFromArray(arr []interface{}, pageNo, limit int) (ret []interface{}) {
	var startIdx, endIdx int
	startIdx = limit * (pageNo - 1)
	endIdx = limit * pageNo

	if len(arr) == limit || len(arr) < limit {
		return arr
	}
	if endIdx < len(arr) {
		ret = arr[startIdx:endIdx]
	} else {
		ret = arr[startIdx:]
	}
	return
}

// 'pdf',
//
//	'doc',
//	'docx',
//	'xlsx',
//	'xls',
//	'ppt',
//	'pptx',
func ReturnFileType(mime string) string {
	if mime == "application/pdf" {
		return "pdf"
	} else if strings.Contains(mime, "audio/") {
		return "audio"
	} else if strings.Contains(mime, "video/") {
		return "video"
	} else if strings.Contains(mime, "image/") {
		return "image"
	} else if mime == "application/json" {
		return "json"
	} else if mime == "application/msword" {
		return "doc"
	} else if mime == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
		return "docx"
	} else if mime == "application/vnd.ms-excel" {
		return "xls"
	} else if mime == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		return "xlsx"
	} else if mime == "application/vnd.ms-powerpoint" {
		return "ppt"
	} else if mime == "application/vnd.openxmlformats-officedocument.presentationml.presentation" {
		return "pptx"
	}
	return ""
}

// ToMap converts a struct to a map[string]interface{}
func ToMap(obj interface{}) (map[string]interface{}, error) {
	objValue := reflect.ValueOf(obj)
	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	if objValue.Kind() != reflect.Struct {
		return nil, fmt.Errorf("toMap: input must be a struct or a pointer to a struct")
	}

	objType := objValue.Type()
	resultMap := make(map[string]interface{})

	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		fieldValue := objValue.Field(i).Interface()
		resultMap[field.Name] = fieldValue
	}

	return resultMap, nil
}

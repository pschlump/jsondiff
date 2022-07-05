package jsondiff

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"sort"

	"github.com/mgutz/ansi"
	"github.com/pschlump/dbgo"
)

// ResolutionType defines a type of comparison: equality, non-equality,
// new sub-diff and so on
type ResolutionType int

const (
	TypeEquals ResolutionType = iota
	TypeNotEquals
	TypeAdded
	TypeRemoved
	TypeDiff

	indentation = "    "
)

var (
	colorStartYellow = ansi.ColorCode("yellow")
	colorStartRed    = ansi.ColorCode("red")
	colorStartGreen  = ansi.ColorCode("green")
	colorReset       = ansi.ColorCode("reset")
)

// Diff is a result of comparison operation. Provides list
// of items that describe difference between objects piece by piece
type Diff struct {
	items   []DiffItem //
	isArray bool       //
	Err     error      // error message if any (can't unmarsal into ...)
	HasDiff bool       // True if there is a difference or error makes it impossible to check
}

// String allows for printing of ResolutionType as a name
func (rt ResolutionType) String() string {
	switch rt {
	case TypeEquals:
		return "TypeEquals"
	case TypeNotEquals:
		return "TypeNotEquals"
	case TypeAdded:
		return "TypeAdded"
	case TypeRemoved:
		return "TypeRemoved"
	case TypeDiff:
		return "TypeDiff"
	}
	return "--internal error--"
}

// Items returns list of diff items
func (d Diff) Items() []DiffItem { return d.items }

// Add adds new item to diff object
func (d *Diff) Add(item DiffItem) {
	d.items = append(d.items, item)
	if item.Resolution != TypeEquals {
		d.HasDiff = true
	}
}

// IsEqual checks if given diff objects does not contain any non-equal
// element. When IsEqual returns "true" that means there is no difference
// between compared objects
func (d Diff) IsEqual() bool { return !d.HasDiff }

func (d *Diff) sort() { sort.Sort(byKey(d.items)) }

// DiffItem defines a difference between 2 items with resolution type
type DiffItem struct {
	Key        string
	ValueA     interface{}
	Resolution ResolutionType
	ValueB     interface{}
}

type byKey []DiffItem

func (m byKey) Len() int           { return len(m) }
func (m byKey) Less(i, j int) bool { return m[i].Key < m[j].Key }
func (m byKey) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }

// Compare produces list of diff items that define difference between
// objects "a" and "b".
// Note: if objects are equal, all diff items will have Resolution of
// type TypeEquals
func Compare(a, b interface{}) Diff {
	mapA := map[string]interface{}{}
	mapB := map[string]interface{}{}

	jsonA, errA := json.Marshal(a)
	if errA != nil {
		return Diff{HasDiff: true, Err: errA}
	}
	jsonB, errB := json.Marshal(b)
	if errB != nil {
		return Diff{HasDiff: true, Err: errB}
	}

	errA = json.Unmarshal(jsonA, &mapA)
	if errA != nil {
		arrA := []interface{}{}
		arrB := []interface{}{}
		errA = json.Unmarshal(jsonA, &arrA)
		if errA != nil {
			return Diff{HasDiff: true, Err: errA}
		}
		errB := json.Unmarshal(jsonB, &arrB)
		if errB != nil {
			return Diff{HasDiff: true, Err: errB}
		}
		rv := compareArrays(arrA, arrB)
		rv.isArray = true
		return rv
	}
	errB = json.Unmarshal(jsonB, &mapB)
	if errB != nil {
		return Diff{HasDiff: true, Err: errB}
	}

	return compareStringMaps(mapA, mapB)
}

// CompareFiels will read in 2 files, hopefully both JSON, and
// perform the compare on them.
func CompareFiles(afn, bfn string) Diff {
	jsonA, aErr := ioutil.ReadFile(afn)
	if aErr != nil {
		fmt.Printf("Unable to open %s, error=%s\n", afn, aErr)
		return Diff{HasDiff: true}
	}
	jsonB, bErr := ioutil.ReadFile(bfn)
	if bErr != nil {
		fmt.Printf("Unable to open %s, error=%s\n", bfn, bErr)
		return Diff{HasDiff: true}
	}

	mapA := map[string]interface{}{}
	mapB := map[string]interface{}{}
	arrA := []interface{}{}
	arrB := []interface{}{}

	errA := json.Unmarshal(jsonA, &mapA)
	if errA != nil {
		if db1 {
			dbgo.Printf("1st errA = %s, at:%(LF)\n", errA)
		}
		errA = json.Unmarshal(jsonA, &arrA)
		errB := json.Unmarshal(jsonB, &arrB)
		if errA != nil {
			fmt.Printf("Unable to parse %s, error=%s\n", afn, errA)
			return Diff{HasDiff: true}
		}
		if errB != nil {
			fmt.Printf("Unable to parse %s, error=%s\n", bfn, errB)
			return Diff{HasDiff: true}
		}
		rv := compareArrays(arrA, arrB)
		rv.isArray = true
		return rv
	}
	errB := json.Unmarshal(jsonB, &mapB)
	if errA != nil && errB != nil {
		fmt.Printf("Neither %s nor %s are in JSON format, %s, %s\n", errA, errB)
		return Diff{HasDiff: false}
	} else if errA != nil {
		fmt.Printf("Unable to parse %s, error=%s\n", afn, errA)
		return Diff{HasDiff: true}
	} else if errB != nil {
		fmt.Printf("Unable to parse %s, error=%s\n", bfn, errB)
		return Diff{HasDiff: true}
	}

	return compareStringMaps(mapA, mapB)
}

// CompareMemToFile compares an in memory structure to a file.  The primary intended
// use for this is in testing code where an in-memory structure has been created
// and a correct reference copy is in a directory on disk.
//
//  d := jsondiff.Compare ( inMem, "./testdir/test1.json" )
//	if d.HasDiff {
//		t.Error ( "failed to pass test1" )
//	}
//
func CompareMemToFile(a interface{}, bfn string) Diff {
	jsonA, errA := json.Marshal(a)
	if errA != nil {
		return Diff{HasDiff: true, Err: errA}
	}
	jsonB, bErr := ioutil.ReadFile(bfn)
	if bErr != nil {
		fmt.Printf("Unable to open %s, error=%s\n", bfn, bErr)
		return Diff{HasDiff: true}
	}

	mapA := map[string]interface{}{}
	mapB := map[string]interface{}{}
	arrA := []interface{}{}
	arrB := []interface{}{}

	errA = json.Unmarshal(jsonA, &mapA)
	if errA != nil {
		if db1 {
			dbgo.Printf("1st errA = %s, %(LF)\n", errA)
		}
		errA = json.Unmarshal(jsonA, &arrA)
		errB := json.Unmarshal(jsonB, &arrB)
		if errA != nil {
			fmt.Printf("Unable to parse generated JSON, error=%s\n", errA)
			return Diff{HasDiff: true}
		}
		if errB != nil {
			fmt.Printf("Unable to parse %s, error=%s\n", bfn, errB)
			return Diff{HasDiff: true}
		}
		rv := compareArrays(arrA, arrB)
		rv.isArray = true
		return rv
	}
	errB := json.Unmarshal(jsonB, &mapB)
	if errA != nil && errB != nil {
		fmt.Printf("Neither %s nor %s are in JSON format, %s, %s\n", errA, errB)
		return Diff{HasDiff: false}
	} else if errA != nil {
		fmt.Printf("Unable to parse generated JSON, error=%s\n", errA)
		return Diff{HasDiff: true}
	} else if errB != nil {
		fmt.Printf("Unable to parse %s, error=%s\n", bfn, errB)
		return Diff{HasDiff: true}
	}

	return compareStringMaps(mapA, mapB)
}

// Format produces formatted output for a diff that can be printed.
// Uses colorization which may not work with terminals that don't
// support ASCII coloring (Windows is under question).
func Format(diff Diff) []byte {
	buf := bytes.Buffer{}

	writeItems(&buf, "", diff.Items(), diff.isArray)

	return buf.Bytes()
}

// xyzzy - error in output, need to see if array, or hash at top
func writeItems(writer io.Writer, prefix string, items []DiffItem, isArray bool) {
	if isArray {
		writer.Write([]byte{'['})
	} else {
		writer.Write([]byte{'{'})
	}
	last := len(items) - 1

	prefixNotEqualsA := prefix + "<> "
	prefixNotEqualsB := prefix + "** "
	prefixAdded := prefix + "<< "
	prefixRemoved := prefix + ">> "

	for i, item := range items {
		writer.Write([]byte{'\n'})

		switch item.Resolution {
		case TypeEquals:
			writeItem(writer, prefix, item.Key, item.ValueA, i < last, isArray)
		case TypeNotEquals:
			writer.Write([]byte(colorStartYellow))

			writeItem(writer, prefixNotEqualsA, item.Key, item.ValueA, i < last, isArray)
			writer.Write([]byte{'\n'})
			writeItem(writer, prefixNotEqualsB, item.Key, item.ValueB, i < last, isArray)

			writer.Write([]byte(colorReset))
		case TypeAdded:
			writer.Write([]byte(colorStartGreen))
			writeItem(writer, prefixAdded, item.Key, item.ValueB, i < last, isArray)
			writer.Write([]byte(colorReset))
		case TypeRemoved:
			writer.Write([]byte(colorStartRed))
			writeItem(writer, prefixRemoved, item.Key, item.ValueA, i < last, isArray)
			writer.Write([]byte(colorReset))
		case TypeDiff:
			subdiff := item.ValueB.([]DiffItem)
			fmt.Fprintf(writer, "%s\"%s\": ", prefix, item.Key)
			writeItems(writer, prefix+indentation, subdiff, false) // xyzzy
			if i < last {
				writer.Write([]byte{','})
			}
		}

	}

	if isArray {
		fmt.Fprintf(writer, "\n%s]", prefix)
	} else {
		fmt.Fprintf(writer, "\n%s}", prefix)
	}
}

func writeItem(writer io.Writer, prefix, key string, value interface{}, isNotLast bool, isArray bool) {
	if isArray {
		fmt.Fprintf(writer, "%s ", prefix)
	} else {
		fmt.Fprintf(writer, "%s\"%s\": ", prefix, key)
	}
	serialized, _ := json.Marshal(value)

	writer.Write(serialized)
	if isNotLast {
		writer.Write([]byte{','})
	}
}

func compare(A, B interface{}) (ResolutionType, Diff) {
	equals := reflect.DeepEqual(A, B)
	if equals {
		return TypeEquals, Diff{}
	}

	mapA, okA := A.(map[string]interface{})
	mapB, okB := B.(map[string]interface{})

	if okA && okB {
		diff := compareStringMaps(mapA, mapB)
		return TypeDiff, diff
	}

	arrayA, okA := A.([]interface{})
	arrayB, okB := B.([]interface{})

	if okA && okB {
		diff := compareArrays(arrayA, arrayB)
		return TypeDiff, diff
	}

	return TypeNotEquals, Diff{}
}

func compareArrays(A, B []interface{}) Diff {
	result := Diff{}

	minLength := len(A)
	if len(A) > len(B) {
		minLength = len(B)
	}

	for i := 0; i < minLength; i++ {
		resolutionType, subdiff := compare(A[i], B[i])

		switch resolutionType {
		case TypeEquals:
			result.Add(DiffItem{"", A[i], TypeEquals, nil})
		case TypeNotEquals:
			result.Add(DiffItem{"", A[i], TypeNotEquals, B[i]})
		case TypeDiff:
			result.Add(DiffItem{"", nil, TypeDiff, subdiff.Items()})
		}
	}

	for i := minLength; i < len(A); i++ {
		result.Add(DiffItem{"", A[i], TypeRemoved, nil})
	}

	for i := minLength; i < len(B); i++ {
		result.Add(DiffItem{"", nil, TypeAdded, B[i]})
	}

	return result
}

func compareStringMaps(A, B map[string]interface{}) Diff {
	keysA := sortedKeys(A)
	keysB := sortedKeys(B)

	result := Diff{}

	for _, kA := range keysA {
		vA := A[kA]

		vB, ok := B[kA]
		if !ok {
			result.Add(DiffItem{kA, vA, TypeRemoved, nil})
			continue
		}

		resolutionType, subdiff := compare(vA, vB)

		switch resolutionType {
		case TypeEquals:
			result.Add(DiffItem{kA, vA, TypeEquals, nil})
		case TypeNotEquals:
			result.Add(DiffItem{kA, vA, TypeNotEquals, vB})
		case TypeDiff:
			result.Add(DiffItem{kA, nil, TypeDiff, subdiff.Items()})
		}
	}

	for _, kB := range keysB {
		if _, ok := A[kB]; !ok {
			result.Add(DiffItem{kB, nil, TypeAdded, B[kB]})
		}
	}

	result.sort()

	return result
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

const db1 = false

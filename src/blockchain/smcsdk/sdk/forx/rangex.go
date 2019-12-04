package forx

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

const (
	Break        = false
	Continue     = true
	maxLoopCount = 10000
)

/*
 * Package forx implements for range function, allows program to visit member of Map or Slice,
 * if want call func with loop count, put in Integer to define loop count;
 *
 * for example:
 * Map:
 * forx.Range( map[t1]t2, func( t1, t2 ) bool {
 *     // do something
 *     return forx.Break
 * 	   or
 *     return forx.Continue
 *     // do something
 *
 *     return true
 *  })
 *
 * Slice:
 * forx.Range( []t, func( int, t ) bool {
 *     // do something
 *     return forx.Break
 * 	   or
 *     return forx.Continue
 *     // do something
 *
 *     return true
 * })
 *
 * loopCount:
 * forx.Range( int, func( int ) bool {
 *     // do something
 *     return forx.Break
 * 	   or
 *     return forx.Continue
 *     // do something
 *
 *     return true
 * })
 *
 * from start to end:
 * forx.Range( int, int, func( int ) bool {
 *     // do something
 *     return forx.Break
 * 	   or
 *     return forx.Continue
 *     // do something
 *
 *     return true
 * })
 *
 * loop forever:
 * forx.Range( func( int ) bool {
 *     // do something
 *     return forx.Break
 * 	   or
 *     return forx.Continue
 *     // do something
 *
 *     return true
 * })
 *
 * loop bool:
 * forx.Range( func() bool {return condition},  func( int ) bool {
 *     // do something
 *     return forx.Break
 * 	   or
 *     return forx.Continue
 *     // do something
 *
 *     return true
 * })
 */

// Range - range object
func Range(args ...interface{}) {

	if len(args) == 1 {
		checkFunc(args[0])

		rangeOfBreak(args[0])
	} else if len(args) == 2 {
		checkFunc(args[1])

		if reflect.TypeOf(args[0]).Kind() == reflect.Map {
			rangeMap(args[0], args[1])
		} else if reflect.TypeOf(args[0]).Kind() == reflect.Slice {
			rangeSlice(args[0], args[1])
		} else if strings.Contains(reflect.TypeOf(args[0]).Kind().String(), "int") {
			rangeOfCount(args[0], args[1])
		} else if reflect.TypeOf(args[0]).Kind() == reflect.Func {
			rangeOfBool(args[0], args[1])
		} else {
			panic("first parameter must be Map,Slice,Integer or Func")
		}
	} else if len(args) == 3 {
		checkFunc(args[2])

		fromType := reflect.TypeOf(args[0]).Kind().String()
		toType := reflect.TypeOf(args[1]).Kind().String()

		if strings.ContainsAny(fromType, "int") && strings.ContainsAny(toType, "int") {
			rangeOfStartEnd(args[0], args[1], args[2])
		} else {
			panic("first and second parameters must be Integer")
		}
	} else {
		panic("invalid parameters")
	}
}

// RangeReverse - reverse range object
func RangeReverse(args ...interface{}) {

	if len(args) == 2 {
		checkFunc(args[1])

		if reflect.TypeOf(args[0]).Kind() == reflect.Slice {
			rangeSliceReverse(args[0], args[1])
		} else {
			panic("first parameter must be Map,Slice,Integer or Func")
		}
	} else {
		panic("invalid parameters")
	}
}

func rangeMap(mapObj, f interface{}) {
	// check map object
	mapObjValue := reflect.ValueOf(mapObj)
	mapObjType := reflect.TypeOf(mapObj)
	keyType := mapObjType.Key()
	valueType := mapObjType.Elem()

	// check operation function
	funcType := reflect.TypeOf(f)
	numIn := funcType.NumIn()
	numOut := funcType.NumOut()
	if numIn != 2 {
		panic("Func must be two in parameters")
	}

	if funcType.In(0) != keyType {
		panic(fmt.Sprintf("Func's first in parameter's type should be %s, obtain %s",
			keyType.String(), funcType.In(0).String()))
	}

	if funcType.In(1) != valueType {
		panic(fmt.Sprintf("Func's second in parameter's type should be %s, obtain %s",
			valueType.String(), funcType.In(1).String()))
	}

	// sort keys
	ks := mapObjValue.MapKeys()
	sort.SliceStable(ks, func(i, j int) bool {
		switch keyType.Kind() {
		case reflect.String:
			return ks[i].String() < ks[j].String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return ks[i].Int() < ks[j].Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return ks[i].Uint() < ks[j].Uint()
		case reflect.Float32, reflect.Float64:
			return ks[i].Float() < ks[j].Float()
		case reflect.Bool:
			return ks[i].Bool() == false && ks[j].Bool() == true
		default:
			panic(fmt.Sprintf("do not support key's type:%s", keyType.String()))
		}

		return false
	})

	// check loop count
	if len(ks) > maxLoopCount {
		panic("loop count cannot greater than " + fmt.Sprintf("%d", maxLoopCount))
	}

	// range map object
	var retVal []reflect.Value
	for _, k := range ks {
		fValue := reflect.ValueOf(f)

		in := make([]reflect.Value, 2)
		in[0] = k
		in[1] = mapObjValue.MapIndex(k)

		if numOut == 1 {
			retVal = fValue.Call(in)
			if retVal[0].Bool() == Break {
				return
			}
		} else {
			fValue.Call(in)
		}
	}
}

func rangeSlice(sliceObj, f interface{}) {
	// check slice object
	sliceObjValue := reflect.ValueOf(sliceObj)
	sliceObjType := reflect.TypeOf(sliceObj)
	valueType := sliceObjType.Elem()

	// check operation function
	funcType := reflect.TypeOf(f)
	numIn := funcType.NumIn()
	numOut := funcType.NumOut()
	if numIn != 2 {
		panic("Func must be two in parameters")
	}

	if funcType.In(0).Kind() != reflect.Int {
		panic(fmt.Sprintf("Func's first in parameter's type should be int, obtain %s", funcType.In(0).String()))
	}

	if funcType.In(1) != valueType {
		panic(fmt.Sprintf("Func's second in parameter's type should be %s, obtain %s",
			valueType.String(), funcType.In(1).String()))
	}

	// check loop data
	length := sliceObjValue.Len()
	if length > maxLoopCount {
		panic("loop count cannot greater than " + fmt.Sprintf("%d", maxLoopCount))
	}

	// range slice object
	index := 0
	var retVal []reflect.Value
	for index < length {
		fValue := reflect.ValueOf(f)

		in := make([]reflect.Value, 2)
		in[0] = reflect.ValueOf(index)
		in[1] = sliceObjValue.Index(index)

		if numOut == 1 {
			retVal = fValue.Call(in)
			if retVal[0].Bool() == Break {
				return
			}
		} else {
			fValue.Call(in)
		}

		index++
	}
}

func rangeSliceReverse(sliceObj, f interface{}) {
	// check slice object
	sliceObjValue := reflect.ValueOf(sliceObj)
	sliceObjType := reflect.TypeOf(sliceObj)
	valueType := sliceObjType.Elem()

	// check operation function
	funcType := reflect.TypeOf(f)
	numIn := funcType.NumIn()
	numOut := funcType.NumOut()
	if numIn != 2 {
		panic("Func must be two in parameters")
	}

	if funcType.In(0).Kind() != reflect.Int {
		panic(fmt.Sprintf("Func's first in parameter's type should be int, obtain %s", funcType.In(0).String()))
	}

	if funcType.In(1) != valueType {
		panic(fmt.Sprintf("Func's second in parameter's type should be %s, obtain %s",
			valueType.String(), funcType.In(1).String()))
	}

	// check loop data
	length := sliceObjValue.Len()
	if length > maxLoopCount {
		panic("loop count cannot greater than " + fmt.Sprintf("%d", maxLoopCount))
	}

	// range slice object
	index := length - 1
	var retVal []reflect.Value
	for index >= 0 {
		fValue := reflect.ValueOf(f)

		in := make([]reflect.Value, 2)
		in[0] = reflect.ValueOf(index)
		in[1] = sliceObjValue.Index(index)

		if numOut == 1 {
			retVal = fValue.Call(in)
			if retVal[0].Bool() == Break {
				return
			}
		} else {
			fValue.Call(in)
		}

		index--
	}
}

func rangeOfCount(loopCount, f interface{}) {
	// check operation function
	funcType := reflect.TypeOf(f)
	numIn := funcType.NumIn()
	numOut := funcType.NumOut()
	if numIn != 1 {
		panic("Func must be one in parameters")
	}

	if funcType.In(0).Kind() != reflect.Int {
		panic(fmt.Sprintf("Func's first in parameter's type should be int, obtain %s", funcType.In(0).String()))
	}

	// check loop data
	count := interfaceToInt(loopCount)
	if count < 0 {
		panic("loop count cannot less than zero")
	}
	if count > maxLoopCount {
		panic("loop count cannot greater than " + fmt.Sprintf("%d", maxLoopCount))
	}

	// loop
	index := 0
	var retVal []reflect.Value
	for index < count {
		fValue := reflect.ValueOf(f)

		in := make([]reflect.Value, 1)
		in[0] = reflect.ValueOf(index)

		if numOut == 1 {
			retVal = fValue.Call(in)
			if retVal[0].Bool() == Break {
				return
			}
		} else {
			fValue.Call(in)
		}

		index++
	}
}

func rangeOfStartEnd(start, end, f interface{}) {

	// check operation function
	funcType := reflect.TypeOf(f)
	numIn := funcType.NumIn()
	numOut := funcType.NumOut()
	if numIn != 1 {
		panic("Func must be one in parameters")
	}

	if funcType.In(0).Kind() != reflect.Int {
		panic(fmt.Sprintf("Func's first in parameter's type should be int, obtain %s", funcType.In(0).String()))
	}

	// check loop data
	startI := interfaceToInt(start)
	endI := interfaceToInt(end)
	if startI > endI {
		panic("start cannot greater than end")
	}
	if endI-startI+1 > maxLoopCount {
		panic("loop count cannot greater than " + fmt.Sprintf("%d", maxLoopCount))
	}

	// loop
	index := startI
	var retVal []reflect.Value
	for index <= endI {
		fValue := reflect.ValueOf(f)

		in := make([]reflect.Value, 1)
		in[0] = reflect.ValueOf(index)

		if numOut == 1 {
			retVal = fValue.Call(in)
			if retVal[0].Bool() == Break {
				return
			}
		} else {
			fValue.Call(in)
		}

		index++
	}
}

func rangeOfBreak(f interface{}) {

	// check operation function
	funcType := reflect.TypeOf(f)
	numIn := funcType.NumIn()
	numOut := funcType.NumOut()
	if numIn != 1 {
		panic("Func must be one in parameters")
	}

	if funcType.In(0).Kind() != reflect.Int {
		panic(fmt.Sprintf("Func's first in parameter's type should be int, obtain %s", funcType.In(0).String()))
	}

	// loop
	index := 0
	var retVal []reflect.Value
	for true {
		fValue := reflect.ValueOf(f)

		in := make([]reflect.Value, 1)
		in[0] = reflect.ValueOf(index)

		if numOut == 1 {
			retVal = fValue.Call(in)
			if retVal[0].Bool() == Break {
				return
			}
		} else {
			fValue.Call(in)
		}

		index++
		if index > maxLoopCount {
			panic("loop count cannot greater than " + fmt.Sprintf("%d", maxLoopCount))
		}
	}
}

func rangeOfBool(fc, f interface{}) {

	// check condition function
	fcType := reflect.TypeOf(fc)
	numIn := fcType.NumIn()
	if numIn != 0 {
		panic("condition Func must be zero in parameters")
	}

	// check operation function
	funcType := reflect.TypeOf(f)
	numIn = funcType.NumIn()
	numOut := funcType.NumOut()
	if numIn != 1 {
		panic("Func must be one in parameters")
	}

	if funcType.In(0).Kind() != reflect.Int {
		panic(fmt.Sprintf("Func's first in parameter's type should be int, obtain %s", funcType.In(0).String()))
	}

	// loop
	index := 0
	var retVal []reflect.Value
	for true {
		fcValue := reflect.ValueOf(fc)
		retVal = fcValue.Call([]reflect.Value{})
		if retVal[0].Bool() == Break {
			return
		}

		fValue := reflect.ValueOf(f)

		in := make([]reflect.Value, 1)
		in[0] = reflect.ValueOf(index)

		if numOut == 1 {
			retVal = fValue.Call(in)
			if retVal[0].Bool() == Break {
				return
			}
		} else {
			fValue.Call(in)
		}

		index++
		if index > maxLoopCount {
			panic("loop count cannot greater than " + fmt.Sprintf("%d", maxLoopCount))
		}
	}
}

func checkFunc(f interface{}) {
	if reflect.TypeOf(f).Kind() != reflect.Func {
		panic("last parameter must be a Func")
	}

	funcType := reflect.TypeOf(f)
	numOut := funcType.NumOut()

	if numOut > 1 {
		panic("Func must have one or zero return value")
	}

	if numOut == 1 {
		if funcType.Out(0).Kind() != reflect.Bool {
			panic("Func's return value must be a bool")
		}
	}
}

func interfaceToInt(v interface{}) int {
	vT := reflect.TypeOf(v).Kind()

	if vT == reflect.Int {
		return v.(int)
	} else if vT == reflect.Int8 {
		return int(v.(int8))
	} else if vT == reflect.Int16 {
		return int(v.(int16))
	} else if vT == reflect.Int32 {
		return int(v.(int32))
	} else if vT == reflect.Int64 {
		return int(v.(int64))
	} else if vT == reflect.Uint {
		return int(v.(uint))
	} else if vT == reflect.Uint8 {
		return int(v.(uint8))
	} else if vT == reflect.Uint16 {
		return int(v.(uint16))
	} else if vT == reflect.Uint32 {
		return int(v.(uint32))
	} else if vT == reflect.Uint64 {
		return int(v.(uint64))
	} else {
		return -1
	}
}

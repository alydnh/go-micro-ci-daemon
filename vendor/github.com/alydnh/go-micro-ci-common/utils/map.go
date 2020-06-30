package utils

import (
	"reflect"
)

func CopyMap(a interface{}) interface{} {
	aValue := reflect.ValueOf(a)
	aType := aValue.Type()
	mapType := reflect.MapOf(aType.Key(), aType.Elem())
	newMap := reflect.MakeMapWithSize(mapType, aValue.Len())
	keys := aValue.MapKeys()
	for _, key := range keys {
		newMap.SetMapIndex(key, aValue.MapIndex(key))
	}
	return newMap.Interface()
}

func WhereInMapValues(m interface{}, predicate interface{}) interface{} {
	predicateFunc := reflect.ValueOf(predicate)
	mValue := reflect.ValueOf(m)
	matchedArray := reflect.MakeSlice(reflect.SliceOf(mValue.Type().Elem()), 0, mValue.Len())
	keys := mValue.MapKeys()
	for _, key := range keys {
		value := mValue.MapIndex(key)
		if predicateFunc.Call([]reflect.Value{value})[0].Bool() {
			matchedArray = reflect.Append(matchedArray, value)
		}
	}
	return matchedArray.Interface()
}

func MapKeys(a interface{}) interface{} {
	aValue := reflect.ValueOf(a)
	aType := aValue.Type()
	sliceType := reflect.SliceOf(aType.Key())
	array := reflect.MakeSlice(sliceType, 0, aValue.Len())
	keys := aValue.MapKeys()
	for _, key := range keys {
		array = reflect.Append(array, key)
	}
	return array.Interface()
}

func MapValues(a interface{}) interface{} {
	mValue := reflect.ValueOf(a)
	matchedArray := reflect.MakeSlice(reflect.SliceOf(mValue.Type().Elem()), 0, mValue.Len())
	keys := mValue.MapKeys()
	for _, key := range keys {
		value := mValue.MapIndex(key)
		matchedArray = reflect.Append(matchedArray, value)
	}
	return matchedArray.Interface()
}

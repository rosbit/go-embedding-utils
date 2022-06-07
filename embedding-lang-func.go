package elutils

import (
	"reflect"
	"fmt"
)

type FnGoFunc func(args []reflect.Value)(results []reflect.Value)

type EmbeddingFuncHelper struct {
	dest reflect.Value
	fnType reflect.Type
}

func NewEmbeddingFuncHelper(funcVarPtr interface{}) (helper *EmbeddingFuncHelper, err error) {
	if funcVarPtr == nil {
		err = fmt.Errorf("funcVarPtr must be a non-nil poiter of func")
		return
	}
	t := reflect.TypeOf(funcVarPtr)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Func {
		err = fmt.Errorf("funcVarPtr expected to be a pointer of func")
		return
	}
	dest := reflect.ValueOf(funcVarPtr).Elem()
	fnType := dest.Type()
	helper = &EmbeddingFuncHelper{dest: dest, fnType: fnType}
	return
}

func (h *EmbeddingFuncHelper) BindEmbeddingFunc(goFunc FnGoFunc) {
	h.dest.Set(reflect.MakeFunc(h.fnType, goFunc))
}

func (h *EmbeddingFuncHelper) MakeGoFuncArgs(args []reflect.Value) (<-chan interface{}) {
	fnType := h.fnType

	itArgs := make(chan interface{})
	go func() {
		lastNumIn := fnType.NumIn() - 1
		variadic := fnType.IsVariadic()
		for i, arg := range args {
			if i < lastNumIn || !variadic {
				itArgs <- arg.Interface()
				continue
			}

			if arg.IsZero() {
				break
			}
			varLen := arg.Len()
			for j:=0; j<varLen; j++ {
				itArgs <- arg.Index(j).Interface()
			}
		}

		close(itArgs)
	}()

	return itArgs
}

func (h *EmbeddingFuncHelper) ToGolangResults(res interface{}, isResArray bool, callErr error) (results []reflect.Value) {
	var err error
	if callErr != nil {
		err = callErr
	}

	fnType := h.fnType
	results = make([]reflect.Value, fnType.NumOut())
	if err == nil {
		if fnType.NumOut() > 0 {
			if isResArray {
				mRes := res.([]interface{})
				n := fnType.NumOut()
				if n == 1 && fnType.Out(0).Kind() == reflect.Slice {
					v := MakeValue(fnType.Out(0))
					if err = SetValue(v, mRes); err == nil {
						results[0] = v
					}
				} else {
					l := len(mRes)
					if n < l {
						l = n
					}
					for i:=0; i<l; i++ {
						// v := reflect.New(fnType.Out(i)).Elem()
						v := MakeValue(fnType.Out(i))
						rv := mRes[i]
						if err = SetValue(v, rv); err == nil {
							results[i] = v
						}
					}
				}
			} else {
				// v := reflect.New(fnType.Out(0)).Elem()
				v := MakeValue(fnType.Out(0))
				if err = SetValue(v, res); err == nil {
					results[0] = v
				}
			}
		}
	}

	if err != nil {
		nOut := fnType.NumOut()
		if nOut > 0 && fnType.Out(nOut-1).Name() == "error" {
			results[nOut-1] = reflect.ValueOf(err).Convert(fnType.Out(nOut-1))
		} else {
			panic(err)
		}
	}

	for i, v := range results {
		if !v.IsValid() {
			results[i] = reflect.Zero(fnType.Out(i))
		}
	}

	return
}


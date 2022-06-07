package elutils

import (
	"reflect"
	"fmt"
	"runtime"
	"strings"
)

type GolangFuncHelper struct {
	fnVal reflect.Value
	fnType reflect.Type
	realName string
}

func NewGolangFuncHelperDiretly(fnVal reflect.Value, fnType reflect.Type) (helper *GolangFuncHelper) {
	return &GolangFuncHelper{fnVal:fnVal, fnType:fnType}
}

func NewGolangFuncHelper(name string, funcVar interface{}) (helper *GolangFuncHelper, err error) {
	if funcVar == nil {
		err = fmt.Errorf("funcVar must be a non-nil value")
		return
	}
	t := reflect.TypeOf(funcVar)
	if t.Kind() != reflect.Func {
		err = fmt.Errorf("funcVar expected to be a func")
		return
	}

	if len(name) == 0 {
		n := runtime.FuncForPC(reflect.ValueOf(funcVar).Pointer()).Name()
		if pos := strings.LastIndex(n, "."); pos >= 0 {
			name = n[pos+1:]
		} else {
			name = n
		}

		if len(name) == 0 {
			name = "noname"
		}
	}

	helper = &GolangFuncHelper{
		fnVal: reflect.ValueOf(funcVar),
		fnType: t,
		realName: name,
	}
	return
}

type FnGetEmbeddingArg func(i int) interface{}

func (h *GolangFuncHelper) CallGolangFunc(embeddingFuncArgsNum int, embddingFuncName string, getArg FnGetEmbeddingArg) (val interface{}, err error) {
		argsNum := embeddingFuncArgsNum
		fnType := h.fnType

		variadic := fnType.IsVariadic()
		lastNumIn := fnType.NumIn() - 1
		if variadic {
			if argsNum < lastNumIn {
				err = fmt.Errorf("at least %d args to call %s", lastNumIn, embddingFuncName)
				return
			}
		} else {
			if argsNum != fnType.NumIn() {
				err = fmt.Errorf("%d args expected to call %s", argsNum, embddingFuncName)
				return
			}
		}

		// make golang func args
		goArgs := make([]reflect.Value, argsNum)
		var fnArgType reflect.Type
		for i:=0; i<argsNum; i++ {
			if i<lastNumIn || !variadic {
				fnArgType = fnType.In(i)
			} else {
				fnArgType = fnType.In(lastNumIn).Elem()
			}

			goArgs[i] = MakeValue(fnArgType)
			SetValue(goArgs[i], getArg(i))
		}

		// call golang func
		res := h.fnVal.Call(goArgs)

		// convert result to embedding
		retc := len(res)
		if retc == 0 {
			val = nil
			return
		}
		lastRetType := fnType.Out(retc-1)
		if lastRetType.Name() == "error" {
			e := res[retc-1].Interface()
			if e != nil {
				err = e.(error)
				return
			}
			retc -= 1
			if retc == 0 {
				val = nil
				return
			}
		}

		if retc == 1 {
			val = res[0].Interface()
			return
		}
		retV := make([]interface{}, retc)
		for i:=0; i<retc; i++ {
			retV[i] = res[i].Interface()
		}
		val = retV
		return
}

func (h *GolangFuncHelper) GetRealName() string {
	return h.realName
}


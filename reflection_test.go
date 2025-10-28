package gocrud

import (
	"reflect"
	"testing"
)

func TestReflection_StructToMap(t *testing.T) {
	t.Run("Test Struct To Map", func(t *testing.T) {
		type Test struct {
			Test  int
			Test2 string
			Test3 float32
			Test4 []byte
		}

		i := &Test{
			Test:  100,
			Test2: "test",
			Test3: 100.11,
			Test4: []byte{0},
		}

		want := map[string]any{
			"test":  &i.Test,
			"test2": &i.Test2,
			"test3": &i.Test3,
			"test4": &i.Test4,
		}

		r := &Reflection{}
		if got := r.StructToMap(i); !reflect.DeepEqual(got, want) {
			t.Errorf("Reflection.StructToMap() = %v, want %v", got, want)
		}
	})

	t.Run("Test Struct To Map - anonymous struct", func(t *testing.T) {
		i := &struct {
			Test  int
			Test2 string
			Test3 float32
			Test4 []byte
		}{
			Test:  100,
			Test2: "test",
			Test3: 100.11,
			Test4: []byte{0},
		}

		want := map[string]any{
			"test":  &i.Test,
			"test2": &i.Test2,
			"test3": &i.Test3,
			"test4": &i.Test4,
		}

		r := &Reflection{}
		if got := r.StructToMap(i); !reflect.DeepEqual(got, want) {
			t.Errorf("Reflection.StructToMap() = %v, want %v", got, want)
		}
	})
}

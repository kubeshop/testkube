// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubeshop/testkube/internal/common"
)

type testObj2 struct {
	Expr  string `expr:"expression"`
	Dummy string
}

type testObj struct {
	Expr            string                        `expr:"expression"`
	Tmpl            string                        `expr:"template"`
	ExprPtr         *string                       `expr:"expression"`
	TmplPtr         *string                       `expr:"template"`
	IntExpr         intstr.IntOrString            `expr:"expression"`
	IntTmpl         intstr.IntOrString            `expr:"template"`
	IntExprPtr      *intstr.IntOrString           `expr:"expression"`
	IntTmplPtr      *intstr.IntOrString           `expr:"template"`
	Obj             testObj2                      `expr:"include"`
	ObjPtr          *testObj2                     `expr:"include"`
	SliceExprStr    []string                      `expr:"expression"`
	SliceExprStrPtr *[]string                     `expr:"expression"`
	SliceExprObj    []testObj2                    `expr:"include"`
	MapKeyVal       map[string]string             `expr:"template,template"`
	MapValIntTmpl   map[string]intstr.IntOrString `expr:"template"`
	MapKeyTmpl      map[string]string             `expr:"template,"`
	MapValTmpl      map[string]string             `expr:"template"`
	MapTmplExpr     map[string]string             `expr:"template,expression"`
	Dummy           string
	DummyPtr        *string
	DummyObj        testObj2
	DummyObjPtr     *testObj2
}

type testObjNested struct {
	Value corev1.Volume `expr:"force"`
	Dummy corev1.Volume
}

var testMachine = NewMachine().
	Register("dummy", "test").
	Register("ten", 10)

func TestGenericString(t *testing.T) {
	obj := testObj{
		Expr:     "5 + 3 + ten",
		Tmpl:     "{{ 10 + 3 }}{{ ten }}",
		ExprPtr:  common.Ptr("1 + 2 + ten"),
		TmplPtr:  common.Ptr("{{ 4 + 3 }}{{ ten }}"),
		Dummy:    "5 + 3 + ten",
		DummyPtr: common.Ptr("5 + 3 + ten"),
	}
	err := Simplify(&obj, testMachine)
	assert.NoError(t, err)
	assert.Equal(t, "18", obj.Expr)
	assert.Equal(t, "1310", obj.Tmpl)
	assert.Equal(t, common.Ptr("13"), obj.ExprPtr)
	assert.Equal(t, common.Ptr("710"), obj.TmplPtr)
	assert.Equal(t, "5 + 3 + ten", obj.Dummy)
	assert.Equal(t, common.Ptr("5 + 3 + ten"), obj.DummyPtr)
}

func TestGenericIntOrString(t *testing.T) {
	obj := testObj{
		IntExpr:    intstr.IntOrString{Type: intstr.String, StrVal: "5 + 3 + ten"},
		IntTmpl:    intstr.IntOrString{Type: intstr.String, StrVal: "{{ 10 + 3 }}{{ ten }}"},
		IntExprPtr: &intstr.IntOrString{Type: intstr.String, StrVal: "1 + 2 + ten"},
		IntTmplPtr: &intstr.IntOrString{Type: intstr.String, StrVal: "{{ 4 + 3 }}{{ ten }}"},
	}
	err := Simplify(&obj, testMachine)
	assert.NoError(t, err)
	assert.Equal(t, "18", obj.IntExpr.String())
	assert.Equal(t, "1310", obj.IntTmpl.String())
	assert.Equal(t, "13", obj.IntExprPtr.String())
	assert.Equal(t, "710", obj.IntTmplPtr.String())
}

func TestGenericSlice(t *testing.T) {
	obj := testObj{
		SliceExprStr:    []string{"200 + 100", "100 + 200", "ten", "abc"},
		SliceExprStrPtr: &[]string{"200 + 100", "100 + 200", "ten", "abc"},
		SliceExprObj:    []testObj2{{Expr: "10 + 5", Dummy: "3 + 2"}},
	}
	err := Simplify(&obj, testMachine)
	assert.NoError(t, err)
	assert.Equal(t, []string{"300", "300", "10", "abc"}, obj.SliceExprStr)
	assert.Equal(t, &[]string{"300", "300", "10", "abc"}, obj.SliceExprStrPtr)
	assert.Equal(t, []testObj2{{Expr: "15", Dummy: "3 + 2"}}, obj.SliceExprObj)
}

func TestGenericMap(t *testing.T) {
	obj := testObj{
		MapKeyVal:     map[string]string{"{{ 10 + 3 }}2": "{{ 3 + 5 }}"},
		MapKeyTmpl:    map[string]string{"{{ 10 + 3 }}2": "{{ 3 + 5 }}"},
		MapValTmpl:    map[string]string{"{{ 10 + 3 }}2": "{{ 3 + 5 }}"},
		MapValIntTmpl: map[string]intstr.IntOrString{"{{ 10 + 3 }}2": {Type: intstr.String, StrVal: "{{ 3 + 5 }}"}},
		MapTmplExpr:   map[string]string{"{{ 10 + 3 }}2": "3 + 5"},
	}
	err := Simplify(&obj, testMachine)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"132": "8"}, obj.MapKeyVal)
	assert.Equal(t, map[string]string{"132": "{{ 3 + 5 }}"}, obj.MapKeyTmpl)
	assert.Equal(t, map[string]string{"{{ 10 + 3 }}2": "8"}, obj.MapValTmpl)
	assert.Equal(t, map[string]intstr.IntOrString{"{{ 10 + 3 }}2": {Type: intstr.String, StrVal: "8"}}, obj.MapValIntTmpl)
	assert.Equal(t, map[string]string{"132": "8"}, obj.MapTmplExpr)
}

func TestNestedObject(t *testing.T) {
	obj := testObj{
		Obj:         testObj2{Expr: "10 + 5", Dummy: "3 + 2"},
		ObjPtr:      &testObj2{Expr: "10 + 8", Dummy: "33 + 2"},
		DummyObj:    testObj2{Expr: "10 + 8", Dummy: "333 + 2"},
		DummyObjPtr: &testObj2{Expr: "10 + 8", Dummy: "3333 + 2"},
	}
	err := Simplify(&obj, testMachine)
	assert.NoError(t, err)
	assert.Equal(t, testObj2{Expr: "15", Dummy: "3 + 2"}, obj.Obj)
	assert.Equal(t, &testObj2{Expr: "18", Dummy: "33 + 2"}, obj.ObjPtr)
	assert.Equal(t, testObj2{Expr: "10 + 8", Dummy: "333 + 2"}, obj.DummyObj)
	assert.Equal(t, &testObj2{Expr: "10 + 8", Dummy: "3333 + 2"}, obj.DummyObjPtr)
}

func TestGenericNotMutateStringPointer(t *testing.T) {
	ptr := common.Ptr("200 + 10")
	obj := testObj{
		ExprPtr: ptr,
	}
	_ = Simplify(&obj, testMachine)
	assert.Equal(t, common.Ptr("200 + 10"), ptr)
}

func TestGenericCompileError(t *testing.T) {
	got := testObj{
		Tmpl: "{{ 1 + 2 }}{{ 3",
	}
	err := Simplify(&got)

	assert.Contains(t, fmt.Sprintf("%v", err), "Tmpl: template error")
}

func TestGenericForceSimplify(t *testing.T) {
	got := corev1.Volume{
		Name: "{{ 3 + 2 }}{{ 5 }}",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "{{ 4433 }}"},
			},
		},
	}
	err := SimplifyForce(&got)

	want := corev1.Volume{
		Name: "55",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "4433"},
			},
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGenericForceSimplifyNested(t *testing.T) {
	got := testObjNested{
		Value: corev1.Volume{
			Name: "{{ 3 + 2 }}{{ 5 }}",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "{{ 4433 }}"},
				},
			},
		},
		Dummy: corev1.Volume{
			Name: "{{ 3 + 2 }}{{ 5 }}",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "{{ 4433 }}"},
				},
			},
		},
	}
	err := Simplify(&got)

	want := testObjNested{
		Value: corev1.Volume{
			Name: "55",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "4433"},
				},
			},
		},
		Dummy: corev1.Volume{
			Name: "{{ 3 + 2 }}{{ 5 }}",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "{{ 4433 }}"},
				},
			},
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

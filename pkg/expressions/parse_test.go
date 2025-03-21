package expressions

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCompileBasic(t *testing.T) {
	assert.Equal(t, "value", must(MustCompile(`"value"`).Static().StringValue()))
}

func TestCompileTernary(t *testing.T) {
	assert.Equal(t, "value", must(MustCompile(`true ? "value" : "another"`).Static().StringValue()))
	assert.Equal(t, "another", must(MustCompile(`false ? "value" : "another"`).Static().StringValue()))
	assert.Equal(t, "another", must(MustCompile(`5 == 3 ? "value" : "another"`).Static().StringValue()))
	assert.Equal(t, "another", must(MustCompile(`5 == 3 && 2 == 4 ? "value" : "another"`).Static().StringValue()))
	assert.Equal(t, "another", must(MustCompile(`5 == 3 || 2 == 4 ? "value" : "another"`).Static().StringValue()))
	assert.Equal(t, "value", must(MustCompile(`3 == 3 || 2 == 4 ? "value" : "another"`).Static().StringValue()))
	assert.Equal(t, "xyz", must(MustCompile(`false ? "value" : true ? "xyz" :"another"`).Static().StringValue()))
	assert.Equal(t, "xyz", must(MustCompile(`false ? "value" : (true ? "xyz" :"another")`).Static().StringValue()))
	assert.Equal(t, 5.78, must(MustCompile(`false ? 3 : (true ? 5.78 : 2)`).Static().FloatValue()))
}

func TestCompileMath(t *testing.T) {
	assert.Equal(t, 5.0, must(MustCompile(`2 + 3`).Static().FloatValue()))
	assert.Equal(t, 0.6, must(MustCompile(`3 / 5`).Static().FloatValue()))
	assert.Equal(t, true, must(MustCompile(`3 <> 5`).Static().BoolValue()))
	assert.Equal(t, true, must(MustCompile(`3 != 5`).Static().BoolValue()))
	assert.Equal(t, false, must(MustCompile(`3 == 5`).Static().BoolValue()))
	assert.Equal(t, false, must(MustCompile(`3 = 5`).Static().BoolValue()))
	assert.Equal(t, `"3/5:"+(a+b)+"/"+5`, MustCompile(`3 + "/" + 5 + ":" + (a + b) + "/" + 5`).String())
	assert.Equal(t, `(a+"/")+b+":"+(a+1)+"/"+b`, MustCompile(`a + "/" + b + ":" + (a + 1) + "/" + b`).String())
}

func TestCompileLogical(t *testing.T) {
	assert.Equal(t, "true", MustCompile(`!(false && r1)`).String())
	assert.Equal(t, "false", MustCompile(`!true && r1`).String())
	assert.Equal(t, "r1", MustCompile(`true && r1`).String())
	assert.Equal(t, "r1", MustCompile(`!true || r1`).String())
	assert.Equal(t, "true", MustCompile(`true || r1`).String())
	assert.Equal(t, "11", MustCompile(`5 - -3 * 2`).String())
	assert.Equal(t, "r1&&false", MustCompile(`r1 && false`).String())
	assert.Equal(t, "bool(r1)", MustCompile(`bool(r1) && true`).String())
	assert.Equal(t, "false", MustCompile(`bool(r1) && false`).String())
	assert.Equal(t, "r1||false", MustCompile(`r1 || false`).String())
	assert.Equal(t, "bool(r1)", MustCompile(`bool(r1) || false`).String())
	assert.Equal(t, "r1||true", MustCompile(`r1 || true`).String())
	assert.Equal(t, "true", MustCompile(`bool(r1) || true`).String())
}

func TestCompileMathOperationsPrecedence(t *testing.T) {
	assert.Equal(t, 7.0, must(MustCompile(`1 + 2 * 3`).Static().FloatValue()))
	assert.Equal(t, 11.0, must(MustCompile(`1 + (2 * 3) + 4`).Static().FloatValue()))
	assert.Equal(t, 11.0, must(MustCompile(`1 + 2 * 3 + 4`).Static().FloatValue()))
	assert.Equal(t, 30.0, must(MustCompile(`1 + 2 * 3 * 4 + 5`).Static().FloatValue()))
	assert.Equal(t, true, must(MustCompile(`1 + 2 * 3 * 4 + 5 <> 3`).Static().BoolValue()))

	assert.Equal(t, false, must(MustCompile(`1 + 2 * 3 * 4 + 5 == 3`).Static().BoolValue()))
	assert.Equal(t, true, must(MustCompile(`1 + 2 * 3 * 4 + 5 = 30`).Static().BoolValue()))
	assert.Equal(t, false, must(MustCompile(`1 + 2 * 3 * 4 + 5 <> 30`).Static().BoolValue()))
	assert.Equal(t, false, must(MustCompile(`1 + 2 * 3 * 4 + 5 <> 20 + 10`).Static().BoolValue()))
	assert.Equal(t, true, must(MustCompile(`1 + 2 * 3 * 4 + 5 = 20 + 10`).Static().BoolValue()))
	assert.Equal(t, false, must(MustCompile(`1 + 2 * 3 * 4 + 5 <> 20 + 10`).Static().BoolValue()))
	assert.Equal(t, true, must(MustCompile(`1 + 2 * 3 * 4 + 5 = 2 + 3 * 6 + 10`).Static().BoolValue()))
	assert.Equal(t, false, must(MustCompile(`1 + 2 * 3 * 4 + 5 <> 2 + 3 * 6 + 10`).Static().BoolValue()))
	assert.Equal(t, 8.0, must(MustCompile(`5 + 3 / 3 * 3`).Static().FloatValue()))
	assert.Equal(t, true, must(MustCompile(`5 + 3 / 3 * 3 = 8`).Static().BoolValue()))
	assert.Equal(t, 8.0, must(MustCompile(`5 + 3 * 3 / 3`).Static().FloatValue()))
	assert.Equal(t, true, must(MustCompile(`5 + 3 * 3 / 3 = 8`).Static().BoolValue()))
	assert.Equal(t, true, must(MustCompile(`5 + 3 * 3 / 3 = 2 + 3 * 2`).Static().BoolValue()))
	assert.Equal(t, false, must(MustCompile(`5 + 3 * 3 / 3 = 3 + 3 * 2`).Static().BoolValue()))

	assert.Equal(t, false, must(MustCompile(`true && false || false && true`).Static().BoolValue()))
	assert.Equal(t, true, must(MustCompile(`true && false || true`).Static().BoolValue()))
	assert.Equal(t, int64(0), must(MustCompile(`1 && 0 && 2`).Static().IntValue()))
	assert.Equal(t, int64(2), must(MustCompile(`1 && 0 || 2`).Static().IntValue()))
	assert.Equal(t, int64(1), must(MustCompile(`1 || 0 || 2`).Static().IntValue()))

	assert.Equal(t, true, must(MustCompile(`10 > 2 && 5 <= 5`).Static().BoolValue()))
	assert.Equal(t, false, must(MustCompile(`10 > 2 && 5 < 5`).Static().BoolValue()))
	assert.Error(t, errOnly(Compile(`10 > 2 > 3`)))

	assert.Equal(t, 817.0, must(MustCompile(`1 + 2 * 3 ** 4 * 5 + 6`).Static().FloatValue()))
	assert.Equal(t, 4.5, must(MustCompile(`72 / 2 ** 4`).Static().FloatValue()))
	assert.InDelta(t, 3.6, must(MustCompile(`3 * 5.2 % 4`).Static().FloatValue()), 0.00001)

	assert.Equal(t, true, must(MustCompile(`!0 && 500`).Static().BoolValue()))
	assert.Equal(t, false, must(MustCompile(`!5 && 500`).Static().BoolValue()))

	assert.Equal(t, "(A+B*(C+D)/E*F)+G<>H**I*J**K", MustCompile(`A + B * (C + D) / E * F + G <> H ** I * J ** K`).String())
}

func TestBuildTemplate(t *testing.T) {
	assert.Equal(t, "abc", MustCompile(`"abc"`).Template())
	assert.Equal(t, "abcdef", MustCompile(`"abc" + "def"`).Template())
	assert.Equal(t, "abc9", MustCompile(`"abc" + 9`).Template())
	assert.Equal(t, "abc{{env.xyz}}", MustCompile(`"abc" + env.xyz`).Template())
	assert.Equal(t, "{{env.xyz}}abc", MustCompile(`env.xyz + "abc"`).Template())
	assert.Equal(t, "{{env.xyz+env.abc}}abc", MustCompile(`env.xyz + env.abc + "abc"`).Template())
	assert.Equal(t, "{{env.xyz+env.abc}}abc", MustCompile(`env.xyz + env.abc + "abc"`).Template())
	assert.Equal(t, "{{3+env.xyz+env.abc}}", MustCompile(`3 + env.xyz + env.abc`).Template())
	assert.Equal(t, "3{{env.xyz}}{{env.abc}}", MustCompile(`string(3) + env.xyz + env.abc`).Template())
	assert.Equal(t, "3{{env.xyz+env.abc}}", MustCompile(`string(3) + (env.xyz + env.abc)`).Template())
	assert.Equal(t, "3{{env.xyz}}{{env.abc}}", MustCompile(`"3" + env.xyz + env.abc`).Template())
	assert.Equal(t, "3{{env.xyz+env.abc}}", MustCompile(`"3" + (env.xyz + env.abc)`).Template())
}

func TestCompileTemplate(t *testing.T) {
	assert.Equal(t, `""`, MustCompileTemplate(``).String())
	assert.Equal(t, `"abc"`, MustCompileTemplate(`abc`).String())
	assert.Equal(t, `"abcxyz5"`, MustCompileTemplate(`abc{{ "xyz" }}{{ 5 }}`).String())
	assert.Equal(t, `"abc50"`, MustCompileTemplate(`abc{{ 5 + 45 }}`).String())
	assert.Equal(t, `"abc50def"`, MustCompileTemplate(`abc{{ 5 + 45 }}def`).String())
	assert.Equal(t, `"abc50def"+string(env.abc*5)+"20"`, MustCompileTemplate(`abc{{ 5 + 45 }}def{{env.abc * 5}}20`).String())

	assert.Equal(t, `abc50def`, must(MustCompileTemplate(`abc{{ 5 + 45 }}def`).Static().StringValue()))
}

func TestCompilePartialResolution(t *testing.T) {
	vm := NewMachine().
		Register("someint", 555).
		Register("somestring", "foo").
		RegisterAccessor(func(name string) (interface{}, bool) {
			if strings.HasPrefix(name, "env.") {
				return "[placeholder:" + name[4:] + "]", true
			}
			return nil, false
		}).
		RegisterAccessor(func(name string) (interface{}, bool) {
			if strings.HasPrefix(name, "secrets.") {
				return MustCompile("secret(" + name[8:] + ")"), true
			}
			return nil, false
		}).
		RegisterFunction("mainEndpoint", func(values ...StaticValue) (interface{}, bool, error) {
			if len(values) != 0 {
				return nil, true, errors.New("the mainEndpoint should have no parameters")
			}
			return MustCompile(`env.apiUrl`), true, nil
		})

	assert.Equal(t, `555`, must(MustCompile(`someint`).Resolve(vm)).String())
	assert.Equal(t, `"[placeholder:name]"`, must(MustCompile(`env.name`).Resolve(vm)).String())
	assert.Equal(t, `secret(name)`, must(MustCompile(`secrets.name`).Resolve(vm)).String())
	assert.Equal(t, `"[placeholder:apiUrl]"`, must(MustCompile(`mainEndpoint()`).Resolve(vm)).String())
}

func TestCompileResolution(t *testing.T) {
	vm := NewMachine().
		Register("someint", 555).
		Register("somestring", "foo").
		RegisterAccessor(func(name string) (interface{}, bool) {
			if strings.HasPrefix(name, "env.") {
				return "[placeholder:" + name[4:] + "]", true
			}
			return nil, false
		}).
		RegisterAccessor(func(name string) (interface{}, bool) {
			if strings.HasPrefix(name, "secrets.") {
				return MustCompile("secret(" + name[8:] + ")"), true
			}
			return nil, false
		}).
		RegisterFunction("mainEndpoint", func(values ...StaticValue) (interface{}, bool, error) {
			if len(values) != 0 {
				return nil, true, errors.New("the mainEndpoint should have no parameters")
			}
			return MustCompile(`env.apiUrl`), true, nil
		})

	assert.Equal(t, `555`, must(MustCompile(`someint`).Resolve(vm, FinalizerFail)).String())
	assert.Equal(t, `"[placeholder:name]"`, must(MustCompile(`env.name`).Resolve(vm, FinalizerFail)).String())
	assert.Error(t, errOnly(MustCompile(`secrets.name`).Resolve(vm, FinalizerFail)))
	assert.Equal(t, `"[placeholder:apiUrl]"`, must(MustCompile(`mainEndpoint()`).Resolve(vm, FinalizerFail)).String())
}

func TestCircularResolution(t *testing.T) {
	vm := NewMachine().
		RegisterFunction("one", func(values ...StaticValue) (interface{}, bool, error) {
			return MustCompile("two()"), true, nil
		}).
		RegisterFunction("two", func(values ...StaticValue) (interface{}, bool, error) {
			return MustCompile("one()"), true, nil
		}).
		RegisterFunction("self", func(values ...StaticValue) (interface{}, bool, error) {
			return MustCompile("self()"), true, nil
		})

	assert.Contains(t, fmt.Sprintf("%v", errOnly(MustCompile(`one()`).Resolve(vm, FinalizerFail))), "call stack exceeded")
	assert.Contains(t, fmt.Sprintf("%v", errOnly(MustCompile(`self()`).Resolve(vm, FinalizerFail))), "call stack exceeded")
}

func TestMinusNumber(t *testing.T) {
	assert.Equal(t, -4.0, must(MustCompile("-4").Static().FloatValue()))
}

func TestCompileMultilineString(t *testing.T) {
	assert.Equal(t, `"\nabc\ndef\n"`, MustCompile(`"
abc
def
"`).String())
}

func TestCompileEscapeTemplate(t *testing.T) {
	assert.Equal(t, `foo{{"{{"}}barbaz{{"{{"}}`, MustCompileTemplate(`foo{{"{{bar"}}baz{{"{{"}}`).Template())
}

func TestCompileEscapeBrace(t *testing.T) {
	assert.Equal(t, `foo{{"{"}}barbaz{{"{"}}`, MustCompileTemplate(`foo{{"{bar"}}baz{{"{"}}`).Template())
}

func TestCompileStandardLib(t *testing.T) {
	assert.Equal(t, `false`, MustCompile(`bool(0)`).String())
	assert.Equal(t, `true`, MustCompile(`bool(500)`).String())
	assert.Equal(t, `"500"`, MustCompile(`string(500)`).String())
	assert.Equal(t, `500`, MustCompile(`int(500)`).String())
	assert.Equal(t, `500`, MustCompile(`int(500.888)`).String())
	assert.Equal(t, `500`, MustCompile(`int("500")`).String())
	assert.Equal(t, `500.44`, MustCompile(`float("500.44")`).String())
	assert.Equal(t, `500`, MustCompile(`json("500")`).String())
	assert.Equal(t, `{"a":500}`, MustCompile(`json("{\"a\": 500}")`).String())
	assert.Equal(t, `"{\"a\":500}"`, MustCompile(`tojson({"a": 500})`).String())
	assert.Equal(t, `"500.8"`, MustCompile(`tojson(500.8)`).String())
	assert.Equal(t, `"\"500.8\""`, MustCompile(`tojson("500.8")`).String())
	assert.Equal(t, `"abc"`, MustCompile(`shellquote("abc")`).String())
	assert.Equal(t, `"'a b c'"`, MustCompile(`shellquote("a b c")`).String())
	assert.Equal(t, `"'a b c' 'd e f'"`, MustCompile(`shellquote("a b c", "d e f")`).String())
	assert.Equal(t, `"''"`, MustCompile(`shellquote(null)`).String())
	assert.Equal(t, `["a","b","c","a b c"]`, MustCompile(`shellparse("a b c 'a b c'")`).String())
	assert.Equal(t, `"abc  d"`, MustCompile(`trim("   abc  d  \n  ")`).String())
	assert.Equal(t, `"abc"`, MustCompile(`yaml("\"abc\"")`).String())
	assert.Equal(t, `{"foo":{"bar":"baz"}}`, MustCompile(`yaml("foo:\n  bar: 'baz'")`).String())
	assert.Equal(t, `"foo:\n    bar: baz\n"`, MustCompile(`toyaml({"foo":{"bar":"baz"}})`).String())
	assert.Equal(t, `{"a":["b","v"]}`, MustCompile(`yaml("
a:
- b
- v
")`).String())
	assert.Equal(t, `["a",10,["a",4]]`, MustCompile(`list("a", 10, ["a", 4])`).String())
	assert.Equal(t, `"a,10,a,4"`, MustCompile(`join(["a",10,["a",4]])`).String())
	assert.Equal(t, `"a---10---a,4"`, MustCompile(`join(["a",10,["a",4]], "---")`).String())
	assert.Equal(t, `[""]`, MustCompile(`split(null)`).String())
	assert.Equal(t, `["a","b","c"]`, MustCompile(`split("a,b,c")`).String())
	assert.Equal(t, `["a","b","c"]`, MustCompile(`split("a---b---c", "---")`).String())
	assert.Equal(t, `5`, MustCompile(`len("abcde")`).String())
	assert.Equal(t, `2`, MustCompile(`len(["a", "b"])`).String())
	assert.Equal(t, `2`, MustCompile(`len({"a": "b", "b": "c"})`).String())
	assert.Equal(t, `2`, MustCompile(`floor(2.6)`).String())
	assert.Equal(t, `2`, MustCompile(`ceil(1.6)`).String())
	assert.Equal(t, `2`, MustCompile(`round(1.6)`).String())
	assert.Equal(t, `2`, MustCompile(`round(1.5)`).String())
	assert.Equal(t, `1`, MustCompile(`round(1.4)`).String())
	assert.Equal(t, `[[1,2],[3,4],[5]]`, MustCompile(`chunk([1,2,3,4,5], 2)`).String())
	assert.Equal(t, `[2,4,6,8,10]`, MustCompile(`map([1,2,3,4,5], "_.value * 2")`).String())
	assert.Equal(t, `[0,2,4,6,8]`, MustCompile(`map([10,20,30,40,50], "_.index * 2")`).String())
	assert.Equal(t, `[2,4,6,8,10]`, MustCompile(`map([1,2,3,4,5], "_.value * 2")`).String())
	assert.Equal(t, `[0,2,4,6,8]`, MustCompile(`map([10,20,30,40,50], "_.index * 2")`).String())
	assert.ElementsMatch(t, []interface{}{MapEntry{Key: "A", Value: "B"}, MapEntry{Key: "C", Value: 5.0}}, must(MustCompile(`entries({"A": "B", "C": 5})`).Static().SliceValue()))
	assert.Equal(t, `[3,4,5]`, MustCompile(`filter([1,2,3,4,5], "_.value > 2")`).String())
	assert.Equal(t, `[5]`, MustCompile(`jq([1,2,3,4,5], ". | max")`).String())
	assert.Equal(t, `[{"b":{"v":2}}]`, MustCompile(`jq([{"a":{"v": 1}},{"b":{"v": 2}}], ". | max_by(.v)")`).String())
	assert.Equal(t, `[[3,4,5]]`, MustCompile(`jq([1,2,3,4,5], "map(select(. > 2))")`).String())
	assert.Equal(t, `5`, MustCompile(`at([1,2,3,4,5], 4)`).String())
	assert.Equal(t, `"value"`, MustCompile(`at({"x": "value"}, "x")`).String())
	assert.Equal(t, `null`, MustCompile(`at({"x": "value"}, "unknown-key")`).String())
	assert.Equal(t, `"abc"`, MustCompile(`eval("\"abc\"")`).String())
	assert.Equal(t, `50`, MustCompile(`eval("5 * 10")`).String())
	assert.Equal(t, `50*something`, MustCompile(`eval("5 * 10 * something")`).String())
	assert.Equal(t, `"abc"`, MustCompile(`relpath("xyz/abc", "xyz")`).String())
	assert.Equal(t, `"abc"`, MustCompile(`relpath("/xyz/../xyz/abc", "/xyz")`).String())
	assert.Equal(t, `"."`, MustCompile(`relpath("/xyz", "/xyz")`).String())
	assert.Equal(t, `"../abc"`, MustCompile(`relpath("/abc", "/xyz")`).String())
	assert.Equal(t, `"abc/def"`, MustCompile(`relpath("/data/abc/def", "/data")`).String())
	assert.Equal(t, `".."`, MustCompile(`relpath("/data/abc/def", "/data/abc/def/xyz")`).String())
	assert.Equal(t, `"../ccc"`, MustCompile(`relpath("/data/abc/def/ccc", "/data/abc/def/xyz")`).String())
	assert.Equal(t, `"/data/abc/def/ccc"`, MustCompile(`abspath("/data/abc/def/ccc", "/data")`).String())
	assert.Equal(t, `"/data/abc/def/ccc"`, MustCompile(`abspath("/data/abc/def/ccc", "/abc")`).String())
	assert.Equal(t, `"/data/abc/def/ccc"`, MustCompile(`abspath("def/ccc", "/data/abc")`).String())
	assert.Equal(t, `"/data"`, MustCompile(`abspath("..", "/data/abc")`).String())
	assert.Equal(t, `[]`, MustCompile(`range(0, 0)`).String())
	assert.Equal(t, `[]`, MustCompile(`range(0)`).String())
	assert.Equal(t, `[]`, MustCompile(`range(0, -3)`).String())
	assert.Equal(t, `[]`, MustCompile(`range(5, 3)`).String())
	assert.Equal(t, `[0,1,2,3,4]`, MustCompile(`range(5)`).String())
	assert.Equal(t, `[5,6,7]`, MustCompile(`range(5, 8)`).String())
	assert.Equal(t, `10`, MustCompile(`any(10, 20, 30, 10)`).String())
	assert.Equal(t, `10`, MustCompile(`any(10, unknownVariable, 30, 10)`).String())
	assert.Equal(t, `30`, MustCompile(`any(unknownVariable, 30, 10)`).String())
	assert.Equal(t, `30`, MustCompile(`any(unknownCall(), 30, 10)`).String())
	assert.Equal(t, `30`, MustCompile(`any(unknownCall(unknownVariable), 30, 10)`).String())
	assert.Equal(t, `30`, MustCompile(`any(30, unknownCall(), 10)`).String())
	assert.Equal(t, `30`, MustCompile(`any(30, unknownCall(unknownVariable), 10)`).String())
	assert.Equal(t, `20`, MustCompile(`any([20, 44, 30]..., 10)`).String())
	assert.Equal(t, `20`, MustCompile(`any(unknown, [20, 44, 30]..., 10)`).String())
	assert.Equal(t, `null`, MustCompile(`any()`).String())
	assert.InDelta(t, time.Now().UnixMilli(), must(time.Parse(RFC3339Millis, must(MustCompile(`date()`).Static().StringValue()))).UnixMilli(), 5)
	assert.Equal(t, time.Now().Truncate(24*time.Hour).UnixMilli(), must(time.Parse("2006-01-02", must(MustCompile(`date("2006-01-02")`).Static().StringValue()))).UnixMilli())
}

func TestCompileWildcard_Unknown(t *testing.T) {
	assert.Equal(t, `map(a.b.c,"_.value.d.e")`, MustCompile("a.b.c.*.d.e").String())
	assert.Equal(t, `map(map(a.b.c,"_.value"),"_.value.d.e")`, MustCompile("a.b.c.*.*.d.e").String())
}

func TestCompileSpread(t *testing.T) {
	assert.Equal(t, `"a b c 'a b c'"`, MustCompile(`shellquote(["a", "b", "c", "a b c"]...)`).String())
	assert.Equal(t, `"a b c 'a b c'"`, MustCompile("shellquote(shellparse(\"a b c\n'a b c'\")...)").String())
	assert.Equal(t, `"axb"`, MustCompile(`join([["a", "b"], "x"]...)`).String())
}

func TestCompileWildcard_Map(t *testing.T) {
	vm := NewMachine().Register("a.b.c", []map[string]interface{}{
		{"d": map[string]string{"e": "v1"}},
		{"d": map[string]string{"e": "v2"}},
	})
	assert.Equal(t, `["v1","v2"]`, must(MustCompile("a.b.c.*.d.e").Resolve(vm)).String())
}

func TestCompileWildcard_Struct(t *testing.T) {
	type S1 struct {
		Else string `json:"e"`
	}
	type S2 struct {
		Something S1 `json:"d"`
	}
	vm := NewMachine().Register("a.b.c", []S2{
		{Something: S1{Else: "v1"}},
		{Something: S1{Else: "v2"}},
	})
	assert.Equal(t, `["v1","v2"]`, must(MustCompile("a.b.c.*.d.e").Resolve(vm)).String())
}

func TestCompileWildcard_Inner(t *testing.T) {
	type S1 struct {
		Else string `json:"e"`
	}
	type S2 struct {
		Something S1 `json:"d"`
	}
	vm := NewMachine().Register("a.b", map[string]interface{}{
		"c": []S2{
			{Something: S1{Else: "v1"}},
			{Something: S1{Else: "v2"}},
		},
	})
	assert.Equal(t, `["v1","v2"]`, must(MustCompile("a.b.c.*.d.e").Resolve(vm)).String())
}

func TestCompileInnerPath(t *testing.T) {
	assert.Equal(t, `"v1"`, MustCompile(`["v1", "v2"].0`).String())
	assert.Equal(t, `"v1abc"`, must(MustCompile(`map(["v1", "v2"], "_.value + \"abc\"").0`).Resolve()).String())
	assert.Equal(t, `"v"`, must(MustCompile(`{"k": "v", "k2":"v2"}.k`).Resolve()).String())
	assert.Equal(t, `"v"`, must(MustCompile(`{"k": {"a": "v"}, "k2":"v2"}.k.a`).Resolve()).String())
}

func TestCompileDetectAccessors(t *testing.T) {
	assert.Equal(t, map[string]struct{}{"something": {}}, MustCompile(`something`).Accessors())
	assert.Equal(t, map[string]struct{}{"something": {}, "other": {}, "another": {}}, MustCompile(`calling(something, 5 * (other + 3), !another)`).Accessors())
}

func TestCompileDetectFunctions(t *testing.T) {
	assert.Equal(t, map[string]struct{}(nil), MustCompile(`something`).Functions())
	assert.Equal(t, map[string]struct{}{"calling": {}, "something": {}, "string": {}, "a": {}}, MustCompile(`calling(something(), 45 + 2 + 10 + string(abc * a(c)))`).Functions())
}

func TestCompileImmutableNone(t *testing.T) {
	assert.Same(t, None, NewValue(noneValue))
	assert.Same(t, NewValue(noneValue), NewValue(noneValue))
}

func TestCompileEscapedTemplate(t *testing.T) {
	input := `{{"{{"}}- with secret "internal/data/database/config" -}}{{"{{"}} .Data.data.username }}@{{"{{"}} .Data.data.password }}{{"{{"}}- end -}}`
	output := `{{- with secret "internal/data/database/config" -}}{{ .Data.data.username }}@{{ .Data.data.password }}{{- end -}}`

	assert.Equal(t, input, MustCompileTemplate(input).Template())
	assert.Equal(t, output, must(MustCompileTemplate(input).Static().StringValue()))
}

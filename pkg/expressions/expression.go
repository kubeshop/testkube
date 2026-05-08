package expressions

//go:generate go tool mockgen -destination=./mock_expression.go -package=expressions "github.com/kubeshop/testkube/pkg/expressions" Expression
type Expression interface {
	String() string
	SafeString() string
	Template() string
	Type() Type
	SafeResolve(...Machine) (Expression, bool, error)
	Resolve(...Machine) (Expression, error)
	Static() StaticValue
	Accessors() map[string]struct{}
	Functions() map[string]struct{}
}

type Type string

const (
	TypeUnknown Type = ""
	TypeBool    Type = "bool"
	TypeString  Type = "string"
	TypeFloat64 Type = "float64"
	TypeInt64   Type = "int64"
)

//go:generate go tool mockgen -destination=./mock_staticvalue.go -package=expressions "github.com/kubeshop/testkube/pkg/expressions" StaticValue
type StaticValue interface {
	Expression
	IsNone() bool
	IsString() bool
	IsBool() bool
	IsInt() bool
	IsNumber() bool
	IsMap() bool
	IsSlice() bool
	Value() interface{}
	BoolValue() (bool, error)
	IntValue() (int64, error)
	FloatValue() (float64, error)
	StringValue() (string, error)
	MapValue() (map[string]interface{}, error)
	SliceValue() ([]interface{}, error)
}

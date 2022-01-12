package storage

import (
	"errors"
	"fmt"
	"reflect"

	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
)

type testStepCodec struct{}

func (testStepCodec) DecodeValue(dctx bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Kind() != reflect.Interface {
		return errors.New("bad type or not settable")
	}

	fmt.Printf("DECODED VALUE: %+v\n", val.Elem())

	return nil
}

func (testStepCodec) EncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	fmt.Printf("ENCODED VALUE: %+v\n", val.Elem())

	return nil
}

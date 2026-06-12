package mongo

import (
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var uuidType = reflect.TypeOf(uuid.UUID{})

func uuidEncodeValue(_ bson.EncodeContext, vw bson.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != uuidType {
		return bson.ValueEncoderError{
			Name:     "uuidEncodeValue",
			Types:    []reflect.Type{uuidType},
			Received: val,
		}
	}
	u := val.Interface().(uuid.UUID)
	return vw.WriteBinaryWithSubtype(u[:], bson.TypeBinaryUUID)
}

func uuidDecodeValue(_ bson.DecodeContext, vr bson.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != uuidType {
		return bson.ValueDecoderError{
			Name:     "uuidDecodeValue",
			Types:    []reflect.Type{uuidType},
			Received: val,
		}
	}

	data, subtype, err := vr.ReadBinary()
	if err != nil {
		return err
	}

	if subtype != bson.TypeBinaryUUID && subtype != bson.TypeBinaryUUIDOld {
		return fmt.Errorf("cannot decode binary subtype %v as UUID", subtype)
	}

	u, err := uuid.FromBytes(data)
	if err != nil {
		return err
	}

	val.Set(reflect.ValueOf(u))
	return nil
}

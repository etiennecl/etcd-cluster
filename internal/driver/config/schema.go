package config

import (
	"bytes"
	_ "embed"
	"io"

	"github.com/clinia/x/logrusx"
	"github.com/clinia/x/otelx"

	"github.com/pkg/errors"
	"github.com/segmentio/ksuid"
	"github.com/tidwall/gjson"
)

//go:embed config.schema.json
var ConfigSchema []byte

var ConfigSchemaID string

func init() {
	ConfigSchemaID = gjson.GetBytes(ConfigSchema, "$id").String()
	if ConfigSchemaID == "" {
		ConfigSchemaID = ksuid.New().String() + ".json"
	}

}

// AddConfigSchema should be used instead of the schema itself to auto-register the dependencies schemas.
func AddConfigSchema(compiler interface {
	AddResource(url string, r io.Reader) error
}) error {
	if err := otelx.AddConfigSchema(compiler); err != nil {
		return err
	}
	if err := logrusx.AddConfigSchema(compiler); err != nil {
		return err
	}

	return errors.WithStack(compiler.AddResource(ConfigSchemaID, bytes.NewReader(ConfigSchema)))
}

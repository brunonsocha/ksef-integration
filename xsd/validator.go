package xsdinvoices

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
)

//go:embed *.xsd
var schemaFiles embed.FS

type Validator struct {
	schema *xsd.Schema
}

func New(ctx context.Context) (*Validator, error) {
	schemaBytes, err := fs.ReadFile(schemaFiles, "xsd_schema.xsd")
	if err != nil {
		return nil, fmt.Errorf("could not read XSD: %w", err)
	}
	doc, err := helium.NewParser().Parse(ctx, schemaBytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse XSD: %w", err)
	}
	schema, err := xsd.NewCompiler().FS(schemaFiles).BaseDir(".").Label("xsd_schema.xsd").Compile(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("could not compile XSD: %w", err)
	}
	return &Validator{schema: schema}, nil
}

func (v *Validator) Validate(ctx context.Context, xmlBytes []byte) error {
	doc, err := helium.NewParser().Parse(ctx, xmlBytes)
	if err != nil {
		return fmt.Errorf("could not parse XML: %w", err)
	}
	if err := xsd.NewValidator(v.schema).Validate(ctx, doc); err != nil {
		return fmt.Errorf("could not validate XML: %w", err)
	}
	return nil
}

package postgres

import (
	"fmt"
	"github.com/cortezaproject/corteza-server/pkg/dal"
	"github.com/cortezaproject/corteza-server/store/adapters/rdbms/ddl"
	"github.com/cortezaproject/corteza-server/store/adapters/rdbms/drivers"
	"github.com/cortezaproject/corteza-server/store/adapters/rdbms/ql"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/dialect/postgres"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/spf13/cast"
	"strings"
)

type (
	postgresDialect struct{}
)

var (
	_ drivers.Dialect = &postgresDialect{}

	dialect            = &postgresDialect{}
	goquDialectWrapper = goqu.Dialect("postgres")
	quoteIdent         = string(postgres.DialectOptions().QuoteRune)
)

func Dialect() *postgresDialect {
	return dialect
}

func (postgresDialect) GOQU() goqu.DialectWrapper  { return goquDialectWrapper }
func (postgresDialect) QuoteIdent(i string) string { return quoteIdent + i + quoteIdent }

func (d postgresDialect) IndexFieldModifiers(attr *dal.Attribute, mm ...dal.IndexFieldModifier) (string, error) {
	return drivers.IndexFieldModifiers(attr, d.QuoteIdent, mm...)
}

func (postgresDialect) DeepIdentJSON(ident exp.IdentifierExpression, pp ...any) (exp.LiteralExpression, error) {
	return drivers.DeepIdentJSON(ident, pp...), nil
}

func (d postgresDialect) TableCodec(m *dal.Model) drivers.TableCodec {
	return drivers.NewTableCodec(m, d)
}

func (d postgresDialect) TypeWrap(dt dal.Type) drivers.Type {
	// Any exception to general type-wrap implementation in the drivers package
	// should be placed here
	switch c := dt.(type) {
	case *dal.TypeTime:
		return &TypeTime{c}
	}

	return drivers.TypeWrap(dt)
}

func (postgresDialect) AttributeCast(attr *dal.Attribute, val exp.LiteralExpression) (exp.LiteralExpression, error) {
	var (
		c exp.CastExpression
	)

	switch attr.Type.(type) {
	case *dal.TypeBoolean:
		// we need to be strictly dealing with strings here!
		// 1) postgresql's JSON op ->> (last one) returns any JSON value as string
		//    so booleans are cast to 'true' & 'false'
		// 2) postgresql will complain about true == 'true' expressions
		ce := exp.NewCaseExpression().
			When(val.In(exp.NewLiteralExpression(`'true'`)), drivers.LiteralTRUE).
			When(val.In(exp.NewLiteralExpression(`'false'`)), drivers.LiteralFALSE).
			Else(drivers.LiteralNULL)

		c = exp.NewCastExpression(ce, "BOOLEAN")

	default:
		return drivers.AttributeCast(attr, val)

	}

	return exp.NewLiteralExpression("?", c), nil
}

func (postgresDialect) AttributeToColumn(attr *dal.Attribute) (col *ddl.Column, err error) {
	col = &ddl.Column{
		Ident:   attr.StoreIdent(),
		Comment: attr.Label,
		Type: &ddl.ColumnType{
			Null: attr.Type.IsNullable(),
		},
	}

	switch t := attr.Type.(type) {
	case *dal.TypeID:
		col.Type.Name = "BIGINT"
		col.Default = ddl.DefaultID(t.HasDefault, t.DefaultValue)
	case *dal.TypeRef:
		col.Type.Name = "BIGINT"
		col.Default = ddl.DefaultID(t.HasDefault, t.DefaultValue)

	case *dal.TypeTimestamp:
		col.Type.Name = "TIMESTAMP"

		if t.Timezone {
			col.Type.Name += "TZ"
		}

		if t.Precision >= 0 {
			col.Type.Name += fmt.Sprintf("(%d)", t.Precision)
		}

		col.Default = ddl.DefaultValueCurrentTimestamp(t.DefaultCurrentTimestamp)

	case *TypeTime:
		col.Type.Name = "TIME"

		if t.Timezone {
			col.Type.Name += "TZ"
		}

		if t.Precision >= 0 {
			col.Type.Name += fmt.Sprintf("(%d)", t.Precision)
		}

		col.Default = ddl.DefaultValueCurrentTimestamp(t.DefaultCurrentTimestamp)

	case *dal.TypeDate:
		col.Type.Name = "DATE"
		col.Default = ddl.DefaultValueCurrentTimestamp(t.DefaultCurrentTimestamp)

	case *dal.TypeNumber:
		if numType := cast.ToString(t.Meta["rdbms:type"]); numType != "" {
			col.Type.Name = numType
			col.Default = ddl.DefaultNumber(t.HasDefault, 0, t.DefaultValue)
			break
		}

		col.Type.Name = "NUMERIC"

		switch {
		case t.Precision > 0 && t.Scale > 0:
			col.Type.Name += fmt.Sprintf("(%d, %d)", t.Precision, t.Scale)
		case t.Precision > 0:
			col.Type.Name += fmt.Sprintf("(%d)", t.Precision)
		}

		col.Default = ddl.DefaultNumber(t.HasDefault, t.Precision, t.DefaultValue)

	case *dal.TypeText:
		if t.Length > 0 {
			col.Type.Name = fmt.Sprintf("VARCHAR(%d)", t.Length)
		} else {
			col.Type.Name = "TEXT"
		}

		if t.HasDefault {
			// @todo use proper quote type
			col.Default = fmt.Sprintf("%q", t.DefaultValue)
		}

	case *dal.TypeJSON:
		col.Type.Name = "JSONB"
		if col.Default, err = ddl.DefaultJSON(t.HasDefault, t.DefaultValue); err != nil {
			return nil, err
		}

	case *dal.TypeGeometry:
		// @todo geometry type
		col.Type.Name = "JSONB"

	case *dal.TypeBlob:
		col.Type.Name = "BYTEA"

	case *dal.TypeBoolean:
		col.Type.Name = "BOOLEAN"
		col.Default = ddl.DefaultBoolean(t.HasDefault, t.DefaultValue)

	case *dal.TypeUUID:
		col.Type.Name = "UUID"

	default:
		return nil, fmt.Errorf("unsupported column type: %s ", t.Type())
	}

	return
}

func (postgresDialect) ExprHandler(n *ql.ASTNode, args ...exp.Expression) (exp.Expression, error) {
	switch strings.ToLower(n.Ref) {
	case "concat":
		// need to force text type on all arguments
		aa := make([]any, len(args))
		for a := range args {
			aa[a] = exp.NewCastExpression(exp.NewLiteralExpression("?", args[a]), "TEXT")
		}

		return exp.NewSQLFunctionExpression("CONCAT", aa...), nil
	}

	return ref2exp.RefHandler(n, args...)
}

package query

import (
	"strings"
	"testing"
)

func TestDateFunctions_PostgresSQLUsesTextNormalization(t *testing.T) {
	tests := []struct {
		name string
		op   FilterOperator
	}{
		{name: "year", op: OpYear},
		{name: "month", op: OpMonth},
		{name: "day", op: OpDay},
		{name: "hour", op: OpHour},
		{name: "minute", op: OpMinute},
		{name: "second", op: OpSecond},
		{name: "date", op: OpDate},
		{name: "time", op: OpTime},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sql, _ := buildFunctionSQL("postgres", tc.op, `"Products"."created_at"`, nil)

			if !strings.Contains(sql, "CAST(NULLIF(CAST(") {
				t.Fatalf("expected postgres SQL to normalize through text cast, got %q", sql)
			}

			if strings.Contains(sql, `NULLIF("Products"."created_at", '')`) {
				t.Fatalf("postgres SQL must not call NULLIF directly on temporal columns: %q", sql)
			}
		})
	}
}

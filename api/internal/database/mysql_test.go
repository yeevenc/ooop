package database

import "testing"

func TestIsIntegerColumnType(t *testing.T) {
	tests := []struct {
		name     string
		dataType string
		want     bool
	}{
		{name: "bigint", dataType: "BIGINT", want: true},
		{name: "int", dataType: "int", want: true},
		{name: "integer", dataType: " integer ", want: true},
		{name: "varchar", dataType: "varchar", want: false},
		{name: "char", dataType: "CHAR", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isIntegerColumnType(test.dataType); got != test.want {
				t.Fatalf("isIntegerColumnType(%q) = %v, want %v", test.dataType, got, test.want)
			}
		})
	}
}

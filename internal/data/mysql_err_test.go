package data

import (
	"errors"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestIsMissingTable(t *testing.T) {
	if isMissingTable(nil) {
		t.Fatal("nil")
	}
	if isMissingTable(errors.New("other")) {
		t.Fatal("other")
	}
	if !isMissingTable(&mysql.MySQLError{Number: 1146, Message: "Table 'cigc.shipping_addresses' doesn't exist"}) {
		t.Fatal("1146")
	}
	if !isMissingTable(errors.New("Error 1146 (42S02): Table 'cigc.shipping_addresses' doesn't exist")) {
		t.Fatal("string 1146")
	}
}

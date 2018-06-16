package acknowledger_test

import (
	"testing"

	"github.com/corvus-ch/rabbitmq-cli-consumer/acknowledger"
)

var defaultTests = []struct {
	name      string
	onFailure int
	method    string
	args      []interface{}
}{
	{"ack", 0, "Ack", []interface{}{}},
	{"reject", 3, "Reject", []interface{}{false}},
	{"rejectRequeue", 4, "Reject", []interface{}{true}},
	{"nack", 5, "Nack", []interface{}{false}},
	{"nackRequeue", 6, "Nack", []interface{}{true}},
	{"undefined", 0, "Nack", []interface{}{true}},
}

func TestDefault(t *testing.T) {
	for code, test := range defaultTests {
		t.Run(test.name, func(t *testing.T) {
			d := new(TestDelivery)
			d.On(test.method, test.args...).Return(nil)
			a := acknowledger.New(false, test.onFailure)
			a.Ack(d, code)
		})
	}
}

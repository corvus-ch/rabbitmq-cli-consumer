package acknowledger_test

import (
	"errors"
	"testing"

	"github.com/corvus-ch/rabbitmq-cli-consumer/acknowledger"
	"github.com/magiconair/properties/assert"
)

var strictTests = []struct {
	name   string
	code   int
	method string
	args   []interface{}
	err    error
}{
	{"ack", 0, "Ack", []interface{}{}, nil},
	{"reject", 3, "Reject", []interface{}{false}, nil},
	{"rejectRequeue", 4, "Reject", []interface{}{true}, nil},
	{"nack", 5, "Nack", []interface{}{false}, nil},
	{"nackRequeue", 6, "Nack", []interface{}{true}, nil},
	{"undefined", 42, "Nack", []interface{}{true}, errors.New("unexpected exit code 42")},
}

func TestStrict(t *testing.T) {
	for _, test := range strictTests {
		t.Run(test.name, func(t *testing.T) {
			d := new(TestDelivery)
			d.On(test.method, test.args...).Return(nil)
			a := acknowledger.New(true, 0)
			assert.Equal(t, test.err, a.Ack(d, test.code))
		})
	}
}

package command_test

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/stretchr/testify/assert"
)

var argumentBuilderGetCommandTests = []struct {
	name         string
	compressed   bool
	withMetadata bool
	capture      bool
	arg          string
}{
	{
		"default",
		false,
		false,
		false,
		"ZGVmYXVsdA==",
	},
	{
		"metadata",
		false,
		true,
		false,
		"eyJwcm9wZXJ0aWVzIjp7ImFwcGxpY2F0aW9uX2hlYWRlcnMiOm51bGwsImNvbnRlbnRfdHlwZSI6IiIsImNvbnRlbnRfZW5jb2RpbmciOiIiLCJkZWxpdmVyeV9tb2RlIjowLCJwcmlvcml0eSI6MCwiY29ycmVsYXRpb25faWQiOiIiLCJyZXBseV90byI6IiIsImV4cGlyYXRpb24iOiIiLCJtZXNzYWdlX2lkIjoiIiwidGltZXN0YW1wIjoiMDAwMS0wMS0wMVQwMDowMDowMFoiLCJ0eXBlIjoiIiwidXNlcl9pZCI6IiIsImFwcF9pZCI6IiJ9LCJkZWxpdmVyeV9pbmZvIjp7Im1lc3NhZ2VfY291bnQiOjAsImNvbnN1bWVyX3RhZyI6IiIsImRlbGl2ZXJ5X3RhZyI6MCwicmVkZWxpdmVyZWQiOmZhbHNlLCJleGNoYW5nZSI6IiIsInJvdXRpbmdfa2V5IjoiIn0sImJvZHkiOiJtZXRhZGF0YSJ9",
	},
	{
		"compressed",
		true,
		false,
		false,
		"eNpKzs8tKEotLk5NAQQAAP//Fz8ENg==",
	},
	{
		"compressedMetadata",
		true,
		true,
		false,
		"eNpcj81OxDAMhN/F50UK17wDN05cqpDMdi3S2HJcRLXad0f9oSAkH+zRWPPNndREYc7oFO+UVCvn5CxtuCEVWKfY5lovlKU5mg++KCgS/SpoWQq3cVcLKn/ClmGSAorhQmosxr5sRxYz1D2By/5i0LoMLvuFL2XbDPs9ofc04nQ7T+ieJqVIIYTnp21eQ4jbvK2Wk3HusPMzqR774w8nt6us3X9ysszND9bW5wk2ePpfblPCin5IKBSvqXasBfIttfEgMJmd2zh8YDmS36Wsa5ZJDb2jvMBTSZ7o8R0AAP//J0eFAw==",
	},
	{
		"complex command",
		false,
		false,
		false,
		"Y29tcGxleCBjb21tYW5k",
	},
	{
		"outputCapturing",
		false,
		false,
		true,
		"b3V0cHV0Q2FwdHVyaW5n",
	},
}

func TestArgumentBuilder_GetCommand(t *testing.T) {
	for _, test := range argumentBuilderGetCommandTests {
		t.Run(test.name, func(t *testing.T) {
			outLog := log.New(&bytes.Buffer{}, "", 0)
			errLog := log.New(&bytes.Buffer{}, "", 0)
			b, err := command.NewBuilder(&command.ArgumentBuilder{
				Compressed:   test.compressed,
				WithMetadata: test.withMetadata,
			}, test.name, test.capture, outLog, errLog)

			if err != nil {
				t.Errorf("failed to create builder: %v", err)
			}

			cmd := createAndAssertCommand(t, b, []byte(test.name))
			assert.Equal(t, append(strings.Split(test.name, " "), test.arg), cmd.Args)
			assert.Nil(t, cmd.Stdin)
			assert.Nil(t, cmd.ExtraFiles)
			assert.Equal(t, os.Environ(), cmd.Env)
			assertLogger(t, outLog, cmd.Stdout, test.capture)
			assertLogger(t, errLog, cmd.Stdout, test.capture)
		})
	}
}

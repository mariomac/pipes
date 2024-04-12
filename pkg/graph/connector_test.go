package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectorRead(t *testing.T) {
	type testCase struct {
		in       string
		expected []dstConnector
	}
	for _, tc := range []testCase{
		{in: "foo", expected: []dstConnector{{dstNode: "foo"}}},
		{in: "ch:foo", expected: []dstConnector{{demuxChan: "ch", dstNode: "foo"}}},
		{in: "ch:foo,bar", expected: []dstConnector{{demuxChan: "ch", dstNode: "foo"}, {dstNode: "bar"}}},
		{in: "ch:foo,ch2:bar", expected: []dstConnector{{demuxChan: "ch", dstNode: "foo"}, {demuxChan: "ch2", dstNode: "bar"}}},
		{in: "foo, ch2:bar", expected: []dstConnector{{dstNode: "foo"}, {demuxChan: "ch2", dstNode: "bar"}}},
	} {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.expected, allConnectorsFrom(tc.in))
		})
	}
}

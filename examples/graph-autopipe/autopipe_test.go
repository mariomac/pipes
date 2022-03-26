package main

import (
	"bufio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAutopipe(t *testing.T) {
	// capturing STDOUT
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()
	g := BuildGraph("./nodes.hcl")
	go g.Run()

	waitForServer(t, 5*time.Second)
	c := http.Client{}
	resp, err := c.Post("http://127.0.0.1:8080", "application/json",
		strings.NewReader(`{"hello":"my friend","password":"sup3rs3cr37","secret":"kadlfjjsdlaf"}`))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	lines := map[string]struct{}{}
	scanner := bufio.NewScanner(r)
	require.True(t, scanner.Scan())
	lines[scanner.Text()] = struct{}{}
	require.True(t, scanner.Scan())
	lines[scanner.Text()] = struct{}{}
	assert.Equal(t, map[string]struct{}{
		`Safe-to-show message: {"hello":"my friend"}`:                                              {},
		`Received message: {"hello":"my friend","password":"sup3rs3cr37","secret":"kadlfjjsdlaf"}`: {},
	}, lines)
}

func waitForServer(t *testing.T, timeout time.Duration) {
	start := time.Now()
	for time.Now().Sub(start) < timeout {
		c := http.Client{}
		if _, err := c.Get("http://127.0.0.1:8080"); err == nil {
			return
		}
	}
	require.Fail(t, "timeout while waiting for server to listen")
}

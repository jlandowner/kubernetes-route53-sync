package main

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckSync(t *testing.T) {
	dnsName := "g.jlandowner.com"
	addr, err := net.LookupHost(dnsName)
	assert.Nil(t, err)

	t.Run("OK", func(t *testing.T) {
		ttl := 10
		ok := checkSync(addr, dnsName, ttl)
		assert.Equal(t, ok, true)
	})
	t.Run("NG", func(t *testing.T) {
		expectedIP := []string{"1.1.1.1", "2,2,2,2"}
		ttl := 10
		ok := checkSync(expectedIP, dnsName, ttl)
		assert.Equal(t, ok, true)
	})
}

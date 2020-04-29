package main

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckSync(t *testing.T) {
	dnsName := "g.jlandowner.com"
	addr, err := net.LookupHost(dnsName)
	assert.Nil(t, err)
	ctx := context.Background()

	t.Run("OK", func(t *testing.T) {
		ttl := 10
		ok := checkSync(ctx, addr, dnsName, ttl)
		assert.Equal(t, true, ok)
	})
	t.Run("NG", func(t *testing.T) {
		expectedIP := []string{"1.1.1.1", "2,2,2,2"}
		ttl := 10
		ok := checkSync(ctx, expectedIP, dnsName, ttl)
		assert.Equal(t, false, ok)
	})
}

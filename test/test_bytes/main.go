package main

import (
	"testing"

	"github.com/asimovsecurity/rosgo/libtest/libtest_bytes"
)

func main() {
	t := new(testing.T)
	libtest_bytes.RTTest(t)
}

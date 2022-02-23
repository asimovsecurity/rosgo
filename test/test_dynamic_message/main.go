package main

import (
	"testing"

	"github.com/asimovsecurity/rosgo/libtest/libtest_dynamic_message"
)

func main() {
	t := new(testing.T)
	libtest_dynamic_message.RTTest(t)
}

package main

//go:generate gengo msg std_msgs/String
import (
	"testing"

	"github.com/asimovsecurity/rosgo/libtest/libtest_service"
)

func main() {
	t := new(testing.T)
	libtest_service.RTTest(t)
}

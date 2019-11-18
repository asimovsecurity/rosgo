package libtest_talker

import (
	"fmt"
	"github.com/edwinhayes/rosgo/ros"
	"github.com/edwinhayes/rosgo/libtest/msgs/std_msgs"
	"github.com/edwinhayes/rosgo/libtest/msgs/test_msgs"
	"os"
	"testing"
	"time"
)

// RTTest performs a run-time test of using rosgo to create a ROS node, and publish messages from that node.  The test passes if the node is created and message publishers publish without error, but does not actually test whether the messages published are visible to other nodes.
func RTTest(t *testing.T) {
	// Instantiate a ROS node.
	node, err := ros.NewNode("/talker", os.Args)
	if err != nil {
		t.Error(err)
		return
	}
	defer node.Shutdown()
	node.Logger().SetSeverity(ros.LogLevelDebug)

	// Create a publisher on the node.
	pub := node.NewPublisher("/chatter", std_msgs.MsgString)
	if pub == nil {
		t.Error("NewPublisher failed; ", pub)
	}
	pub2 := node.NewPublisher("/all", test_msgs.MsgAllFieldTypes)
	if pub2 == nil {
		t.Error("NewPublisher failed; ", pub)
	}

	// Try to publish a message.
	var m1 std_msgs.String
	m1.Data = fmt.Sprintf("hello %s", time.Now().String())
	pub.Publish(&m1)
	var m2 test_msgs.AllFieldTypes
	pub2.Publish(&m2)

	// All done.
	return
}

// ALL DONE.

#!/bin/bash
source /opt/ros/melodic/setup.bash
export PATH=$PWD/bin:/usr/local/go/bin:$PATH
export GOPATH=$PWD:/usr/local/go

roscore &
go install github.com/asimovsecurity/rosgo/gengo
go generate github.com/asimovsecurity/rosgo/test/test_message
go test github.com/asimovsecurity/rosgo/xmlrpc
go test github.com/asimovsecurity/rosgo/ros
go test github.com/asimovsecurity/rosgo/test/test_message


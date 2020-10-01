package ros

import (
	"fmt"
	"reflect"
	"time"

	modular "github.com/edwinhayes/logrus-modular"
)

const (
	SimpleStatePending uint8 = 0
	SimpleStateActive  uint8 = 1
	SimpleStateDone    uint8 = 2
)

type simpleActionClient struct {
	ac          *defaultActionClient
	simpleState uint8
	gh          ClientGoalHandler
	doneCb      interface{}
	activeCb    interface{}
	feedbackCb  interface{}
	doneChan    chan struct{}
	logger      *modular.ModuleLogger
}

func newSimpleActionClient(node Node, action string, actionType ActionType) (*simpleActionClient, error) {
	client, err := newDefaultActionClient(node, action, actionType)
	if err != nil {
		return nil, err
	}
	return &simpleActionClient{
		ac:          client,
		simpleState: SimpleStateDone,
		doneChan:    make(chan struct{}, 10),
		logger:      node.Logger(),
	}, nil
}

func (sc *simpleActionClient) SendGoal(goal Message, doneCb, activeCb, feedbackCb interface{}, goalID string) error {
	sc.StopTrackingGoal()
	sc.doneCb = doneCb
	sc.activeCb = activeCb
	sc.feedbackCb = feedbackCb

	sc.setSimpleState(SimpleStatePending)
	gh, err := sc.ac.SendGoal(goal, sc.transitionHandler, sc.feedbackHandler, goalID)
	if err != nil {
		return err
	}
	sc.gh = gh
	return nil
}

func (sc *simpleActionClient) SendGoalAndWait(goal Message, executeTimeout, preeptTimeout Duration) (uint8, error) {
	logger := *sc.logger
	sc.SendGoal(goal, nil, nil, nil, "")
	if !sc.WaitForResult(executeTimeout) {
		logger.Debug("Cancelling goal")
		sc.CancelGoal()
		if sc.WaitForResult(preeptTimeout) {
			logger.Debug("Preempt finished within specified timeout")
		} else {
			logger.Debug("Preempt did not finish within specified timeout")
		}
	}

	return sc.GetState()
}

func (sc *simpleActionClient) ShutdownClient(status bool, feedback bool, result bool) {
	sc.ac.ShutdownClient(status, feedback, result)
}

func (sc *simpleActionClient) WaitForServer(timeout Duration) bool {
	return sc.ac.WaitForServer(timeout)
}

func (sc *simpleActionClient) WaitForResult(timeout Duration) bool {
	logger := *sc.logger
	if sc.gh == nil {
		logger.Errorf("[SimpleActionClient] Called WaitForResult when no goal exists")
		return false
	}

	waitStart := Now()
	waitStart = waitStart.Add(timeout)

LOOP:
	for {
		select {
		case <-sc.doneChan:
			break LOOP
		case <-time.After(100 * time.Millisecond):
		}

		if !timeout.IsZero() && waitStart.Cmp(Now()) <= 0 {
			break LOOP
		}
	}

	return sc.simpleState == SimpleStateDone
}

func (sc *simpleActionClient) GetResult() (Message, error) {
	if sc.gh == nil {
		return nil, fmt.Errorf("called get result when no goal running")
	}

	return sc.gh.GetResult()
}

func (sc *simpleActionClient) GetState() (uint8, error) {
	if sc.gh == nil {
		return uint8(9), fmt.Errorf("called get state when no goal running")
	}

	status, err := sc.gh.GetGoalStatus()
	if err != nil {
		return uint8(9), err
	}

	if status == uint8(7) {
		status = uint8(0)
	} else if status == uint8(6) {
		status = uint8(1)
	}

	return status, nil
}

func (sc *simpleActionClient) GetGoalStatusText() (string, error) {
	if sc.gh == nil {
		return "", fmt.Errorf("called GetGoalStatusText when no goal is running")
	}

	return sc.gh.GetGoalStatusText()
}

func (sc *simpleActionClient) CancelAllGoals() {
	sc.ac.CancelAllGoals()
}

func (sc *simpleActionClient) CancelAllGoalsBeforeTime(stamp Time) {
	sc.ac.CancelAllGoalsBeforeTime(stamp)
}

func (sc *simpleActionClient) CancelGoal() error {
	if sc.gh == nil {
		return nil
	}

	return sc.gh.Cancel()
}

func (sc *simpleActionClient) StopTrackingGoal() {
	sc.gh = nil
}

func (sc *simpleActionClient) transitionHandler(gh ClientGoalHandler) {
	logger := *sc.logger
	commState, err := gh.GetCommState()
	if err != nil {
		logger.Errorf("Error getting CommState: %v", err)
		return
	}
	logger.Infof("in transitionHandler with comm state %s and simple state %s with SimpleActionClient in NS %s", commState, sc.simpleState, sc.ac.node.Name())
	errMsg := fmt.Errorf("received comm state %s when in simple state %d with SimpleActionClient in NS %s",
		commState, sc.simpleState, sc.ac.node.Name())

	var callbackType string
	var args []reflect.Value
	switch commState {
	case Active:
		switch sc.simpleState {
		case SimpleStatePending:
			sc.setSimpleState(SimpleStateActive)
			callbackType = "active"

		case SimpleStateDone:
			logger.Errorf("[SimpleActionClient] %v", errMsg)
		}

	case Recalling:
		switch sc.simpleState {
		case SimpleStateActive, SimpleStateDone:
			logger.Errorf("[SimpleActionClient] %v", errMsg)
		}

	case Preempting:
		switch sc.simpleState {
		case SimpleStatePending:
			sc.setSimpleState(SimpleStateActive)
			callbackType = "active"

		case SimpleStateDone:
			logger.Errorf("[SimpleActionClient] %v", errMsg)
		}

	case Done:
		switch sc.simpleState {
		case SimpleStatePending, SimpleStateActive:
			sc.setSimpleState(SimpleStateDone)
			sc.sendDone()

			if sc.doneCb == nil {
				break
			}

			status, err := gh.GetGoalStatus()
			if err != nil {
				logger.Errorf("[SimpleActionClient] Error getting status: %v", err)
				break
			}
			logger.Infof("[SimpleActionClient] Got goal status: %v", status)
			result, err := gh.GetResult()
			if err != nil {
				logger.Errorf("[SimpleActionClient] Error getting result: %v; GoalStatus: %v", err, status)
				break
			}

			callbackType = "done"
			args = append(args, reflect.ValueOf(status))
			args = append(args, reflect.ValueOf(result))

		case SimpleStateDone:
			logger.Errorf("[SimpleActionClient] received DONE twice")
		}
	}

	if len(callbackType) > 0 {
		sc.runCallback(callbackType, args)
	}
}

func (sc *simpleActionClient) sendDone() {
	logger := *sc.logger
	select {
	case sc.doneChan <- struct{}{}:
	default:
		logger.Errorf("[SimpleActionClient] Error sending done notification. Channel full.")
	}
}

func (sc *simpleActionClient) feedbackHandler(gh ClientGoalHandler, msg Message) {
	if sc.gh == nil || sc.gh != gh {
		return
	}

	sc.runCallback("feedback", []reflect.Value{reflect.ValueOf(msg)})
}

func (sc *simpleActionClient) setSimpleState(state uint8) {
	logger := *sc.logger
	logger.Debugf("[SimpleActionClient] Transitioning from %d to %d", sc.simpleState, state)
	sc.simpleState = state
}

func (sc *simpleActionClient) runCallback(cbType string, args []reflect.Value) {
	var callback interface{}
	logger := *sc.logger
	switch cbType {
	case "active":
		callback = sc.activeCb
	case "feedback":
		callback = sc.feedbackCb
	case "done":
		callback = sc.doneCb
	default:
		logger.Errorf("[SimpleActionClient] Unknown callback %s", cbType)
	}

	if callback == nil {
		return
	}

	fun := reflect.ValueOf(callback)
	numArgsNeeded := fun.Type().NumIn()

	if numArgsNeeded > len(args) {
		logger.Errorf("[SimpleActionClient] Unexpected arguments:"+
			"callback %s expects %d arguments but %d arguments provided", cbType, numArgsNeeded, len(args))
		return
	}

	logger.Debugf("[SimpleActionClient] Calling %s callback with %d arguments", cbType, len(args))

	fun.Call(args[0:numArgsNeeded])
}

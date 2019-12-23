// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package integrations

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/notification"
	"code.gitea.io/gitea/modules/notification/base"
	"code.gitea.io/gitea/modules/queue"
)

var notifierListener *NotifierListener

var once = sync.Once{}

type NotifierListener struct {
	lock      sync.RWMutex
	callbacks map[string][]*func(string, [][]byte)
	notifier  base.Notifier
}

func NotifierListenerInit() {
	once.Do(func() {
		notifierListener = &NotifierListener{
			callbacks: map[string][]*func(string, [][]byte){},
		}
		notifierListener.notifier = base.NewQueueNotifierWithHandle("test-notifier", notifierListener.handle)
		notification.RegisterNotifier(notifierListener.notifier)
	})
}

// Register will register a callback with the provided notifier function
func (n *NotifierListener) Register(functionName string, callback *func(string, [][]byte)) {
	n.lock.Lock()
	n.callbacks[functionName] = append(n.callbacks[functionName], callback)
	n.lock.Unlock()
}

// Deregister will remove the provided callback from the provided notifier function
func (n *NotifierListener) Deregister(functionName string, callback *func(string, [][]byte)) {
	n.lock.Lock()
	found := -1
	for i, callbackPtr := range n.callbacks[functionName] {
		if callbackPtr == callback {
			found = i
			break
		}
	}
	if found > -1 {
		n.callbacks[functionName] = append(n.callbacks[functionName][0:found], n.callbacks[functionName][found+1:]...)
	}
	n.lock.Unlock()
}

// RegisterChannel will register a provided channel with function name and return a function to deregister it
func (n *NotifierListener) RegisterChannel(name string, channel chan<- interface{}, argNumber int, exemplar interface{}) (deregister func()) {
	t := reflect.TypeOf(exemplar)
	callback := func(_ string, args [][]byte) {
		n := reflect.New(t).Elem()
		err := json.Unmarshal(args[argNumber], n.Addr().Interface())
		if err != nil {
			log.Error("Wrong Argument passed to register channel: %v ", err)
		}
		channel <- n.Interface()
	}
	n.Register(name, &callback)

	return func() {
		n.Deregister(name, &callback)
	}
}

func (n *NotifierListener) handle(data ...queue.Data) {
	n.lock.RLock()
	defer n.lock.RUnlock()
	for _, datum := range data {
		call := datum.(*base.FunctionCall)
		callbacks, ok := n.callbacks[call.Name]
		if ok && len(callbacks) > 0 {
			for _, callback := range callbacks {
				(*callback)(call.Name, call.Args)
			}
		}
	}
}

func TestNotifierListener(t *testing.T) {
	defer prepareTestEnv(t)()

	createPullNotified := make(chan interface{}, 10)
	deregister := notifierListener.RegisterChannel("NotifyNewPullRequest", createPullNotified, 0, &models.PullRequest{})
	bs, _ := json.Marshal(&models.PullRequest{})
	notifierListener.handle(&base.FunctionCall{
		Name: "NotifyNewPullRequest",
		Args: [][]byte{
			bs,
		},
	})
	<-createPullNotified

	notifierListener.notifier.NotifyNewPullRequest(&models.PullRequest{})
	<-createPullNotified

	notification.NotifyNewPullRequest(&models.PullRequest{})
	<-createPullNotified

	deregister()
	close(createPullNotified)

	notification.NotifyNewPullRequest(&models.PullRequest{})
	// would panic if not deregistered
}

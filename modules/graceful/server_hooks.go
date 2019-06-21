// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package graceful

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"code.gitea.io/gitea/modules/log"
)

// shutdown closes the listener so that no new connections are accepted. it also
// starts a goroutine that will hammer (stop all running requests) the server
// after DefaultHammerTime.
func (srv *Server) shutdown() {
	if srv.getState() != stateRunning {
		return
	}

	srv.setState(stateShuttingDown)
	if DefaultHammerTime >= 0 {
		go srv.hammerTime(DefaultHammerTime)
	}

	if srv.OnShutdown != nil {
		srv.OnShutdown()
	}
	err := srv.listener.Close()
	if err != nil {
		log.Error("PID: %d Listener.Close() error: %v", os.Getpid(), err)
	} else {
		log.Info("PID: %d Listener (%s) closed.", os.Getpid(), srv.listener.Addr())
	}
}

// hammerTime forces the server to shutdown in a given timeout - whether it
// finished outstanding requests or not. if Read/WriteTimeout are not set or the
// max header size is very big a connection could hang...
//
// srv.Serve() will not return until all connections are served. this will
// unblock the srv.wg.Wait() in Serve() thus causing ListenAndServe* functions to
// return.
func (srv *Server) hammerTime(d time.Duration) {
	defer func() {
		// we are calling srv.wg.Done() until it panics which means we called
		// Done() when the counter was already at 0 and we're done.
		// (and thus Serve() will return and the parent will exit)
		if r := recover(); r != nil {
			log.Error("WaitGroup at 0: Error: %v", r)
		}
	}()
	if srv.getState() != stateShuttingDown {
		return
	}
	time.Sleep(d)
	log.Warn("[STOP - Hammer Time] Forcefully shutting down parent")
	for {
		if srv.getState() == stateTerminate {
			break
		}
		srv.wg.Done()
		runtime.Gosched()
	}
}

func (srv *Server) fork() error {
	runningServerReg.Lock()
	defer runningServerReg.Unlock()

	// only one server instance should fork!
	if runningServersForked {
		return errors.New("another process already forked. Ignoring this one")
	}

	runningServersForked = true

	_, err := RestartProcess()

	return err
}

// RegisterPreSignalHook registers a function to be run before the signal handler for
// a given signal. These are not mutex locked and should therefore be only called before Serve.
func (srv *Server) RegisterPreSignalHook(sig os.Signal, f func()) (err error) {
	for _, s := range hookableSignals {
		if s == sig {
			srv.PreSignalHooks[sig] = append(srv.PreSignalHooks[sig], f)
			return
		}
	}
	err = fmt.Errorf("Signal %v is not supported", sig)
	return
}

// RegisterPostSignalHook registers a function to be run after the signal handler for
// a given signal. These are not mutex locked and should therefore be only called before Serve.
func (srv *Server) RegisterPostSignalHook(sig os.Signal, f func()) (err error) {
	for _, s := range hookableSignals {
		if s == sig {
			srv.PostSignalHooks[sig] = append(srv.PostSignalHooks[sig], f)
			return
		}
	}
	err = fmt.Errorf("Signal %v is not supported", sig)
	return
}

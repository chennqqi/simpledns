package main

import (
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"github.com/chennqqi/goutils/consul.v2"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

//a file to watch, and Arbitrary extra data
type NameValue struct {
	Name  string
	Extra interface{}
}

//return watcherEvent
type WatcherEvent struct {
	Name  string
	Data  []byte
	Extra interface{}
}

//Watcher can watch multiple files or consul-stored files changed event
type Watcher struct {
	consulNames  []string
	consulIndexs []uint64
	extraData    map[string]interface{}

	w        *fsnotify.Watcher
	c        *consul.ConsulOperator
	ch       chan *WatcherEvent
	stopChan chan struct{}
}

//create a new Watcher, consulOperater is an object to watch consul stored file
func NewWatcher(names []NameValue, c *consul.ConsulOperator) (*Watcher, error) {
	var w Watcher
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w.c = c
	w.ch = make(chan *WatcherEvent)
	w.extraData = make(map[string]interface{})
	w.stopChan = make(chan struct{})
	w.w = fsw

	for i := 0; i < len(names); i++ {
		name := names[i].Name
		if strings.HasPrefix(name, "consul://") {
			u, err := url.Parse(name)
			if err != nil {
				logrus.Errorf("[watcher.go::Watcher.NewWatcher] parse consul %v error: %v", name, err)
			} else {
				w.consulNames = append(w.consulNames, u.Path)
				w.extraData[u.Path] = names[i].Extra
			}
		} else {
			fsw.Add(name)
			w.extraData[name] = names[i].Extra
		}
		logrus.Debugf("add watch: %v %v", name, names[i].Extra)
	}
	for i := 0; i < len(w.consulNames); i++ {
		_, index, _ := c.GetEx(w.consulNames[i])
		w.consulIndexs = append(w.consulIndexs, index)
	}
	return &w, nil
}

//run this watcher
func (w *Watcher) Run() {
	fsw := w.w
	c := w.c
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
FOR_LOOP:
	for {
		select {
		case e, ok := <-fsw.Events:
			if !ok {
				break FOR_LOOP
			}
			if e.Op != fsnotify.Write {
				txt, err := ioutil.ReadFile(e.Name)
				if err != nil {
					logrus.Errorf("[watcher.go::Watcher.Run] ioutil.ReadAll %v error: %v", e.Name, e)
				} else {
					//emit watcherEvent
					w.ch <- &WatcherEvent{
						e.Name,
						txt,
						w.extraData[e.Name],
					}
				}
			}

		case e, ok := <-fsw.Errors:
			if !ok {
				break
			}
			logrus.Infof("[watcher.go::Watcher.Run] recv watcher error: %v", e)

		case <-ticker.C:
			for i := 0; i < len(w.consulNames); i++ {
				name := w.consulNames[i]
				txt, index, err := c.GetEx(name)
				if err != nil {
					logrus.Errorf("[watcher.go::Watcher.Run] consul.GetEx(%v) error: %v", name, err)
				} else if w.consulIndexs[i] != index { //lastIndex
					//emit watcherEvent
					logrus.Debugf("consul event: %v %v %v", name, string(txt), w.extraData[name])
					w.ch <- &WatcherEvent{
						name,
						txt,
						w.extraData[name],
					}
					w.consulIndexs[i] = index
				}
			}
		}
	}
	close(w.ch)
	close(w.stopChan)
}

//WatcherEvent output chan
func (w *Watcher) Events() <-chan *WatcherEvent {
	return w.ch
}

//stop watcher
func (w *Watcher) Stop() {
	w.w.Close()
	<-w.stopChan
}

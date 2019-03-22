package main

import (
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/chennqqi/goutils/consul"
	"github.com/fsnotify/fsnotify"
)

type NameValue struct {
	Name  string
	Extra interface{}
}

type WatcherEvent struct {
	Name  string
	Data  []byte
	Extra interface{}
}

type Watcher struct {
	consulNames  []string
	consulIndexs []uint64
	extraData    map[string]interface{}

	w        *fsnotify.Watcher
	c        *consul.ConsulOperator
	ch       chan *WatcherEvent
	stopChan chan struct{}
}

func NewWatcher(names []NameValue) (*Watcher, error) {
	var w Watcher
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w.ch = make(chan *WatcherEvent)
	w.extraData = make(map[string]interface{})
	w.w = fsw

	for i := 0; i < len(names); i++ {
		name := names[i].Name
		if strings.HasPrefix(name, "consul://") {
			u, err := url.Parse(name)
			if err != nil {
				logrus.Errorf("[watcher.go::Watcher.NewWatcher] parse consul %v error: %v", name, err)
			} else {
				w.consulNames = append(w.consulNames, u.Path)
			}
		} else {
			fsw.Add(name)
		}
		w.extraData[name] = names[i].Extra
	}
	return &w, nil
}

func (w *Watcher) Run() {
	fsw := w.w
	c := w.c
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case e, ok := <-fsw.Events:
			if !ok {
				break
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
}

func (w *Watcher) Events() <-chan *WatcherEvent {
	return w.ch
}

func (w *Watcher) Stop() {
	w.w.Close()
	<-w.stopChan
}

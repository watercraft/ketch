package ketch

import (
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/hashicorp/memberlist"

	"github.com/watercraft/ketch/api"
	"github.com/watercraft/ketch/msg"
)

// Ketch
// State of Ketch service.
type Ketch struct {
	// Logger to send logs to
	log *logrus.Logger

	// HashiCorp's Memberlist
	list *memberlist.Memberlist

	// User config
	config *Config

	// Host server boot time
	bootTime *time.Time

	// Private database for Ketch config
	db *bolt.DB

	// Read/write lock to protect Ketch state
	sync.RWMutex

	// Runtime information
	runtime *api.Runtime

	// resourceMgr is a map from type to manager object
	resourceMgr map[api.Type]*ResourceMgr

	// incomingMsgCh is a channel to dispatch messages
	// without blocking memberlist NotifyMsg() call
	incomingMsgCh chan msg.Msg

	// wakeCh is the channel used to wake the processing loop
	wakeServiceLoopCh chan bool

	// uptime is the current host uptime in seconds
	uptime int64
}

// Join
// Sync state with list of candidate Ketch members.
// Returns number of members successfully joined.
// Wrapper around Memberlist Join()
func (k *Ketch) Join(list []string) (int, error) {
	return k.list.Join(list)
}

// Members
// Return list of Ketch members.
// Wrapper around Memberlist Members()
func (k *Ketch) Members() []*memberlist.Node {
	return k.list.Members()
}

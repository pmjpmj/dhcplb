/**
 * Copyright (c) 2016-present, Facebook, Inc.
 * All rights reserved.
 *
 * This source code is licensed under the BSD-style license found in the
 * LICENSE file in the root directory of this source tree. An additional grant
 * of patent rights can be found in the PATENTS file in the same directory.
 */

package dhcplb

import (
	"errors"
	"hash/fnv"
	"sync"
	"sync/atomic"

	"github.com/golang/glog"
)

type giaddrModulo struct {
	lock    sync.RWMutex
	stable  []*DHCPServer
	rc      []*DHCPServer
	rcRatio uint32
}

func (m *giaddrModulo) Name() string {
	return "giaddr"
}

func (m *giaddrModulo) getHash(token []byte) uint32 {
	hasher := fnv.New32a()
	hasher.Write(token)
	hash := hasher.Sum32()
	return hash
}

func (m *giaddrModulo) SetRCRatio(ratio uint32) {
	atomic.StoreUint32(&m.rcRatio, ratio)
}

func (m *giaddrModulo) SelectServerFromList(list []*DHCPServer, message *DHCPMessage) (*DHCPServer, error) {
	hash := m.getHash(message.GiAddr)
	if len(list) == 0 {
		return nil, errors.New("Server list is empty")
	}
	return list[hash%uint32(len(list))], nil
}

func (m *giaddrModulo) SelectRatioBasedDhcpServer(message *DHCPMessage) (*DHCPServer, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	hash := m.getHash(message.GiAddr)

	// convert to a number 0-100 and then see if it should be RC
	if hash%100 < m.rcRatio {
		return m.SelectServerFromList(m.rc, message)
	}
	// otherwise go to stable
	return m.SelectServerFromList(m.stable, message)
}

func (m *giaddrModulo) UpdateServerList(name string, list []*DHCPServer, ptr *[]*DHCPServer) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	*ptr = list
	glog.Infof("List of available %s servers:", name)
	for _, server := range *ptr {
		glog.Infof("%s", server)
	}
	return nil
}

func (m *giaddrModulo) UpdateStableServerList(list []*DHCPServer) error {
	return m.UpdateServerList("stable", list, &m.stable)
}

func (m *giaddrModulo) UpdateRCServerList(list []*DHCPServer) error {
	return m.UpdateServerList("rc", list, &m.rc)
}

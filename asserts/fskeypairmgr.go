// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2015-2016 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package asserts

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
)

// the default simple filesystem based keypair manager/backstore

const (
	privateKeysLayoutVersion = "v0"
	privateKeysRoot          = "private-keys-" + privateKeysLayoutVersion
)

type filesystemKeypairManager struct {
	top string
	mu  sync.RWMutex
}

// OpenFSKeypairManager opens a filesystem backed assertions backstore under path.
func OpenFSKeypairManager(path string) (KeypairManager, error) {
	top := filepath.Join(path, privateKeysRoot)
	err := ensureTop(top)
	if err != nil {
		return nil, err
	}
	return &filesystemKeypairManager{top: top}, nil
}

var errKeypairAlreadyExists = errors.New("key pair with given key id already exists")

func (fskm *filesystemKeypairManager) Put(authorityID string, privKey PrivateKey) error {
	keyID := privKey.PublicKey().ID()
	escapedAuthorityID := url.QueryEscape(authorityID)
	if entryExists(fskm.top, escapedAuthorityID, keyID) {
		return errKeypairAlreadyExists
	}
	encoded, err := encodePrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("failed to store private key: %v", err)
	}

	fskm.mu.Lock()
	defer fskm.mu.Unlock()

	err = atomicWriteEntry(encoded, true, fskm.top, escapedAuthorityID, keyID)
	if err != nil {
		return fmt.Errorf("failed to store private key: %v", err)
	}
	return nil
}

var errKeypairNotFound = errors.New("no matching key pair found")

func (fskm *filesystemKeypairManager) Get(authorityID, keyID string) (PrivateKey, error) {
	fskm.mu.RLock()
	defer fskm.mu.RUnlock()

	encoded, err := readEntry(fskm.top, url.QueryEscape(authorityID), keyID)
	if os.IsNotExist(err) {
		return nil, errKeypairNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read key pair: %v", err)
	}
	privKey, err := decodePrivateKey(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key pair: %v", err)
	}
	return privKey, nil
}

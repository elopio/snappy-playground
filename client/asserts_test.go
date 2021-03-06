// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016 Canonical Ltd
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

package client_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"

	. "gopkg.in/check.v1"

	"github.com/ubuntu-core/snappy/asserts"
)

func (cs *clientSuite) TestClientAssert(c *C) {
	cs.rsp = `{
		"type": "sync",
		"result": {}
	}`
	a := []byte("Assertion.")
	err := cs.cli.Assert(a)
	c.Assert(err, IsNil)
	body, err := ioutil.ReadAll(cs.req.Body)
	c.Assert(err, IsNil)
	c.Check(body, DeepEquals, a)
	c.Check(cs.req.Method, Equals, "POST")
	c.Check(cs.req.URL.Path, Equals, "/2.0/assertions")
}

func (cs *clientSuite) TestClientAssertsCallsEndpoint(c *C) {
	_, _ = cs.cli.Asserts("snap-revision", nil)
	c.Check(cs.req.Method, Equals, "GET")
	c.Check(cs.req.URL.Path, Equals, "/2.0/assertions/snap-revision")
}

func (cs *clientSuite) TestClientAssertsCallsEndpointWithFilter(c *C) {
	_, _ = cs.cli.Asserts("snap-revision", map[string]string{
		"snap-id":     "snap-id-1",
		"snap-digest": "sha256 digest...",
	})
	u, err := url.ParseRequestURI(cs.req.URL.String())
	c.Assert(err, IsNil)
	c.Check(u.Path, Equals, "/2.0/assertions/snap-revision")
	c.Check(u.Query(), DeepEquals, url.Values{
		"snap-digest": []string{"sha256 digest..."},
		"snap-id":     []string{"snap-id-1"},
	})
}

func (cs *clientSuite) TestClientAssertsHttpError(c *C) {
	cs.err = errors.New("fail")
	_, err := cs.cli.Asserts("snap-build", nil)
	c.Assert(err, ErrorMatches, "failed to query assertions: fail")
}

func (cs *clientSuite) TestClientAssertsJSONError(c *C) {
	cs.status = http.StatusBadRequest
	cs.header = http.Header{}
	cs.header.Add("Content-type", "application/json")
	cs.rsp = `{
		"status_code": 400,
		"type": "error",
		"result": {
			"message": "invalid"
		}
	}`
	_, err := cs.cli.Asserts("snap-build", nil)
	c.Assert(err, ErrorMatches, "invalid")
}

func (cs *clientSuite) TestClientAsserts(c *C) {
	cs.header = http.Header{}
	cs.header.Add("X-Ubuntu-Assertions-Count", "2")
	cs.rsp = `type: snap-revision
authority-id: store-id1
snap-id: snap-id-1
snap-digest: sha256 ...
snap-revision: 1
snap-build: sha256 ...
developer-id: dev-id1
revision: 1
timestamp: 2015-11-25T20:00:00Z
body-length: 0

openpgp ...

type: snap-revision
authority-id: store-id1
snap-id: snap-id-2
snap-digest: sha256 ...
snap-revision: 1
snap-build: sha256 ...
developer-id: dev-id1
revision: 1
timestamp: 2015-11-30T20:00:00Z
body-length: 0

openpgp ...
`

	a, err := cs.cli.Asserts("snap-revision", nil)
	c.Assert(err, IsNil)
	c.Check(a, HasLen, 2)

	c.Check(a[0].Type(), Equals, asserts.SnapRevisionType)
}

func (cs *clientSuite) TestClientAssertsNoAssertions(c *C) {
	cs.header = http.Header{}
	cs.header.Add("X-Ubuntu-Assertions-Count", "0")
	cs.rsp = ""
	cs.status = http.StatusOK
	a, err := cs.cli.Asserts("snap-revision", nil)
	c.Assert(err, IsNil)
	c.Check(a, HasLen, 0)
}

func (cs *clientSuite) TestClientAssertsMissingAssertions(c *C) {
	cs.header = http.Header{}
	cs.header.Add("X-Ubuntu-Assertions-Count", "4")
	cs.rsp = ""
	cs.status = http.StatusOK
	_, err := cs.cli.Asserts("snap-build", nil)
	c.Assert(err, ErrorMatches, "response did not have the expected number of assertions")
}

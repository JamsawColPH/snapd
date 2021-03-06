// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2017 Canonical Ltd
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

package snapstate_test

import (
	"io"
	"io/ioutil"
	"time"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/osutil"
	"github.com/snapcore/snapd/overlord/auth"
	"github.com/snapcore/snapd/overlord/snapstate"
	"github.com/snapcore/snapd/overlord/state"
	"github.com/snapcore/snapd/store/storetest"
)

type catalogStore struct {
	storetest.Store

	ops []string
}

func (r *catalogStore) WriteCatalogs(w io.Writer) error {
	r.ops = append(r.ops, "write-catalog")
	w.Write([]byte("pkg1\npkg2"))
	return nil
}

func (r *catalogStore) Sections(*auth.UserState) ([]string, error) {
	r.ops = append(r.ops, "sections")
	return []string{"section1", "section2"}, nil
}

type catalogRefreshTestSuite struct {
	state *state.State

	store  *catalogStore
	tmpdir string
}

var _ = Suite(&catalogRefreshTestSuite{})

func (s *catalogRefreshTestSuite) SetUpTest(c *C) {
	s.tmpdir = c.MkDir()
	dirs.SetRootDir(s.tmpdir)
	s.state = state.New(nil)

	s.store = &catalogStore{}
	s.state.Lock()
	snapstate.ReplaceStore(s.state, s.store)
	s.state.Unlock()

	snapstate.CanAutoRefresh = func(*state.State) (bool, error) { return true, nil }
}

func (s *catalogRefreshTestSuite) TearDownTest(c *C) {
	snapstate.CanAutoRefresh = nil
}

func (s *catalogRefreshTestSuite) TestCatalogRefresh(c *C) {
	cr7 := snapstate.NewCatalogRefresh(s.state)
	err := cr7.Ensure()
	c.Check(err, IsNil)

	c.Check(s.store.ops, DeepEquals, []string{"sections", "write-catalog"})

	c.Check(osutil.FileExists(dirs.SnapSectionsFile), Equals, true)
	content, err := ioutil.ReadFile(dirs.SnapSectionsFile)
	c.Assert(err, IsNil)
	c.Check(string(content), Equals, "section1\nsection2")

	c.Check(osutil.FileExists(dirs.SnapNamesFile), Equals, true)
	content, err = ioutil.ReadFile(dirs.SnapNamesFile)
	c.Assert(err, IsNil)
	c.Check(string(content), Equals, "pkg1\npkg2")
}

func (s *catalogRefreshTestSuite) TestCatalogRefreshNotNeeded(c *C) {
	cr7 := snapstate.NewCatalogRefresh(s.state)
	snapstate.MockCatalogRefreshNextRefresh(cr7, time.Now().Add(1*time.Hour))
	err := cr7.Ensure()
	c.Check(err, IsNil)
	c.Check(s.store.ops, HasLen, 0)
	c.Check(osutil.FileExists(dirs.SnapSectionsFile), Equals, false)
	c.Check(osutil.FileExists(dirs.SnapNamesFile), Equals, false)
}

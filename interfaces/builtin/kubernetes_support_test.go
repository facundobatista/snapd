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

package builtin_test

import (
	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/interfaces"
	"github.com/snapcore/snapd/interfaces/apparmor"
	"github.com/snapcore/snapd/interfaces/builtin"
	"github.com/snapcore/snapd/interfaces/kmod"
	"github.com/snapcore/snapd/snap"
	"github.com/snapcore/snapd/snap/snaptest"
	"github.com/snapcore/snapd/testutil"
)

type KubernetesSupportInterfaceSuite struct {
	iface interfaces.Interface
	slot  *interfaces.Slot
	plug  *interfaces.Plug
}

const k8sMockPlugSnapInfoYaml = `name: other
version: 1.0
apps:
 app2:
  command: foo
  plugs: [kubernetes-support]
`

var _ = Suite(&KubernetesSupportInterfaceSuite{})

func (s *KubernetesSupportInterfaceSuite) SetUpTest(c *C) {
	s.iface = builtin.NewKubernetesSupportInterface()
	s.slot = &interfaces.Slot{
		SlotInfo: &snap.SlotInfo{
			Snap:      &snap.Info{SuggestedName: "core", Type: snap.TypeOS},
			Name:      "kubernetes-support",
			Interface: "kubernetes-support",
		},
	}
	plugSnap := snaptest.MockInfo(c, k8sMockPlugSnapInfoYaml, nil)
	s.plug = &interfaces.Plug{PlugInfo: plugSnap.Plugs["kubernetes-support"]}
}

func (s *KubernetesSupportInterfaceSuite) TestName(c *C) {
	c.Assert(s.iface.Name(), Equals, "kubernetes-support")
}

func (s *KubernetesSupportInterfaceSuite) TestSanitizeSlot(c *C) {
	err := s.iface.SanitizeSlot(s.slot)
	c.Assert(err, IsNil)
	err = s.iface.SanitizeSlot(&interfaces.Slot{SlotInfo: &snap.SlotInfo{
		Snap:      &snap.Info{SuggestedName: "some-snap"},
		Name:      "kubernetes-support",
		Interface: "kubernetes-support",
	}})
	c.Assert(err, ErrorMatches, "kubernetes-support slots are reserved for the operating system snap")
}

func (s *KubernetesSupportInterfaceSuite) TestSanitizePlug(c *C) {
	err := s.iface.SanitizePlug(s.plug)
	c.Assert(err, IsNil)
}

func (s *KubernetesSupportInterfaceSuite) TestSanitizeIncorrectInterface(c *C) {
	c.Assert(func() { s.iface.SanitizeSlot(&interfaces.Slot{SlotInfo: &snap.SlotInfo{Interface: "other"}}) },
		PanicMatches, `slot is not of interface "kubernetes-support"`)
	c.Assert(func() { s.iface.SanitizePlug(&interfaces.Plug{PlugInfo: &snap.PlugInfo{Interface: "other"}}) },
		PanicMatches, `plug is not of interface "kubernetes-support"`)
}

func (s *KubernetesSupportInterfaceSuite) TestUsedSecuritySystems(c *C) {
	kmodSpec := &kmod.Specification{}
	err := kmodSpec.AddConnectedPlug(s.iface, s.plug, s.slot)
	c.Assert(err, IsNil)
	c.Assert(kmodSpec.Modules(), DeepEquals, map[string]bool{
		"llc": true,
		"stp": true,
	})

	apparmorSpec := &apparmor.Specification{}
	err = apparmorSpec.AddConnectedPlug(s.iface, s.plug, s.slot)
	c.Assert(err, IsNil)
	c.Assert(apparmorSpec.SecurityTags(), DeepEquals, []string{"snap.other.app2"})
	c.Check(apparmorSpec.SnippetForTag("snap.other.app2"), testutil.Contains, "# Allow reading the state of modules kubernetes needs\n")
}

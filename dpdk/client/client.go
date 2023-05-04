// Copyright 2022 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/onmetal/dpservice-cli/dpdk/api"
	apierrors "github.com/onmetal/dpservice-cli/dpdk/api/errors"
	"github.com/onmetal/dpservice-cli/netiputil"
	dpdkproto "github.com/onmetal/net-dpservice-go/proto"
)

type Client interface {
	GetLoadBalancer(ctx context.Context, id string) (*api.LoadBalancer, error)
	CreateLoadBalancer(ctx context.Context, lb *api.LoadBalancer) (*api.LoadBalancer, error)
	DeleteLoadBalancer(ctx context.Context, id string) error

	ListLoadBalancerPrefixes(ctx context.Context, interfaceID string) (*api.PrefixList, error)
	CreateLoadBalancerPrefix(ctx context.Context, prefix *api.Prefix) (*api.Prefix, error)
	DeleteLoadBalancerPrefix(ctx context.Context, interfaceID string, prefix netip.Prefix) error

	GetLoadBalancerTargets(ctx context.Context, interfaceID string) (*api.LoadBalancerTargetList, error)
	CreateLoadBalancerTarget(ctx context.Context, lbtarget *api.LoadBalancerTarget) (*api.LoadBalancerTarget, error)
	DeleteLoadBalancerTarget(ctx context.Context, id string, targetIP netip.Addr) error

	GetInterface(ctx context.Context, id string) (*api.Interface, error)
	ListInterfaces(ctx context.Context) (*api.InterfaceList, error)
	CreateInterface(ctx context.Context, iface *api.Interface) (*api.Interface, error)
	DeleteInterface(ctx context.Context, id string) error

	GetVirtualIP(ctx context.Context, interfaceID string) (*api.VirtualIP, error)
	CreateVirtualIP(ctx context.Context, virtualIP *api.VirtualIP) (*api.VirtualIP, error)
	DeleteVirtualIP(ctx context.Context, interfaceID string) error

	ListPrefixes(ctx context.Context, interfaceID string) (*api.PrefixList, error)
	CreatePrefix(ctx context.Context, prefix *api.Prefix) (*api.Prefix, error)
	DeletePrefix(ctx context.Context, interfaceID string, prefix netip.Prefix) error

	ListRoutes(ctx context.Context, vni uint32) (*api.RouteList, error)
	CreateRoute(ctx context.Context, route *api.Route) (*api.Route, error)
	DeleteRoute(ctx context.Context, vni uint32, prefix netip.Prefix, nextHopVNI uint32, nextHopIP netip.Addr) error

	GetNat(ctx context.Context, interfaceID string) (*api.Nat, error)
	CreateNat(ctx context.Context, nat *api.Nat) (*api.Nat, error)
}

type client struct {
	dpdkproto.DPDKonmetalClient
}

func NewClient(protoClient dpdkproto.DPDKonmetalClient) Client {
	return &client{protoClient}
}

func (c *client) GetLoadBalancer(ctx context.Context, id string) (*api.LoadBalancer, error) {
	res, err := c.DPDKonmetalClient.GetLoadBalancer(ctx, &dpdkproto.GetLoadBalancerRequest{LoadBalancerID: []byte(id)})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetStatus().GetError(); errorCode != 0 {
		return nil, apierrors.NewStatusError(errorCode, res.GetStatus().GetMessage())
	}
	lb, err := api.ProtoLoadBalancerToLoadBalancer(res, id)
	return lb, err
}

func (c *client) CreateLoadBalancer(ctx context.Context, lb *api.LoadBalancer) (*api.LoadBalancer, error) {
	var lbPorts = make([]*dpdkproto.LBPort, 0, len(lb.Spec.Lbports))
	for _, p := range lb.Spec.Lbports {
		lbPort := &dpdkproto.LBPort{Port: p.Port, Protocol: dpdkproto.Protocol(p.Protocol)}
		lbPorts = append(lbPorts, lbPort)
	}
	res, err := c.DPDKonmetalClient.CreateLoadBalancer(ctx, &dpdkproto.CreateLoadBalancerRequest{
		LoadBalancerID: []byte(lb.LoadBalancerMeta.ID),
		Vni:            lb.Spec.VNI,
		LbVipIP:        api.LbipToProtoLbip(lb.Spec.LbVipIP),
		Lbports:        lbPorts,
	})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetStatus().GetError(); errorCode != 0 {
		return nil, apierrors.NewStatusError(errorCode, res.GetStatus().GetMessage())
	}

	underlayRoute, err := netip.ParseAddr(string(res.GetUnderlayRoute()))
	if err != nil {
		return nil, fmt.Errorf("error parsing underlay route: %w", err)
	}
	lb.Spec.UnderlayRoute = underlayRoute

	return &api.LoadBalancer{
		TypeMeta:         api.TypeMeta{Kind: api.LoadBalancerKind},
		LoadBalancerMeta: lb.LoadBalancerMeta,
		Spec:             lb.Spec,
		Status: api.LoadBalancerStatus{
			Error:   res.Status.Error,
			Message: res.Status.Message,
		},
	}, nil
}

func (c *client) DeleteLoadBalancer(ctx context.Context, id string) error {
	res, err := c.DPDKonmetalClient.DeleteLoadBalancer(ctx, &dpdkproto.DeleteLoadBalancerRequest{LoadBalancerID: []byte(id)})
	if err != nil {
		return err
	}
	if errorCode := res.GetError(); errorCode != 0 {
		return apierrors.NewStatusError(errorCode, res.GetMessage())
	}
	return nil
}

func (c *client) ListLoadBalancerPrefixes(ctx context.Context, interfaceID string) (*api.PrefixList, error) {
	res, err := c.DPDKonmetalClient.ListInterfaceLoadBalancerPrefixes(ctx, &dpdkproto.ListInterfaceLoadBalancerPrefixesRequest{
		InterfaceID: []byte(interfaceID),
	})
	if err != nil {
		return nil, err
	}

	prefixes := make([]api.Prefix, len(res.GetPrefixes()))
	for i, dpdkPrefix := range res.GetPrefixes() {
		prefix, err := api.ProtoPrefixToPrefix(interfaceID, api.ProtoLBPrefixToProtoPrefix(*dpdkPrefix))
		if err != nil {
			return nil, err
		}

		prefixes[i] = *prefix
	}

	return &api.PrefixList{
		TypeMeta: api.TypeMeta{Kind: api.PrefixListKind},
		Items:    prefixes,
	}, nil
}

func (c *client) CreateLoadBalancerPrefix(ctx context.Context, prefix *api.Prefix) (*api.Prefix, error) {
	res, err := c.DPDKonmetalClient.CreateInterfaceLoadBalancerPrefix(ctx, &dpdkproto.CreateInterfaceLoadBalancerPrefixRequest{
		InterfaceID: &dpdkproto.InterfaceIDMsg{
			InterfaceID: []byte(prefix.InterfaceID),
		},
		Prefix: &dpdkproto.Prefix{
			IpVersion:    api.NetIPAddrToProtoIPVersion(prefix.Prefix.Addr()),
			Address:      []byte(prefix.Prefix.Addr().String()),
			PrefixLength: uint32(prefix.Prefix.Bits()),
		},
	})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetStatus().GetError(); errorCode != 0 {
		return nil, apierrors.NewStatusError(errorCode, res.GetStatus().GetMessage())
	}
	return &api.Prefix{
		TypeMeta:   api.TypeMeta{Kind: api.PrefixKind},
		PrefixMeta: prefix.PrefixMeta,
		Spec:       prefix.Spec,
	}, nil
}

func (c *client) DeleteLoadBalancerPrefix(ctx context.Context, interfaceID string, prefix netip.Prefix) error {
	res, err := c.DPDKonmetalClient.DeleteInterfaceLoadBalancerPrefix(ctx, &dpdkproto.DeleteInterfaceLoadBalancerPrefixRequest{
		InterfaceID: &dpdkproto.InterfaceIDMsg{
			InterfaceID: []byte(interfaceID),
		},
		Prefix: &dpdkproto.Prefix{
			IpVersion:    api.NetIPAddrToProtoIPVersion(prefix.Addr()),
			Address:      []byte(prefix.Addr().String()),
			PrefixLength: uint32(prefix.Bits()),
		},
	})
	if err != nil {
		return err
	}
	if errorCode := res.GetError(); errorCode != 0 {
		return apierrors.NewStatusError(errorCode, res.GetMessage())
	}
	return nil
}

func (c *client) GetLoadBalancerTargets(ctx context.Context, loadBalancerID string) (*api.LoadBalancerTargetList, error) {
	res, err := c.DPDKonmetalClient.GetLoadBalancerTargets(ctx, &dpdkproto.GetLoadBalancerTargetsRequest{
		LoadBalancerID: []byte(loadBalancerID),
	})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetStatus().GetError(); errorCode != 0 {
		return nil, apierrors.NewStatusError(errorCode, res.GetStatus().GetMessage())
	}

	lbtargets := make([]api.LoadBalancerTarget, len(res.GetTargetIPs()))
	for i, dpdkLBtarget := range res.GetTargetIPs() {
		var lbtarget api.LoadBalancerTarget
		lbtarget.TypeMeta.Kind = api.LoadBalancerTargetKind
		lbtarget.Spec.TargetIP = *api.ProtoLbipToLbip(*dpdkLBtarget)
		lbtarget.LoadBalancerTargetMeta.ID = loadBalancerID

		lbtargets[i] = lbtarget
	}

	return &api.LoadBalancerTargetList{
		TypeMeta: api.TypeMeta{Kind: api.LoadBalancerTargetListKind},
		Items:    lbtargets,
	}, nil
}

func (c *client) CreateLoadBalancerTarget(ctx context.Context, lbtarget *api.LoadBalancerTarget) (*api.LoadBalancerTarget, error) {
	res, err := c.DPDKonmetalClient.AddLoadBalancerTarget(ctx, &dpdkproto.AddLoadBalancerTargetRequest{
		LoadBalancerID: []byte(lbtarget.LoadBalancerTargetMeta.ID),
		TargetIP:       api.LbipToProtoLbip(lbtarget.Spec.TargetIP.Address),
	})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetError(); errorCode != 0 {
		return nil, apierrors.NewStatusError(errorCode, res.GetMessage())
	}

	return &api.LoadBalancerTarget{
		TypeMeta:               api.TypeMeta{Kind: api.LoadBalancerTargetKind},
		LoadBalancerTargetMeta: lbtarget.LoadBalancerTargetMeta,
		Spec:                   lbtarget.Spec,
	}, nil
}

func (c *client) DeleteLoadBalancerTarget(ctx context.Context, id string, targetIP netip.Addr) error {
	res, err := c.DPDKonmetalClient.DeleteLoadBalancerTarget(ctx, &dpdkproto.DeleteLoadBalancerTargetRequest{
		LoadBalancerID: []byte(id),
		TargetIP:       api.LbipToProtoLbip(targetIP),
	})
	if err != nil {
		return err
	}
	if errorCode := res.GetError(); errorCode != 0 {
		return apierrors.NewStatusError(errorCode, res.GetMessage())
	}
	return nil
}

func (c *client) GetInterface(ctx context.Context, name string) (*api.Interface, error) {
	res, err := c.DPDKonmetalClient.GetInterface(ctx, &dpdkproto.InterfaceIDMsg{InterfaceID: []byte(name)})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetStatus().GetError(); errorCode != 0 {
		return nil, apierrors.NewStatusError(errorCode, res.GetStatus().GetMessage())
	}
	return api.ProtoInterfaceToInterface(res.GetInterface())
}

func (c *client) ListInterfaces(ctx context.Context) (*api.InterfaceList, error) {
	res, err := c.DPDKonmetalClient.ListInterfaces(ctx, &dpdkproto.Empty{})
	if err != nil {
		return nil, err
	}

	ifaces := make([]api.Interface, len(res.GetInterfaces()))
	for i, dpdkIface := range res.GetInterfaces() {
		iface, err := api.ProtoInterfaceToInterface(dpdkIface)
		if err != nil {
			return nil, err
		}

		ifaces[i] = *iface
	}

	return &api.InterfaceList{
		TypeMeta: api.TypeMeta{Kind: api.InterfaceListKind},
		Items:    ifaces,
	}, nil
}

func (c *client) CreateInterface(ctx context.Context, iface *api.Interface) (*api.Interface, error) {
	res, err := c.DPDKonmetalClient.CreateInterface(ctx, &dpdkproto.CreateInterfaceRequest{
		InterfaceType: dpdkproto.InterfaceType_VirtualInterface,
		InterfaceID:   []byte(iface.ID),
		Vni:           iface.Spec.VNI,
		Ipv4Config:    api.NetIPAddrToProtoIPConfig(netiputil.FindIPv4(iface.Spec.IPs)),
		Ipv6Config:    api.NetIPAddrToProtoIPConfig(netiputil.FindIPv6(iface.Spec.IPs)),
		DeviceName:    iface.Spec.Device,
	})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetResponse().GetStatus().GetError(); errorCode != 0 {
		return nil, apierrors.NewStatusError(errorCode, res.GetResponse().GetStatus().GetMessage())
	}

	underlayIP, err := netip.ParseAddr(string(res.GetResponse().GetUnderlayRoute()))
	if err != nil {
		return nil, fmt.Errorf("error parsing underlay route: %w", err)
	}

	return &api.Interface{
		TypeMeta:      api.TypeMeta{Kind: api.InterfaceKind},
		InterfaceMeta: iface.InterfaceMeta,
		Spec:          iface.Spec, // TODO: Enable dynamic device allocation
		Status: api.InterfaceStatus{
			UnderlayIP: underlayIP,
		},
	}, nil
}

func (c *client) DeleteInterface(ctx context.Context, name string) error {
	res, err := c.DPDKonmetalClient.DeleteInterface(ctx, &dpdkproto.InterfaceIDMsg{InterfaceID: []byte(name)})
	if err != nil {
		return err
	}
	if errorCode := res.GetError(); errorCode != 0 {
		return apierrors.NewStatusError(errorCode, res.GetMessage())
	}
	return nil
}

func (c *client) GetVirtualIP(ctx context.Context, interfaceName string) (*api.VirtualIP, error) {
	res, err := c.DPDKonmetalClient.GetInterfaceVIP(ctx, &dpdkproto.InterfaceIDMsg{
		InterfaceID: []byte(interfaceName),
	})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetStatus().GetError(); errorCode != 0 {
		return nil, apierrors.NewStatusError(errorCode, res.GetStatus().GetMessage())
	}

	return api.ProtoVirtualIPToVirtualIP(interfaceName, res)
}

func (c *client) CreateVirtualIP(ctx context.Context, virtualIP *api.VirtualIP) (*api.VirtualIP, error) {
	res, err := c.DPDKonmetalClient.AddInterfaceVIP(ctx, &dpdkproto.InterfaceVIPMsg{
		InterfaceID: []byte(virtualIP.InterfaceID),
		InterfaceVIPIP: &dpdkproto.InterfaceVIPIP{
			IpVersion: api.NetIPAddrToProtoIPVersion(virtualIP.IP),
			Address:   []byte(virtualIP.IP.String()),
		},
	})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetStatus().GetError(); errorCode != 0 {
		return nil, apierrors.NewStatusError(errorCode, res.GetStatus().GetMessage())
	}

	return &api.VirtualIP{
		TypeMeta: api.TypeMeta{Kind: api.VirtualIPKind},
		Spec:     virtualIP.Spec,
	}, nil
}

func (c *client) DeleteVirtualIP(ctx context.Context, interfaceID string) error {
	res, err := c.DPDKonmetalClient.DeleteInterfaceVIP(ctx, &dpdkproto.InterfaceIDMsg{
		InterfaceID: []byte(interfaceID),
	})
	if err != nil {
		return err
	}
	if errorCode := res.GetError(); errorCode != 0 {
		return apierrors.NewStatusError(errorCode, res.GetMessage())
	}
	return nil
}

func (c *client) ListPrefixes(ctx context.Context, interfaceID string) (*api.PrefixList, error) {
	res, err := c.DPDKonmetalClient.ListInterfacePrefixes(ctx, &dpdkproto.InterfaceIDMsg{
		InterfaceID: []byte(interfaceID),
	})
	if err != nil {
		return nil, err
	}

	prefixes := make([]api.Prefix, len(res.GetPrefixes()))
	for i, dpdkPrefix := range res.GetPrefixes() {
		prefix, err := api.ProtoPrefixToPrefix(interfaceID, dpdkPrefix)
		if err != nil {
			return nil, err
		}

		prefixes[i] = *prefix
	}

	return &api.PrefixList{
		TypeMeta: api.TypeMeta{Kind: api.PrefixListKind},
		Items:    prefixes,
	}, nil
}

func (c *client) CreatePrefix(ctx context.Context, prefix *api.Prefix) (*api.Prefix, error) {
	res, err := c.DPDKonmetalClient.AddInterfacePrefix(ctx, &dpdkproto.InterfacePrefixMsg{
		InterfaceID: &dpdkproto.InterfaceIDMsg{
			InterfaceID: []byte(prefix.InterfaceID),
		},
		Prefix: &dpdkproto.Prefix{
			IpVersion:    api.NetIPAddrToProtoIPVersion(prefix.Prefix.Addr()),
			Address:      []byte(prefix.Prefix.Addr().String()),
			PrefixLength: uint32(prefix.Prefix.Bits()),
		},
	})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetStatus().GetError(); errorCode != 0 {
		return nil, apierrors.NewStatusError(errorCode, res.GetStatus().GetMessage())
	}
	return &api.Prefix{
		TypeMeta:   api.TypeMeta{Kind: api.PrefixKind},
		PrefixMeta: prefix.PrefixMeta,
		Spec:       prefix.Spec,
	}, nil
}

func (c *client) DeletePrefix(ctx context.Context, interfaceID string, prefix netip.Prefix) error {
	res, err := c.DPDKonmetalClient.DeleteInterfacePrefix(ctx, &dpdkproto.InterfacePrefixMsg{
		InterfaceID: &dpdkproto.InterfaceIDMsg{
			InterfaceID: []byte(interfaceID),
		},
		Prefix: &dpdkproto.Prefix{
			IpVersion:    api.NetIPAddrToProtoIPVersion(prefix.Addr()),
			Address:      []byte(prefix.Addr().String()),
			PrefixLength: uint32(prefix.Bits()),
		},
	})
	if err != nil {
		return err
	}
	if errorCode := res.GetError(); errorCode != 0 {
		return apierrors.NewStatusError(errorCode, res.GetMessage())
	}
	return nil
}

func (c *client) CreateRoute(ctx context.Context, route *api.Route) (*api.Route, error) {
	res, err := c.DPDKonmetalClient.AddRoute(ctx, &dpdkproto.VNIRouteMsg{
		Vni: &dpdkproto.VNIMsg{Vni: route.VNI},
		Route: &dpdkproto.Route{
			IpVersion: api.NetIPAddrToProtoIPVersion(route.NextHop.IP),
			Weight:    100,
			Prefix: &dpdkproto.Prefix{
				IpVersion:    api.NetIPAddrToProtoIPVersion(route.Prefix.Addr()),
				Address:      []byte(route.Prefix.String()),
				PrefixLength: uint32(route.Prefix.Bits()),
			},
			NexthopVNI:     route.NextHop.VNI,
			NexthopAddress: []byte(route.NextHop.IP.String()),
		},
	})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetError(); errorCode != 0 {
		return nil, apierrors.NewStatusError(errorCode, res.GetMessage())
	}
	return &api.Route{
		TypeMeta: api.TypeMeta{Kind: api.RouteKind},
		Spec:     route.Spec,
	}, nil
}

func (c *client) DeleteRoute(ctx context.Context, vni uint32, prefix netip.Prefix, nextHopVNI uint32, nextHopIP netip.Addr) error {
	res, err := c.DPDKonmetalClient.DeleteRoute(ctx, &dpdkproto.VNIRouteMsg{
		Vni: &dpdkproto.VNIMsg{Vni: vni},
		Route: &dpdkproto.Route{
			IpVersion: api.NetIPAddrToProtoIPVersion(nextHopIP),
			Weight:    100,
			Prefix: &dpdkproto.Prefix{
				IpVersion:    api.NetIPAddrToProtoIPVersion(prefix.Addr()),
				Address:      []byte(prefix.String()),
				PrefixLength: uint32(prefix.Bits()),
			},
			NexthopVNI:     nextHopVNI,
			NexthopAddress: []byte(nextHopIP.String()),
		},
	})
	if err != nil {
		return err
	}
	if errorCode := res.GetError(); errorCode != 0 {
		return apierrors.NewStatusError(errorCode, res.GetMessage())
	}
	return nil
}

func (c *client) ListRoutes(ctx context.Context, vni uint32) (*api.RouteList, error) {
	res, err := c.DPDKonmetalClient.ListRoutes(ctx, &dpdkproto.VNIMsg{
		Vni: vni,
	})
	if err != nil {
		return nil, err
	}

	routes := make([]api.Route, len(res.GetRoutes()))
	for i, dpdkRoute := range res.GetRoutes() {
		route, err := api.ProtoRouteToRoute(vni, dpdkRoute)
		if err != nil {
			return nil, err
		}

		routes[i] = *route
	}

	return &api.RouteList{
		TypeMeta: api.TypeMeta{Kind: api.RouteListKind},
		Items:    routes,
	}, nil
}

func (c *client) GetNat(ctx context.Context, interfaceID string) (*api.Nat, error) {
	res, err := c.DPDKonmetalClient.GetNAT(ctx, &dpdkproto.GetNATRequest{InterfaceID: []byte(interfaceID)})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetStatus().GetError(); errorCode != 0 {
		return nil, apierrors.NewStatusError(errorCode, res.GetStatus().GetMessage())
	}
	nat, err := api.ProtoNatToNat(res, interfaceID)
	return nat, err
}

func (c *client) CreateNat(ctx context.Context, nat *api.Nat) (*api.Nat, error) {
	res, err := c.DPDKonmetalClient.AddNAT(ctx, &dpdkproto.AddNATRequest{
		InterfaceID: []byte(nat.NatMeta.InterfaceID),
		NatVIPIP: &dpdkproto.NATIP{
			IpVersion: api.NetIPAddrToProtoIPVersion(nat.Spec.NatVIPIP),
			Address:   []byte(nat.Spec.NatVIPIP.String()),
		},
		MinPort: nat.Spec.MinPort,
		MaxPort: nat.Spec.MaxPort,
	})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetStatus().GetError(); errorCode != 0 {
		return nil, apierrors.NewStatusError(errorCode, res.GetStatus().GetMessage())
	}

	underlayRoute, err := netip.ParseAddr(string(res.GetUnderlayRoute()))
	if err != nil {
		return nil, fmt.Errorf("error parsing underlay route: %w", err)
	}
	nat.Spec.UnderlayRoute = underlayRoute

	return &api.Nat{
		TypeMeta: api.TypeMeta{Kind: api.NatKind},
		NatMeta:  nat.NatMeta,
		Spec:     nat.Spec,
		Status: api.NatStatus{
			Error:   res.Status.Error,
			Message: res.Status.Message,
		},
	}, nil
}

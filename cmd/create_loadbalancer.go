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

package cmd

import (
	"context"
	"fmt"
	"net/netip"
	"os"

	"github.com/onmetal/dpservice-cli/dpdk/api"
	"github.com/onmetal/dpservice-cli/dpdk/api/errors"
	"github.com/onmetal/dpservice-cli/flag"
	"github.com/onmetal/dpservice-cli/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func CreateLoadBalancer(dpdkClientFactory DPDKClientFactory, rendererFactory RendererFactory) *cobra.Command {
	var (
		opts CreateLoadBalancerOptions
	)

	cmd := &cobra.Command{
		Use:     "loadbalancer <--id> <--vni> <--vip> <--lbports>",
		Short:   "Create a loadbalancer",
		Example: "dpservice-cli add loadbalancer --id=4 --vni=100 --vip=10.20.30.40 --lbports=TCP/443,UDP/53",
		Aliases: LoadBalancerAliases,
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunCreateLoadBalancer(
				cmd.Context(),
				dpdkClientFactory,
				rendererFactory,
				opts,
			)
		},
	}

	opts.AddFlags(cmd.Flags())

	util.Must(opts.MarkRequiredFlags(cmd))

	return cmd
}

type CreateLoadBalancerOptions struct {
	Id      string
	VNI     uint32
	LbVipIP netip.Addr
	Lbports []string
}

func (o *CreateLoadBalancerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Id, "id", o.Id, "Loadbalancer ID to add.")
	fs.Uint32Var(&o.VNI, "vni", o.VNI, "VNI to add the loadbalancer to.")
	flag.AddrVar(fs, &o.LbVipIP, "vip", o.LbVipIP, "VIP to assign to the loadbalancer.")
	fs.StringSliceVar(&o.Lbports, "lbports", o.Lbports, "LB ports to assign to the loadbalancer.")
}

func (o *CreateLoadBalancerOptions) MarkRequiredFlags(cmd *cobra.Command) error {
	for _, name := range []string{"id", "vni", "vip", "lbports"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			return err
		}
	}
	return nil
}

func RunCreateLoadBalancer(ctx context.Context, dpdkClientFactory DPDKClientFactory, rendererFactory RendererFactory, opts CreateLoadBalancerOptions) error {
	client, cleanup, err := dpdkClientFactory.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("error creating dpdk client: %w", err)
	}
	defer DpdkClose(cleanup)

	var ports = make([]api.LBPort, 0, len(opts.Lbports))
	for _, p := range opts.Lbports {
		port, err := api.StringLbportToLbport(p)
		if err != nil {
			return fmt.Errorf("error converting port: %w", err)
		}
		ports = append(ports, port)
	}

	lb, err := client.CreateLoadBalancer(ctx, &api.LoadBalancer{
		LoadBalancerMeta: api.LoadBalancerMeta{
			ID: opts.Id,
		},
		Spec: api.LoadBalancerSpec{
			VNI:     opts.VNI,
			LbVipIP: &opts.LbVipIP,
			Lbports: ports,
		},
	})
	if err != nil && err != errors.ErrServerError {
		return fmt.Errorf("error adding loadbalancer: %w", err)
	}

	lb.TypeMeta.Kind = api.LoadBalancerKind
	lb.LoadBalancerMeta.ID = opts.Id
	return rendererFactory.RenderObject(fmt.Sprintf("added, underlay route: %s", lb.Spec.UnderlayRoute), os.Stdout, lb)
}

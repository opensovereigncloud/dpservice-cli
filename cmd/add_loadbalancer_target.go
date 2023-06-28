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
	"strings"

	"github.com/onmetal/dpservice-cli/flag"
	"github.com/onmetal/dpservice-cli/util"
	"github.com/onmetal/net-dpservice-go/api"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func AddLoadBalancerTarget(
	dpdkClientFactory DPDKClientFactory,
	rendererFactory RendererFactory,
) *cobra.Command {
	var (
		opts AddLoadBalancerTargetOptions
	)

	cmd := &cobra.Command{
		Use:     "lbtarget <target-ip> <--lb-id>",
		Short:   "Add a loadbalancer target",
		Example: "dpservice-cli add lbtarget --target-ip=ff80::5 --lb-id=2",
		Args:    cobra.ExactArgs(0),
		Aliases: LoadBalancerTargetAliases,
		RunE: func(cmd *cobra.Command, args []string) error {

			return RunAddLoadBalancerTarget(
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

type AddLoadBalancerTargetOptions struct {
	TargetIP       netip.Addr
	LoadBalancerID string
}

func (o *AddLoadBalancerTargetOptions) AddFlags(fs *pflag.FlagSet) {
	flag.AddrVar(fs, &o.TargetIP, "target-ip", o.TargetIP, "Loadbalancer Target IP.")
	fs.StringVar(&o.LoadBalancerID, "lb-id", o.LoadBalancerID, "ID of the loadbalancer to add the target for.")
}

func (o *AddLoadBalancerTargetOptions) MarkRequiredFlags(cmd *cobra.Command) error {
	for _, name := range []string{"target-ip", "lb-id"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			return err
		}
	}
	return nil
}

func RunAddLoadBalancerTarget(
	ctx context.Context,
	dpdkClientFactory DPDKClientFactory,
	rendererFactory RendererFactory,
	opts AddLoadBalancerTargetOptions,
) error {
	client, cleanup, err := dpdkClientFactory.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("error creating dpdk client: %w", err)
	}
	defer DpdkClose(cleanup)

	lbtarget, err := client.AddLoadBalancerTarget(ctx, &api.LoadBalancerTarget{
		TypeMeta:               api.TypeMeta{Kind: api.LoadBalancerTargetKind},
		LoadBalancerTargetMeta: api.LoadBalancerTargetMeta{LoadbalancerID: opts.LoadBalancerID},
		Spec:                   api.LoadBalancerTargetSpec{TargetIP: &opts.TargetIP},
	})
	if err != nil && !strings.Contains(err.Error(), "error code") {
		return fmt.Errorf("error adding loadbalancer target: %w", err)
	}

	return rendererFactory.RenderObject("added", os.Stdout, lbtarget)
}

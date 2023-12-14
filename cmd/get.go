// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func Get(factory DPDKClientFactory) *cobra.Command {
	rendererOptions := &RendererOptions{Output: "table"}

	cmd := &cobra.Command{
		Use:  "get",
		Args: cobra.NoArgs,
		RunE: SubcommandRequired,
	}

	rendererOptions.AddFlags(cmd.PersistentFlags())

	subcommands := []*cobra.Command{
		GetInterface(factory, rendererOptions),
		GetVirtualIP(factory, rendererOptions),
		GetLoadBalancer(factory, rendererOptions),
		GetNat(factory, rendererOptions),
		GetFirewallRule(factory, rendererOptions),
		GetVni(factory, rendererOptions),
		GetVersion(factory, rendererOptions),
		GetInit(factory, rendererOptions),
	}

	cmd.Short = fmt.Sprintf("Gets one of %v", CommandNames(subcommands))
	cmd.Long = fmt.Sprintf("Gets one of %v", CommandNames(subcommands))

	cmd.AddCommand(
		subcommands...,
	)

	return cmd
}

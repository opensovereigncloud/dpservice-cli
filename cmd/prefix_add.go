package cmd

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"time"

	dpdkproto "github.com/onmetal/net-dpservice-go/proto"
	"github.com/spf13/cobra"
)

// addPrefixCmd represents the prefix add command
var addPrefixCmd = &cobra.Command{
	Use: "add",
	Run: func(cmd *cobra.Command, args []string) {
		client, closer := getDpClient(cmd)
		defer closer.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		machinId, err := cmd.Flags().GetString("interface_id")
		if err != nil {
			fmt.Println("Err:", err)
			os.Exit(1)
		}

		ipv4, err := cmd.Flags().GetString("ipv4")
		if err != nil {
			fmt.Println("Err:", err)
			os.Exit(1)
		}
		ipv6, err := cmd.Flags().GetString("ipv6")
		if err != nil {
			fmt.Println("Err:", err)
			os.Exit(1)
		}
		prefix := &dpdkproto.Prefix{}

		var ipPrefix netip.Prefix
		if ipv4 != "" {
			ipPrefix, err = netip.ParsePrefix(ipv4)
			prefix.IpVersion = dpdkproto.IPVersion_IPv4
		} else {
			ipPrefix, err = netip.ParsePrefix(ipv6)
			prefix.IpVersion = dpdkproto.IPVersion_IPv6
		}

		if err != nil {
			fmt.Println("Err:", err)
			os.Exit(1)
		}

		prefix.Address = []byte(ipPrefix.Addr().String())
		prefix.PrefixLength = uint32(ipPrefix.Bits())
		reg := &dpdkproto.InterfacePrefixMsg{
			InterfaceID: &dpdkproto.InterfaceIDMsg{
				InterfaceID: []byte(machinId),
			},
			Prefix: prefix,
		}

		msg, err := client.AddInterfacePrefix(ctx, reg)
		if err != nil {
			fmt.Println("Err:", err)
			os.Exit(1)
		}
		fmt.Println("AddInterfacePrefix", msg)

	},
}

func init() {
	prefixCmd.AddCommand(addPrefixCmd)

	addPrefixCmd.Flags().StringP("interface_id", "i", "", "")
	addPrefixCmd.Flags().String("ipv4", "", "192.168.1.1/32")
	addPrefixCmd.Flags().String("ipv6", "", "")

	_ = addPrefixCmd.MarkFlagRequired("interface_id")
}

package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	dpdkproto "github.com/onmetal/net-dpservice-go/proto"

	"github.com/spf13/cobra"
)

// delInterfaceCmd represents the machine del command
var delInterfaceCmd = &cobra.Command{
	Use: "del",
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

		req := &dpdkproto.InterfaceIDMsg{
			InterfaceID: []byte(machinId),
		}

		msg, err := client.DeleteInterface(ctx, req)
		if err != nil {
			fmt.Println("Err:", err)
			os.Exit(1)
		}
		fmt.Println("DeleteInterface", msg)
	},
}

func init() {
	machineCmd.AddCommand(delInterfaceCmd)
	delInterfaceCmd.Flags().StringP("interface_id", "i", "", "")
}

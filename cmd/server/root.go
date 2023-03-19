package server

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/etiennemtl/etcd-mini-cluster/internal/driver"
	"github.com/etiennemtl/etcd-mini-cluster/pkg/node"
	"github.com/spf13/cobra"
)

func NewServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "serve",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := driver.NewDefaultRegistry(cmd.Context(), cmd.Flags())
			if err != nil {
				return err
			}

			n := node.NewNode(cmd.Context(), reg)
			err = n.Bootstrap(cmd.Context())
			if err != nil {
				return err
			}

			return n.Start(cmd.Context(), InterruptCh())
			// if err != nil {
			// 	return err
			// }

			// client, err := clientv3.New(clientv3.Config{
			// 	DialTimeout: 10 * time.Second,
			// 	Endpoints:   []string{cfg.TransportClientURL().String()},
			// })
			// if err != nil {
			// 	return err
			// }
			// defer client.Close()

			// members, err := client.MemberList(cmd.Context())
			// if err != nil {
			// 	return err
			// }

			// for _, member := range members.Members {
			// 	log.Printf("Member: %s", member.Name)
			// }

			// return nil
		},
	}

	return cmd
}

// InterruptCh returns channel which will get data when system receives interrupt signal.
func InterruptCh() <-chan interface{} {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ret := make(chan interface{}, 1)
	go func() {
		s := <-c
		ret <- s
		close(ret)
	}()

	return ret
}

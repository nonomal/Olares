package user

import (
	"context"
	"encoding/json"
	"fmt"
	iamv1alpha2 "github.com/beclab/api/iam/v1alpha2"
	"github.com/spf13/cobra"
	"log"
	"strconv"
	"time"
)

type listUsersOptions struct {
	kubeConfig string
	output     string
}

func NewCmdListUsers() *cobra.Command {
	o := &listUsersOptions{}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "list all users",
		Run: func(cmd *cobra.Command, args []string) {
			if err := o.Run(); err != nil {
				log.Fatal(err)
			}
		},
	}
	o.AddFlags(cmd)
	return cmd
}

func (o *listUsersOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.kubeConfig, "kubeconfig", "", "path to kubeconfig file")
	cmd.Flags().StringVarP(&o.output, "output", "o", "table", "output format (table, json)")
}

func (o *listUsersOptions) Run() error {
	ctx := context.Background()

	userClient, err := newUserClientFromKubeConfig(o.kubeConfig)
	if err != nil {
		return err
	}

	var userList iamv1alpha2.UserList
	err = userClient.List(ctx, &userList)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]userInfo, 0, len(userList.Items))
	for _, user := range userList.Items {
		info := userInfo{
			UID:               string(user.UID),
			Name:              user.Name,
			DisplayName:       user.Spec.DisplayName,
			Description:       user.Spec.Description,
			Email:             user.Spec.Email,
			State:             string(user.Status.State),
			CreationTimestamp: user.CreationTimestamp.Unix(),
		}

		if user.Annotations != nil {
			if role, ok := user.Annotations["bytetrade.io/owner-role"]; ok {
				info.Roles = []string{role}
			}
			if terminusName, ok := user.Annotations["bytetrade.io/terminus-name"]; ok {
				info.TerminusName = terminusName
			}
			if avatar, ok := user.Annotations["bytetrade.io/avatar"]; ok {
				info.Avatar = avatar
			}
			if memoryLimit, ok := user.Annotations["bytetrade.io/user-memory-limit"]; ok {
				info.MemoryLimit = memoryLimit
			}
			if cpuLimit, ok := user.Annotations["bytetrade.io/user-cpu-limit"]; ok {
				info.CpuLimit = cpuLimit
			}
		}

		if user.Status.LastLoginTime != nil {
			lastLogin := user.Status.LastLoginTime.Unix()
			info.LastLoginTime = &lastLogin
		}

		users = append(users, info)
	}

	if o.output == "json" {
		jsonOutput, _ := json.MarshalIndent(users, "", "  ")
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Printf("%-15s %-10s %-10s %-30s %-10s %-10s %-10s\n", "NAME", "ROLE", "STATE", "CREATE TIME", "ACTIVATED", "MEMORY", "CPU")
		for _, user := range users {
			role := "normal"
			if len(user.Roles) > 0 {
				role = user.Roles[0]
			}
			fmt.Printf("%-15s %-10s %-10s %-30s %-10s %-10s %-10s\n",
				user.Name, role, user.State, time.Unix(user.CreationTimestamp, 0).Format(time.RFC3339), strconv.FormatBool(user.WizardComplete), user.MemoryLimit, user.CpuLimit)
		}
	}

	return nil
}

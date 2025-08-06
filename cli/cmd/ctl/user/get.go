package user

import (
	"context"
	"encoding/json"
	"fmt"
	iamv1alpha2 "github.com/beclab/api/iam/v1alpha2"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"strconv"
	"time"
)

type getUserOptions struct {
	name       string
	kubeConfig string
	output     string
}

func NewCmdGetUser() *cobra.Command {
	o := &getUserOptions{}
	cmd := &cobra.Command{
		Use:   "get {name}",
		Short: "get user details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			o.name = args[0]
			if err := o.Validate(); err != nil {
				log.Fatal(err)
			}
			if err := o.Run(); err != nil {
				log.Fatal(err)
			}
		},
	}
	o.AddFlags(cmd)
	return cmd
}

func (o *getUserOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.kubeConfig, "kubeconfig", "", "path to kubeconfig file")
	cmd.Flags().StringVarP(&o.output, "output", "o", "table", "output format (table, json)")
}

func (o *getUserOptions) Validate() error {
	if o.name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (o *getUserOptions) Run() error {
	ctx := context.Background()

	userClient, err := newUserClientFromKubeConfig(o.kubeConfig)
	if err != nil {
		return err
	}

	var user iamv1alpha2.User
	err = userClient.Get(ctx, types.NamespacedName{Name: o.name}, &user)
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("user '%s' not found", o.name)
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

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

	if o.output == "json" {
		jsonOutput, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Printf("%-10s %-10s %-10s %-30s %-10s %-10s %-10s\n", "NAME", "ROLE", "STATE", "CREATE TIME", "ACTIVATED", "MEMORY", "CPU")
		role := "normal"
		if len(info.Roles) > 0 {
			role = info.Roles[0]
		}
		fmt.Printf("%-10s %-10s %-10s %-30s %-10s %-10s %-10s\n",
			info.Name, role, info.State, time.Unix(info.CreationTimestamp, 0).Format(time.RFC3339), strconv.FormatBool(info.WizardComplete), info.MemoryLimit, info.CpuLimit)
	}

	return nil
}

package user

import (
	"bytetrade.io/web3os/app-service/api/sys.bytetrade.io/v1alpha1"
	"context"
	"fmt"
	"github.com/beclab/Olares/cli/pkg/utils"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"log"
	"sort"
	"strings"

	iamv1alpha2 "github.com/beclab/api/iam/v1alpha2"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type createUserOptions struct {
	name        string
	displayName string
	domain      string
	role        string
	resourceLimit
	password    string
	description string
	kubeConfig  string
}

func NewCmdCreateUser() *cobra.Command {
	o := &createUserOptions{}
	cmd := &cobra.Command{
		Use:     "create {name}",
		Aliases: []string{"add", "new"},
		Short:   "create a new user",
		Args:    cobra.ExactArgs(1),
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

func (o *createUserOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.displayName, "display-name", "", "display name (optional)")
	cmd.Flags().StringVar(&o.domain, "domain", "", "domain (optional, defaults to the Olares system's domain)")
	cmd.Flags().StringVarP(&o.role, "role", "r", "normal", "owner role (optional, one of owner, admin, normal)")
	cmd.Flags().StringVarP(&o.memoryLimit, "memory-limit", "m", defaultMemoryLimit, "memory limit (optional)")
	cmd.Flags().StringVarP(&o.cpuLimit, "cpu-limit", "c", defaultCPULimit, "cpu limit (optional)")
	cmd.Flags().StringVarP(&o.password, "password", "p", "", "initial password (optional)")
	cmd.Flags().StringVar(&o.description, "description", "", "user description (optional)")
	cmd.Flags().StringVar(&o.kubeConfig, "kubeconfig", "", "path to kubeconfig file (optional)")
}

func (o *createUserOptions) Validate() error {
	if o.name == "" {
		return fmt.Errorf("name is required")
	}

	if errs := validation.IsDNS1123Subdomain(o.name); len(errs) > 0 {
		return fmt.Errorf("invalid name: %s", strings.Join(errs, ","))
	}

	if o.domain != "" {
		if errs := validation.IsDNS1123Subdomain(o.domain); len(errs) > 0 {
			return fmt.Errorf("invalid domain: %s", strings.Join(errs, ","))
		}
		if len(strings.Split(o.domain, ".")) < 2 {
			return errors.New("invalid domain: should be a domain with at least two segments separated by dots")
		}
		for _, label := range strings.Split(o.domain, ".") {
			if errs := validation.IsDNS1123Label(label); len(errs) > 0 {
				return fmt.Errorf("invalid domain: %s", strings.Join(errs, ","))
			}
		}
	}

	if o.role != "" {
		if o.role != "owner" && o.role != "admin" && o.role != "normal" {
			return fmt.Errorf("invalid role: should be one of owner, admin, or normal")
		}
	}

	if err := validateResourceLimit(o.resourceLimit); err != nil {
		return err
	}

	return nil
}

func (o *createUserOptions) Run() error {
	ctx := context.Background()
	userClient, err := newUserClientFromKubeConfig(o.kubeConfig)
	if err != nil {
		return err
	}

	if o.memoryLimit == "" {
		o.memoryLimit = defaultMemoryLimit
	}

	if o.cpuLimit == "" {
		o.cpuLimit = defaultCPULimit
	}

	if o.domain == "" {
		var system v1alpha1.Terminus
		err := userClient.Get(ctx, types.NamespacedName{Name: systemObjectName}, &system)
		if err != nil {
			return fmt.Errorf("failed to get system info: %v", err)
		}
		o.domain = system.Spec.Settings[systemObjectDomainKey]
	}

	var userList iamv1alpha2.UserList
	err = userClient.List(ctx, &userList)
	if err != nil {
		return fmt.Errorf("failed to list users to set creator: %w", err)
	}
	var owners []iamv1alpha2.User
	for _, user := range userList.Items {
		if role, ok := user.Annotations["bytetrade.io/owner-role"]; ok && role == "owner" {
			owners = append(owners, user)
		}
	}
	sort.Slice(owners, func(i, j int) bool {
		return owners[i].CreationTimestamp.Before(&owners[j].CreationTimestamp)
	})
	if len(owners) == 0 {
		return fmt.Errorf("no owners found")
	}
	creator := owners[0].Name

	if o.password == "" {
		password, passwordEncrypted, err := utils.GenerateEncryptedPassword(8)
		if err != nil {
			return fmt.Errorf("error generating password: %v", err)
		}
		log.Println("generated initial password:", password)
		o.password = passwordEncrypted
	} else {
		o.password = utils.MD5(o.password + "@Olares2025")
	}

	olaresName := fmt.Sprintf("%s@%s", o.name, o.domain)

	user := &iamv1alpha2.User{
		TypeMeta: metav1.TypeMeta{
			APIVersion: iamv1alpha2.SchemeGroupVersion.String(),
			Kind:       iamv1alpha2.ResourceKindUser,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: o.name,
			Annotations: map[string]string{
				"iam.kubesphere.io/uninitialized":    "true",
				"bytetrade.io/creator":               creator,
				"bytetrade.io/owner-role":            o.role,
				"bytetrade.io/is-ephemeral":          "true",
				"bytetrade.io/terminus-name":         olaresName,
				"bytetrade.io/launcher-auth-policy":  "two_factor",
				"bytetrade.io/launcher-access-level": "1",
				"bytetrade.io/user-memory-limit":     o.memoryLimit,
				"bytetrade.io/user-cpu-limit":        o.cpuLimit,
				"iam.kubesphere.io/sync-to-lldap":    "true",
				"iam.kubesphere.io/synced-to-lldap":  "false",
			},
		},
		Spec: iamv1alpha2.UserSpec{
			DisplayName:     o.displayName,
			Email:           olaresName,
			InitialPassword: o.password,
			Description:     o.description,
		},
	}

	if o.role == "owner" || o.role == "admin" {
		user.Spec.Groups = append(user.Spec.Groups, "lldap_admin")
	}

	err = userClient.Create(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("User '%s' created successfully\n", o.name)
	return nil
}

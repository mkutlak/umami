package main

import (
	"io/ioutil"

	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func readFileOrPanic(path string) pulumi.String {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err.Error())
	}
	return pulumi.String(string(data))
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		umamiConf := config.New(ctx, "")

		// * * * * * * * * * * * * * * * * * * * * * * * * * * * *
		// Create a VPC Umami Network
		umamiNetwork, err := compute.NewNetwork(ctx, "umami-private-network",
			&compute.NetworkArgs{
				AutoCreateSubnetworks: pulumi.Bool(true),
			},
		)
		if err != nil {
			return err
		}

		// * * * * * * * * * * * * * * * * * * * * * * * * * * * *
		// Allow internal comunication between the virtual machines
		const GCP_COMPUTE_RANGES string = "10.128.0.0/9"

		_, err = compute.NewFirewall(ctx, "umami-internal-firewall",
			&compute.FirewallArgs{
				Network:  umamiNetwork.SelfLink,
				Priority: pulumi.Int(900),
				SourceRanges: pulumi.StringArray{
					pulumi.String(GCP_COMPUTE_RANGES),
				},
				Allows: &compute.FirewallAllowArray{
					&compute.FirewallAllowArgs{
						Protocol: pulumi.String("icmp"),
					},
					&compute.FirewallAllowArgs{
						Protocol: pulumi.String("tcp"),
						Ports: pulumi.StringArray{
							pulumi.String("0-65535"),
						},
					},
					&compute.FirewallAllowArgs{
						Protocol: pulumi.String("udp"),
						Ports: pulumi.StringArray{
							pulumi.String("0-65535"),
						},
					},
				},
			},
		)
		if err != nil {
			return err
		}

		// * * * * * * * * * * * * * * * * * * * * * * * * * * * *
		// Allow ICMP and connections to ports (22, 30000-40000) from a single IP address.
		umamiFirewall, err := compute.NewFirewall(ctx, "umami-external-firewall",
			&compute.FirewallArgs{
				Network: umamiNetwork.SelfLink,
				//Priority: pulumi.Int(990),
				SourceRanges: pulumi.StringArray{
					pulumi.String(umamiConf.Require("ip-cidr")),
				},
				Allows: &compute.FirewallAllowArray{
					&compute.FirewallAllowArgs{
						Protocol: pulumi.String("icmp"),
					},
					&compute.FirewallAllowArgs{
						Protocol: pulumi.String("tcp"),
						Ports: pulumi.StringArray{
							pulumi.String("22"),
							pulumi.String("30000-40000"),
						},
					},
				},
			},
		)
		if err != nil {
			return err
		}

		// * * * * * * * * * * * * * * * * * * * * * * * * * * * *
		// Service Account
		umamiServiceAccount, err := serviceaccount.NewAccount(ctx, "umami-sa",
			&serviceaccount.AccountArgs{
				AccountId:   pulumi.String("umami-sa-id"),
				DisplayName: pulumi.String("Umami Service Account"),
			})
		if err != nil {
			return err
		}

		// * * * * * * * * * * * * * * * * * * * * * * * * * * * *
		// Two virtual machines used as Kubernetes nodes
		umamiPubKey := readFileOrPanic(umamiConf.Require("ssh-pubkey-path"))

		type InstanceConfig struct {
			name   string
			labels pulumi.StringMap
		}

		for _, e := range []InstanceConfig{
			// MASTER node instance config
			{
				name: "umami-master",
				labels: pulumi.StringMap{
					"env":  pulumi.String("cks"),
					"node": pulumi.String("master"),
				},
			},
			// WORKER node instance config
			{
				name: "umami-worker",
				labels: pulumi.StringMap{
					"env":  pulumi.String("cks"),
					"node": pulumi.String("worker"),
				},
			}} {
			_, err := compute.NewInstance(ctx, e.name,
				&compute.InstanceArgs{
					Zone:        pulumi.String(umamiConf.Require("zone")),
					MachineType: pulumi.String("e2-medium"),
					BootDisk: &compute.InstanceBootDiskArgs{
						InitializeParams: &compute.InstanceBootDiskInitializeParamsArgs{
							Image: pulumi.String("ubuntu-1804-lts"),
							Size:  pulumi.Int(50),
						},
					},
					NetworkInterfaces: compute.InstanceNetworkInterfaceArray{
						&compute.InstanceNetworkInterfaceArgs{
							Network: umamiNetwork.ID(),
							// Must be empty to request an ephemeral IP
							AccessConfigs: &compute.InstanceNetworkInterfaceAccessConfigArray{
								&compute.InstanceNetworkInterfaceAccessConfigArgs{},
							},
						},
					},
					Metadata: pulumi.StringMap{
						"block-project-ssh-keys": pulumi.String("yes"),
						"ssh-keys":               umamiPubKey,
					},
					ServiceAccount: &compute.InstanceServiceAccountArgs{
						Email: umamiServiceAccount.Email,
						Scopes: pulumi.StringArray{
							pulumi.String("cloud-platform"),
						},
					},
					Labels: e.labels,
				},
				pulumi.DependsOn([]pulumi.Resource{umamiFirewall}),
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

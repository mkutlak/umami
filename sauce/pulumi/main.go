package main

import (
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const GCP_REGION = ""

const GCP_PROJECT_NAME = ""

const MY_IP_RANGE = "0.0.0.0/0"

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create a UMAMI network
		umamiNetwork, err := compute.NewNetwork(ctx, "priv_umami_network",
			&compute.NetworkArgs{
				AutoCreateSubnetworks: pulumi.Bool(true),
			},
		)
		if err != nil {
			return err
		}

		const GCP_COMPUTE_RANGES string = "10.128.0.0/9"
		type FirewallConfig struct {
			name         string
			sourceRanges string
			description  string
		}

		umamiFirewall, err := compute.NewFirewall(ctx, "umami_firewall",
			&compute.FirewallArgs{
				Network:  umamiNetwork.SelfLink,
				Priority: pulumi.Int(990),
				SourceRanges: pulumi.StringArray{
					pulumi.String(MY_IP_RANGE),
				},
				Allows: &compute.FirewallAllowArray{
					&compute.FirewallAllowArgs{
						Protocol: pulumi.String("icmp"),
					},
					&compute.FirewallAllowArgs{
						Protocol: pulumi.String("tcp"),
						Ports: pulumi.StringArray{
							pulumi.String("22"),
						},
					},
					&compute.FirewallAllowArgs{
						Protocol: pulumi.String("tcp"),
						Ports: pulumi.StringArray{
							pulumi.String("30000-40000"),
						},
					},
				},
			},
		)
		if err != nil {
			return err
		}

		type InstanceConfig struct {
			name   string
			labels pulumi.StringMap
		}

		for _, e := range []InstanceConfig{
			// MASTER node instance config
			{
				name: "umami_master",
				labels: pulumi.StringMap{
					"env":  pulumi.String("cks"),
					"node": pulumi.String("master"),
				},
			},
			// WORKER node instance config
			{
				name: "umami_worker",
				labels: pulumi.StringMap{
					"env":  pulumi.String("cks"),
					"node": pulumi.String("worker"),
				},
			}} {
			_, err := compute.NewInstance(ctx, e.name,
				&compute.InstanceArgs{
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
					ServiceAccount: &compute.InstanceServiceAccountArgs{
						Scopes: pulumi.StringArray{
							pulumi.String("https://www.googleapis.com/auth/cloud-platform"),
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

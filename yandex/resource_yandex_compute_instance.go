package yandex

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mitchellh/hashstructure"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	"github.com/yandex-cloud/go-sdk/operation"
	"github.com/yandex-cloud/terraform-provider-yandex/common"
)

const (
	yandexComputeInstanceDefaultTimeout       = 5 * time.Minute
	yandexComputeInstanceDiskOperationTimeout = 5 * time.Minute
	yandexComputeInstanceDeallocationTimeout  = 15 * time.Second
	yandexComputeInstanceMoveTimeout          = 1 * time.Minute
)

func resourceYandexComputeInstance() *schema.Resource {
	return &schema.Resource{
		Description: "A VM instance resource. For more information, see [the official documentation](https://yandex.cloud/docs/compute/concepts/vm).\n",

		Create: resourceYandexComputeInstanceCreate,
		Read:   resourceYandexComputeInstanceRead,
		Update: resourceYandexComputeInstanceUpdate,
		Delete: resourceYandexComputeInstanceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(yandexComputeInstanceDefaultTimeout),
			Update: schema.DefaultTimeout(yandexComputeInstanceDefaultTimeout),
			Delete: schema.DefaultTimeout(yandexComputeInstanceDefaultTimeout),
		},

		SchemaVersion: 1,

		MigrateState: resourceComputeInstanceMigrateState,

		Schema: map[string]*schema.Schema{
			"resources": {
				Type:        schema.TypeList,
				Description: "Compute resources that are allocated for the instance.",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"memory": {
							Type:         schema.TypeFloat,
							Description:  "Memory size in GB.",
							Required:     true,
							ForceNew:     false,
							ValidateFunc: FloatAtLeast(0.0),
						},

						"cores": {
							Type:        schema.TypeInt,
							Description: "CPU cores for the instance.",
							Required:    true,
							ForceNew:    false,
						},

						"gpus": {
							Type:        schema.TypeInt,
							Description: "If provided, specifies the number of GPU devices for the instance.",
							Optional:    true,
							ForceNew:    false,
						},

						"core_fraction": {
							Type:        schema.TypeInt,
							Description: "If provided, specifies baseline performance for a core as a percent.",
							Optional:    true,
							ForceNew:    false,
							Default:     100,
						},
					},
				},
			},

			"boot_disk": {
				Type:        schema.TypeList,
				Description: "The boot disk for the instance. Either `initialize_params` or `disk_id` must be specified.",
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"auto_delete": {
							Type:        schema.TypeBool,
							Description: "Defines whether the disk will be auto-deleted when the instance is deleted. The default value is `True`.",
							Optional:    true,
							Default:     true,
							ForceNew:    true,
						},

						"device_name": {
							Type:        schema.TypeString,
							Description: "Name that can be used to access an attached disk.",
							Optional:    true,
							Computed:    true,
							ForceNew:    true,
						},

						"mode": {
							Type:        schema.TypeString,
							Description: "Type of access to the disk resource. By default, a disk is attached in `READ_WRITE` mode.",
							Optional:    true,
							Computed:    true,
						},

						"disk_id": {
							Type:          schema.TypeString,
							Description:   "The ID of the existing disk (such as those managed by `yandex_compute_disk`) to attach as a boot disk.",
							Optional:      true,
							Computed:      true,
							ForceNew:      true,
							ConflictsWith: []string{"boot_disk.initialize_params"},
						},

						"initialize_params": {
							Type:          schema.TypeList,
							Description:   "Parameters for a new disk that will be created alongside the new instance. Either `initialize_params` or `disk_id` must be set. Either `image_id` or `snapshot_id` must be specified.",
							Optional:      true,
							Computed:      true,
							ForceNew:      true,
							MaxItems:      1,
							ConflictsWith: []string{"boot_disk.disk_id"},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Description: "Name of the boot disk.",
										Optional:    true,
										Computed:    true,
										ForceNew:    true,
									},

									"description": {
										Type:        schema.TypeString,
										Description: "Description of the boot disk.",
										Optional:    true,
										Computed:    true,
										ForceNew:    true,
									},

									"size": {
										Type:         schema.TypeInt,
										Description:  "Size of the disk in GB.",
										Optional:     true,
										Computed:     true,
										ForceNew:     true,
										ValidateFunc: validation.IntAtLeast(1),
									},

									"block_size": {
										Type:        schema.TypeInt,
										Description: "Block size of the disk, specified in bytes.",
										Optional:    true,
										Computed:    true,
										ForceNew:    true,
									},

									"type": {
										Type:        schema.TypeString,
										Description: "Disk type.",
										Optional:    true,
										ForceNew:    true,
										Default:     "network-hdd",
									},

									"image_id": {
										Type:          schema.TypeString,
										Description:   "A disk image to initialize this disk from.",
										Optional:      true,
										Computed:      true,
										ForceNew:      true,
										ConflictsWith: []string{"boot_disk.initialize_params.snapshot_id"},
									},

									"snapshot_id": {
										Type:          schema.TypeString,
										Description:   "A snapshot to initialize this disk from.",
										Optional:      true,
										Computed:      true,
										ForceNew:      true,
										ConflictsWith: []string{"boot_disk.initialize_params.image_id"},
									},

									"kms_key_id": {
										Type:        schema.TypeString,
										Description: "ID of KMS symmetric key used to encrypt disk.",
										ForceNew:    true,
										Optional:    true,
									},
								},
							},
						},
					},
				},
			},

			"network_acceleration_type": {
				Type:         schema.TypeString,
				Description:  "Type of network acceleration. Can be `standard` or `software_accelerated`. The default is `standard`.",
				Optional:     true,
				Default:      "standard",
				ValidateFunc: validation.StringInSlice([]string{"standard", "software_accelerated"}, false),
			},

			"network_interface": {
				Type:        schema.TypeList,
				Description: "Networks to attach to the instance. This can be specified multiple times.",
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet_id": {
							Type:        schema.TypeString,
							Description: "ID of the subnet to attach this interface to. The subnet must exist in the same zone where this instance will be created.",
							Required:    true,
						},

						"ipv4": {
							Type:        schema.TypeBool,
							Description: "Allocate an IPv4 address for the interface. The default value is `true`.",
							Optional:    true,
							Default:     true,
						},

						"ip_address": {
							Type:        schema.TypeString,
							Description: "The private IP address to assign to the instance. If empty, the address will be automatically assigned from the specified subnet.",
							Optional:    true,
							Computed:    true,
						},

						"ipv6": {
							Type:        schema.TypeBool,
							Description: "If `true`, allocate an IPv6 address for the interface. The address will be automatically assigned from the specified subnet.",
							Optional:    true,
							Computed:    true,
						},

						"ipv6_address": {
							Type:        schema.TypeString,
							Description: "The private IPv6 address to assign to the instance.",
							Optional:    true,
							Computed:    true,
						},

						"nat": {
							Type:        schema.TypeBool,
							Description: "Provide a public address, for instance, to access the internet over NAT.",
							Optional:    true,
							Default:     false,
						},

						"index": {
							Type:        schema.TypeInt,
							Description: "Index of network interface, will be calculated automatically for instance create or update operations if not specified. Required for attach/detach operations.",
							Optional:    true,
							Computed:    true,
						},

						"mac_address": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"nat_ip_address": {
							Type:        schema.TypeString,
							Description: "Provide a public address, for instance, to access the internet over NAT. Address should be already reserved in web UI.",
							Optional:    true,
							Computed:    true,
						},

						"nat_ip_version": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"security_group_ids": {
							Type:        schema.TypeSet,
							Description: "Security Group (SG) IDs for network interface.",
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
							Optional:    true,
						},

						"dns_record": {
							Type:        schema.TypeList,
							Description: "List of configurations for creating ipv4 DNS records.",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"fqdn": {
										Type:        schema.TypeString,
										Description: "DNS record FQDN (must have a dot at the end).",
										Required:    true,
									},
									"dns_zone_id": {
										Type:        schema.TypeString,
										Description: "DNS zone ID (if not set, private zone used).",
										Optional:    true,
									},
									"ttl": {
										Type:        schema.TypeInt,
										Description: "DNS record TTL in seconds.",
										Optional:    true,
									},
									"ptr": {
										Type:        schema.TypeBool,
										Description: "When set to `true`, also create a PTR DNS record.",
										Optional:    true,
									},
								},
							},
						},

						"ipv6_dns_record": {
							Type:        schema.TypeList,
							Description: "List of configurations for creating ipv6 DNS records.",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"fqdn": {
										Type:        schema.TypeString,
										Description: "DNS record FQDN (must have a dot at the end).",
										Required:    true,
									},
									"dns_zone_id": {
										Type:        schema.TypeString,
										Description: "DNS zone ID (if not set, private zone used).",
										Optional:    true,
									},
									"ttl": {
										Type:        schema.TypeInt,
										Description: "DNS record TTL in seconds.",
										Optional:    true,
									},
									"ptr": {
										Type:        schema.TypeBool,
										Description: "When set to `true`, also create a PTR DNS record.",
										Optional:    true,
									},
								},
							},
						},

						"nat_dns_record": {
							Type:        schema.TypeList,
							Description: "List of configurations for creating ipv4 NAT DNS records.",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"fqdn": {
										Type:        schema.TypeString,
										Description: "DNS record FQDN (must have a dot at the end).",
										Required:    true,
									},
									"dns_zone_id": {
										Type:        schema.TypeString,
										Description: "DNS zone ID (if not set, private zone used).",
										Optional:    true,
									},
									"ttl": {
										Type:        schema.TypeInt,
										Description: "DNS record TTL in seconds.",
										Optional:    true,
									},
									"ptr": {
										Type:        schema.TypeBool,
										Description: "When set to `true`, also create a PTR DNS record.",
										Optional:    true,
									},
								},
							},
						},
					},
				},
			},

			"name": {
				Type:        schema.TypeString,
				Description: common.ResourceDescriptions["name"],
				Optional:    true,
				Default:     "",
			},

			"description": {
				Type:        schema.TypeString,
				Description: common.ResourceDescriptions["description"],
				Optional:    true,
			},

			"folder_id": {
				Type:        schema.TypeString,
				Description: common.ResourceDescriptions["folder_id"],
				Computed:    true,
				Optional:    true,
			},

			"labels": {
				Type:        schema.TypeMap,
				Description: common.ResourceDescriptions["labels"],
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
			},

			"zone": {
				Type:        schema.TypeString,
				Description: common.ResourceDescriptions["zone"],
				Computed:    true,
				Optional:    true,
				ForceNew:    true,
			},

			"hostname": {
				Type:             schema.TypeString,
				Description:      "Host name for the instance. This field is used to generate the instance `fqdn` value. The host name must be unique within the network and region. If not specified, the host name will be equal to `id` of the instance and `fqdn` will be `<id>.auto.internal`. Otherwise FQDN will be `<hostname>.<region_id>.internal`.",
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				DiffSuppressFunc: hostnameDiffSuppressFunc,
			},

			"metadata": {
				Type:        schema.TypeMap,
				Description: "Metadata key/value pairs to make available from within the instance.",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
			},

			"platform_id": {
				Type:        schema.TypeString,
				Description: "The type of virtual machine to create.",
				Optional:    true,
				ForceNew:    false,
				Default:     "standard-v1",
			},

			"allow_stopping_for_update": {
				Type:        schema.TypeBool,
				Description: "If `true`, allows Terraform to stop the instance in order to update its properties. If you try to update a property that requires stopping the instance without setting this field, the update will fail.",
				Optional:    true,
			},

			"allow_recreate": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"secondary_disk": {
				Type:        schema.TypeSet,
				Description: "A set of disks to attach to the instance. The structure is documented below.\n\n~> The [`allow_stopping_for_update`](#allow_stopping_for_update) property must be set to `true` in order to update this structure.",
				Set:         hashInstanceSecondaryDisks,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"disk_id": {
							Type:        schema.TypeString,
							Description: "ID of the disk that is attached to the instance.",
							Required:    true,
						},

						"auto_delete": {
							Type:        schema.TypeBool,
							Description: "Whether the disk is auto-deleted when the instance is deleted. The default value is `false`.",
							Optional:    true,
							Default:     false,
						},

						"device_name": {
							Type:        schema.TypeString,
							Description: "Name that can be used to access an attached disk under `/dev/disk/by-id/`.",
							Optional:    true,
							Computed:    true,
						},

						"mode": {
							Type:         schema.TypeString,
							Description:  "Type of access to the disk resource. By default, a disk is attached in `READ_WRITE` mode.",
							Optional:     true,
							Default:      "READ_WRITE",
							ValidateFunc: validation.StringInSlice([]string{"READ_WRITE", "READ_ONLY"}, false),
						},
					},
				},
			},

			"scheduling_policy": {
				Type:        schema.TypeList,
				Description: "Scheduling policy configuration.",
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"preemptible": {
							Type:        schema.TypeBool,
							Description: "Specifies if the instance is preemptible. Defaults to `false`.",
							Optional:    true,
							Default:     false,
						},
					},
				},
			},

			"service_account_id": {
				Type:        schema.TypeString,
				Description: common.ResourceDescriptions["service_account_id"],
				Computed:    true,
				Optional:    true,
			},

			"placement_policy": {
				Type:        schema.TypeList,
				Description: "The placement policy configuration.",
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"placement_group_id": {
							Type:        schema.TypeString,
							Description: "Specifies the id of the Placement Group to assign to the instance.",
							Optional:    true,
						},
						"placement_group_partition": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"host_affinity_rules": {
							Type:        schema.TypeList,
							Description: "List of host affinity rules.\n\n~> Due to terraform limitations, simply deleting the `placement_policy` fields does not work. To reset the values of these fields, you need to set them empty:\n\nplacement_policy {\n    placement_group_id = \"\"\n    host_affinity_rules = []\n}",
							Computed:    true,
							Optional:    true,
							ConfigMode:  schema.SchemaConfigModeAttr,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:        schema.TypeString,
										Description: "Affinity label or one of reserved values - `yc.hostId`, `yc.hostGroupId`.",
										Required:    true,
									},
									"op": {
										Type:        schema.TypeString,
										Description: "Affinity action. The only value supported is `IN`.",
										Required:    true,
										ValidateFunc: validation.StringInSlice(
											generateHostAffinityRuleOperators(), false),
									},
									"values": {
										Type:        schema.TypeList,
										Description: "List of values (host IDs or host group IDs).",
										Required:    true,
										MinItems:    1,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
					},
				},
			},

			"fqdn": {
				Type:        schema.TypeString,
				Description: "The fully qualified DNS name of this instance.",
				Computed:    true,
			},

			"status": {
				Type:        schema.TypeString,
				Description: "The status of this instance.",
				Computed:    true,
			},

			"created_at": {
				Type:        schema.TypeString,
				Description: common.ResourceDescriptions["created_at"],
				Computed:    true,
			},

			"local_disk": {
				Type:        schema.TypeList,
				Description: "List of local disks that are attached to the instance.\n\n~> Local disks are not available for all users by default.\n",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"size_bytes": {
							Type:        schema.TypeInt,
							Description: "Size of the disk, specified in bytes.",
							Required:    true,
							ForceNew:    true,
						},
						"device_name": {
							Type:        schema.TypeString,
							Description: "The name of the local disk device.",
							Computed:    true,
						},
					},
				},
			},

			"metadata_options": {
				Type:        schema.TypeList,
				Description: "Options allow user to configure access to instance's metadata.",
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"gce_http_endpoint": {
							Type:         schema.TypeInt,
							ValidateFunc: validation.IntBetween(0, 2),
							Optional:     true,
							Computed:     true,
						},
						"aws_v1_http_endpoint": {
							Type:         schema.TypeInt,
							ValidateFunc: validation.IntBetween(0, 2),
							Optional:     true,
							Computed:     true,
						},
						"gce_http_token": {
							Type:         schema.TypeInt,
							ValidateFunc: validation.IntBetween(0, 2),
							Optional:     true,
							Computed:     true,
						},
						"aws_v1_http_token": {
							Type:         schema.TypeInt,
							ValidateFunc: validation.IntBetween(0, 2),
							Optional:     true,
							Computed:     true,
						},
					},
				},
			},

			"filesystem": {
				Type:        schema.TypeSet,
				Description: "List of filesystems that are attached to the instance.",
				Optional:    true,
				Set:         hashFilesystem,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"filesystem_id": {
							Type:        schema.TypeString,
							Description: "ID of the filesystem that should be attached.",
							Required:    true,
						},

						"device_name": {
							Type:        schema.TypeString,
							Description: "Name of the device representing the filesystem on the instance.",
							Optional:    true,
							Computed:    true,
						},

						"mode": {
							Type:         schema.TypeString,
							Description:  "Mode of access to the filesystem that should be attached. By default, filesystem is attached in `READ_WRITE` mode.",
							Optional:     true,
							Default:      "READ_WRITE",
							ValidateFunc: validation.StringInSlice([]string{"READ_WRITE", "READ_ONLY"}, false),
						},
					},
				},
			},

			"gpu_cluster_id": {
				Type:        schema.TypeString,
				Description: "ID of the GPU cluster to attach this instance to.",
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},

			"maintenance_policy": {
				Type:        schema.TypeString,
				Description: "Behavior on maintenance events. Can be: `unspecified`, `migrate`, `restart`. The default is `unspecified`.",
				Optional:    true,
				Computed:    true,
			},
			"maintenance_grace_period": {
				Type:        schema.TypeString,
				Description: "Time between notification via metadata service and maintenance. E.g., `60s`.",
				Optional:    true,
				Computed:    true,
			},

			// Computed is true while Required and Optional are both false, for a read only field.
			"hardware_generation": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"generation2_features": {
							Type: schema.TypeList,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{},
							},
							Computed: true,
						},

						"legacy_features": {
							Type: schema.TypeList,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"pci_topology": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
							Computed: true,
						},
					},
				},
				Computed: true,
			},
		},
	}
}

func resourceYandexComputeInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	req, err := prepareCreateInstanceRequest(d, config)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(config.Context(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Compute().Instance().Create(ctx, req))
	if err != nil {
		return fmt.Errorf("Error while requesting API to create instance: %s", err)
	}

	protoMetadata, err := op.Metadata()
	if err != nil {
		return fmt.Errorf("Error while get instance create operation metadata: %s", err)
	}

	md, ok := protoMetadata.(*compute.CreateInstanceMetadata)
	if !ok {
		return fmt.Errorf("could not get Instance ID from create operation metadata")
	}

	d.SetId(md.InstanceId)

	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Error while waiting operation to create instance: %s", err)
	}

	if _, err := op.Response(); err != nil {
		return fmt.Errorf("Instance creation failed: %s", err)
	}

	return resourceYandexComputeInstanceRead(d, meta)
}

func resourceYandexComputeInstanceRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ctx, cancel := context.WithTimeout(config.Context(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	instance, err := config.sdk.Compute().Instance().Get(ctx, &compute.GetInstanceRequest{
		InstanceId: d.Id(),
		View:       compute.InstanceView_FULL,
	})

	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Instance %q", d.Get("name").(string)))
	}

	resources, err := flattenInstanceResources(instance)
	if err != nil {
		return err
	}

	bootDisk, err := flattenInstanceBootDisk(ctx, instance, config.sdk.Compute().Disk())
	if err != nil {
		return err
	}

	secondaryDisks, err := flattenInstanceSecondaryDisks(instance)
	if err != nil {
		return err
	}

	schedulingPolicy, err := flattenInstanceSchedulingPolicy(instance)
	if err != nil {
		return err
	}

	placementPolicy, err := flattenInstancePlacementPolicy(instance)
	if err != nil {
		return err
	}

	networkInterfaces, externalIP, internalIP, err := flattenInstanceNetworkInterfaces(instance)
	if err != nil {
		return err
	}

	localDisks := flattenLocalDisks(instance)

	metadataOptions := flattenInstanceMetadataOptions(instance)

	filesystems := flattenInstanceFilesystems(instance)

	hardwareGeneration, err := flattenComputeHardwareGeneration(instance.HardwareGeneration)
	if err != nil {
		return err
	}

	d.Set("created_at", getTimestamp(instance.CreatedAt))
	d.Set("platform_id", instance.PlatformId)
	d.Set("folder_id", instance.FolderId)
	d.Set("zone", instance.ZoneId)
	d.Set("name", instance.Name)
	d.Set("fqdn", instance.Fqdn)
	d.Set("description", instance.Description)
	d.Set("service_account_id", instance.ServiceAccountId)
	d.Set("status", strings.ToLower(instance.Status.String()))
	d.Set("metadata_options", metadataOptions)

	hostname, err := parseHostnameFromFQDN(instance.Fqdn)
	if err != nil {
		return err
	}
	d.Set("hostname", hostname)

	if err := d.Set("metadata", instance.Metadata); err != nil {
		return err
	}

	if err := d.Set("labels", instance.Labels); err != nil {
		return err
	}

	if err := d.Set("resources", resources); err != nil {
		return err
	}

	if err := d.Set("boot_disk", bootDisk); err != nil {
		return err
	}

	if err := d.Set("secondary_disk", secondaryDisks); err != nil {
		return err
	}

	if err := d.Set("scheduling_policy", schedulingPolicy); err != nil {
		return err
	}

	if err := d.Set("placement_policy", placementPolicy); err != nil {
		return err
	}

	if err := d.Set("local_disk", localDisks); err != nil {
		return err
	}

	if err := d.Set("filesystem", filesystems); err != nil {
		return err
	}

	if instance.NetworkSettings != nil {
		d.Set("network_acceleration_type", strings.ToLower(instance.NetworkSettings.Type.String()))
	}

	if err := d.Set("network_interface", networkInterfaces); err != nil {
		return err
	}

	connIP := externalIP
	if connIP == "" {
		connIP = internalIP
	}

	d.SetConnInfo(map[string]string{
		"type": "ssh",
		"host": connIP,
	})

	if instance.GpuSettings != nil {
		d.Set("gpu_cluster_id", instance.GpuSettings.GpuClusterId)
	}

	if instance.MaintenancePolicy != compute.MaintenancePolicy_MAINTENANCE_POLICY_UNSPECIFIED {
		if err := d.Set("maintenance_policy", strings.ToLower(instance.MaintenancePolicy.String())); err != nil {
			return err
		}
	}

	if err := d.Set("maintenance_grace_period", formatDuration(instance.MaintenanceGracePeriod)); err != nil {
		return err
	}

	if err := d.Set("hardware_generation", hardwareGeneration); err != nil {
		return err
	}

	return nil
}

// revive:enable:var-naming

func resourceYandexComputeInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ctx := config.Context()

	instance, err := config.sdk.Compute().Instance().Get(ctx, &compute.GetInstanceRequest{
		InstanceId: d.Id(),
	})

	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Instance %q", d.Get("name").(string)))
	}

	d.Partial(true)

	folderPropName := "folder_id"
	if d.HasChange(folderPropName) {
		if !d.Get("allow_recreate").(bool) {
			if err := ensureAllowStoppingForUpdate(d, folderPropName); err != nil {
				return err
			}

			if instance.Status != compute.Instance_STOPPED {
				if err := makeInstanceActionRequest(instanceActionStop, d, meta); err != nil {
					return err
				}
			}

			req := &compute.MoveInstanceRequest{
				InstanceId:          d.Id(),
				DestinationFolderId: d.Get(folderPropName).(string),
			}

			if err := makeInstanceMoveRequest(req, d, meta); err != nil {
				return err
			}

			if err := makeInstanceActionRequest(instanceActionStart, d, meta); err != nil {
				return err
			}

		} else {
			if err := resourceYandexComputeInstanceDelete(d, meta); err != nil {
				return err
			}
			if err := resourceYandexComputeInstanceCreate(d, meta); err != nil {
				return err
			}
		}
	}

	labelPropName := "labels"
	if d.HasChange(labelPropName) {
		labelsProp, err := expandLabels(d.Get(labelPropName))
		if err != nil {
			return err
		}

		req := &compute.UpdateInstanceRequest{
			InstanceId: d.Id(),
			Labels:     labelsProp,
			UpdateMask: &field_mask.FieldMask{
				Paths: []string{labelPropName},
			},
		}

		err = makeInstanceUpdateRequest(req, d, meta)
		if err != nil {
			return err
		}

	}

	metadataPropName := "metadata"
	if d.HasChange(metadataPropName) {
		metadataProp, err := expandLabels(d.Get(metadataPropName))
		if err != nil {
			return err
		}

		req := &compute.UpdateInstanceRequest{
			InstanceId: d.Id(),
			Metadata:   metadataProp,
			UpdateMask: &field_mask.FieldMask{
				Paths: []string{metadataPropName},
			},
		}

		err = makeInstanceUpdateRequest(req, d, meta)
		if err != nil {
			return err
		}

	}

	metadataOptionsPropName := "metadata_options"
	if d.HasChange(metadataOptionsPropName) {
		metadataOptionsProp := expandInstanceMetadataOptions(d)

		req := &compute.UpdateInstanceRequest{
			InstanceId:      d.Id(),
			MetadataOptions: metadataOptionsProp,
			UpdateMask: &field_mask.FieldMask{
				Paths: []string{metadataOptionsPropName},
			},
		}

		err = makeInstanceUpdateRequest(req, d, meta)
		if err != nil {
			return err
		}

	}

	namePropName := "name"
	if d.HasChange(namePropName) {
		req := &compute.UpdateInstanceRequest{
			InstanceId: d.Id(),
			Name:       d.Get(namePropName).(string),
			UpdateMask: &field_mask.FieldMask{
				Paths: []string{namePropName},
			},
		}

		err := makeInstanceUpdateRequest(req, d, meta)
		if err != nil {
			return err
		}

	}

	descPropName := "description"
	if d.HasChange(descPropName) {
		req := &compute.UpdateInstanceRequest{
			InstanceId:  d.Id(),
			Description: d.Get(descPropName).(string),
			UpdateMask: &field_mask.FieldMask{
				Paths: []string{descPropName},
			},
		}

		err := makeInstanceUpdateRequest(req, d, meta)
		if err != nil {
			return err
		}

	}

	serviceAccountPropName := "service_account_id"
	if d.HasChange(serviceAccountPropName) {
		req := &compute.UpdateInstanceRequest{
			InstanceId:       d.Id(),
			ServiceAccountId: d.Get(serviceAccountPropName).(string),
			UpdateMask: &field_mask.FieldMask{
				Paths: []string{serviceAccountPropName},
			},
		}

		err := makeInstanceUpdateRequest(req, d, meta)
		if err != nil {
			return err
		}

	}

	maintenancePolicyPropName := "maintenance_policy"
	maintenanceGracePeriodPropName := "maintenance_grace_period"
	if d.HasChange(maintenancePolicyPropName) || d.HasChange(maintenanceGracePeriodPropName) {
		req := &compute.UpdateInstanceRequest{
			InstanceId: d.Id(),
			UpdateMask: &field_mask.FieldMask{
				Paths: []string{},
			},
		}

		if d.HasChange(maintenancePolicyPropName) {
			req.MaintenancePolicy, err = expandMaintenancePolicy(d)
			if err != nil {
				return err
			}
			req.UpdateMask.Paths = append(req.UpdateMask.Paths, maintenancePolicyPropName)
		}

		if d.HasChange(maintenanceGracePeriodPropName) {
			req.MaintenanceGracePeriod, err = parseDuration(d.Get(maintenanceGracePeriodPropName).(string))
			if err != nil {
				return err
			}
			req.UpdateMask.Paths = append(req.UpdateMask.Paths, maintenanceGracePeriodPropName)
		}

		err = makeInstanceUpdateRequest(req, d, meta)
		if err != nil {
			return err
		}
	}

	networkInterfacesPropName := "network_interface"
	needUpdateInterfacesOnStoppedInstance := false
	var addNatRequests []*compute.AddInstanceOneToOneNatRequest
	var removeNatRequests []*compute.RemoveInstanceOneToOneNatRequest
	var updateInterfaceRequests []*compute.UpdateInstanceNetworkInterfaceRequest
	var attachInterfaceRequests []*compute.AttachInstanceNetworkInterfaceRequest
	var detachInterfaceRequests []*compute.DetachInstanceNetworkInterfaceRequest
	if d.HasChange(networkInterfacesPropName) {
		o, n := d.GetChange(networkInterfacesPropName)
		oldList := o.([]interface{})
		newList := n.([]interface{})

		if len(oldList) != len(newList) {
			log.Printf("[DEBUG] Number of network interfaces has changed, processing attach/detach interfaces. " +
				"Instance will be stopped")
			needUpdateInterfacesOnStoppedInstance = true

			attachInterfaceRequests, detachInterfaceRequests, err = getSpecsForAttachDetachNetworkInterfaces(newList,
				d.Id(), instance.NetworkInterfaces)
			if err != nil {
				return err
			}

		} else {
			updateInterfaceRequests, needUpdateInterfacesOnStoppedInstance, err = getSpecsForUpdateNetworkInterfaces(d, networkInterfacesPropName, oldList, newList)
			if err != nil {
				return err
			}
			addNatRequests, removeNatRequests, err = getSpecsForAddRemoveNatNetworkInterfaces(d.Id(), oldList, newList)
			if err != nil {
				return err
			}
		}

		if !needUpdateInterfacesOnStoppedInstance && (len(removeNatRequests) > 0 || len(addNatRequests) > 0 || len(updateInterfaceRequests) > 0) {
			for _, req := range removeNatRequests {
				err := makeInstanceRemoveOneToOneNatRequest(req, d, meta)
				if err != nil {
					return err
				}
			}
			for _, req := range addNatRequests {
				err := makeInstanceAddOneToOneNatRequest(req, d, meta)
				if err != nil {
					return err
				}
			}
			for _, req := range updateInterfaceRequests {
				err := makeInstanceUpdateNetworkInterfaceRequest(req, d, meta)
				if err != nil {
					return err
				}
			}
		}
	}

	secDiskPropName := "secondary_disk"
	if d.HasChange(secDiskPropName) {
		_, n := d.GetChange(secDiskPropName)

		// Keep track of disks currently in the instance. Because the yandex_compute_disk resource
		// can detach disks, it's possible that there are fewer disks currently attached than there
		// were at the time we ran terraform plan.
		currDisks := map[string]*compute.AttachedDisk{}
		for _, disk := range instance.SecondaryDisks {
			currDisks[disk.DiskId] = disk
		}

		// Keep track of new config's disks.
		// Since changing any field within the disk needs to detach+reattach it,
		// keep track of the hash of the full disk.
		// If a disk with a certain hash is only in the new config, it should be attached.
		nDisks := map[string]*compute.AttachedDiskSpec{}
		var attach []*compute.AttachedDiskSpec
		var detach []*compute.DetachInstanceDiskRequest
		for _, disk := range n.(*schema.Set).List() {
			diskConfig := disk.(map[string]interface{})
			diskSpec, err := expandSecondaryDiskSpec(diskConfig)
			if err != nil {
				return err
			}
			nDisks[diskSpec.GetDiskId()] = diskSpec

			if currDisk, ok := currDisks[diskSpec.GetDiskId()]; !ok {
				// attach new disks
				attach = append(attach, diskSpec)
			} else if diskSpecChanged(currDisk, diskSpec) {
				// disk spec has been changed
				// detach and attach it
				detach = append(detach, &compute.DetachInstanceDiskRequest{
					InstanceId: d.Id(),
					Disk: &compute.DetachInstanceDiskRequest_DiskId{
						DiskId: currDisk.GetDiskId(),
					},
				})
				attach = append(attach, diskSpec)
			}
		}

		// Detach disks that are not in new config
		for diskID := range currDisks {
			if _, ok := nDisks[diskID]; !ok {
				detach = append(detach, &compute.DetachInstanceDiskRequest{
					InstanceId: d.Id(),
					Disk: &compute.DetachInstanceDiskRequest_DiskId{
						DiskId: diskID,
					},
				})
			}
		}

		// Make all detach calls
		for _, req := range detach {
			err = makeDetachDiskRequest(req, meta)
			if err != nil {
				return err
			}
			log.Printf("[DEBUG] Successfully detached disk %q from instance %q", req.GetDiskId(), req.GetInstanceId())
		}

		// Attach the new disks
		for _, diskSpec := range attach {
			req := &compute.AttachInstanceDiskRequest{
				InstanceId:       d.Id(),
				AttachedDiskSpec: diskSpec,
			}

			err := makeAttachDiskRequest(req, meta)
			if err != nil {
				return err
			}
			log.Printf("[DEBUG] Successfully attached disk %s", diskSpec.GetDiskId())
		}
	}
	filesystemPropName := "filesystem"
	if d.HasChange(filesystemPropName) {
		o, n := d.GetChange(filesystemPropName)

		currFs := map[string]struct{}{}
		for _, fs := range instance.Filesystems {
			currFs[fs.FilesystemId] = struct{}{}
		}

		oldFs := map[uint64]string{}
		for _, fs := range o.(*schema.Set).List() {
			fsConfig := fs.(map[string]interface{})
			fsSpec, err := expandFilesystemSpec(fsConfig)
			if err != nil {
				return err
			}
			hash, err := hashstructure.Hash(fsSpec, nil)
			if err != nil {
				return err
			}
			if _, ok := currFs[fsSpec.GetFilesystemId()]; ok {
				oldFs[hash] = fsSpec.GetFilesystemId()
			}
		}

		newFs := map[uint64]struct{}{}
		var attach []*compute.AttachedFilesystemSpec
		for _, fs := range n.(*schema.Set).List() {
			fsConfig := fs.(map[string]interface{})
			fsSpec, err := expandFilesystemSpec(fsConfig)
			if err != nil {
				return err
			}
			hash, err := hashstructure.Hash(fsSpec, nil)
			if err != nil {
				return err
			}
			newFs[hash] = struct{}{}

			if _, ok := oldFs[hash]; !ok {
				attach = append(attach, fsSpec)
			}
		}

		// Detach old filesystems
		for hash, fsID := range oldFs {
			if _, ok := newFs[hash]; !ok {
				req := &compute.DetachInstanceFilesystemRequest{
					InstanceId: d.Id(),
					Filesystem: &compute.DetachInstanceFilesystemRequest_FilesystemId{
						FilesystemId: fsID,
					},
				}

				err = makeDetachFilesystemRequest(req, meta)
				if err != nil {
					return err
				}
				log.Printf("[DEBUG] Successfully detached filesystem %s", fsID)
			}
		}

		// Attach the new filesystems
		for _, fsSpec := range attach {
			req := &compute.AttachInstanceFilesystemRequest{
				InstanceId:             d.Id(),
				AttachedFilesystemSpec: fsSpec,
			}

			err := makeAttachFilesystemRequest(req, meta)
			if err != nil {
				return err
			}
			log.Printf("[DEBUG] Successfully attached filesystem %s", fsSpec.GetFilesystemId())
		}
	}

	resourcesPropName := "resources"
	platformIDPropName := "platform_id"
	networkAccelerationTypePropName := "network_acceleration_type"
	schedulingPolicyName := "scheduling_policy"
	placementPolicyPropName := "placement_policy"

	properties := []string{
		resourcesPropName,
		platformIDPropName,
		networkAccelerationTypePropName,
		schedulingPolicyName,
		placementPolicyPropName,
	}
	if d.HasChange(resourcesPropName) || d.HasChange(platformIDPropName) || d.HasChange(networkAccelerationTypePropName) ||
		needUpdateInterfacesOnStoppedInstance || d.HasChange(schedulingPolicyName) || d.HasChange(placementPolicyPropName) {
		if err := ensureAllowStoppingForUpdate(d, properties...); err != nil {
			return err
		}
		if err := makeInstanceActionRequest(instanceActionStop, d, meta); err != nil {
			return err
		}

		instanceStoppedAt := time.Now()

		// update platform, resources, network_settings and maintenance_policy in one request
		if d.HasChange(resourcesPropName) || d.HasChange(platformIDPropName) || d.HasChange(networkAccelerationTypePropName) ||
			d.HasChange(placementPolicyPropName) || d.HasChange(schedulingPolicyName) {
			req := &compute.UpdateInstanceRequest{
				InstanceId: d.Id(),
				UpdateMask: &field_mask.FieldMask{
					Paths: []string{},
				},
			}

			if d.HasChange(resourcesPropName) {
				spec, err := expandInstanceResourcesSpec(d)
				if err != nil {
					return err
				}

				req.ResourcesSpec = spec
				req.UpdateMask.Paths = append(req.UpdateMask.Paths, "resources_spec")
			}

			if d.HasChange(platformIDPropName) {
				req.PlatformId = d.Get(platformIDPropName).(string)
				req.UpdateMask.Paths = append(req.UpdateMask.Paths, platformIDPropName)
			}

			if d.HasChange(networkAccelerationTypePropName) {
				networkSettings, err := expandInstanceNetworkSettingsSpecs(d)
				if err != nil {
					return err
				}

				req.NetworkSettings = networkSettings
				req.UpdateMask.Paths = append(req.UpdateMask.Paths, "network_settings")
			}

			if d.HasChange(schedulingPolicyName) {
				schedulingPolicy, err := expandInstanceSchedulingPolicy(d)
				if err != nil {
					return err
				}

				req.SchedulingPolicy = schedulingPolicy
				req.UpdateMask.Paths = append(req.UpdateMask.Paths, "scheduling_policy.preemptible")
			}

			if d.HasChange(placementPolicyPropName) {
				placementPolicy, paths := preparePlacementPolicyForUpdateRequest(d)
				req.PlacementPolicy = placementPolicy
				req.UpdateMask.Paths = append(req.UpdateMask.Paths, paths...)
			}

			err = makeInstanceUpdateRequest(req, d, meta)
			if err != nil {
				return err
			}
		}

		// update interfaces on stopped instance
		if needUpdateInterfacesOnStoppedInstance {
			// wait for resource deallocation
			timeSinceInstanceStopped := time.Since(instanceStoppedAt)
			if timeSinceInstanceStopped < yandexComputeInstanceDeallocationTimeout {
				sleepTime := yandexComputeInstanceDeallocationTimeout - timeSinceInstanceStopped
				log.Printf("[DEBUG] Sleeping %s, waiting for deallocation", sleepTime)
				time.Sleep(sleepTime)
			}
			for _, req := range removeNatRequests {
				err := makeInstanceRemoveOneToOneNatRequest(req, d, meta)
				if err != nil {
					return err
				}
			}
			for _, req := range addNatRequests {
				err := makeInstanceAddOneToOneNatRequest(req, d, meta)
				if err != nil {
					return err
				}
			}
			for _, req := range updateInterfaceRequests {
				err := makeInstanceUpdateNetworkInterfaceRequest(req, d, meta)
				if err != nil {
					return err
				}
			}
			// Attach network interfaces
			for _, req := range attachInterfaceRequests {
				err := makeInstanceAttachNetworkInterfaceRequest(req, d, meta)
				if err != nil {
					return err
				}
			}

			// Detach network interfaces
			for _, req := range detachInterfaceRequests {
				err := makeInstanceDetachNetworkInterfaceRequest(req, d, meta)
				if err != nil {
					return err
				}
			}

		}

		if err := makeInstanceActionRequest(instanceActionStart, d, meta); err != nil {
			return err
		}
	}

	d.Partial(false)

	return resourceYandexComputeInstanceRead(d, meta)
}

func resourceYandexComputeInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	log.Printf("[DEBUG] Deleting Instance %q", d.Id())

	req := &compute.DeleteInstanceRequest{
		InstanceId: d.Id(),
	}

	ctx, cancel := context.WithTimeout(config.Context(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Compute().Instance().Delete(ctx, req))
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Instance %q", d.Get("name").(string)))
	}

	err = op.Wait(ctx)
	if err != nil {
		return err
	}

	_, err = op.Response()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Finished deleting Instance %q", d.Id())
	return nil
}

func prepareCreateInstanceRequest(d *schema.ResourceData, meta *Config) (*compute.CreateInstanceRequest, error) {
	zone, err := getZone(d, meta)
	if err != nil {
		return nil, fmt.Errorf("Error getting zone while creating instance: %s", err)
	}

	folderID, err := getFolderID(d, meta)
	if err != nil {
		return nil, fmt.Errorf("Error getting folder ID while creating instance: %s", err)
	}

	labels, err := expandLabels(d.Get("labels"))
	if err != nil {
		return nil, fmt.Errorf("Error expanding labels while creating instance: %s", err)
	}

	metadata, err := expandLabels(d.Get("metadata"))
	if err != nil {
		return nil, fmt.Errorf("Error expanding metadata while creating instance: %s", err)
	}

	resourcesSpec, err := expandInstanceResourcesSpec(d)
	if err != nil {
		return nil, fmt.Errorf("Error create 'resources_spec' object of api request: %s", err)
	}

	bootDiskSpec, err := expandInstanceBootDiskSpec(d, meta)
	if err != nil {
		return nil, fmt.Errorf("Error create 'boot_disk' object of api request: %s", err)
	}

	secondaryDiskSpecs, err := expandInstanceSecondaryDiskSpecs(d)
	if err != nil {
		return nil, fmt.Errorf("Error create 'secondary_disk' object of api request: %s", err)
	}

	networkSettingsSpecs, err := expandInstanceNetworkSettingsSpecs(d)
	if err != nil {
		return nil, fmt.Errorf("Error create 'network' object of api request: %s", err)
	}

	nicSpecs, err := expandInstanceNetworkInterfaceSpecs(d)
	if err != nil {
		return nil, fmt.Errorf("Error create 'network' object of api request: %s", err)
	}

	schedulingPolicy, err := expandInstanceSchedulingPolicy(d)
	if err != nil {
		return nil, fmt.Errorf("Error create 'scheduling_policy' object of api request: %s", err)
	}

	placementPolicy, err := expandInstancePlacementPolicy(d)
	if err != nil {
		return nil, fmt.Errorf("Error create 'placement_policy' object of api request: %s", err)
	}

	metadataOptions := expandInstanceMetadataOptions(d)

	localDisks := expandLocalDiskSpecs(d.Get("local_disk"))

	filesystemSpecs, err := expandInstanceFilesystemSpecs(d)
	if err != nil {
		return nil, fmt.Errorf("Error create 'filesystem' object of api request: %s", err)
	}

	gpuSettingsSpec, err := expandInstanceGpuSettingsSpec(d)
	if err != nil {
		return nil, fmt.Errorf("Error create 'gpu_settings' object of api request: %s", err)
	}

	maintenancePolicy, err := expandMaintenancePolicy(d)
	if err != nil {
		return nil, fmt.Errorf("Error create 'maintenance_policy' object of api request: %s", err)
	}

	maintenanceGracePeriod, err := parseDuration(d.Get("maintenance_grace_period").(string))
	if err != nil {
		return nil, fmt.Errorf("Error create 'maintenance_grace_period' object of api request: %s", err)
	}

	req := &compute.CreateInstanceRequest{
		FolderId:               folderID,
		Hostname:               d.Get("hostname").(string),
		Name:                   d.Get("name").(string),
		Description:            d.Get("description").(string),
		PlatformId:             d.Get("platform_id").(string),
		ServiceAccountId:       d.Get("service_account_id").(string),
		ZoneId:                 zone,
		Labels:                 labels,
		Metadata:               metadata,
		ResourcesSpec:          resourcesSpec,
		BootDiskSpec:           bootDiskSpec,
		SecondaryDiskSpecs:     secondaryDiskSpecs,
		NetworkSettings:        networkSettingsSpecs,
		NetworkInterfaceSpecs:  nicSpecs,
		SchedulingPolicy:       schedulingPolicy,
		PlacementPolicy:        placementPolicy,
		LocalDiskSpecs:         localDisks,
		MetadataOptions:        metadataOptions,
		FilesystemSpecs:        filesystemSpecs,
		GpuSettings:            gpuSettingsSpec,
		MaintenancePolicy:      maintenancePolicy,
		MaintenanceGracePeriod: maintenanceGracePeriod,
	}

	return req, nil
}

func expandMaintenancePolicy(d *schema.ResourceData) (compute.MaintenancePolicy, error) {
	if v, ok := d.GetOk("maintenance_policy"); ok {
		switch v {
		case "unspecified":
			return compute.MaintenancePolicy_MAINTENANCE_POLICY_UNSPECIFIED, nil
		case "restart":
			return compute.MaintenancePolicy_RESTART, nil
		case "migrate":
			return compute.MaintenancePolicy_MIGRATE, nil
		default:
			return compute.MaintenancePolicy_MAINTENANCE_POLICY_UNSPECIFIED, fmt.Errorf("unknown maintenance_policy: %q", v)
		}
	}
	return compute.MaintenancePolicy_MAINTENANCE_POLICY_UNSPECIFIED, nil
}

func parseHostnameFromFQDN(fqdn string) (string, error) {
	if !strings.Contains(fqdn, ".") {
		return fqdn + ".", nil
	}
	if strings.HasSuffix(fqdn, ".auto.internal") {
		return "", nil
	}
	if strings.HasSuffix(fqdn, ".internal") {
		p := strings.Split(fqdn, ".")
		return p[0], nil
	}

	return fqdn, nil
}

func wantChangeAddressSpec(old *compute.PrimaryAddressSpec, new *compute.PrimaryAddressSpec) bool {
	if old == nil && new == nil {
		return false
	}

	if (old != nil && new == nil) || (old == nil && new != nil) {
		return true
	}

	if new.Address != "" && old.Address != new.Address {
		return true
	}

	return dnsSpecChanged(old, new) || natDnsSpecChanged(old.OneToOneNatSpec, new.OneToOneNatSpec)
}

func dnsSpecChanged(old *compute.PrimaryAddressSpec, new *compute.PrimaryAddressSpec) bool {
	if len(old.DnsRecordSpecs) != len(new.DnsRecordSpecs) {
		return true
	}

	for i, oldrs := range old.DnsRecordSpecs {
		newrs := new.DnsRecordSpecs[i]
		if differentRecordSpec(oldrs, newrs) {
			return true
		}
	}
	return false
}

func needToRestartDueToAddressChange(old *compute.PrimaryAddressSpec, new *compute.PrimaryAddressSpec) bool {
	if old == nil && new == nil {
		return false
	}

	if (old != nil && new == nil) || (old == nil && new != nil) {
		return true
	}

	return new.Address != "" && old.Address != new.Address
}

func natAddressSpecChanged(old *compute.OneToOneNatSpec, new *compute.OneToOneNatSpec) bool {
	if old == nil && new == nil {
		return false
	}

	if (old != nil && new == nil) || (old == nil && new != nil) {
		return true
	}

	return new.Address != "" && old.Address != new.Address
}

func natDnsSpecChanged(old *compute.OneToOneNatSpec, new *compute.OneToOneNatSpec) bool {
	if old == nil && new == nil {
		return false
	}

	if (old != nil && new == nil) || (old == nil && new != nil) {
		//the whole NAT section changed, need to make separate requests
		return false
	}

	if len(old.DnsRecordSpecs) != len(new.DnsRecordSpecs) {
		return true
	}

	for i, oldrs := range old.DnsRecordSpecs {
		newrs := new.DnsRecordSpecs[i]
		if differentRecordSpec(oldrs, newrs) {
			return true
		}
	}
	return false
}
func makeInstanceUpdateRequest(req *compute.UpdateInstanceRequest, d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ctx, cancel := context.WithTimeout(config.Context(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Compute().Instance().Update(ctx, req))
	if err != nil {
		return fmt.Errorf("Error while requesting API to update Instance %q: %s", d.Id(), err)
	}

	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Error updating Instance %q: %s", d.Id(), err)
	}

	return nil
}

func makeInstanceUpdateNetworkInterfaceRequest(req *compute.UpdateInstanceNetworkInterfaceRequest, d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ctx, cancel := context.WithTimeout(config.Context(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Compute().Instance().UpdateNetworkInterface(ctx, req))
	if err != nil {
		return fmt.Errorf("Error while requesting API to update network interface for Instance %q: %s", d.Id(), err)
	}

	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Error updating Instance %q: %s", d.Id(), err)
	}

	return nil
}

func makeInstanceAddOneToOneNatRequest(req *compute.AddInstanceOneToOneNatRequest, d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ctx, cancel := context.WithTimeout(config.Context(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Compute().Instance().AddOneToOneNat(ctx, req))
	if err != nil {
		return fmt.Errorf("Error while requesting API to add one-to-one nat for Instance %q: %s", d.Id(), err)
	}

	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Error updating Instance %q: %s", d.Id(), err)
	}

	return nil
}

func makeInstanceRemoveOneToOneNatRequest(req *compute.RemoveInstanceOneToOneNatRequest, d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ctx, cancel := context.WithTimeout(config.Context(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Compute().Instance().RemoveOneToOneNat(ctx, req))
	if err != nil {
		return fmt.Errorf("Error while requesting API to remove one-to-one nat for Instance %q: %s", d.Id(), err)
	}

	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Error updating Instance %q: %s", d.Id(), err)
	}

	return nil
}

func makeInstanceAttachNetworkInterfaceRequest(req *compute.AttachInstanceNetworkInterfaceRequest, d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ctx, cancel := context.WithTimeout(config.Context(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Compute().Instance().AttachNetworkInterface(ctx, req))
	if err != nil {
		return fmt.Errorf("Error while requesting API to attach network interface to the Instance %q: %s", d.Id(), err)
	}

	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Error updating Instance %q: %s", d.Id(), err)
	}

	return nil
}

func makeInstanceDetachNetworkInterfaceRequest(req *compute.DetachInstanceNetworkInterfaceRequest, d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ctx, cancel := context.WithTimeout(config.Context(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Compute().Instance().DetachNetworkInterface(ctx, req))
	if err != nil {
		return fmt.Errorf("Error while requesting API to detach network interface from the Instance %q: %s", d.Id(), err)
	}

	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Error updating Instance %q: %s", d.Id(), err)
	}

	return nil
}

func makeInstanceActionRequest(action instanceAction, d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ctx, cancel := context.WithTimeout(config.Context(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	instanceID := d.Id()
	var err error
	var op *operation.Operation

	log.Printf("[DEBUG] Prepare to run %s action on instance %s", action, instanceID)

	switch action {
	case instanceActionStop:
		{
			op, err = config.sdk.WrapOperation(config.sdk.Compute().Instance().
				Stop(ctx, &compute.StopInstanceRequest{
					InstanceId: instanceID,
				}))
		}
	case instanceActionStart:
		{
			op, err = config.sdk.WrapOperation(config.sdk.Compute().Instance().
				Start(ctx, &compute.StartInstanceRequest{
					InstanceId: instanceID,
				}))
		}
	default:
		return fmt.Errorf("Action %s not supported", action)
	}

	if err != nil {
		log.Printf("[DEBUG] Error while run %s action on instance %s: %s", action, instanceID, err)
		return fmt.Errorf("Error while run %s action on Instance %s: %s", action, instanceID, err)
	}

	err = op.Wait(ctx)
	if err != nil {
		log.Printf("[DEBUG] Error while wait %s action on instance %s: %s", action, instanceID, err)
		return fmt.Errorf("Error while wait %s action on Instance %s: %s", action, instanceID, err)
	}

	return nil
}

func makeDetachDiskRequest(req *compute.DetachInstanceDiskRequest, meta interface{}) error {
	config := meta.(*Config)

	ctx, cancel := context.WithTimeout(config.Context(), yandexComputeInstanceDiskOperationTimeout)
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Compute().Instance().DetachDisk(ctx, req))
	if err != nil {
		return fmt.Errorf("Error while requesting API to detach Disk %s from Instance %q: %s", req.GetDiskId(), req.GetInstanceId(), err)
	}

	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Error detach Disk %s from Instance %q: %s", req.GetDiskId(), req.GetInstanceId(), err)
	}

	return nil
}

func makeAttachDiskRequest(req *compute.AttachInstanceDiskRequest, meta interface{}) error {
	config := meta.(*Config)

	ctx, cancel := context.WithTimeout(config.Context(), yandexComputeInstanceDiskOperationTimeout)
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Compute().Instance().AttachDisk(ctx, req))
	if err != nil {
		return fmt.Errorf("Error while requesting API to attach Disk %s to Instance %q: %s", req.AttachedDiskSpec.GetDiskId(), req.GetInstanceId(), err)
	}

	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Error attach Disk %s to Instance %q: %s", req.AttachedDiskSpec.GetDiskId(), req.GetInstanceId(), err)
	}

	return nil
}

func makeDetachFilesystemRequest(req *compute.DetachInstanceFilesystemRequest, meta interface{}) error {
	config := meta.(*Config)

	ctx, cancel := context.WithTimeout(config.Context(), yandexComputeInstanceDefaultTimeout)
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Compute().Instance().DetachFilesystem(ctx, req))
	if err != nil {
		return fmt.Errorf("Error while requesting API to detach Filesystem %s from Instance %q: %s",
			req.GetFilesystemId(), req.GetInstanceId(), err)
	}

	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Error detach Filesystem %s from Instance %q: %s",
			req.GetFilesystemId(), req.GetInstanceId(), err)
	}

	return nil
}

func makeAttachFilesystemRequest(req *compute.AttachInstanceFilesystemRequest, meta interface{}) error {
	config := meta.(*Config)

	ctx, cancel := context.WithTimeout(config.Context(), yandexComputeInstanceDefaultTimeout)
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Compute().Instance().AttachFilesystem(ctx, req))
	if err != nil {
		return fmt.Errorf("Error while requesting API to attach Filesystem %s to Instance %q: %s",
			req.AttachedFilesystemSpec.GetFilesystemId(), req.GetInstanceId(), err)
	}

	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Error attach Filesystem %s to Instance %q: %s",
			req.AttachedFilesystemSpec.GetFilesystemId(), req.GetInstanceId(), err)
	}

	return nil
}

func makeInstanceMoveRequest(req *compute.MoveInstanceRequest, d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ctx, cancel := context.WithTimeout(config.Context(), yandexComputeInstanceMoveTimeout)
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Compute().Instance().Move(ctx, req))
	if err != nil {
		return fmt.Errorf("Error while requesting API to move Instance %q: %s", d.Id(), err)
	}

	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Error moving Instance %q: %s", d.Id(), err)
	}

	return nil
}

func differentRecordSpec(r1, r2 *compute.DnsRecordSpec) bool {
	return r1.GetFqdn() != r2.GetFqdn() ||
		r1.GetDnsZoneId() != r2.GetDnsZoneId() ||
		r1.GetTtl() != r2.GetTtl() ||
		r1.GetPtr() != r2.GetPtr()
}

func generateHostAffinityRuleOperators() []string {
	operators := make([]string, 0, len(compute.PlacementPolicy_HostAffinityRule_Operator_value))
	for operatorName := range compute.PlacementPolicy_HostAffinityRule_Operator_value {
		operators = append(operators, operatorName)
	}
	return operators
}

func preparePlacementPolicyForUpdateRequest(d *schema.ResourceData) (*compute.PlacementPolicy, []string) {
	var placementPolicy compute.PlacementPolicy
	var paths []string
	if d.HasChange("placement_policy.0.placement_group_id") {
		placementPolicy.PlacementGroupId = d.Get("placement_policy.0.placement_group_id").(string)
		paths = append(paths, "placement_policy.placement_group_id")
	}

	if d.HasChange("placement_policy.0.host_affinity_rules") {
		rules := d.Get("placement_policy.0.host_affinity_rules").([]interface{})
		placementPolicy.HostAffinityRules = expandHostAffinityRulesSpec(rules)
		paths = append(paths, "placement_policy.host_affinity_rules")
	}
	if d.HasChange("placement_policy.0.placement_group_partition") {
		placementPolicy.PlacementGroupPartition = int64(d.Get("placement_policy.0.placement_group_partition").(int))
		paths = append(paths, "placement_policy.placement_group_partition")
	}
	return &placementPolicy, paths
}

func ensureAllowStoppingForUpdate(d *schema.ResourceData, propNames ...string) error {
	message := fmt.Sprintf("Changing the %s in an instance requires stopping it. ", strings.Join(propNames, ", "))
	if !d.Get("allow_stopping_for_update").(bool) {
		return fmt.Errorf(message + "To acknowledge this action, please set allow_stopping_for_update = true in your config file.")
	}
	return nil
}

func getSpecsForAttachDetachNetworkInterfaces(newList []interface{}, instanceId string, instanceNetworkInterfaces []*compute.NetworkInterface) (attachInterfaceRequests []*compute.AttachInstanceNetworkInterfaceRequest, detachInterfaceRequests []*compute.DetachInstanceNetworkInterfaceRequest, err error) {
	curIfaces := make(map[string]*compute.NetworkInterface, len(instanceNetworkInterfaces))
	newIfaces := make(map[string]*compute.NetworkInterfaceSpec)

	for _, iface := range instanceNetworkInterfaces {
		curIfaces[iface.Index] = iface
	}
	for ifaceIndex := 0; ifaceIndex < len(newList); ifaceIndex++ {
		newIface := newList[ifaceIndex].(map[string]interface{})
		newIfaceindex, ok := newIface["index"].(int)
		if !ok {
			return nil, nil, fmt.Errorf("NIC number #%d does not have a 'index' attribute defined, you have "+
				"to specify it", ifaceIndex)
		}
		index := strconv.Itoa(newIfaceindex)
		iface, err := expandNetworkInterfaceSpec(newIface)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to process NIC number: #%d: %s", ifaceIndex, err)
		}
		newIfaces[index] = iface
		if _, ok := curIfaces[index]; !ok {
			attachInterfaceRequests = append(attachInterfaceRequests, &compute.AttachInstanceNetworkInterfaceRequest{
				InstanceId:            instanceId,
				NetworkInterfaceIndex: index,
				PrimaryV4AddressSpec:  iface.PrimaryV4AddressSpec,
				SecurityGroupIds:      iface.SecurityGroupIds,
				SubnetId:              iface.SubnetId,
			})
		}
	}
	for index := range curIfaces {
		if _, ok := newIfaces[index]; !ok {
			detachInterfaceRequests = append(detachInterfaceRequests,
				&compute.DetachInstanceNetworkInterfaceRequest{
					InstanceId:            instanceId,
					NetworkInterfaceIndex: index,
				})
		}
	}
	return attachInterfaceRequests, detachInterfaceRequests, nil
}

func getSpecsForUpdateNetworkInterfaces(d *schema.ResourceData, networkInterfacesPropName string, oldList []interface{}, newList []interface{}) (
	updateInterfaceRequests []*compute.UpdateInstanceNetworkInterfaceRequest, stopInstance bool, err error) {
	for ifaceIndex := 0; ifaceIndex < len(oldList); ifaceIndex++ {
		log.Printf("[DEBUG] Processing interface #%d", ifaceIndex)
		oldIface := oldList[ifaceIndex].(map[string]interface{})
		newIface := newList[ifaceIndex].(map[string]interface{})

		req := &compute.UpdateInstanceNetworkInterfaceRequest{
			InstanceId:            d.Id(),
			NetworkInterfaceIndex: fmt.Sprint(ifaceIndex),
			UpdateMask: &field_mask.FieldMask{
				Paths: []string{},
			},
		}

		oldV4Spec, err := expandPrimaryV4AddressSpec(oldIface)
		if err != nil {
			return nil, stopInstance, err
		}
		oldV6Spec, err := expandPrimaryV6AddressSpec(oldIface)
		if err != nil {
			return nil, stopInstance, err
		}
		newV4Spec, err := expandPrimaryV4AddressSpec(newIface)
		if err != nil {
			return nil, stopInstance, err
		}
		newV6Spec, err := expandPrimaryV6AddressSpec(newIface)
		if err != nil {
			return nil, stopInstance, err
		}

		if oldIface["subnet_id"].(string) != newIface["subnet_id"].(string) {
			// change subnet, update all the properties!
			req.UpdateMask.Paths = append(req.UpdateMask.Paths, "subnet_id", "primary_v4_address_spec", "primary_v6_address_spec")
			// ...on stopped instance
			stopInstance = true

			req.SubnetId = newIface["subnet_id"].(string)
			req.PrimaryV4AddressSpec = newV4Spec
			if newV4Spec != nil && !d.HasChange(fmt.Sprintf("%s.%d.%s", networkInterfacesPropName, ifaceIndex, "ip_address")) {
				req.PrimaryV4AddressSpec.Address = ""
			}
			req.PrimaryV6AddressSpec = newV6Spec
			if newV6Spec != nil && d.HasChange(fmt.Sprintf("%s.%d.%s", networkInterfacesPropName, ifaceIndex, "ipv6_address")) {
				req.PrimaryV6AddressSpec.Address = ""
			}
		} else {
			if wantChangeAddressSpec(oldV4Spec, newV4Spec) {
				// change primary v4 address
				// ...on stopped instance?
				if needToRestartDueToAddressChange(oldV4Spec, newV4Spec) {
					req.UpdateMask.Paths = append(req.UpdateMask.Paths, "primary_v4_address_spec")
					stopInstance = true
				} else if dnsSpecChanged(oldV4Spec, newV4Spec) {
					req.UpdateMask.Paths = append(req.UpdateMask.Paths, "primary_v4_address_spec.dns_record_specs")
					if natDnsSpecChanged(oldV4Spec.OneToOneNatSpec, newV4Spec.OneToOneNatSpec) {
						req.UpdateMask.Paths = append(req.UpdateMask.Paths, "primary_v4_address_spec.one_to_one_nat_spec.dns_record_specs")
					}
				} else if natDnsSpecChanged(oldV4Spec.OneToOneNatSpec, newV4Spec.OneToOneNatSpec) {
					req.UpdateMask.Paths = append(req.UpdateMask.Paths, "primary_v4_address_spec.one_to_one_nat_spec.dns_record_specs")
				}

				req.PrimaryV4AddressSpec = newV4Spec
			}

			if wantChangeAddressSpec(oldV6Spec, newV6Spec) {
				// change primary v6 address
				// ...on stopped instance?
				if needToRestartDueToAddressChange(oldV6Spec, newV6Spec) {
					req.UpdateMask.Paths = append(req.UpdateMask.Paths, "primary_v6_address_spec")
					stopInstance = true
				} else {
					req.UpdateMask.Paths = append(req.UpdateMask.Paths, "primary_v6_address_spec.dns_record_specs")
				}

				req.PrimaryV6AddressSpec = newV6Spec
			}
		}

		oldSgs := expandSecurityGroupIds(oldIface["security_group_ids"])
		newSgs := expandSecurityGroupIds(newIface["security_group_ids"])
		if !reflect.DeepEqual(oldSgs, newSgs) {
			log.Printf("[DEBUG]  changing sgs form %s to %s", oldSgs, newSgs)
			// change security groups
			req.UpdateMask.Paths = append(req.UpdateMask.Paths, "security_group_ids")

			req.SecurityGroupIds = newSgs
		}

		if len(req.UpdateMask.Paths) > 0 {
			updateInterfaceRequests = append(updateInterfaceRequests, req)
		}
	}
	return updateInterfaceRequests, stopInstance, nil
}

func getSpecsForAddRemoveNatNetworkInterfaces(instanceId string, oldList []interface{}, newList []interface{}) (
	addNatRequests []*compute.AddInstanceOneToOneNatRequest, removeNatRequests []*compute.RemoveInstanceOneToOneNatRequest, err error) {
	for ifaceIndex := 0; ifaceIndex < len(oldList); ifaceIndex++ {
		log.Printf("[DEBUG] Processing interface #%d", ifaceIndex)
		oldIface := oldList[ifaceIndex].(map[string]interface{})
		newIface := newList[ifaceIndex].(map[string]interface{})

		oldV4Spec, err := expandPrimaryV4AddressSpec(oldIface)
		if err != nil {
			return nil, nil, err
		}
		newV4Spec, err := expandPrimaryV4AddressSpec(newIface)
		if err != nil {
			return nil, nil, err
		}
		if oldV4Spec == nil || newV4Spec == nil {
			return nil, nil, nil
		}
		if natAddressSpecChanged(oldV4Spec.OneToOneNatSpec, newV4Spec.OneToOneNatSpec) {
			// changing nat address on maybe running instance, safer to use add/remove nat calls
			if oldV4Spec.OneToOneNatSpec != nil {
				removeNatRequests = append(removeNatRequests, &compute.RemoveInstanceOneToOneNatRequest{
					InstanceId:            instanceId,
					NetworkInterfaceIndex: fmt.Sprint(ifaceIndex),
				})
			}
			if newV4Spec.OneToOneNatSpec != nil {
				addNatRequests = append(addNatRequests, &compute.AddInstanceOneToOneNatRequest{
					InstanceId:            instanceId,
					NetworkInterfaceIndex: fmt.Sprint(ifaceIndex),
					OneToOneNatSpec:       newV4Spec.OneToOneNatSpec,
				})
			}
		}

	}
	return addNatRequests, removeNatRequests, err
}

func hostnameDiffSuppressFunc(_, oldValue, newValue string, _ *schema.ResourceData) bool {
	return strings.TrimRight(oldValue, ".") == strings.TrimRight(newValue, ".")
}

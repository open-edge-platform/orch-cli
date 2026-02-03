## orch-cli create host

Provisions a host or hosts

```
orch-cli create host --import-from-csv] [flags]
```

### Examples

```
# Provision a host or a number of hosts from a CSV file

# Generate CSV input file using the --generate-csv flag - the default output will be a base test.csv file.
orch-cli create host --project some-project --generate-csv

# Generate CSV input file using the --generate-csv flag - the defined output will be a base myhosts.csv file.
orch-cli create host --project some-project --generate-csv=myhosts.csv

# Sample input csv file hosts.csv

Serial - Serial Number of the machine - mandatory field (both or one of Serial or UUID must be provided)
UUID - UUID of the machine - mandatory field (both or one of Serial or UUID must be provided), UUID must be provided if K8s cluster is going to be auto provisioned
OSProfile - OS Profile to be used for provisioning of the host - name of the profile or it's resource ID - mandatory field
Site - The resource ID of the site to which the host will be provisioned - mandatory field
Secure - Optional security feature to configure for the host - must be supported by OS Profile if enabled
Remote User - Optional remote user name or resource ID to configure for the host
Metadata - Optional metadata to configure for the host
LVMSize - Optional LVM size to be configured for the host
CloudInitMeta - Optional Cloud Init Metadata to be configured for the host
K8sEnable - Optional command to enable cluster deployment (only used if Cluster Orchestration feature is enabled in the Edge Orchestrator)
K8sClusterTemplate - Optional Cluster template to be used for K8s deployment on the host, must be provided if K8sEnable is true
K8sClusterConfig - Optional Cluster config to be used to specify role and cluster name and/or cluster labels

Serial,UUID,OSProfile,Site,Secure,RemoteUser,Metadata,LVMSize,CloudInitMeta,K8sEnable,K8sClusterTemplate,K8sConfig,Error - do not fill
2500JF3,4c4c4544-2046-5310-8052-cac04f515233,"Edge Microvisor Toolkit 3.0.20250617",site-c69a3c81,,localaccount-4c2c5f5a
1500JF3,1c4c4544-2046-5310-8052-cac04f515233,"Edge Microvisor Toolkit 3.0.20250617",site-c69a3c81,false,,key1=value1&key2=value2
15002F3,114c4544-2046-5310-8052-cac04f512233,"Edge Microvisor Toolkit 3.0.20250617",site-c69a3c81,false,,key1=value2&key3=value4
11002F3,2c4c4544-2046-5310-8052-cac04f512233,"Edge Microvisor Toolkit 3.0.20250617",site-c69a3c81,false,,key1=value2&key3=value4,,cloudinitname&customconfig-1234abcd
25002F3,214c4544-2046-5310-8052-cac04f512233,"Edge Microvisor Toolkit 3.0.20250617",site-c69a3c81,false,user,key1=value2&key3=value4,60,,true,baseline:v2.0.2,,role:all;name:mycluster;labels:key1=val1&key2=val2

# --dry-run allows for verification of the validity of the input csv file without creating hosts
orch-cli create host --project some-project --import-from-csv test.csv --dry-run

# Create hosts - --import-from-csv is a mandatory flag pointing to the input file. Successfully provisioned host indicated by output - errors provided in output file
orch-cli create host --project some-project --import-from-csv test.csv

# Optional flag ovverides - the flag will override all instances of an attribute inside the CSV file

--remote-user - name or id of a SSH user
--site - site ID
--secure - true or false - security feature configuration
--os-profile - name or ID of the OS profile
--metadata - key value paired metatada separated by &, must be put in quotes.
--cluster-deploy - true or false - cluster deployment configuration
--cluster-template - name and version of the cluster template to be used for cluster cration (separated by :)
--cluster-config - extra configuration for cluster creation empty defaults to "role:all", if not empty role must be defined, name and labels are optional (labels separated by &)
--cloud-init - name or resource ID of custom config - multiple configs must be separated by &
--lvm-size - size of the LVM to be configured for the host

# Create hosts from CSV and override provided values
/orch-cli create host --project some-project --import-from-csv test.csv --os-profile ubuntu-22.04-lts-generic-ext --secure false --site site-7ca0a77c --remote-user user --metadata "key7=val7key3=val3"

```

### Options

```
  -j, --cloud-init string                  Override the cloud init metadata provided in CSV file for all hosts
  -f, --cluster-config string              Override the cluster configuration provided in CSV file for all hosts
  -c, --cluster-deploy string              Override the cluster deployment flag provided in CSV file for all hosts
  -t, --cluster-template string            Override the cluster template provided in CSV file for all hosts
  -d, --dry-run                            Verify the validity of input CSV file
  -g, --generate-csv string[="test.csv"]   Generates a template CSV file for host import
  -h, --help                               help for host
  -i, --import-from-csv string             CSV file containing information about to be provisioned hosts
  -l, --lvm-size string                    Override the LVM size configuration provided in CSV file for all hosts
  -m, --metadata string                    Override the metadata provided in CSV file for all hosts
  -o, --os-profile string                  Override the OSProfile provided in CSV file for all hosts
  -r, --remote-user string                 Override the metadata provided in CSV file for all hosts
  -x, --secure string                      Override the security feature configuration provided in CSV file for all hosts
  -s, --site string                        Override the site provided in CSV file for all hosts
```

### Options inherited from parent commands

```
      --api-endpoint string   API Service Endpoint (default "https://api.kind.internal/")
      --debug-headers         emit debug-style headers separating columns via '|' character
  -n, --noauth                use without authentication checks
  -p, --project string        Active project name
  -v, --verbose               produce verbose output
```

### SEE ALSO

* [orch-cli create](orch-cli_create.md)	 - Create various orchestrator service entities

###### Auto generated by spf13/cobra on 3-Feb-2026

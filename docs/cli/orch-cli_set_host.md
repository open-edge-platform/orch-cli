## orch-cli set host

Sets a host attribute or action

```
orch-cli set host [name|resourceID] [flags]
```

### Examples

```
#Set an attribute of a host or execute an action - at least one flag must be specified

#Set host power state to on
orch-cli set host host-1234abcd  --project itep --power on

#Set host power command policy
orch-cli set host host-1234abcd  --project itep --power-policy ordered

--power - Set desired power state of host to on|off|reset|power-cycle
--power-policy - Set the desired power command policy to ordered|immediate

#Set host AMT state to provisioned
orch-cli set host host-1234abcd --project some-project --amt-state provisioned

--amt-state - Set desired AMT state of host to provisioned|unprovisioned

#Set host AMT Control Mode to admin control mode
orch-cli set host host-1234abcd --project some-project --control-mode admin

--control-mode - Set desired AMT control mode of host to admin|client

#Set KVM session
orch-cli set host host-1234abcd --project some-project --session-type kvm --session-state start
#Set SOL session
orch-cli set host host-1234abcd --project some-project --session-type sol --session-state stop
--session-type - Set session type (kvm|sol)
--session-state - Set desired session state (start|stop)

# Generate CSV input file using the --generate-csv flag - the default output will be a base test.csv file.
orch-cli set host --project some-project --generate-csv

# Generate CSV input file using the --generate-csv flag - the defined output will be a base myhosts.csv file.
orch-cli set host --project some-project --generate-csv=myhosts.csv

# Sample input csv file hosts.csv

Name - Name of the machine - mandatory field
ResourceID - Unique Identifier of host - mandatory field
DesiredAmtState - Desired AMT state of host - provisioned|unprovisioned or AMT_STATE_PROVISIONED|AMT_STATE_UNPROVISIONED - optional, leave blank to skip
ControlMode - Desired AMT control mode of host - admin|client or AMT_CONTROL_MODE_ACM|AMT_CONTROL_MODE_CCM - optional, leave blank to skip
DesiredPowerState - Desired power state of host - on|off|reset|power-cycle - optional, leave blank to skip

Name,ResourceID,DesiredAmtState,ControlMode,DesiredPowerState
host-1,host-1234abcd,provisioned
host-1,host-1234abcd,provisioned,admin,power-cycle

# --dry-run allows for verification of the validity of the input csv file without updating hosts
orch-cli set host --project some-project --import-from-csv test.csv --dry-run

# Set hosts - --import-from-csv is a mandatory flag pointing to the input file
orch-cli set host --project some-project --import-from-csv test.csv

# Bulk actions using filters - apply changes to all matching hosts
orch-cli set host --project some-project --filter "hostStatus='onboarded'" --power power-cycle
orch-cli set host --project some-project --site site-1234abcd --power on
orch-cli set host --project some-project --region region-1234abcd --power reset
orch-cli set host --project some-project --site site-1234abcd --amt-state provisioned --control-mode admin
orch-cli set host --project some-project --filter "hostStatus='onboarded'" --power on --amt-state provisioned

# Dry run to see which hosts would be affected
orch-cli set host --project some-project --filter "hostStatus='onboarded'" --power off --dry-run

#Set host OS Update policy
orch-cli set host host-1234abcd  --project itep --osupdatepolicy <resourceID>

#Bulk set OS Update policy using filters
orch-cli set host --project itep --site site-1234abcd --osupdatepolicy <resourceID>

--osupdatepolicy - Set the OS Update policy for the host, must be a valid resource ID of an OS Update policy

```

### Options

```
  -a, --amt-state string                   Set AMT state <provisioned|unprovisioned>
  -m, --control-mode string                Set AMT control mode client|admin
  -d, --dry-run                            Verify the validity of input CSV file
  -f, --filter string                      Filter hosts for bulk operations using AIP-160 filter expressions
  -g, --generate-csv string[="test.csv"]   Generates a template CSV file for host import
  -h, --help                               help for host
  -i, --import-from-csv string             CSV file containing information about provisioned hosts
      --orch-ca string                     Path to the cluster CA certificate (e.g. orch-ca.crt)
  -u, --osupdatepolicy string              Set OS update policy <resourceID>
  -r, --power string                       Power on|off|reset|power-cycle
  -c, --power-policy string                Set power policy immediate|ordered
      --region string                      Filter hosts by region for bulk operations
      --session-state string               Set remote session state <start|stop>
      --session-type string                Set remote session type <kvm|sol>
  -s, --site string                        Filter hosts by site for bulk operations
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

* [orch-cli set](orch-cli_set.md)	 - Update various orchestrator service entities

###### Auto generated by spf13/cobra on 29-May-2026

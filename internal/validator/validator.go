// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	e "github.com/open-edge-platform/cli/internal/errors"
	"github.com/open-edge-platform/cli/internal/files"
	"github.com/open-edge-platform/cli/internal/types"
)

const (
	TRIMSET = "\t\r\n "
)

// serial number is alphanumeric ranging from 5 to 20 charaters
// and composed of A-Z | a-z | 0-9. It is a required field hence
// Tolerate empty Serial number.
const SNPATTERN = `^([A-Za-z0-9]{5,20})?$`

// UUID has a fixed pattern as defined in https://www.rfc-editor.org/rfc/rfc4122.txt
// Tolerate empty uuid.
const UPATTERN = `^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})?$`

// Pattern for OS Resource Id as defined in inventory/api/os/v1/os.proto.
const OSPIDPATTERN = `^os-[0-9a-f]{8}$`

// Pattern for Site Id as defined in inventory/api/os/v1/location.proto.
const SITEIDPATTERN = `^site-[0-9a-f]{8}$`

// Pattern for Localaccount Id as defined in inventory/api/os/v1/localaccount.proto.
const LAIDPATTERN = `^localaccount-[0-9a-f]{8}$`

func SanitizeEntries(entries []types.HostRecord) ([]types.HostRecord, error) {
	var failure error
	sanitizedRecords := []types.HostRecord{}
	mapSN := make(map[string]int)
	mapUUID := make(map[string]int)

	snRe := regexp.MustCompile(SNPATTERN)
	uRe := regexp.MustCompile(UPATTERN)
	siteRe := regexp.MustCompile(SITEIDPATTERN)

	for i := 1; i <= len(entries); i++ {
		record := entries[i-1]

		errMsg := ""
		sanitizedRecord := record

		// if line has anything other than Serial,UUID,OSProfile,Site,Secure,RemoteUser,Metadata,AMTEnable,CloudInitMeta,K8sClusterTemplate terminate
		if record.Error != "" {
			return nil, e.NewCustomError(e.ErrNoComment)
		}

		sn := strings.Trim(record.Serial, TRIMSET)
		// check if serial number is present and valid
		errMsg = validateSN(snRe, sn, errMsg, mapSN, i)
		sanitizedRecord.Serial = sn

		var uuid string

		// check if uuid is valid
		if record.UUID != "" {
			uuid = strings.Trim(record.UUID, TRIMSET)
			errMsg = validateUUID(uRe, uuid, errMsg, mapUUID, i)
			sanitizedRecord.UUID = uuid
		}

		// if none of serial number or uuid are available, add error for at least one field
		if errMsg == "" && sanitizedRecord.Serial == "" && sanitizedRecord.UUID == "" {
			errMsg = fmt.Sprintf("%s;", e.NewCustomError(e.ErrOneFieldRequired).Error())
		}

		// Add osProfile id/name if available. Since name can also be supplied,
		// pattern match cannot be used to validate the field here as name does
		// not have a known pattern.
		// Refer inventory/api/os/v1/os.proto for more details
		if record.OSProfile != "" {
			osProfileID := strings.Trim(record.OSProfile, TRIMSET)
			sanitizedRecord.OSProfile = osProfileID
		} else {
			errMsg = fmt.Sprintf("%s %s;", errMsg, e.NewCustomError(e.ErrOSProfileRequired).Error())
		}

		//Check if Site is provided
		if record.Site != "" {
			siteID := strings.Trim(record.Site, TRIMSET)
			errMsg = validateSite(siteRe, siteID, errMsg)
			sanitizedRecord.Site = siteID
		} else {
			errMsg = fmt.Sprintf("%s %s;", errMsg, e.NewCustomError(e.ErrSiteRequired).Error())
		}

		// if there are error messages, append at last
		if errMsg != "" {
			failure = e.NewCustomError(e.ErrCheckFailed)
			sanitizedRecord.Error = errMsg
		}
		sanitizedRecords = append(sanitizedRecords, sanitizedRecord)
	}

	return sanitizedRecords, failure
}

func validateUUID(uRe *regexp.Regexp, uuid, errMsg string, mapUUID map[string]int, i int) string {
	if matched := uRe.MatchString(uuid); !matched {
		errMsg = fmt.Sprintf("%s%s;", errMsg, e.NewCustomError(e.ErrInvalidUUID).Error())
	} else if idx, exists := mapUUID[strings.ToLower(uuid)]; exists {
		errMsg = fmt.Sprintf("%s%s : Row %d;", errMsg, e.NewCustomError(e.ErrDuplicateUUID).Error(), idx)
	} else if uuid != "" {
		mapUUID[strings.ToLower(uuid)] = i
	}
	return errMsg
}

func validateSN(snRe *regexp.Regexp, sn, errMsg string, mapSn map[string]int, i int) string {
	if matched := snRe.MatchString(sn); !matched {
		errMsg = fmt.Sprintf("%s;", e.NewCustomError(e.ErrInvalidSN).Error())
	} else if idx, exists := mapSn[strings.ToLower(sn)]; exists {
		errMsg = fmt.Sprintf("%s : Row %d;", e.NewCustomError(e.ErrDuplicateSN).Error(), idx)
	} else if sn != "" {
		mapSn[strings.ToLower(sn)] = i
	}
	return errMsg
}

func validateSite(siteRe *regexp.Regexp, site, errMsg string) string {
	if matched := siteRe.MatchString(site); !matched {
		errMsg = fmt.Sprintf("%s;", e.NewCustomError(e.ErrInvalidSite).Error())
	}
	return errMsg
}

// checkCSV checks the contents of the given CSV file (or additional overides) & generates an error
// if errors are found in the CSV (or in overides).
func CheckCSV(filename string, globalOverrides types.HostRecord) ([]types.HostRecord, error) {
	fmt.Printf("Checking CSV file: %s\n", filename)

	content, err := files.ReadHostRecords(filename)
	if err != nil {
		return nil, err
	}

	//replace content with overrides if not empty
	for i := range content {
		recordValue := reflect.ValueOf(&content[i]).Elem()
		overrideValue := reflect.ValueOf(globalOverrides)

		for j := range recordValue.NumField() {
			field := recordValue.Field(j)
			overrideField := overrideValue.Field(j)
			if overrideField.Kind() == reflect.String && overrideField.String() != "" {
				field.SetString(overrideField.String())
			}
		}
	}

	validated, errVal := SanitizeEntries(content)
	if errVal != nil && validated == nil {
		return nil, errVal
	}

	if errVal != nil {
		newFilename := fmt.Sprintf("%s_%s_%s", "preflight_error",
			time.Now().Format(time.RFC3339), filepath.Base(filename))
		if err := files.WriteHostRecords(newFilename, validated); err != nil {
			return nil, err
		}
		fmt.Printf("Generating error file: %s\n", newFilename)
	}
	return validated, errVal
}

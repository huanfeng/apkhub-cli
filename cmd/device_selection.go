package cmd

import (
	"fmt"
	"strings"

	"github.com/huanfeng/apkhub/pkg/client"
)

func parseDeviceList(devices []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, id := range devices {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func resolveTargetDevices(adbMgr *client.ADBManager, explicit []string, all bool, allowPrompt bool) ([]string, error) {
	if all {
		status, err := adbMgr.GetDeviceStatus()
		if err != nil {
			return nil, fmt.Errorf("failed to load devices: %w", err)
		}
		var online []string
		for _, device := range status.Online {
			online = append(online, device.ID)
		}
		if len(online) == 0 {
			return nil, fmt.Errorf("no online devices available")
		}
		return online, nil
	}

	ids := parseDeviceList(explicit)
	if len(ids) > 0 {
		return ids, nil
	}

	if !allowPrompt {
		return nil, fmt.Errorf("no devices specified")
	}

	selected, err := adbMgr.SelectDevice()
	if err != nil {
		return nil, err
	}
	return []string{selected}, nil
}

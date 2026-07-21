/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package helm

import (
	"encoding/json"
	"fmt"

	trusteev1alpha1 "github.com/confidential-containers/trustee-operator/api/v1alpha1"
)

func SpecToValues(spec *trusteev1alpha1.TrusteeSpec) (map[string]interface{}, error) {
	raw, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshaling spec: %w", err)
	}

	var vals map[string]interface{}
	if err := json.Unmarshal(raw, &vals); err != nil {
		return nil, fmt.Errorf("unmarshaling spec to values: %w", err)
	}

	remap := map[string]string{
		"logLevel":               "log_level",
		"sessionStorageType":     "sessionStorageType",
		"dnsHostAliasWorkaround": "dnsHostAliasWorkaround",
	}
	for goKey, helmKey := range remap {
		if v, ok := vals[goKey]; ok && goKey != helmKey {
			vals[helmKey] = v
			delete(vals, goKey)
		}
	}

	cleanEmpty(vals)

	return vals, nil
}

func cleanEmpty(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			cleanEmpty(val)
			if len(val) == 0 {
				delete(m, k)
			}
		case string:
			if val == "" {
				delete(m, k)
			}
		case nil:
			delete(m, k)
		case float64:
			if val == 0 {
				delete(m, k)
			}
		case bool:
			if !val {
				delete(m, k)
			}
		case []interface{}:
			if len(val) == 0 {
				delete(m, k)
			}
		}
	}
}

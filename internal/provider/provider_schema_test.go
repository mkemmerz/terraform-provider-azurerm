package provider

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
)

func TestDataSourcesHaveSensitiveFieldsMarkedAsSensitive(t *testing.T) {
	provider := TestAzureProvider()

	// intentionally sorting these so the output is consistent
	dataSourceNames := make([]string, 0)
	for dataSourceName := range provider.DataSourcesMap {
		dataSourceNames = append(dataSourceNames, dataSourceName)
	}
	sort.Strings(dataSourceNames)

	for _, dataSourceName := range dataSourceNames {
		dataSource := provider.DataSourcesMap[dataSourceName]
		if err := schemaContainsSensitiveFieldsNotMarkedAsSensitive(dataSource.Schema); err != nil {
			t.Fatalf("the Data Source %q contains a sensitive field which isn't marked as sensitive: %+v", dataSourceName, err)
		}
	}
}

func TestResourcesHaveSensitiveFieldsMarkedAsSensitive(t *testing.T) {
	provider := TestAzureProvider()

	// intentionally sorting these so the output is consistent
	resourceNames := make([]string, 0)
	for resourceName := range provider.ResourcesMap {
		resourceNames = append(resourceNames, resourceName)
	}
	sort.Strings(resourceNames)

	for _, resourceName := range resourceNames {
		resource := provider.ResourcesMap[resourceName]
		if err := schemaContainsSensitiveFieldsNotMarkedAsSensitive(resource.Schema); err != nil {
			t.Fatalf("the Resource %q contains a sensitive field which isn't marked as sensitive: %+v", resourceName, err)
		}
	}
}

func schemaContainsSensitiveFieldsNotMarkedAsSensitive(input map[string]*pluginsdk.Schema) error {
	exactMatchFieldNames := []string{
		"api_key",
		"api_secret_key",
		"password",
		"private_key",
		"ssh_private_key",
	}

	// intentionally sorting these so the output is consistent
	fieldNames := make([]string, 0)
	for fieldName := range input {
		fieldNames = append(fieldNames, fieldName)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		key := strings.ToLower(fieldName)
		field := input[fieldName]

		for _, val := range exactMatchFieldNames {
			if strings.EqualFold(key, val) && !field.Sensitive {
				return fmt.Errorf("field %q is a sensitive value and should be marked as Sensitive", fieldName)
			}
		}

		if strings.HasSuffix(key, "_api_key") && field.Type == pluginsdk.TypeString && !field.Sensitive {
			return fmt.Errorf("field %q is a sensitive value and should be marked as Sensitive", fieldName)
		}

		if field.Type == pluginsdk.TypeList && field.Elem != nil {
			if val, ok := field.Elem.(*pluginsdk.Resource); ok && val.Schema != nil {
				if err := schemaContainsSensitiveFieldsNotMarkedAsSensitive(val.Schema); err != nil {
					return fmt.Errorf("the field %q is a List: %+v", fieldName, err)
				}
			}
		}

		if field.Type == pluginsdk.TypeSet && field.Elem != nil {
			if val, ok := field.Elem.(*pluginsdk.Resource); ok && val.Schema != nil {
				if err := schemaContainsSensitiveFieldsNotMarkedAsSensitive(val.Schema); err != nil {
					return fmt.Errorf("the field %q is a Set: %+v", fieldName, err)
				}
			}
		}
	}

	return nil
}

func TestDataSourcesHaveEnabledFieldsMarkedAsBooleans(t *testing.T) {
	provider := TestAzureProvider()

	// intentionally sorting these so the output is consistent
	dataSourceNames := make([]string, 0)
	for dataSourceName := range provider.DataSourcesMap {
		dataSourceNames = append(dataSourceNames, dataSourceName)
	}
	sort.Strings(dataSourceNames)

	for _, dataSourceName := range dataSourceNames {
		dataSource := provider.DataSourcesMap[dataSourceName]
		if err := schemaContainsEnabledFieldsNotDefinedAsABoolean(dataSource.Schema, map[string]struct{}{}); err != nil {
			t.Fatalf("the Data Source %q contains an `_enabled` field which isn't defined as a boolean: %+v", dataSourceName, err)
		}
	}
}

func TestResourcesHaveEnabledFieldsMarkedAsBooleans(t *testing.T) {
	provider := TestAzureProvider()

	// intentionally sorting these so the output is consistent
	resourceNames := make([]string, 0)
	for resourceName := range provider.ResourcesMap {
		resourceNames = append(resourceNames, resourceName)
	}
	sort.Strings(resourceNames)

	// TODO: 4.0 - work through this list
	resourceFieldsWhichNeedToBeAddressed := map[string]map[string]struct{}{
		// 1: Fields which require renaming etc
		"azurerm_datadog_monitor_sso_configuration": {
			// should be fixed in 4.0, presumably ditching `_enabled` and adding Enum validation
			"single_sign_on_enabled": {},
		},
		"azurerm_netapp_volume": {
			// should be fixed in 4.0, presumably ditching `_enabled` and making this `protocols_to_use` or something?
			"protocols_enabled": {},
		},
		"azurerm_kubernetes_cluster": {
			// this either wants `enabled` removing, or to be marked as a false-positive
			"transparent_huge_page_enabled": {},
		},
		"azurerm_kubernetes_cluster_node_pool": {
			// this either wants `enabled` removing, or to be marked as a false-positive
			"transparent_huge_page_enabled": {},
		},

		// 2: False Positives
		"azurerm_iot_security_solution": {
			// this is a list of recommendations
			"recommendations_enabled": {},
		},
	}

	for _, resourceName := range resourceNames {
		resource := provider.ResourcesMap[resourceName]
		fieldsToBeAddressed := resourceFieldsWhichNeedToBeAddressed[resourceName]

		if err := schemaContainsEnabledFieldsNotDefinedAsABoolean(resource.Schema, fieldsToBeAddressed); err != nil {
			t.Fatalf("the Resource %q contains an `_enabled` field which isn't defined as a boolean: %+v", resourceName, err)
		}
	}
}

func schemaContainsEnabledFieldsNotDefinedAsABoolean(input map[string]*schema.Schema, fieldsToBeAddressed map[string]struct{}) error {
	// intentionally sorting these so the output is consistent
	fieldNames := make([]string, 0)
	for fieldName := range input {
		fieldNames = append(fieldNames, fieldName)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		key := strings.ToLower(fieldName)
		field := input[fieldName]

		if strings.HasSuffix(key, "_enabled") {
			// @tombuildsstuff: we have some Resources which will need to be addressed in the next major version (v4.0)
			// if this field name matches one we're intentionally ignoring, let's ignore it for now
			if _, shouldIgnore := fieldsToBeAddressed[key]; shouldIgnore {
				continue
			}
			if field.Type != pluginsdk.TypeBool {
				return fmt.Errorf("field %q is an `_enabled` field so should be defined as a Boolean but got %+v", fieldName, field.Type)
			}
		}

		if field.Type == pluginsdk.TypeList && field.Elem != nil {
			if val, ok := field.Elem.(*pluginsdk.Resource); ok && val.Schema != nil {
				if err := schemaContainsEnabledFieldsNotDefinedAsABoolean(val.Schema, fieldsToBeAddressed); err != nil {
					return fmt.Errorf("the field %q is a List: %+v", fieldName, err)
				}
			}
		}

		if field.Type == pluginsdk.TypeSet && field.Elem != nil {
			if val, ok := field.Elem.(*pluginsdk.Resource); ok && val.Schema != nil {
				if err := schemaContainsEnabledFieldsNotDefinedAsABoolean(val.Schema, fieldsToBeAddressed); err != nil {
					return fmt.Errorf("the field %q is a Set: %+v", fieldName, err)
				}
			}
		}
	}

	return nil
}

func TestDataSourcesDoNotContainANameFieldWithADefaultOfDefault(t *testing.T) {
	provider := TestAzureProvider()

	// intentionally sorting these so the output is consistent
	dataSourceNames := make([]string, 0)
	for dataSourceName := range provider.DataSourcesMap {
		dataSourceNames = append(dataSourceNames, dataSourceName)
	}
	sort.Strings(dataSourceNames)

	for _, dataSourceName := range dataSourceNames {
		dataSource := provider.DataSourcesMap[dataSourceName]
		if err := schemaContainsANameFieldWithADefaultValueOfDefault(dataSource.Schema, map[string]struct{}{}); err != nil {
			t.Fatalf("the Data Source %q contains a `name` field with a default value of `default` - this Data Source should be exposed as part of the parent Data Source it's located within: %+v", dataSourceName, err)
		}
	}
}

func TestResourcesDoNotContainANameFieldWithADefaultOfDefault(t *testing.T) {
	provider := TestAzureProvider()

	// intentionally sorting these so the output is consistent
	resourceNames := make([]string, 0)
	for resourceName := range provider.ResourcesMap {
		resourceNames = append(resourceNames, resourceName)
	}
	sort.Strings(resourceNames)

	// TODO: 4.0 - work through this list
	resourceFieldsWhichNeedToBeAddressed := map[string]map[string]struct{}{
		// 1: to be addressed in 4.0
		"azurerm_datadog_monitor_sso_configuration": {
			// TODO: in 4.0 this resource probably wants embedding within `azurerm_datadog_monitor`
			// which'll also need the Monitor resource to have Create call Update
			"name": {},
		},
		"azurerm_datadog_monitor_tag_rule": {
			// TODO: in 4.0 this resource probably wants embedding within `azurerm_datadog_monitor`
			// which'll also need the Monitor resource to have Create call Update
			"name": {},
		},

		// 2. False Positives?
		"azurerm_redis_enterprise_database": {
			"name": {},
		},
	}

	for _, resourceName := range resourceNames {
		resource := provider.ResourcesMap[resourceName]
		fieldsToBeAddressed := resourceFieldsWhichNeedToBeAddressed[resourceName]

		if err := schemaContainsANameFieldWithADefaultValueOfDefault(resource.Schema, fieldsToBeAddressed); err != nil {
			t.Fatalf("the Resource %q contains a `name` field with a default value of `default` - this Resource should be exposed as part of the parent Resource it's located within: %+v", resourceName, err)
		}
	}
}

func schemaContainsANameFieldWithADefaultValueOfDefault(input map[string]*schema.Schema, fieldsToBeAddressed map[string]struct{}) error {
	// intentionally sorting these so the output is consistent
	fieldNames := make([]string, 0)
	for fieldName := range input {
		fieldNames = append(fieldNames, fieldName)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		key := strings.ToLower(fieldName)
		field := input[fieldName]

		// @tombuildsstuff: we have some Resources which will need to be addressed in the next major version (v4.0)
		// if this field name matches one we're intentionally ignoring, let's ignore it for now
		if _, shouldIgnore := fieldsToBeAddressed[key]; shouldIgnore {
			continue
		}

		if strings.EqualFold(key, "name") {
			var defaultValue any
			if field.Default != nil {
				defaultValue = field.Default
			}
			if v, ok := defaultValue.(string); ok {
				if strings.EqualFold(v, "default") {
					return fmt.Errorf("field %q is a `name` field which contains a default value of `default`", fieldName)
				}
			}
		}

		if field.Type == pluginsdk.TypeList && field.Elem != nil {
			if val, ok := field.Elem.(*pluginsdk.Resource); ok && val.Schema != nil {
				if err := schemaContainsANameFieldWithADefaultValueOfDefault(val.Schema, fieldsToBeAddressed); err != nil {
					return fmt.Errorf("the field %q is a List: %+v", fieldName, err)
				}
			}
		}

		if field.Type == pluginsdk.TypeSet && field.Elem != nil {
			if val, ok := field.Elem.(*pluginsdk.Resource); ok && val.Schema != nil {
				if err := schemaContainsANameFieldWithADefaultValueOfDefault(val.Schema, fieldsToBeAddressed); err != nil {
					return fmt.Errorf("the field %q is a Set: %+v", fieldName, err)
				}
			}
		}
	}

	return nil
}

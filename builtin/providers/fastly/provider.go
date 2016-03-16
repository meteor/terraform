package fastly

import (
        "github.com/hashicorp/terraform/helper/schema"
        "github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
        return &schema.Provider{
                Schema: map[string]*schema.Schema{
                        "api_key": &schema.Schema{
                                Type:        schema.TypeString,
                                Optional:    true,
                                Default:     "",
                                Description: "Fastly API Key from https://app.fastly.com/#account",
                        },
                },
                ResourcesMap: map[string]*schema.Resource{
                        "fastly_service_v1": resourceServiceV1(),
                },
        }
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
        config := Config{}
        return config.Client()
}

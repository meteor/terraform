package fastly

import (
        "github.com/hashicorp/terraform/helper/resource"
        "github.com/hashicorp/terraform/helper/schema"
)

func resourceServiceV1() *schema.Resource {
        return &schema.Resource{
                Create: resourceServiceV1Create,
                Read:   resourceServiceV1Read,
                Update: resourceServiceV1Update,
                Delete: resourceServiceV1Delete,

                Schema: map[string]*schema.Schema{
                        "name": &schema.Schema{
                                Type:     schema.TypeString,
                                Required: true,
                        },
                },
        }
}

func resourceServiceV1Create(d *schema.ResourceData, m interface{}) error {
        d.SetId(resource.UniqueId())
        return nil
}

func resourceServiceV1Read(d *schema.ResourceData, m interface{}) error {
        return nil
}

func resourceServiceV1Update(d *schema.ResourceData, m interface{}) error {
        return nil
}

func resourceServiceV1Delete(d *schema.ResourceData, m interface{}) error {
        return nil
}

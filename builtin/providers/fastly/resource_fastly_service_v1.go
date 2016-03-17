package fastly

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	gofastly "github.com/sethvargo/go-fastly"
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
			"ActiveVersion": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceServiceV1Create(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*FastlyClient).conn

	s, err := conn.CreateService(&gofastly.CreateServiceInput{
		Name:    d.Get("name").(string),
		Comment: "Created by Terraform",
	})

	if err != nil {
		return err
	}

	d.SetId(s.ID)

	return resourceServiceV1Read(d, meta)
}

func resourceServiceV1Read(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*FastlyClient).conn

	s, err := conn.GetService(&gofastly.GetServiceInput{
		ID: d.Id(),
	})

	if err != nil {
		return err
	}

	log.Printf("\n---\nService Details: %#v\n---\n", s)

	d.Set("name", s.Name)
	d.Set("ActiveVersion", s.ActiveVersion)

	return nil
}

func resourceServiceV1Update(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("name") {
		conn := meta.(*FastlyClient).conn
		_, err := conn.UpdateService(&gofastly.UpdateServiceInput{
			ID:   d.Id(),
			Name: d.Get("name").(string),
		})

		if err != nil {
			return err
		}
	}
	return resourceServiceV1Read(d, meta)
}

func resourceServiceV1Delete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*FastlyClient).conn

	err := conn.DeleteService(&gofastly.DeleteServiceInput{
		ID: d.Id(),
	})

	if err != nil {
		return err
	}

	l, err := conn.ListServices(&gofastly.ListServicesInput{})
	if err != nil {
		return fmt.Errorf("[WARN] Error listing servcies when deleting Fastly Service (%s): %s", d.Id(), err)
	}

	for _, s := range l {
		if s.ID == d.Id() {
			// service still found
			return fmt.Errorf("[WARN] Tried deleting Service (%s), but was still found", d.Id())
		}
	}
	d.SetId("")
	return nil
}

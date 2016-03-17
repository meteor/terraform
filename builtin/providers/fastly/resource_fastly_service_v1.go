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

			"active_version": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},

			"domain": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"comment": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"backend": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"address": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceServiceV1Create(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*FastlyClient).conn

	log.Printf("\n---\nDEBUG id: %s\n---\n", d.Id())
	// var latestVersion string
	// var service *gofastly.Service
	// var createVersion bool
	// if d.Id() == "" {
	// Create the service
	var err error
	service, err := conn.CreateService(&gofastly.CreateServiceInput{
		Name:    d.Get("name").(string),
		Comment: "Created by Terraform",
	})

	if err != nil {
		return err
	}

	d.SetId(service.ID)
	// Since this is a new creation, there will be an inactive version 1 waiting
	d.Set("active_version", "1")
	latestVersion := "1"

	log.Printf("\n---\nDEBUG Service in Create: %#v\n---\n", service)
	log.Printf("\n---\nDEBUG Lastest Version in Create: %#v\n---\n", latestVersion)

	return resourceServiceV1Update(d, meta)
}

func resourceServiceV1Update(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*FastlyClient).conn
	service, err := conn.GetService(&gofastly.GetServiceInput{
		ID: d.Id(),
	})

	if err != nil {
		return err
	}

	var needsChange bool
	if d.HasChange("domain") {
		log.Printf("\n------ has domainchange")
		needsChange = true
	} else {
		log.Printf("\n-00----- no domain change")
	}

	if d.HasChange("backend") {
		log.Printf("\n------ has backend change")
		needsChange = true
	} else {
		log.Printf("\n-00----- no backend change")
	}

	// if domains or backends have changed, then we create a new version and
	// post the updates
	latestVersion := "1"
	if attr, ok := d.GetOk("active_version"); ok {
		latestVersion = attr.(string)
	}

	if needsChange {
		log.Printf("\n00000\n\n\tshould change\n\n000000\n")
		log.Printf("\n---\nDEBUG Service in needs change: %#v\n---\n", service)
		log.Printf("\n---\nDEBUG Lastest Version in needs change: %#v\n---\n", latestVersion)
		if latestVersion != "1" {
			log.Printf("\n\t---- creating version\n---\n")
			newVersion, err := conn.CloneVersion(&gofastly.CloneVersionInput{
				Service: d.Id(),
				Version: latestVersion,
			})
			if err != nil {
				return err
			}
			latestVersion = fmt.Sprintf("%d", newVersion.Number)
		} else {
			log.Printf("\n\t---- not creating version, using %s\n---\n", latestVersion)
		}

		// find differences in domains
		od, nd := d.GetChange("domain")
		if od == nil {
			od = new(schema.Set)
		}
		if nd == nil {
			nd = new(schema.Set)
		}

		ods := od.(*schema.Set)
		nds := nd.(*schema.Set)
		log.Printf("\n\t### old domains: %#v\n", ods)
		log.Printf("\n\t### new domains: %#v\n", nds)

		// delete removed domains
		log.Printf("--- ods dif ns : %#v\n", ods.Difference(nds).List())
		log.Printf("--- nds dif os : %#v\n", nds.Difference(ods).List())

		// PUT new domains
		// var dA []map[string]interface{}
		for _, dRaw := range nds.Difference(ods).List() {
			df := dRaw.(map[string]interface{})
			log.Printf("\n\t--- domain to add: %s\n", df["name"].(string))
			opts := gofastly.CreateDomainInput{
				Service: d.Id(),
				Version: latestVersion,
				Name:    df["name"].(string),
			}
			if v, ok := df["comment"]; ok {
				opts.Comment = v.(string)
			}

			_, err := conn.CreateDomain(&opts)
			if err != nil {
				return err
			}
		}

		// find difference in backends
		ob, nb := d.GetChange("backend")
		if ob == nil {
			ob = new(schema.Set)
		}
		if nb == nil {
			nb = new(schema.Set)
		}

		obs := ob.(*schema.Set)
		nbs := nb.(*schema.Set)
		log.Printf("\n\t### old domains: %#v\n", obs)
		log.Printf("\n\t### new domains: %#v\n", nbs)

		// delete removed backends
		log.Printf("--- obs dif nbs : %#v\n", obs.Difference(nbs).List())
		log.Printf("--- nbs dif obs : %#v\n", nbs.Difference(obs).List())

		// PUT new domains
		// var dA []map[string]interface{}
		for _, dRaw := range nbs.Difference(obs).List() {
			df := dRaw.(map[string]interface{})
			log.Printf("\n\t--- backend to add: %s\n", df["name"].(string))
			opts := gofastly.CreateBackendInput{
				Service: d.Id(),
				Version: latestVersion,
				Name:    df["name"].(string),
				Address: df["address"].(string),
			}

			_, err := conn.CreateBackend(&opts)
			if err != nil {
				return err
			}
		}

		// if err != nil {
		// 	return err
		// }

		// validateversion
		valid, msg, err := conn.ValidateVersion(&gofastly.ValidateVersionInput{
			Service: d.Id(),
			Version: latestVersion,
		})

		if err != nil {
			return fmt.Errorf("[ERR] Error checking validation: %s", err)
		}
		if !valid {
			return fmt.Errorf("[ERR] Invalid configuration: %s", msg)
		}

		_, err = conn.ActivateVersion(&gofastly.ActivateVersionInput{
			Service: d.Id(),
			Version: latestVersion,
		})
		if err != nil {
			return fmt.Errorf("[ERR] Error activating version (%s): %s", latestVersion, err)
		}
		d.Set("active_version", latestVersion)
	} else { // end needsChange
		// Debugging
		log.Printf("\n--------- no changes needed------")
	}

	log.Printf("\n---\nDEBUG Service: %#v\n---\n", service)
	log.Printf("\n---\nDEBUG Lastest Version: %#v\n---\n", latestVersion)

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
	log.Printf("\n---\nService versions: %d\n---\n", s.ActiveVersion)

	if s.ActiveVersion != 0 {
		// get latest version info

		var version *gofastly.Version
		for _, v := range s.Versions {
			// the versions from the /service endpoint are returned as strings :/
			// TODO: Update go-fastly for this
			// vN, err := strconv.ParseUint(v.Number, 10, 32)
			// if err != nil {
			// 	return fmt.Errorf("[ERR] Error converting the version number: %s", err)
			// }
			if v.Number == fmt.Sprintf("%d", s.ActiveVersion) {
				log.Printf("\n--\nFound version: %#v\n", v)
				*version = *v
			}
		}

		if version == nil {
			log.Printf("[WARN] No active versions found yet")
		}
		// for each domain, write to state

		// for each backend, write to state
	} else {
		log.Printf("\n---\nDEBUG Active Version is 0\n")
	}

	// dA = append(dA, map[string]interface{}{"name": domain.Name, "comment": domain.Comment})
	// if err := d.Set("domain", dA); err != nil {
	// 	log.Printf("[ERR] Error setting domains: %s", err)
	// }
	d.Set("name", s.Name)
	d.Set("active_version", s.ActiveVersion)

	return nil
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

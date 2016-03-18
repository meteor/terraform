package fastly

import (
	"fmt"
	"log"
	"time"

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
				Type:     schema.TypeString,
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

			"force_destroy": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
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
	// d.Set("active_version", "-1")
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

	// update settings/names
	// No new verions is required for this
	if d.HasChange("name") {
		_, err := conn.UpdateService(&gofastly.UpdateServiceInput{
			ID:   d.Id(),
			Name: d.Get("name").(string),
		})
		if err != nil {
			return err
		}
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
	// latestVersion := "1"
	// if attr, ok := d.GetOk("active_version"); ok {
	// latestVersion = attr.(string)
	latestVersion := d.Get("active_version").(string)
	// }

	if needsChange {
		log.Printf("\n00000\n\n\tshould change\n\n000000\n")
		log.Printf("\n---\nDEBUG Service in needs change: %#v\n---\n", service)
		log.Printf("\n---\nDEBUG Lastest Version in needs change: %#v\n---\n", latestVersion)
		if latestVersion != "" {
			log.Printf("\n\t---- creating version\n---\n")
			newVersion, err := conn.CloneVersion(&gofastly.CloneVersionInput{
				Service: d.Id(),
				Version: latestVersion,
			})
			if err != nil {
				return err
			}
			latestVersion = newVersion.Number
			time.Sleep(10 * time.Second)
		} else {
			latestVersion = "1"
			log.Printf("\n\t---- not creating version, using %s\n---\n", latestVersion)
		}

		log.Printf("\n---\nDEBUG Lastest Version :::: %s\n---\n", latestVersion)
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
		// log.Printf("\n\t### old domains: %#v\n", ods)
		// log.Printf("\n\t### new domains: %#v\n", nds)

		// delete removed domains
		remove := ods.Difference(nds).List()
		add := nds.Difference(ods).List()
		log.Printf("--- ods dif ns : %#v\n", remove)
		log.Printf("--- nds dif os : %#v\n", add)

		for _, dRaw := range remove {
			df := dRaw.(map[string]interface{})
			log.Printf("\n\t--- domain to remove: %s\n", df["name"].(string))
			opts := gofastly.DeleteDomainInput{
				Service: d.Id(),
				Version: latestVersion,
				Name:    df["name"].(string),
			}

			log.Printf("[DEBUG] Fastly Domain Removal opts: %#v", opts)
			err := conn.DeleteDomain(&opts)
			if err != nil {
				return err
			}
		}

		// POST new Domains
		// Note: we don't utilize the PUT endpoint to update a Domain, we simply
		// destory it and create a new one. This is how Terraform works with nested
		// sub resources, we only get the full diff not a partial set item diff.
		// Because this is done on a new version of the configuration, this is
		// considered safe
		for _, dRaw := range add {
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

			log.Printf("[DEBUG] Fastly Domain Addition opts: %#v", opts)
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
		removeBackends := obs.Difference(nbs).List()
		addBackends := nbs.Difference(obs).List()
		log.Printf("--- obs dif nbs : %#v\n", removeBackends)
		log.Printf("--- nbs dif obs : %#v\n", addBackends)

		// DELETE old Backends
		for _, bRaw := range removeBackends {
			bf := bRaw.(map[string]interface{})
			log.Printf("\n\t--- backend to remove: %s\n", bf["name"].(string))
			opts := gofastly.DeleteBackendInput{
				Service: d.Id(),
				Version: latestVersion,
				Name:    bf["name"].(string),
			}

			log.Printf("[DEBUG] Fastly Backend Removal opts: %#v", opts)
			err := conn.DeleteBackend(&opts)
			if err != nil {
				return err
			}
		}

		// POST new Backends
		// Note: we don't utilize the PUT endpoint to update a Backend, we simply
		// destory it and create a new one. This is how Terraform works with nested
		// sub resources, we only get the full diff not a partial set item diff.
		// Because this is done on a new version of the configuration, this is
		// considered safe
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

		// validate version
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

	s, err := conn.GetServiceDetails(&gofastly.GetServiceInput{
		ID: d.Id(),
	})

	if err != nil {
		return err
	}

	log.Printf("\n---\nService Details: %#v\n---\n", s)
	log.Printf("\n---\nService versions: %d\n---\n", s.ActiveVersion)

	if s.ActiveVersion.Number != "" {
		// get latest version info
		// TODO: update go-fastly to support a ActiveVersion struct, which contains
		// domain and backend info in the response. Here we do 2 additional queries
		// to find out that info
		domainList, err := conn.ListDomains(&gofastly.ListDomainsInput{
			Service: d.Id(),
			Version: s.ActiveVersion.Number,
		})

		if err != nil {
			return err
		}

		// Refresh Domains
		var dl []map[string]interface{}
		for _, d := range domainList {
			dl = append(dl, map[string]interface{}{"name": d.Name, "comment": d.Comment})
		}

		if err := d.Set("domain", dl); err != nil {
			log.Printf("\n@@@@@@\nerror setting domains: %s\n@@@\n", err)
			log.Printf("[WARN] Error setting Domains for (%s): %s", d.Id(), err)
		}

		// Refresh Backends
		backendList, err := conn.ListBackends(&gofastly.ListBackendsInput{
			Service: d.Id(),
			Version: s.ActiveVersion.Number,
		})

		if err != nil {
			return err
		}
		var bl []map[string]interface{}
		for _, b := range backendList {
			bl = append(bl, map[string]interface{}{"name": b.Name, "address": b.Address})
		}

		if err := d.Set("backend", bl); err != nil {
			log.Printf("\n@@@@@@\nerror setting Backends: %s\n@@@\n", err)
			log.Printf("[WARN] Error setting Backends for (%s): %s", d.Id(), err)
		}
	} else {
		log.Printf("\n---\nDEBUG Active Version is 0\n")
	}

	d.Set("name", s.Name)
	d.Set("active_version", s.ActiveVersion.Number)

	return nil
}

func resourceServiceV1Delete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*FastlyClient).conn

	if d.Get("force_destroy").(bool) {
		s, err := conn.GetServiceDetails(&gofastly.GetServiceInput{
			ID: d.Id(),
		})

		if err != nil {
			return err
		}

		log.Printf("\n---\nService Details: %#v\n---\n", s)
		log.Printf("\n---\nService versions: %d\n---\n", s.ActiveVersion)

		if s.ActiveVersion.Number != "" {
			_, err := conn.DeactivateVersion(&gofastly.DeactivateVersionInput{
				Service: d.Id(),
				Version: s.ActiveVersion.Number,
			})
			if err != nil {
				return err
			}
		}
	}

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

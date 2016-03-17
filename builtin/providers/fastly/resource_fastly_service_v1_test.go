package fastly

import (
	"fmt"
	"log"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	gofastly "github.com/sethvargo/go-fastly"
)

func TestAccFastlyServiceV1_basic(t *testing.T) {
	var service gofastly.Service
	name := acctest.RandString(10)
	nameUpdate := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1Config(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1Attributes(&service, name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", name),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "ActiveVersion", "0"),
				),
			},

			resource.TestStep{
				Config: testAccServiceV1Config(nameUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1Attributes(&service, nameUpdate),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", nameUpdate),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "ActiveVersion", "0"),
				),
			},
		},
	})
}

func testAccCheckServiceV1Exists(n string, service *gofastly.Service) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Service ID is set")
		}

		conn := testAccProvider.Meta().(*FastlyClient).conn
		latest, err := conn.GetService(&gofastly.GetServiceInput{
			ID: rs.Primary.ID,
		})

		if err != nil {
			return err
		}

		*service = *latest

		return nil
	}
}

func testAccCheckFastlyServiceV1Attributes(service *gofastly.Service, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if service.Name != name {
			return fmt.Errorf("Bad name, expected (%s), got (%s)", name, service.Name)
		}

		return nil
	}
}

func testAccCheckServiceV1Destroy(s *terraform.State) error {
	// conn := testAccProvider.Meta().(*FastlyClient).conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fastly_service_v1" {
			continue
		}
		log.Printf("\n---\nDEBUG in delete for fastly service\n---\n")

		conn := testAccProvider.Meta().(*FastlyClient).conn

		l, err := conn.ListServices(&gofastly.ListServicesInput{})
		if err != nil {
			return fmt.Errorf("[WARN] Error listing servcies when deleting Fastly Service (%s): %s", rs.Primary.ID, err)
		}

		for _, s := range l {
			if s.ID == rs.Primary.ID {
				// service still found
				return fmt.Errorf("[WARN] Tried deleting Service (%s), but was still found", rs.Primary.ID)
			}
		}
	}
	return nil
}

func testAccServiceV1Config(name string) string {
	return fmt.Sprintf(`
resource "fastly_service_v1" "foo" {
        name = "%s"
}`, name)
}

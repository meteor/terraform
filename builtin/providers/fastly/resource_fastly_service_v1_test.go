package fastly

import (
	"fmt"
	"log"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	gofastly "github.com/sethvargo/go-fastly"
)

func TestAccFastlyServiceV1_basic(t *testing.T) {
	var service gofastly.ServiceDetail
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1Config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1Attributes(&service),
					resource.TestCheckResourceAttr(
						"fastly_service_v1.foo", "name", "testservice"),
				),
			},
		},
	})
}

func testAccCheckServiceV1Exists(n string, service *gofastly.ServiceDetail) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Service ID is set")
		}

		log.Printf("\n---\nDEBUG\n---\nFound: %s\n---\n", rs.Primary.ID)
		log.Printf("\n---\nWhat is meta: %#v\n\n---\n", testAccProvider.Meta())
		conn := testAccProvider.Meta().(*FastlyClient).conn

		latest, err := conn.GetService(&gofastly.GetServiceInput{
			ID: rs.Primary.ID,
		})

		if err != nil {
			log.Printf("\n---\nError: %s\n---\n", err)
			return err
		}

		*service = *latest

		return nil
	}
}

func testAccCheckFastlyServiceV1Attributes(service *gofastly.ServiceDetail) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if service.Name != "testservice" {
			return fmt.Errorf("Bad name: %s", service.Name)
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

		// Try to find the Service

		return nil
	}
	return fmt.Errorf("No Service found error")
}

const testAccServiceV1Config = `
resource "fastly_service_v1" "foo" {
        name = "testservice"
}
`
